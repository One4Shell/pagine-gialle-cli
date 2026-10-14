package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/h4shell/paginegialle-cli/internal/model"
	"github.com/h4shell/paginegialle-cli/internal/output"
)

func init() {
	rootCmd.AddCommand(newSearchCmd())
}

func newSearchCmd() *cobra.Command {
	var (
		what     string
		where    string
		page     int
		maxPages int
		fields   string
	)

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search businesses by category/keyword and location",
		Long: `Search paginegialle.it listings by what (category or keyword) and where
(city or area). Returns name, category, address, phones, rating and profile URL
for each business.

JSON shape (stdout, one document):
  {"schema_version":1,
   "query":{"what":"idraulici","where":"roma","page":1},
   "items":[{"id":"...","name":"...","categories":["..."],
             "address":{"street":"...","postal_code":"...","city":"...","province":"RM"},
             "phones":["..."],"snippet":"...","rating":{"value":4.8,"reviews":37},
             "url":"https://www.paginegialle.it/..."}],
   "pagination":{"page":1,"per_page":25,"has_more":true,"next_page":2}}

Env vars: PAGINEGIALLE_BASE_URL, PAGINEGIALLE_CONTACT.
Exit codes: 0 success, 1 runtime failure, 2 usage error.`,
		Example: `  paginegialle search --what idraulici --where roma
  paginegialle search --what "centro estetico" --where milano --max-pages 3
  paginegialle search --what ristoranti --where torino --fields name,phones,url`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if what == "" || where == "" {
				return errors.New("both --what and --where are required")
			}
			if page < 1 {
				page = 1
			}
			if maxPages < 1 {
				maxPages = 1
			}

			svc := newService()
			items := make([]model.Business, 0)
			hasMore := false
			for i := 0; i < maxPages; i++ {
				res, err := svc.Search(what, where, page+i)
				if err != nil {
					reportError(err)
					return nil
				}
				items = append(items, res.Businesses...)
				hasMore = res.HasMore
				if !res.HasMore {
					break
				}
			}

			pagination := &output.Pagination{
				Page:    page,
				PerPage: 25,
				HasMore: hasMore,
			}
			if hasMore {
				pagination.NextPage = page + maxPages
			}

			if !jsonMode() {
				rows := make([][]string, 0, len(items))
				for _, b := range items {
					city := ""
					phone := ""
					if b.Address != nil {
						city = b.Address.City
					}
					if len(b.Phones) > 0 {
						phone = b.Phones[0]
					}
					rows = append(rows, []string{b.Name, joinOr(b.Categories, "; "), city, phone, b.URL})
				}
				table([]string{"NAME", "CATEGORY", "CITY", "PHONE", "URL"}, rows)
				return nil
			}

			env := output.Envelope{
				SchemaVersion: output.SchemaVersion,
				Query:         model.SearchQuery{What: what, Where: where, Page: page},
				Items:         projectFields(items, splitFields(fields)),
				Pagination:    pagination,
			}
			if err := output.WriteJSON(env); err != nil {
				output.Report(true, output.Err{Code: "internal", Message: err.Error()})
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&what, "what", "", "category or keyword to search for (required)")
	cmd.Flags().StringVar(&where, "where", "", "city or area to search in (required)")
	cmd.Flags().IntVar(&page, "page", 1, "results page to start from")
	cmd.Flags().IntVar(&maxPages, "max-pages", 1, "maximum number of pages to fetch")
	cmd.Flags().StringVar(&fields, "fields", "", "comma-separated keys to keep in each item")
	return cmd
}
