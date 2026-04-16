package support

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func ParseFlags(name string, args []string) (*flag.FlagSet, *bool, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return nil, nil, err
	}
	return fs, jsonOut, nil
}

func RequireArg(fs *flag.FlagSet, usage string) error {
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: %s", usage)
	}
	return nil
}

func Decode(body []byte, target interface{}) error {
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

func NormalizeInterspersedFlags(args []string) []string {
	if len(args) < 2 {
		return args
	}
	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			continue
		}
		positionals = append(positionals, arg)
	}
	return append(flags, positionals...)
}

func JSONLines(body []byte) []string {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, body, "", "  "); err == nil {
		return nonEmptyLines(pretty.String())
	}
	return nonEmptyLines(string(body))
}

func StringValue(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", value)
	}
}

func nonEmptyLines(raw string) []string {
	parts := strings.Split(raw, "\n")
	lines := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimRight(part, " \t\r"); strings.TrimSpace(trimmed) != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}
