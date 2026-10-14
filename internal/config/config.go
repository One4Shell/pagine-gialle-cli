package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the resolved runtime settings. Resolution order is
// flag > environment variable > default; cmd/ applies flag values after
// calling Defaults().
type Config struct {
	BaseURL  string
	Contact  string
	Timeout  time.Duration
	Retries  int
	Delay    time.Duration
	CacheTTL time.Duration
	NoCache  bool
}

// DefaultBaseURL is the public directory site this CLI reads from.
const DefaultBaseURL = "https://www.paginegialle.it"

// Env returns env[name] if set and non-empty, otherwise def.
func Env(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

// EnvInt returns env[name] parsed as an int, otherwise def.
func EnvInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// Defaults returns the baseline configuration read from environment variables
// prefixed with the CLI name (e.g. PAGINEGIALLE_BASE_URL) plus sane defaults.
func Defaults(envPrefix string) Config {
	return Config{
		BaseURL:  Env(envPrefix+"_BASE_URL", DefaultBaseURL),
		Contact:  Env(envPrefix+"_CONTACT", ""),
		Timeout:  30 * time.Second,
		Retries:  EnvInt(envPrefix+"_RETRIES", 3),
		Delay:    time.Second,
		CacheTTL: 24 * time.Hour,
	}
}
