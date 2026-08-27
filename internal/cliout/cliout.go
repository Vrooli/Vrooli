package cliout

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// RenderJSONOr selects the JSON wire renderer or the human-readable renderer
// for a command. Keeping the format decision here prevents each command from
// growing a subtly different JSON prologue.
func RenderJSONOr(w io.Writer, format Format, jsonRender func(io.Writer) error, humanRender func(io.Writer) error) error {
	if format == FormatJSON {
		return jsonRender(w)
	}
	return humanRender(w)
}

const (
	clioutParameterA = 2
)

// Format describes the output mode requested by the caller.
type Format string

const (
	FormatHuman Format = "human"
	FormatJSON  Format = "json"
)

// ParseFormat resolves --json and --format style inputs to a single format.
func ParseFormat(format string, jsonFlag bool) (Format, error) {
	if jsonFlag {
		return FormatJSON, nil
	}

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", string(FormatHuman):
		return FormatHuman, nil
	case string(FormatJSON):
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("unsupported output format %q", format)
	}
}

// DefaultColorEnabled returns whether ANSI color should be used for a stream.
func DefaultColorEnabled(stream *os.File) bool {
	if stream == nil {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}

	info, err := stream.Stat()
	if err != nil {
		return false
	}

	return colorEnabledForFileMode(info.Mode())
}

// colorEnabledForFileMode keeps the terminal heuristic explicit and testable.
// We intentionally use a narrow, dependency-free signal here: only character
// devices are treated as interactive candidates.
func colorEnabledForFileMode(mode os.FileMode) bool {
	return (mode & os.ModeCharDevice) != 0
}

// WriteJSON emits formatted JSON followed by a newline.
func WriteJSON(w io.Writer, value any) error {
	if w == nil {
		return errors.New("writer is required")
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// Section describes one human-or-JSON output section. JSON is kept as a
// callback so callers can preserve their existing response shape without
// duplicating format branching around every renderer.
type Section struct {
	Format  Format
	JSON    func() error
	Empty   string
	Headers []string
	Rows    [][]string
}

// WriteSection owns the shared JSON, empty-state, and tabular rendering shape.
func WriteSection(w io.Writer, section Section) error {
	if w == nil {
		return errors.New("writer is required")
	}
	if section.Format == FormatJSON {
		if section.JSON == nil {
			return errors.New("JSON writer is required")
		}
		return section.JSON()
	}
	if len(section.Rows) == 0 {
		if section.Empty == "" {
			return nil
		}
		_, err := fmt.Fprintln(w, section.Empty)
		return err
	}
	return RenderTable(w, section.Headers, section.Rows)
}

// RenderTable writes a simple tab-aligned table.
func RenderTable(w io.Writer, headers []string, rows [][]string) error {
	if w == nil {
		return errors.New("writer is required")
	}
	tw := tabwriter.NewWriter(w, 0, 0, clioutParameterA, ' ', 0)
	if len(headers) > 0 {
		if _, err := fmt.Fprintln(tw, strings.Join(headers, "\t")); err != nil {
			return err
		}
	}
	for _, row := range rows {
		cells := row
		if len(headers) > 0 {
			cells = make([]string, len(headers))
			copy(cells, row)
		}
		if _, err := fmt.Fprintln(tw, strings.Join(cells, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}

// BoolLabel keeps simple human-readable booleans consistent across CLI output.
func BoolLabel(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

// EmptyLabel creates the shared noun-phrase form used for empty sections.
func EmptyLabel(noun string) string { return "no " + noun }
