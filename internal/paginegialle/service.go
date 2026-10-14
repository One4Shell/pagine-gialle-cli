package paginegialle

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/h4shell/paginegialle-cli/internal/model"
)

// Fetcher is the minimal HTTP capability the service needs (satisfied by
// *client.Client, and trivially faked in tests).
type Fetcher interface {
	GetBody(rawurl string) (int, []byte, error)
}

// Service fetches and parses paginegialle.it pages.
type Service struct {
	fetcher Fetcher
	baseURL string
}

// NewService wires a Fetcher and the site base URL.
func NewService(f Fetcher, baseURL string) *Service {
	return &Service{fetcher: f, baseURL: baseURL}
}

// SearchPath builds the canonical search URL path for a query and page.
func SearchPath(what, where string, page int) string {
	p := "/ricerca/" + url.PathEscape(what) + "/" + url.PathEscape(where)
	if page > 1 {
		p += "/p-" + strconv.Itoa(page)
	}
	return p
}

// Search fetches one results page and parses it.
func (s *Service) Search(what, where string, page int) (*SearchPage, error) {
	_, body, err := s.fetcher.GetBody(SearchPath(what, where, page))
	if err != nil {
		return nil, err
	}
	res, err := ParseSearch(body, page)
	if err != nil {
		return nil, fmt.Errorf("parse search results: %w", err)
	}
	return res, nil
}

// ResolveURL turns a slug, path or absolute URL into a full profile URL.
func ResolveURL(baseURL, ref string) string {
	ref = strings.TrimSpace(ref)
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref
	}
	base := strings.TrimRight(baseURL, "/")
	if strings.HasPrefix(ref, "/") {
		return base + ref
	}
	return base + "/" + ref
}

// Profile fetches and parses a single business profile.
func (s *Service) Profile(ref string) (*model.Profile, string, error) {
	target := ResolveURL(s.baseURL, ref)
	_, body, err := s.fetcher.GetBody(target)
	if err != nil {
		return nil, target, err
	}
	p, err := ParseProfile(body, target)
	if err != nil {
		return nil, target, err
	}
	return p, target, nil
}
