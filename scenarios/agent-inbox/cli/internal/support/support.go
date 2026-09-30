package support

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const CLIName = "agent-inbox"

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

func GetJSON[T any](core *cliapp.ScenarioApp, path string, out *T) error {
	body, err := core.Get(path, nil)
	if err != nil {
		return err
	}
	return Decode(body, out)
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

func ParseOptionalBool(raw string) (*bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid boolean value %q", raw)
	}
	return &value, nil
}

func BoolLabel(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}

func Truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func FormatTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.Format("2006-01-02 15:04")
	}
	return value
}

func AbsPath(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	resolved, err := filepath.Abs(input)
	if err != nil {
		return input
	}
	return resolved
}
