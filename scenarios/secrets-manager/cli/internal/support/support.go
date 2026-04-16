package support

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	return fs
}

func ParseFlags(fs *flag.FlagSet, args []string) error {
	return cliutil.ParseInterspersed(fs, args)
}

func GetJSON[T any](core *cliapp.ScenarioApp, path string, query url.Values, out *T) error {
	body, err := core.Get(path, query)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func GetRootJSON[T any](core *cliapp.ScenarioApp, path string, query url.Values, out *T) error {
	body, err := core.GetRoot(path, query)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func RequestJSON[T any](core *cliapp.ScenarioApp, method, path string, query url.Values, body any, out *T) error {
	resp, err := core.Request(method, path, query, body)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp, out)
}

func RequestRootJSON[T any](core *cliapp.ScenarioApp, method, path string, query url.Values, body any, out *T) error {
	resp, err := core.RequestRoot(method, path, query, body)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp, out)
}

func PrintList(jsonOutput bool, payload any, report cliapp.ListReport) error {
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, payload)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func PrintOperational(jsonOutput bool, payload any, report cliapp.OperationalReport) error {
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, payload)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func PrintMutation(jsonOutput bool, payload any, report cliapp.MutationReport) error {
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, payload)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func FormatTime(ts time.Time) string {
	if ts.IsZero() {
		return "unknown"
	}
	return ts.Format(time.RFC3339)
}

func FormatTimePtr(ts *time.Time) string {
	if ts == nil {
		return "never"
	}
	return FormatTime(*ts)
}

func BoolLabel(v bool, yes, no string) string {
	if v {
		return yes
	}
	return no
}

func JoinNonEmpty(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	return strings.Join(cleaned, " | ")
}

func ParseJSONMap(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("parse JSON object: %w", err)
	}
	return out, nil
}

func ParseKV(values []string) (map[string]string, error) {
	out := make(map[string]string, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key, raw, ok := strings.Cut(value, "=")
		if !ok {
			return nil, fmt.Errorf("expected KEY=VALUE, got %q", value)
		}
		key = strings.TrimSpace(key)
		raw = strings.TrimSpace(raw)
		if key == "" || raw == "" {
			return nil, fmt.Errorf("expected KEY=VALUE, got %q", value)
		}
		out[key] = raw
	}
	return out, nil
}

func ParseMultiValues(csv string, repeated []string) []string {
	values := append([]string{}, cliutil.ParseCSV(csv)...)
	for _, value := range repeated {
		values = append(values, cliutil.ParseCSV(value)...)
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func SortedKeys[K ~string, V any](m map[K]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, string(key))
	}
	sort.Strings(keys)
	return keys
}

func Query(key, value string) url.Values {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	q := url.Values{}
	q.Set(key, value)
	return q
}

func Fallback(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func OptionalResourceFlag(resource string) string {
	resource = strings.TrimSpace(resource)
	if resource == "" {
		return ""
	}
	return " --resource " + resource
}
