package support

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// CLIName is the invocation name used in retrieval hints and next-command strings.
const CLIName = "algorithm-library"

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

// EnvelopeMessage returns the "message" string if body is an envelope, else empty.
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

// ReadFileText reads a source file and returns its content as a string.
func ReadFileText(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("a file path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(data), nil
}

// DetectLanguage infers a language name from a file extension. Returns an empty
// string when the extension is unknown.
func DetectLanguage(path string) string {
	ext := strings.ToLower(strings.TrimPrefix(extOf(path), "."))
	switch ext {
	case "py":
		return "python"
	case "js":
		return "javascript"
	case "ts":
		return "typescript"
	case "java":
		return "java"
	case "cpp", "cc", "cxx":
		return "cpp"
	case "c":
		return "c"
	case "go":
		return "go"
	case "rs":
		return "rust"
	case "rb":
		return "ruby"
	case "cs":
		return "csharp"
	}
	return ""
}

func extOf(path string) string {
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		return path[idx:]
	}
	return ""
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

// CheckMark returns a human-readable indicator for a bool.
func CheckMark(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
