// Package cmdutil provides shared utilities for CLI command packages.
package cmdutil

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/vrooli/cli-core/cliutil"
)

var globalFormat = "json"

// APIPath builds a versioned API path from a relative path.
func APIPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return "/api/v1" + path
}

// MapToValues converts a simple string map to url.Values format.
func MapToValues(m map[string]string) map[string][]string {
	if m == nil {
		return nil
	}
	result := make(map[string][]string)
	for k, v := range m {
		result[k] = []string{v}
	}
	return result
}

// ParseArgs allows flag parsing even when the first argument is positional.
// Users can run `cmd <id> --flag value` or `cmd --flag value <id>` interchangeably.
func ParseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		positional := args[0]
		if err := fs.Parse(args[1:]); err != nil {
			return nil, err
		}
		return append([]string{positional}, fs.Args()...), nil
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return fs.Args(), nil
}

// ResolveFormat returns the effective output format based on a global default and a local override.
func ResolveFormat(local string) string {
	if strings.TrimSpace(local) != "" {
		return local
	}
	return globalFormat
}

// SetGlobalFormat sets the default output format (applies when commands leave --format empty).
func SetGlobalFormat(format string) {
	if strings.TrimSpace(format) == "" {
		return
	}
	globalFormat = strings.ToLower(strings.TrimSpace(format))
}

// GlobalFormat returns the currently active global format.
func GlobalFormat() string {
	return globalFormat
}

// PrintByFormat prints data according to the resolved format.
func PrintByFormat(format string, body []byte) {
	format = ResolveFormat(format)
	if strings.ToLower(format) == "json" {
		cliutil.PrintJSON(body)
		return
	}
	fmt.Println(string(body))
}

// PrintTable renders a simple tabular view for human scanning.
func PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	_ = w.Flush()
}
