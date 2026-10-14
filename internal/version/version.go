package version

// Build information. The Makefile stamps these via -ldflags; the scaffold
// defaults are the un-stamped fallback.
var (
	Version = "0.1.0"
	Commit  = "none"
	Date    = "unknown"
)
