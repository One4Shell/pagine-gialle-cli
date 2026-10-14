package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/h4shell/paginegialle-cli/internal/client"
	"github.com/h4shell/paginegialle-cli/internal/config"
	"github.com/h4shell/paginegialle-cli/internal/paginegialle"
	"github.com/h4shell/paginegialle-cli/internal/version"
)

var (
	cfg config.Config

	flagBaseURL  string
	flagContact  string
	flagTimeout  time.Duration
	flagRetries  int
	flagDelay    time.Duration
	flagCacheTTL time.Duration
	flagNoCache  bool
	flagJSON     bool
	flagHuman    bool
)

var rootCmd = &cobra.Command{
	Use:   "paginegialle",
	Short: "B2B contact research on paginegialle.it with machine-readable JSON output",
	Long: `paginegialle is a command-line client for the Italian business directory
paginegialle.it. It searches and enriches company profiles into stable,
machine-readable records for B2B contact research.

Data commands emit exactly one compact JSON document on stdout by default and
nothing else. Pass --human for readable tables. Errors are structured JSON on
stderr with semantic exit codes: 0 success, 1 runtime failure, 2 usage error.

Note: reads paginegialle.it over HTTP. Use responsibly: the CLI throttles
requests (--delay), caches responses, and sends an honest User-Agent.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version.Version,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&flagBaseURL, "base-url", "", "site base URL (env: PAGINEGIALLE_BASE_URL)")
	pf.StringVar(&flagContact, "contact", "", "contact URL/email for the User-Agent (env: PAGINEGIALLE_CONTACT)")
	pf.DurationVar(&flagTimeout, "timeout", 0, "per-request timeout, e.g. 30s (env: PAGINEGIALLE_TIMEOUT)")
	pf.IntVar(&flagRetries, "retries", 0, "max attempts per request on 429/5xx (env: PAGINEGIALLE_RETRIES)")
	pf.DurationVar(&flagDelay, "delay", 0, "minimum delay between requests, e.g. 1s")
	pf.DurationVar(&flagCacheTTL, "cache-ttl", 0, "on-disk response cache TTL, e.g. 24h")
	pf.BoolVar(&flagNoCache, "no-cache", false, "disable the on-disk response cache")
	pf.BoolVar(&flagJSON, "json", false, "emit machine-readable JSON (default)")
	pf.BoolVar(&flagHuman, "human", false, "human-readable tables instead of JSON")
}

// jsonMode reports whether output should be machine-readable JSON. JSON is the
// default; --json is accepted for explicitness and --human switches to tables.
func jsonMode() bool {
	if flagJSON {
		return true
	}
	return !flagHuman
}

// initConfig runs once before any command, applying flag > env > default.
func initConfig() {
	cfg = config.Defaults("PAGINEGIALLE")
	if flagBaseURL != "" {
		cfg.BaseURL = flagBaseURL
	}
	if flagContact != "" {
		cfg.Contact = flagContact
	}
	if flagTimeout > 0 {
		cfg.Timeout = flagTimeout
	}
	if flagRetries > 0 {
		cfg.Retries = flagRetries
	}
	if flagDelay > 0 {
		cfg.Delay = flagDelay
	}
	if flagCacheTTL > 0 {
		cfg.CacheTTL = flagCacheTTL
	}
	if flagNoCache {
		cfg.NoCache = true
	}
}

// newService builds the fetching + parsing service from resolved config.
func newService() *paginegialle.Service {
	c := client.New(client.Config{
		BaseURL:  cfg.BaseURL,
		Contact:  cfg.Contact,
		Timeout:  cfg.Timeout,
		Retries:  cfg.Retries,
		Delay:    cfg.Delay,
		CacheTTL: cfg.CacheTTL,
		NoCache:  cfg.NoCache,
	})
	return paginegialle.NewService(c, cfg.BaseURL)
}

// Execute runs the root command. Usage errors (unknown flag/command) exit 2;
// runtime/API failures exit 1 via internal/output from within commands.
func Execute() {
	cobra.OnInitialize(initConfig)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
