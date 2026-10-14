package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/h4shell/paginegialle-cli/internal/model"
	"github.com/h4shell/paginegialle-cli/internal/output"
	"github.com/h4shell/paginegialle-cli/internal/paginegialle"
)

func init() {
	rootCmd.AddCommand(newGetCmd())
}

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <slug|url>",
		Short: "Fetch one enriched business profile",
		Long: `Fetch a single business profile and enrich it from the page's schema.org
JSON-LD: legal name, VAT/tax id, address, geo, phones, email, website, social
links, rating, opening hours and description.

<slug|url> is a profile slug (e.g. ristorante-al-mago-albairate-mi) or a full
https://www.paginegialle.it/... URL.

JSON shape (stdout, bare object, one document):
  {"id":"...","name":"...","categories":["..."],"legal_name":"...","vat":"...",
   "tax_id":"...","address":{...},"geo":{"lat":45.4,"lon":8.9},
   "phones":["..."],"email":"...","website":"...","social":["..."],
   "rating":{"value":5,"reviews":2},"opening_hours":["..."],
   "description":"...","url":"https://www.paginegialle.it/..."}

Exit codes: 0 success, 1 runtime failure, 2 usage error.`,
		Example: `  paginegialle get ristorante-al-mago-albairate-mi
  paginegialle get https://www.paginegialle.it/osteria-macondo-roma`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			svc := newService()
			profile, _, err := svc.Profile(args[0])
			if err != nil {
				if errors.Is(err, paginegialle.ErrNoProfile) {
					output.Report(jsonMode(), output.Err{Code: "not_found", Message: "no business profile found for " + args[0]})
					return nil
				}
				reportError(err)
				return nil
			}

			if !jsonMode() {
				printProfile(profile)
				return nil
			}
			if err := output.WriteJSON(profile); err != nil {
				output.Report(true, output.Err{Code: "internal", Message: err.Error()})
			}
			return nil
		},
	}
	return cmd
}

func printProfile(p *model.Profile) {
	printField("name", p.Name)
	printField("categories", strings.Join(p.Categories, "; "))
	printField("legal_name", p.LegalName)
	printField("vat", p.VAT)
	printField("tax_id", p.TaxID)
	if p.Address != nil {
		printField("address", strings.Join([]string{p.Address.Street, p.Address.PostalCode, p.Address.City, p.Address.Province}, " "))
	}
	printField("phones", strings.Join(p.Phones, ", "))
	printField("email", p.Email)
	printField("website", p.Website)
	printField("url", p.URL)
}

func printField(label, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(os.Stdout, "%-12s %s\n", label+":", value)
}
