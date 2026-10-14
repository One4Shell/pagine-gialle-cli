package paginegialle

import (
	"os"
	"testing"

	"github.com/h4shell/paginegialle-cli/internal/output"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func TestParseSearch(t *testing.T) {
	page, err := ParseSearch(readFixture(t, "search_ristoranti_milano.html"), 1)
	if err != nil {
		t.Fatalf("ParseSearch: %v", err)
	}
	if len(page.Businesses) == 0 {
		t.Fatal("expected businesses, got none")
	}
	first := page.Businesses[0]
	if first.Name != "Ristorante al Mago" {
		t.Errorf("name = %q", first.Name)
	}
	if first.ID != "ristorante-al-mago-albairate-mi" {
		t.Errorf("id = %q", first.ID)
	}
	if first.Address == nil || first.Address.City != "Albairate" || first.Address.Province != "MI" {
		t.Errorf("address = %+v", first.Address)
	}
	if len(first.Phones) == 0 || first.Phones[0] != "02 94960166" {
		t.Errorf("phones = %v", first.Phones)
	}
	if first.Rating == nil || first.Rating.Reviews == 0 {
		t.Errorf("rating = %+v", first.Rating)
	}
	if !page.HasMore || page.NextPage != 2 {
		t.Errorf("pagination hasMore=%v next=%d", page.HasMore, page.NextPage)
	}

	got, err := output.Marshal(first)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"id":"ristorante-al-mago-albairate-mi","name":"Ristorante al Mago","categories":["ristoranti e trattorie"],"address":{"street":"Frazione Castelletto, 27","postal_code":"20080","city":"Albairate","province":"MI"},"phones":["02 94960166","320 6832426"],"snippet":"Ad Albairate AL MAGO il luogo perfetto per serate indimenticabili, dove falconieri, giullari, musici e maghi animano l’ambiente rendendolo magico e suggestivo.","rating":{"value":3.9,"reviews":269},"url":"https://www.paginegialle.it/ristorante-al-mago-albairate-mi"}`
	if got != want {
		t.Errorf("golden mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func TestParseProfile(t *testing.T) {
	p, err := ParseProfile(readFixture(t, "profile_ristorante_al_mago.html"), "https://www.paginegialle.it/ristorante-al-mago-albairate-mi")
	if err != nil {
		t.Fatalf("ParseProfile: %v", err)
	}
	if p.Name != "Ristorante al Mago" {
		t.Errorf("name = %q", p.Name)
	}
	if p.VAT != "10539550151" {
		t.Errorf("vat = %q", p.VAT)
	}
	if p.Email != "info@ristorantealmago.com" {
		t.Errorf("email = %q", p.Email)
	}
	if p.Website != "https://www.almagoristorante.com" {
		t.Errorf("website = %q", p.Website)
	}
	if p.Address == nil || p.Address.City != "Albairate" || p.Address.Region != "Lombardia" {
		t.Errorf("address = %+v", p.Address)
	}
	if p.Geo == nil || p.Geo.Lat == 0 {
		t.Errorf("geo = %+v", p.Geo)
	}
	if p.Rating == nil || p.Rating.Value != 5 {
		t.Errorf("rating = %+v", p.Rating)
	}
}

func TestParseAddress(t *testing.T) {
	a := ParseAddress("Via dei Mille, 10 - 00185 Roma (RM)")
	if a == nil || a.Street != "Via dei Mille, 10" || a.PostalCode != "00185" || a.City != "Roma" || a.Province != "RM" {
		t.Errorf("address = %+v", a)
	}
}
