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

// RenderTable writes a simple tab-aligned table.
func RenderTable(w io.Writer, headers []string, rows [][]string) error {
	if w == nil {
		return errors.New("writer is required")
	}
	if len(headers) == 0 {
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, strings.Join(headers, "\t")); err != nil {
		return err
	}
	for _, row := range rows {
		cells := make([]string, len(headers))
		copy(cells, row)
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
