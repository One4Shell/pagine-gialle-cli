package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/h4shell/paginegialle-cli/internal/client"
	"github.com/h4shell/paginegialle-cli/internal/output"
)

// reportError converts any error into the structured failure contract and
// exits 1. In JSON mode the error is a single-line JSON document on stderr.
func reportError(err error) {
	fe := client.MapError(err)
	output.Report(jsonMode(), output.Err{
		Code:       fe.Code,
		Message:    fe.Message,
		HTTPStatus: fe.HTTPStatus,
	})
}

// splitFields parses a comma-separated --fields value into a clean list.
func splitFields(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, f := range strings.Split(s, ",") {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

// projectFields narrows each item to the requested keys. Keys are emitted in
// sorted order by encoding/json, which keeps the output deterministic.
// When fields is empty the original typed slice is returned unchanged (so the
// canonical struct ordering is preserved).
func projectFields(items any, fields []string) any {
	if len(fields) == 0 {
		return items
	}
	want := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		want[f] = struct{}{}
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return items
	}
	var objs []map[string]any
	if err := json.Unmarshal(raw, &objs); err != nil {
		return items
	}
	out := make([]map[string]any, 0, len(objs))
	for _, obj := range objs {
		filtered := make(map[string]any, len(fields))
		for k, v := range obj {
			if _, ok := want[k]; ok {
				filtered[k] = v
			}
		}
		out = append(out, filtered)
	}
	return out
}

// table writes aligned tab-separated rows to stdout for --human mode.
func table(header []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(header, "\t"))
	for _, r := range rows {
		fmt.Fprintln(w, strings.Join(r, "\t"))
	}
	_ = w.Flush()
}

// joinOr renders a slice as a comma-separated string, or a dash when empty.
func joinOr(values []string, sep string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, sep)
}
