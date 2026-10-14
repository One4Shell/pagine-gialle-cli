package client

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(baseURL string, retries int) *Client {
	return New(Config{BaseURL: baseURL, Timeout: 5 * time.Second, Retries: retries, Delay: 0, NoCache: true})
}

func TestGetBodySuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept-Language"); got == "" {
			t.Errorf("missing Accept-Language")
		}
		_, _ = w.Write([]byte("<html>ok</html>"))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, 3)
	status, body, err := c.GetBody("/some/page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 200 || string(body) != "<html>ok</html>" {
		t.Fatalf("status=%d body=%q", status, body)
	}
}

func TestGetBodyRetriesOn5xx(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("second"))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, 3)
	_, body, err := c.GetBody("/x")
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if string(body) != "second" {
		t.Fatalf("body = %q", body)
	}
}

func TestGetBodyErrorMapping(t *testing.T) {
	cases := []struct {
		status int
		code   string
	}{
		{http.StatusNotFound, "not_found"},
		{http.StatusForbidden, "auth_failed"},
		{http.StatusBadRequest, "invalid_argument"},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
		}))
		c := newTestClient(srv.URL, 1)
		_, _, err := c.GetBody("/x")
		fe, ok := err.(*FetchError)
		if !ok {
			srv.Close()
			t.Fatalf("status %d: expected *FetchError, got %T", tc.status, err)
		}
		if fe.Code != tc.code || fe.HTTPStatus != tc.status {
			t.Errorf("status %d: got code=%q http=%d", tc.status, fe.Code, fe.HTTPStatus)
		}
		srv.Close()
	}
}

func TestAbsoluteURLNotPrefixed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/page" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := newTestClient("http://example.invalid", 1)
	if _, _, err := c.GetBody(srv.URL + "/page"); err != nil {
		t.Fatalf("absolute URL should be used as-is: %v", err)
	}
}
