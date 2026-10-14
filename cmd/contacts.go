package cmd

import (
	"errors"
	"sync"

	"github.com/spf13/cobra"

	"github.com/h4shell/paginegialle-cli/internal/model"
	"github.com/h4shell/paginegialle-cli/internal/output"
	"github.com/h4shell/paginegialle-cli/internal/paginegialle"
)

const contactsPerPage = 25

func init() {
	rootCmd.AddCommand(newContactsCmd())
}

func newContactsCmd() *cobra.Command {
	var (
		what        string
		where       string
		max         int
		page        int
		enrich      bool
		concurrency int
	)

	cmd := &cobra.Command{
		Use:   "contacts",
		Short: "Build a B2B contact list (search + optional profile enrichment)",
		Long: `Search paginegialle.it and normalize the results into contact records for
B2B outreach. By default each business profile is fetched to enrich the record
with email, website and VAT; pass --enrich=false for a fast, search-only run.

JSON shape (stdout, one document):
  {"schema_version":1,
   "query":{"what":"idraulici","where":"roma","page":1,"max":50},
   "items":[{"name":"...","categories":["..."],
             "address":{"street":"...","postal_code":"...","city":"...","province":"RM"},
             "phones":["..."],"email":"...","website":"...","vat":"...",
             "url":"https://www.paginegialle.it/..."}],
   "pagination":{"page":1,"per_page":25,"has_more":true,"next_page":3},
   "stats":{"requested":50,"enriched":50,"with_email":31,"with_website":44}}

Env vars: PAGINEGIALLE_BASE_URL, PAGINEGIALLE_CONTACT.
Exit codes: 0 success, 1 runtime failure, 2 usage error.`,
		Example: `  paginegialle contacts --what idraulici --where roma --max 50
  paginegialle contacts --what "studi legali" --where milano --max 20 --enrich=false`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if what == "" || where == "" {
				return errors.New("both --what and --where are required")
			}
			if page < 1 {
				page = 1
			}
			if max < 1 {
				max = contactsPerPage
			}
			if concurrency < 1 {
				concurrency = 1
			}

			svc := newService()

			// Collect businesses across as many pages as needed.
			businesses := make([]model.Business, 0, max)
			hasMore := false
			for p := page; len(businesses) < max; p++ {
				res, err := svc.Search(what, where, p)
				if err != nil {
					reportError(err)
					return nil
				}
				businesses = append(businesses, res.Businesses...)
				hasMore = res.HasMore
				if !res.HasMore {
					break
				}
			}
			if len(businesses) > max {
				businesses = businesses[:max]
			}

			contacts := make([]model.Contact, len(businesses))
			stats := model.ContactStats{Requested: len(businesses)}

			if enrich {
				stats.Enriched = enrichProfiles(svc, businesses, contacts, concurrency)
			} else {
				for i, b := range businesses {
					contacts[i] = contactFromBusiness(b)
				}
			}
			for _, c := range contacts {
				if c.Email != "" {
					stats.WithEmail++
				}
				if c.Website != "" {
					stats.WithWebsite++
				}
			}

			if !jsonMode() {
				rows := make([][]string, 0, len(contacts))
				for _, c := range contacts {
					city := ""
					if c.Address != nil {
						city = c.Address.City
					}
					rows = append(rows, []string{c.Name, city, joinOr(c.Phones, ", "), c.Email, c.Website})
				}
				table([]string{"NAME", "CITY", "PHONES", "EMAIL", "WEBSITE"}, rows)
				return nil
			}

			pagination := &output.Pagination{Page: page, PerPage: contactsPerPage, HasMore: hasMore}
			if hasMore {
				pagination.NextPage = page + (len(businesses)+contactsPerPage-1)/contactsPerPage
			}

			env := output.Envelope{
				SchemaVersion: output.SchemaVersion,
				Query:         model.ContactsQuery{What: what, Where: where, Page: page, Max: max},
				Items:         contacts,
				Pagination:    pagination,
				Stats:         stats,
			}
			if err := output.WriteJSON(env); err != nil {
				output.Report(true, output.Err{Code: "internal", Message: err.Error()})
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&what, "what", "", "category or keyword to search for (required)")
	cmd.Flags().StringVar(&where, "where", "", "city or area to search in (required)")
	cmd.Flags().IntVar(&max, "max", contactsPerPage, "maximum number of contacts to return")
	cmd.Flags().IntVar(&page, "page", 1, "results page to start from")
	cmd.Flags().BoolVar(&enrich, "enrich", true, "fetch each profile to add email/website/VAT")
	cmd.Flags().IntVar(&concurrency, "concurrency", 1, "parallel profile fetches when enriching")
	return cmd
}

// enrichProfiles fetches profiles concurrently, preserving input order, and
// returns the number of profiles successfully retrieved.
func enrichProfiles(svc *paginegialle.Service, businesses []model.Business, contacts []model.Contact, concurrency int) int {
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	enriched := 0
	for i := range businesses {
		contacts[i] = contactFromBusiness(businesses[i])
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			profile, _, err := svc.Profile(businesses[i].URL)
			if err != nil {
				return
			}
			applyProfile(&contacts[i], profile)
			mu.Lock()
			enriched++
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	return enriched
}

func contactFromBusiness(b model.Business) model.Contact {
	return model.Contact{
		Name:       b.Name,
		Categories: b.Categories,
		Address:    b.Address,
		Phones:     b.Phones,
		URL:        b.URL,
	}
}

func applyProfile(c *model.Contact, p *model.Profile) {
	c.Email = p.Email
	c.Website = p.Website
	c.VAT = p.VAT
	if c.Address == nil {
		c.Address = p.Address
	}
	c.Phones = model.DedupeStrings(append(append([]string{}, c.Phones...), p.Phones...))
	if len(c.Categories) == 0 {
		c.Categories = p.Categories
	}
}
