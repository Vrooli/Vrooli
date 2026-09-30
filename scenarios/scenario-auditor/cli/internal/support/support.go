package support

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const CLIName = "scenario-auditor"

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

func PrintList(jsonOutput bool, report cliapp.ListReport) error {
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func PrintOperational(jsonOutput bool, report cliapp.OperationalReport) error {
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func PrintMutation(jsonOutput bool, report cliapp.MutationReport) error {
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func StringValue(v any) string {
	switch value := v.(type) {
	case string:
		return strings.TrimSpace(value)
	case fmt.Stringer:
		return strings.TrimSpace(value.String())
	case json.Number:
		return value.String()
	case float64:
		if value == float64(int64(value)) {
			return strconv.FormatInt(int64(value), 10)
		}
		return strconv.FormatFloat(value, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(value)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func BoolValue(v any) bool {
	switch value := v.(type) {
	case bool:
		return value
	case string:
		parsed, _ := strconv.ParseBool(strings.TrimSpace(value))
		return parsed
	default:
		return false
	}
}

func IntValue(v any) int {
	switch value := v.(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		i, _ := value.Int64()
		return int(i)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(value))
		return i
	default:
		return 0
	}
}

func FloatValue(v any) float64 {
	switch value := v.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case json.Number:
		f, _ := value.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
		return f
	default:
		return 0
	}
}

func MapValue(v any) map[string]any {
	if v == nil {
		return nil
	}
	switch value := v.(type) {
	case map[string]any:
		return value
	default:
		return nil
	}
}

func SliceValue(v any) []any {
	switch value := v.(type) {
	case []any:
		return value
	default:
		return nil
	}
}

func SliceMaps(v any) []map[string]any {
	items := SliceValue(v)
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m := MapValue(item); m != nil {
			result = append(result, m)
		}
	}
	return result
}

func StringsFromAny(v any) []string {
	items := SliceValue(v)
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text := StringValue(item); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func ParseMultiValue(csv string, repeated []string) []string {
	combined := append([]string{}, cliutil.ParseCSV(csv)...)
	for _, value := range repeated {
		combined = append(combined, cliutil.ParseCSV(value)...)
	}
	seen := make(map[string]struct{}, len(combined))
	out := make([]string, 0, len(combined))
	for _, value := range combined {
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

func FormatKV(details map[string]any, keys ...string) []string {
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		if text := StringValue(details[key]); text != "" {
			lines = append(lines, fmt.Sprintf("%s: %s", prettifyKey(key), text))
		}
	}
	return lines
}

func SortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func PrettifyMapLines(m map[string]any) []string {
	keys := SortedMapKeys(m)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		value := m[key]
		if child := MapValue(value); child != nil {
			lines = append(lines, fmt.Sprintf("%s: %s", prettifyKey(key), strings.Join(PrettifyMapLines(child), "; ")))
			continue
		}
		if items := StringsFromAny(value); len(items) > 0 {
			lines = append(lines, fmt.Sprintf("%s: %s", prettifyKey(key), strings.Join(items, ", ")))
			continue
		}
		text := StringValue(value)
		if text == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", prettifyKey(key), text))
	}
	return lines
}

func ResolveScenarioPath(name string) string {
	return cliutil.ResolveScenarioPath(name)
}

func WaitForStatus(ctx context.Context, interval time.Duration, fetch func() (map[string]any, error), statusPath ...string) (map[string]any, error) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		resp, err := fetch()
		if err != nil {
			return nil, err
		}
		status := NestedString(resp, statusPath...)
		if IsTerminalStatus(status) {
			return resp, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func IsTerminalStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "failed", "cancelled", "rolled_back":
		return true
	default:
		return false
	}
}

func NestedMap(m map[string]any, path ...string) map[string]any {
	current := m
	for _, key := range path {
		next := MapValue(current[key])
		if next == nil {
			return nil
		}
		current = next
	}
	return current
}

func NestedString(m map[string]any, path ...string) string {
	if len(path) == 0 {
		return ""
	}
	current := m
	for i := 0; i < len(path)-1; i++ {
		next := MapValue(current[path[i]])
		if next == nil {
			return ""
		}
		current = next
	}
	return StringValue(current[path[len(path)-1]])
}

func prettifyKey(key string) string {
	key = strings.ReplaceAll(key, "_", " ")
	key = strings.ReplaceAll(key, "-", " ")
	return strings.Title(key)
}
