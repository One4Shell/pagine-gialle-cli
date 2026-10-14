// Package model defines the stable, ordered data shapes emitted by the CLI.
// Field order in these structs is the deterministic JSON key order guaranteed
// by the output contract.
package model

// Address is a parsed Italian postal address.
type Address struct {
	Street     string `json:"street,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	City       string `json:"city,omitempty"`
	Province   string `json:"province,omitempty"`
	Region     string `json:"region,omitempty"`
}

// Rating is a business rating aggregate.
type Rating struct {
	Value   float64 `json:"value"`
	Reviews int     `json:"reviews"`
}

// Geo is a WGS84 coordinate pair.
type Geo struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Business is one search-result listing.
type Business struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Categories []string `json:"categories,omitempty"`
	Address    *Address `json:"address,omitempty"`
	Phones     []string `json:"phones,omitempty"`
	Snippet    string   `json:"snippet,omitempty"`
	Rating     *Rating  `json:"rating,omitempty"`
	URL        string   `json:"url"`
}

// Profile is a fully enriched business profile (get command).
type Profile struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Categories   []string `json:"categories,omitempty"`
	LegalName    string   `json:"legal_name,omitempty"`
	VAT          string   `json:"vat,omitempty"`
	TaxID        string   `json:"tax_id,omitempty"`
	Address      *Address `json:"address,omitempty"`
	Geo          *Geo     `json:"geo,omitempty"`
	Phones       []string `json:"phones,omitempty"`
	Email        string   `json:"email,omitempty"`
	Website      string   `json:"website,omitempty"`
	Social       []string `json:"social,omitempty"`
	Rating       *Rating  `json:"rating,omitempty"`
	OpeningHours []string `json:"opening_hours,omitempty"`
	Description  string   `json:"description,omitempty"`
	URL          string   `json:"url"`
}

// Contact is a normalized B2B contact record.
type Contact struct {
	Name       string   `json:"name"`
	Categories []string `json:"categories,omitempty"`
	Address    *Address `json:"address,omitempty"`
	Phones     []string `json:"phones,omitempty"`
	Email      string   `json:"email,omitempty"`
	Website    string   `json:"website,omitempty"`
	VAT        string   `json:"vat,omitempty"`
	URL        string   `json:"url"`
}

// SearchQuery echoes the inputs of a search for machine consumers.
type SearchQuery struct {
	What  string `json:"what"`
	Where string `json:"where"`
	Page  int    `json:"page"`
}

// ContactsQuery echoes the inputs of a contacts run.
type ContactsQuery struct {
	What  string `json:"what"`
	Where string `json:"where"`
	Page  int    `json:"page"`
	Max   int    `json:"max"`
}

// ContactStats summarizes a contacts run.
type ContactStats struct {
	Requested   int `json:"requested"`
	Enriched    int `json:"enriched"`
	WithEmail   int `json:"with_email"`
	WithWebsite int `json:"with_website"`
}

// DedupeStrings returns values with empties removed and order-preserving
// de-duplication applied. A nil input yields nil so JSON omits the field.
func DedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
