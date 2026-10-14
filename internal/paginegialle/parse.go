// Package paginegialle parses the server-rendered HTML of paginegialle.it into
// the stable model shapes consumed by the CLI.
package paginegialle

import (
	"bytes"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/h4shell/paginegialle-cli/internal/model"
)

// SearchPage is the parsed result of one search-results page.
type SearchPage struct {
	Businesses []model.Business
	HasMore    bool
	NextPage   int
}

var (
	ratingClassRe = regexp.MustCompile(`rating-stars--(\d+)`)
	digitsRe      = regexp.MustCompile(`\d+`)
	capRe         = regexp.MustCompile(`\b(\d{5})\b`)
	provRe        = regexp.MustCompile(`\(([A-Za-z]{2})\)`)
	pageRe        = regexp.MustCompile(`/p-(\d+)`)
	tagRe         = regexp.MustCompile(`<[^>]+>`)
	spaceRe       = regexp.MustCompile(`\s+`)
)

// ParseSearch parses a search-results HTML page.
func ParseSearch(body []byte, currentPage int) (*SearchPage, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	page := &SearchPage{}
	doc.Find(".search-itm").Each(func(_ int, sel *goquery.Selection) {
		b, ok := parseSearchItem(sel)
		if ok {
			page.Businesses = append(page.Businesses, b)
		}
	})

	if v, ok := doc.Find("[data-pageurl]").First().Attr("data-pageurl"); ok {
		if m := pageRe.FindStringSubmatch(v); m != nil {
			page.HasMore = true
			if n, err := strconv.Atoi(m[1]); err == nil {
				page.NextPage = n
			}
		}
	}
	if page.NextPage == 0 && page.HasMore {
		page.NextPage = currentPage + 1
	}
	return page, nil
}

func parseSearchItem(sel *goquery.Selection) (model.Business, bool) {
	name := cleanText(sel.Find(".search-itm__rag").First().Text())
	if name == "" {
		return model.Business{}, false
	}

	var b model.Business
	b.Name = name

	if href, ok := sel.Find(".search-itm__content a[href]").First().Attr("href"); ok {
		b.URL = strings.TrimSpace(href)
		b.ID = slugFromURL(b.URL)
	}
	if b.ID == "" {
		if v, ok := sel.Attr("data-cd_id_sede"); ok {
			b.ID = v
		}
	}

	if c := cleanText(sel.Find(".search-itm__category").First().Text()); c != "" {
		b.Categories = []string{c}
	}
	if a := cleanText(sel.Find(".search-itm__adr").First().Text()); a != "" {
		b.Address = ParseAddress(a)
	}
	sel.Find(".search-itm__phone-item").Each(func(_ int, p *goquery.Selection) {
		if v := cleanText(p.Text()); v != "" {
			b.Phones = append(b.Phones, v)
		}
	})
	b.Phones = model.DedupeStrings(b.Phones)
	if s := cleanText(sel.Find(".search-itm__dsc").First().Text()); s != "" {
		b.Snippet = s
	}
	if r := parseRating(sel); r != nil {
		b.Rating = r
	}
	return b, true
}

func parseRating(sel *goquery.Selection) *model.Rating {
	stars := sel.Find(".search-itm__rating .rating-stars").First()
	if stars.Length() == 0 {
		return nil
	}
	class, _ := stars.Attr("class")
	m := ratingClassRe.FindStringSubmatch(class)
	if m == nil {
		return nil
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return nil
	}
	r := &model.Rating{Value: round1(float64(n) / 20.0)}
	if cnt := sel.Find(".rating-stars__count").First().Text(); cnt != "" {
		if d := digitsRe.FindString(cnt); d != "" {
			if v, err := strconv.Atoi(d); err == nil {
				r.Reviews = v
			}
		}
	}
	return r
}

