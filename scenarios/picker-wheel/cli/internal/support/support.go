package support

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliresponse"
	"github.com/vrooli/cli-core/cliutil"
)

// CLIName is the invocation name used in retrieval hints and next-command strings.
const CLIName = "picker-wheel"

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

// Decode delegates the shared response-body contract to cli-core.
func Decode(body []byte, dest interface{}) error {
	return cliresponse.Decode(body, dest)
}

// EnvelopeMessage delegates shared envelope decoding to cli-core.
func EnvelopeMessage(body []byte) string {
	return cliresponse.EnvelopeMessage(body)
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

// ShortID returns the first 8 chars of an id for compact display.
func ShortID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
