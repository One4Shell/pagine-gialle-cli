package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// SchemaVersion is stamped into collection envelopes so machine consumers can
// detect breaking shape changes.
const SchemaVersion = 1

// Pagination carries paging state for collection envelopes.
type Pagination struct {
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	HasMore    bool `json:"has_more"`
	NextPage   int  `json:"next_page,omitempty"`
	TotalItems int  `json:"total_items,omitempty"`
}

// Envelope wraps a collection result so machine consumers can rely on a
// stable, self-describing shape (see the cli-forge-skill output contract).
type Envelope struct {
	SchemaVersion int         `json:"schema_version"`
	Query         any         `json:"query,omitempty"`
	Items         any         `json:"items"`
	Pagination    *Pagination `json:"pagination,omitempty"`
	Stats         any         `json:"stats,omitempty"`
}

// Err is the structured error payload emitted on stderr on failure.
type Err struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Details    any    `json:"details,omitempty"`
}

type failure struct {
	OK    bool `json:"ok"`
	Error Err  `json:"error"`
}

// WriteJSON emits v to stdout as exactly one JSON document followed by a
// newline. It is the only way data leaves a command.
func WriteJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(b))
	return err
}

// Marshal is an alias used by tests to pin the exact JSON contract.
func Marshal(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

// Fail writes the machine-readable failure to stderr (used in --json mode)
// and exits with status 1. stdout is left untouched.
func Fail(err Err) {
	if err.Code == "" {
		err.Code = "internal"
	}
	_ = json.NewEncoder(os.Stderr).Encode(failure{OK: false, Error: err})
	os.Exit(1)
}

// FailText exits 1 after printing a human-readable error line to stderr.
func FailText(code, message string) {
	if code == "" {
		code = "internal"
	}
	fmt.Fprintf(os.Stderr, "error[%s]: %s\n", code, message)
	os.Exit(1)
}

// Report emits a structured error on stderr when jsonMode is true, or a plain
// line otherwise, then exits 1. It centralizes the failure contract for cmd/.
func Report(jsonMode bool, err Err) {
	if jsonMode {
		Fail(err)
	}
	FailText(err.Code, err.Message)
}
