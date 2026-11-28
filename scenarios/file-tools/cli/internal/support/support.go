package support

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliresponse"
	"github.com/vrooli/cli-core/cliutil"
)

// CLIName is the invocation name used in retrieval hints and next-command strings.
const CLIName = "file-tools"

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

// ReadJSONBody loads a JSON file from disk for --body-file flag usage.
// Returns a json.RawMessage suitable for passing to core.Request as the body.
// If path is empty and required is true, returns an error.
func ReadJSONBody(path string, required bool) (json.RawMessage, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		if required {
			return nil, fmt.Errorf("--body-file PATH is required")
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