// ParseProfile parses a business profile page (JSON-LD with HTML fallbacks).
func ParseProfile(body []byte, pageURL string) (*model.Profile, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var ld *jsonld
	doc.Find("script").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		t, _ := s.Attr("type")
		if !strings.EqualFold(strings.TrimSpace(t), "application/ld+json") {
			return true
		}
		if cand := bestJSONLD([]byte(s.Text())); cand != nil {
			ld = cand
			return false
		}
		return true
	})
	if ld == nil {
		return nil, ErrNoProfile
	}

	p := &model.Profile{
		ID:        slugFromURL(pageURL),
		Name:      cleanText(ld.Name),
		LegalName: cleanText(ld.LegalName),
		VAT:       cleanText(ld.VatID),
		TaxID:     cleanText(ld.TaxID),
		URL:       pageURL,
	}
	if p.Name == "" {
		p.Name = cleanText(doc.Find("h1").First().Text())
	}
	p.Description = cleanHTML(ld.Description)

	if ld.Address != nil {
		p.Address = &model.Address{
			Street:     cleanText(ld.Address.Street),
			PostalCode: cleanText(ld.Address.PostalCode),
			City:       cleanText(ld.Address.City),
			Region:     cleanText(ld.Address.Region),
		}
	}
	if ld.Geo != nil {
		lat, _ := strconv.ParseFloat(strings.TrimSpace(string(ld.Geo.Lat)), 64)
		lon, _ := strconv.ParseFloat(strings.TrimSpace(string(ld.Geo.Lon)), 64)
		if lat != 0 || lon != 0 {
			p.Geo = &model.Geo{Lat: lat, Lon: lon}
		}
	}

	var phones []string
	if ld.Telephone != "" {
		phones = append(phones, cleanText(ld.Telephone))
	}
	for _, cp := range ld.ContactPoint {
		if cp.Telephone != "" {
			phones = append(phones, cleanText(cp.Telephone))
		}
		if p.Email == "" && cp.Email != "" {
			p.Email = cleanText(cp.Email)
		}
	}
	p.Phones = model.DedupeStrings(phones)

	for _, u := range decodeStringList(ld.SameAs) {
		u = strings.TrimSpace(u)
		if u == "" || strings.Contains(u, "paginegialle.it") {
			continue
		}
		p.Social = append(p.Social, u)
	}
	p.Social = model.DedupeStrings(p.Social)
	if len(p.Social) > 0 {
		p.Website = p.Social[0]
	}

	if ld.AggregateRating != nil {
		v, _ := strconv.ParseFloat(strings.TrimSpace(string(ld.AggregateRating.Value)), 64)
		c, _ := strconv.Atoi(strings.TrimSpace(string(ld.AggregateRating.Count)))
		if v > 0 || c > 0 {
			p.Rating = &model.Rating{Value: round1(v), Reviews: c}
		}
	}
	for _, h := range ld.OpeningHours {
		if h = cleanText(h); h != "" {
			p.OpeningHours = append(p.OpeningHours, h)
		}
	}
	p.OpeningHours = model.DedupeStrings(p.OpeningHours)

	if p.Email == "" {
		p.Email = firstMailto(doc)
	}

	// Category from breadcrumb ("Aziende" is the root, drop it).
	var cats []string
	doc.Find(".breadcrumbs__link").Each(func(i int, s *goquery.Selection) {
		if t := cleanText(s.Text()); t != "" && !strings.EqualFold(t, "Aziende") {
			cats = append(cats, t)
		}
	})
	p.Categories = model.DedupeStrings(cats)

	return p, nil
}

func firstMailto(doc *goquery.Document) string {
	var email string
	doc.Find(`a[href^="mailto:"]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		href, _ := s.Attr("href")
		addr := strings.TrimSpace(strings.TrimPrefix(href, "mailto:"))
		if i := strings.IndexByte(addr, '?'); i >= 0 {
			addr = addr[:i]
		}
		if strings.Contains(addr, "@") {
			email = addr
			return false
		}
		return true
	})
	return email
}

// ParseAddress parses the Italian address format used in search results, e.g.
// "Frazione Castelletto, 27 - 20080 Albairate (MI)".
func ParseAddress(raw string) *model.Address {
	raw = cleanText(raw)
	if raw == "" {
		return nil
	}
	a := &model.Address{}
	street := raw
	if i := strings.LastIndex(raw, " - "); i >= 0 {
		street = raw[:i]
		right := raw[i+3:]
		if m := capRe.FindStringSubmatch(right); m != nil {
			a.PostalCode = m[1]
			right = strings.Replace(right, m[1], "", 1)
		}
		if m := provRe.FindStringSubmatch(right); m != nil {
			a.Province = strings.ToUpper(m[1])
			right = provRe.ReplaceAllString(right, "")
		}
		a.City = cleanText(strings.Trim(right, " ,"))
	}
	a.Street = cleanText(street)
	if a.City == "" && a.PostalCode == "" && a.Province == "" {
		a.Street = cleanText(raw)
	}
	if a.Street == "" && a.City == "" {
		return nil
	}
	return a
}

func slugFromURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Path == "" {
		return ""
	}
	return path.Base(strings.TrimRight(u.Path, "/"))
}

func cleanText(s string) string {
	return strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
}

func cleanHTML(s string) string {
	s = tagRe.ReplaceAllString(s, " ")
	return cleanText(s)
}

func round1(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}
