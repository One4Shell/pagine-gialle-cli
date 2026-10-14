package paginegialle

import (
	"os"
	"testing"
)

type fakeFetcher struct {
	pages map[string][]byte
}

func (f *fakeFetcher) GetBody(rawurl string) (int, []byte, error) {
	if b, ok := f.pages[rawurl]; ok {
		return 200, b, nil
	}
	return 404, nil, &testFetchError{}
}

type testFetchError struct{}

func (*testFetchError) Error() string { return "not found" }

func TestSearchPath(t *testing.T) {
	cases := []struct {
		what, where string
		page        int
		want        string
	}{
		{"idraulici", "roma", 1, "/ricerca/idraulici/roma"},
		{"idraulici", "roma", 3, "/ricerca/idraulici/roma/p-3"},
		{"centro estetico", "milano", 1, "/ricerca/centro%20estetico/milano"},
	}
	for _, tc := range cases {
		if got := SearchPath(tc.what, tc.where, tc.page); got != tc.want {
			t.Errorf("SearchPath(%q,%q,%d) = %q, want %q", tc.what, tc.where, tc.page, got, tc.want)
		}
	}
}

func TestResolveURL(t *testing.T) {
	base := "https://www.paginegialle.it"
	cases := map[string]string{
		"foo-bar":                             base + "/foo-bar",
		"/foo-bar":                            base + "/foo-bar",
		"https://www.paginegialle.it/foo-bar": "https://www.paginegialle.it/foo-bar",
		"http://x.test/y":                     "http://x.test/y",
	}
	for ref, want := range cases {
		if got := ResolveURL(base, ref); got != want {
			t.Errorf("ResolveURL(%q) = %q, want %q", ref, got, want)
		}
	}
}

func TestServiceSearchAndProfile(t *testing.T) {
	search, err := os.ReadFile("testdata/search_ristoranti_milano.html")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := os.ReadFile("testdata/profile_ristorante_al_mago.html")
	if err != nil {
		t.Fatal(err)
	}
	base := "https://www.paginegialle.it"
	f := &fakeFetcher{pages: map[string][]byte{
		"/ricerca/ristoranti/milano":              search,
		base + "/ristorante-al-mago-albairate-mi": profile,
	}}
	svc := NewService(f, base)

	res, err := svc.Search("ristoranti", "milano", 1)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Businesses) == 0 {
		t.Fatal("expected businesses")
	}

	p, resolved, err := svc.Profile("ristorante-al-mago-albairate-mi")
	if err != nil {
		t.Fatalf("Profile: %v", err)
	}
	if resolved != base+"/ristorante-al-mago-albairate-mi" {
		t.Errorf("resolved = %q", resolved)
	}
	if p.Email != "info@ristorantealmago.com" {
		t.Errorf("email = %q", p.Email)
	}
}
