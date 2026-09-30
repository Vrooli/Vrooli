// Package support holds small helpers shared by every CLI domain.
// Keep it focused: flag/argument parsing, JSON body loading, output
// path handling, and human-friendly formatting. Anything API-shape
// related lives in the generated proto types, not here.
//
// Audit (2026-05-13): each helper here is used by ≥2 domains and is
// strictly cross-cutting (no domain-specific knowledge). New entries
// must clear the same bar — if it's used in one place, inline it; if
// it's API-shape-aware, push it to the API.
package support

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// CLIName is the invocation name used in retrieval hints and next-command strings.
const CLIName = "web-console"

// ApplyAliases re-attaches command aliases to a manifest-loaded subcommand
// group. cli-manifest/v1 has no per-command alias field, so domains that
// shipped subcommand aliases before the manifest migration (e.g. `session
// list` aliased as `ls`) restore them here, post-LoadFromManifest, to keep
// the observable CLI surface byte-identical. The map is subcommand name →
// aliases; names not present in the group are ignored.
func ApplyAliases(subs []cliapp.Command, aliases map[string][]string) {
	for i := range subs {
		if a, ok := aliases[subs[i].Name]; ok {
			subs[i].Aliases = append(subs[i].Aliases, a...)
		}
	}
}

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

// ReadJSONFile loads a JSON file and returns the parsed RawMessage. An empty
// path returns nil unless required is true.
func ReadJSONFile(path string, required bool) (json.RawMessage, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		if required {
			return nil, fmt.Errorf("a JSON body file path is required (use --body-file)")
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

// ShortID returns the first 8 chars of an id for compact display.
func ShortID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
