package client

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config configures the HTTP client used for scraping.
type Config struct {
	BaseURL  string
	Contact  string
	Timeout  time.Duration
	Retries  int
	Delay    time.Duration
	CacheTTL time.Duration
	NoCache  bool
}

// FetchError is a typed failure the output layer maps to the contract's
// code/http_status fields.
type FetchError struct {
	Code       string
	Message    string
	HTTPStatus int
	RetryAfter int
}

func (e *FetchError) Error() string {
	if e.HTTPStatus > 0 {
		return fmt.Sprintf("%s (http %d): %s", e.Code, e.HTTPStatus, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Client is a polite, retrying, caching HTTP client for HTML pages.
type Client struct {
	cfg      Config
	http     *http.Client
	cacheDir string
	lastReq  time.Time
}

// New returns a Client with sane defaults applied. It performs no network I/O.
func New(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.Retries <= 0 {
		cfg.Retries = 3
	}
	c := &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
	c.http.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// Follow the site's own trailing-slash / canonical redirects.
		return nil
	}
	if dir, err := os.UserCacheDir(); err == nil {
		c.cacheDir = filepath.Join(dir, "paginegialle")
	}
	if cfg.NoCache {
		c.cacheDir = ""
	}
	return c
}

// cachePath returns the cache file for a URL, or "" when caching is disabled.
func (c *Client) cachePath(rawurl string) string {
	if c.cacheDir == "" {
		return ""
	}
	sum := sha1.Sum([]byte(rawurl))
	return filepath.Join(c.cacheDir, hex.EncodeToString(sum[:])+".html")
}

func (c *Client) readCache(rawurl string) ([]byte, bool) {
	p := c.cachePath(rawurl)
	if p == "" {
		return nil, false
	}
	fi, err := os.Stat(p)
	if err != nil {
		return nil, false
	}
	if c.cfg.CacheTTL > 0 && time.Since(fi.ModTime()) > c.cfg.CacheTTL {
		return nil, false
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	return b, true
}

func (c *Client) writeCache(rawurl string, body []byte) {
	p := c.cachePath(rawurl)
	if p == "" {
		return
	}
	if err := os.MkdirAll(c.cacheDir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(p, body, 0o644)
}

// sleep throttles requests so consecutive calls are at least Delay apart.
func (c *Client) throttle() {
	if c.cfg.Delay <= 0 {
		return
	}
	if !c.lastReq.IsZero() {
		if wait := c.cfg.Delay - time.Since(c.lastReq); wait > 0 {
			time.Sleep(wait)
		}
	}
	c.lastReq = time.Now()
}

// GetBody fetches rawurl (absolute or path-relative to BaseURL) with retries,
// throttling and caching. On a non-2xx response it returns a *FetchError.
func (c *Client) GetBody(rawurl string) (int, []byte, error) {
	if !strings.HasPrefix(rawurl, "http://") && !strings.HasPrefix(rawurl, "https://") {
		rawurl = strings.TrimRight(c.cfg.BaseURL, "/") + "/" + strings.TrimLeft(rawurl, "/")
	}

	if body, ok := c.readCache(rawurl); ok {
		return http.StatusOK, body, nil
	}

	var (
		lastErr error
		status  int
	)
	for attempt := 1; attempt <= c.cfg.Retries; attempt++ {
		c.throttle()
		req, err := http.NewRequest(http.MethodGet, rawurl, nil)
		if err != nil {
			return 0, nil, &FetchError{Code: "invalid_argument", Message: err.Error()}
		}
		req.Header.Set("User-Agent", c.userAgent())
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
		req.Header.Set("Accept-Language", "it-IT,it;q=0.9,en;q=0.8")

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = &FetchError{Code: "network", Message: err.Error()}
			if attempt < c.cfg.Retries {
				backoff(0, attempt)
				continue
			}
			return 0, nil, lastErr
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		status = resp.StatusCode
		if readErr != nil {
			lastErr = &FetchError{Code: "network", Message: readErr.Error()}
			if attempt < c.cfg.Retries {
				backoff(0, attempt)
				continue
			}
			return status, nil, lastErr
		}

		if status == http.StatusOK {
			c.writeCache(rawurl, body)
			return status, body, nil
		}

		retryable := status == http.StatusTooManyRequests ||
			status == http.StatusBadGateway ||
			status == http.StatusServiceUnavailable ||
			status == http.StatusGatewayTimeout ||
			(status >= 500 && status < 600)
		if retryable && attempt < c.cfg.Retries {
			backoff(retryAfter(resp.Header), attempt)
			continue
		}

		return status, body, classify(status, resp)
	}
	if lastErr == nil {
		lastErr = &FetchError{Code: "network", Message: "request failed"}
	}
	return status, nil, lastErr
}

func (c *Client) userAgent() string {
	if c.cfg.Contact != "" {
		return fmt.Sprintf("paginegialle-cli/0.1 (+%s)", c.cfg.Contact)
	}
	return "paginegialle-cli/0.1 (+https://github.com/h4shell/paginegialle-cli)"
}

func classify(status int, resp *http.Response) error {
	code := "api_error"
	switch status {
	case http.StatusBadRequest:
		code = "invalid_argument"
	case http.StatusUnauthorized, http.StatusForbidden:
		code = "auth_failed"
	case http.StatusNotFound:
		code = "not_found"
	case http.StatusTooManyRequests:
		code = "rate_limited"
	}
	fe := &FetchError{
		Code:       code,
		Message:    fmt.Sprintf("%s returned HTTP %d", resp.Request.URL.Host, status),
		HTTPStatus: status,
	}
	if ra := retryAfter(resp.Header); ra > 0 {
		fe.RetryAfter = int(ra.Seconds())
	}
	return fe
}

func retryAfter(h http.Header) time.Duration {
	raw := h.Get("Retry-After")
	if raw == "" {
		return 0
	}
	if secs, err := strconv.Atoi(raw); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(raw); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

func backoff(after time.Duration, attempt int) {
	if after > 0 {
		time.Sleep(after)
		return
	}
	d := time.Duration(attempt) * 300 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	time.Sleep(d)
}

// MapError converts any error into an *FetchError, defaulting to internal.
func MapError(err error) *FetchError {
	if err == nil {
		return nil
	}
	if fe, ok := err.(*FetchError); ok {
		return fe
	}
	return &FetchError{Code: "internal", Message: err.Error()}
}
