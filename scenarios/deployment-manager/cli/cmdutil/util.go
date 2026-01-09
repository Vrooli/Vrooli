package cmdutil

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/vrooli/cli-core/cliutil"
)

var globalFormat = "json"

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

func TierToNumber(t string) int {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "local", "1":
		return 1
	case "desktop", "2":
		return 2
	case "mobile", "ios", "android", "3":
		return 3
	case "saas", "cloud", "web", "4":
		return 4
	case "enterprise", "on-prem", "5":
		return 5
	default:
		if t == "" {
			return 2
		}
	}
	return 3
}

func EnsureMap(obj map[string]interface{}, key string) map[string]interface{} {
	if obj == nil {
		obj = map[string]interface{}{}
	}
	if val, ok := obj[key]; ok {
		if asMap, ok := val.(map[string]interface{}); ok {
			return asMap
		}
	}
	newMap := map[string]interface{}{}
	obj[key] = newMap
	return newMap
}

func PrintByFormat(format string, body []byte) {
	format = ResolveFormat(format)
	if strings.ToLower(format) == "json" {
		cliutil.PrintJSON(body)
		return
	}
	fmt.Println(string(body))
}

func PrintJSONMap(m map[string]interface{}) {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		cliutil.PrintJSONMap(m, 0)
		return
	}
	fmt.Println(string(data))
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

// PrintTable renders a simple tabular view for human scanning.
func PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	_ = w.Flush()
}
