package support

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

func NewFlagSet(name string) *flag.FlagSet {
	return flag.NewFlagSet(name, flag.ContinueOnError)
}

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

func Unmarshal(body []byte, target interface{}) error {
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

func Truncate(text string, max int) string {
	if max <= 0 || len(text) <= max {
		return text
	}
	if max <= 3 {
		return text[:max]
	}
	return text[:max-3] + "..."
}

func ShortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
