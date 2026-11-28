package support

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func ParseFlags(fs *flag.FlagSet, args []string) error {
	return cliutil.ParseInterspersed(fs, args)
}

func Decode(body []byte, dest interface{}) error {
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

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

func Confirm(prompt string) (bool, error) {
	fmt.Fprintf(os.Stdout, "%s (y/N) ", strings.TrimSpace(prompt))
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

func APIRootPreviewURL(apiRootBase, funnelID string) string {
	apiRootBase = strings.TrimRight(strings.TrimSpace(apiRootBase), "/")
	funnelID = strings.TrimSpace(funnelID)
	if apiRootBase == "" || funnelID == "" {
		return ""
	}
	return apiRootBase + "/preview/" + funnelID
}

func FormatTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return t.Format(time.RFC3339)
}

func FormatTimeValue(value time.Time) string {
	if value.IsZero() {
		return "unknown"
	}
	return value.Format(time.RFC3339)
}

func PtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func StepsCount(steps []FunnelStep) int {
	return len(steps)
}
