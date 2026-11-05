package support

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const AppName = "scenario-dependency-analyzer"

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func ParseFlags(fs *flag.FlagSet, args []string) error {
	return cliutil.ParseInterspersed(fs, args)
}

func BuildQuery(params map[string]string) url.Values {
	values := url.Values{}
	for key, value := range params {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		values.Set(key, value)
	}
	return values
}

func Decode(body []byte, dest interface{}) error {
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

func PrintAPIJSON(body []byte) error {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, body, "", "  "); err == nil {
		_, err = fmt.Fprintln(os.Stdout, pretty.String())
		return err
	}
	_, err := fmt.Fprintln(os.Stdout, string(body))
	return err
}

func PrintReportJSON(report interface{}) error {
	return cliapp.PrintReportJSON(os.Stdout, report)
}

func PrintList(jsonOutput bool, report cliapp.ListReport, raw interface{}) error {
	if jsonOutput {
		if raw != nil {
			return PrintReportJSON(raw)
		}
		return PrintReportJSON(report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func PrintOperational(jsonOutput bool, report cliapp.OperationalReport, raw interface{}) error {
	if jsonOutput {
		if raw != nil {
			return PrintReportJSON(raw)
		}
		return PrintReportJSON(report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func PrintMutation(jsonOutput bool, report cliapp.MutationReport, raw interface{}) error {
	if jsonOutput {
		if raw != nil {
			return PrintReportJSON(raw)
		}
		return PrintReportJSON(report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func String(value interface{}) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func Bool(value interface{}) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		return err == nil && parsed
	case float64:
		return typed != 0
	case int:
		return typed != 0
	default:
		return false
	}
}

func Int(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return 0
}

func Float(value interface{}) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func Map(value interface{}) map[string]interface{} {
	if typed, ok := value.(map[string]interface{}); ok {
		return typed
	}
	return nil
}

func Maps(value interface{}) []map[string]interface{} {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if mapped := Map(item); mapped != nil {
			out = append(out, mapped)
		}
	}
	return out
}

func Strings(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := String(item)
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func KeysSorted(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func GraphType(input string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "", "all", "combined":
		return "combined", nil
	case "resource", "resources":
		return "resource", nil
	case "scenario", "scenarios":
		return "scenario", nil
	default:
		return "", fmt.Errorf("invalid graph type %q; valid types: resource, scenario, combined", input)
	}
}

func JoinCSV(input string) []string {
	raw := strings.Split(strings.TrimSpace(input), ",")
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func BoolWord(value bool, yes string, no string) string {
	if value {
		return yes
	}
	return no
}
