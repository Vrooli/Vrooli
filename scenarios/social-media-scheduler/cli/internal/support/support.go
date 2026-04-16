package support

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// CLIName is the invocation name used in retrieval hints and next-command strings.
const CLIName = "social-media-scheduler"

// NewFlagSet returns a flag set configured for library-style usage with suppressed output.
func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// ParseFlags parses args with interspersed positional/flag support.
func ParseFlags(fs *flag.FlagSet, args []string) error {
	return cliutil.ParseInterspersed(fs, args)
}

// Decode unmarshals body as the concrete shape expected by the caller.
// If the response is a {success, data} envelope, the data field is unwrapped;
// otherwise the raw body is decoded directly.
func Decode(body []byte, dest interface{}) error {
	if raw, ok := unwrapEnvelope(body); ok {
		if err := json.Unmarshal(raw, dest); err != nil {
			return fmt.Errorf("parse response data: %w", err)
		}
		return nil
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

// DecodeRaw returns the inner data payload if the body is a {success, data}
// envelope, else returns the body unchanged.
func DecodeRaw(body []byte) json.RawMessage {
	if raw, ok := unwrapEnvelope(body); ok {
		return raw
	}
	return body
}

// EnvelopeMessage returns the "message" string if body is a
// {success, data: {message}} envelope, else empty.
func EnvelopeMessage(body []byte) string {
	raw, ok := unwrapEnvelope(body)
	if !ok {
		return ""
	}
	var wrapper struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return ""
	}
	return strings.TrimSpace(wrapper.Message)
}

func unwrapEnvelope(body []byte) (json.RawMessage, bool) {
	trimmed := strings.TrimSpace(string(body))
	if !strings.HasPrefix(trimmed, "{") {
		return nil, false
	}
	var env struct {
		Success *bool           `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, false
	}
	if env.Success == nil || len(env.Data) == 0 {
		return nil, false
	}
	return env.Data, true
}

// BuildQuery returns url.Values with only non-empty trimmed values set.
func BuildQuery(params map[string]string) url.Values {
	values := url.Values{}
	for key, value := range params {
		value = strings.TrimSpace(value)
		if value != "" {
			values.Set(key, value)
		}
	}
	return values
}

// ReadJSONFile loads a JSON file and returns the parsed RawMessage. An empty path
// returns nil unless required is true.
func ReadJSONFile(path string, required bool) (json.RawMessage, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		if required {
			return nil, fmt.Errorf("a JSON file path is required")
		}
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse %s as JSON: %w", path, err)
	}
	return raw, nil
}

// WriteOutput writes data to path, or stdout when path is empty.
func WriteOutput(path string, data []byte) error {
	path = strings.TrimSpace(path)
	if path == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// FormatTime renders an RFC3339 timestamp, falling back to the raw string.
func FormatTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return t.UTC().Format(time.RFC3339)
}

// FormatTimeValue renders a time.Time; zero times render as "unknown".
func FormatTimeValue(value time.Time) string {
	if value.IsZero() {
		return "unknown"
	}
	return value.UTC().Format(time.RFC3339)
}

// PtrString returns the dereferenced string or an empty string.
func PtrString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

// ShortID returns the first 8 chars of an id for compact display.
func ShortID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

// MapRows renders a map as sorted `key: value` lines for ListReport output.
func MapRows(data map[string]interface{}) []string {
	if len(data) == 0 {
		return []string{"(empty payload)"}
	}
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rows := make([]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, fmt.Sprintf("%s: %s", k, RenderValue(data[k])))
	}
	return rows
}

// RenderValue formats a JSON-decoded value for human display.
func RenderValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case string:
		return v
	case bool:
		return fmt.Sprintf("%t", v)
	case float64:
		if float64(int64(v)) == v {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprintf("%v", value)
		}
		return string(data)
	}
}

// SplitCSV returns trimmed non-empty tokens from a comma-separated list.
func SplitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
