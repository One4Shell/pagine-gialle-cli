package paginegialle

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

// ErrNoProfile is returned when a page carries no parseable structured data.
var ErrNoProfile = errors.New("no structured business data found on page")

// flexString decodes a JSON field that may be a string or a number.
type flexString string

// UnmarshalJSON accepts strings, numbers and null.
func (f *flexString) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*f = ""
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexString(s)
		return nil
	}
	*f = flexString(string(b))
	return nil
}

// jsonld is the subset of schema.org LocalBusiness fields the site emits.
type jsonld struct {
	Type         json.RawMessage `json:"@type"`
	Name         string          `json:"name"`
	LegalName    string          `json:"legalName"`
	VatID        string          `json:"vatID"`
	TaxID        string          `json:"taxID"`
	Telephone    string          `json:"telephone"`
	Email        string          `json:"email"`
	Description  string          `json:"description"`
	SameAs       json.RawMessage `json:"sameAs"`
	OpeningHours []string        `json:"openingHours"`
	Geo          *struct {
		Lat flexString `json:"latitude"`
		Lon flexString `json:"longitude"`
	} `json:"geo"`
	Address *struct {
		Street     string `json:"streetAddress"`
		PostalCode string `json:"postalCode"`
		City       string `json:"addressLocality"`
		Region     string `json:"addressRegion"`
	} `json:"address"`
	ContactPoint []struct {
		Email     string `json:"email"`
		Telephone string `json:"telephone"`
	} `json:"contactPoint"`
	AggregateRating *struct {
		Value flexString `json:"ratingValue"`
		Count flexString `json:"reviewCount"`
	} `json:"aggregateRating"`
}

// bestJSONLD picks the most business-like object from a JSON-LD script block.
// The block may be a single object or an array of objects.
func bestJSONLD(raw []byte) *jsonld {
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return nil
	}

	var single jsonld
	if err := json.Unmarshal(raw, &single); err == nil {
		if isUseful(&single) {
			return &single
		}
	}

	var many []jsonld
	if err := json.Unmarshal(raw, &many); err == nil {
		var best *jsonld
		bestScore := 0
		for i := range many {
			if s := score(&many[i]); s > bestScore {
				bestScore = s
				best = &many[i]
			}
		}
		return best
	}
	return nil
}

func isUseful(l *jsonld) bool { return score(l) > 0 }

// score ranks candidates: a named legal/business entity with an address or a
// phone beats one that has only a name.
func score(l *jsonld) int {
	if l == nil || l.Name == "" {
		return 0
	}
	s := 1
	if l.Address != nil {
		s += 4
	}
	if l.Telephone != "" || len(l.ContactPoint) > 0 {
		s += 2
	}
	if l.VatID != "" || l.TaxID != "" {
		s++
	}
	if l.AggregateRating != nil {
		s++
	}
	return s
}

// decodeStringList accepts a JSON string or array of strings.
func decodeStringList(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []string{s}
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return list
	}
	return nil
}
