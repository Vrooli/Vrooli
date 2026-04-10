// Package output provides formatting helpers for CLI output.
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// JSON outputs data as pretty-printed JSON.
func JSON(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// Table outputs data in a simple table format.
func Table(headers []string, rows [][]string) {
	if len(rows) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range headers {
		fmt.Printf("%-*s  ", widths[i], h)
	}
	fmt.Println()

	// Print separator
	for i := range headers {
		fmt.Print(strings.Repeat("-", widths[i]))
		fmt.Print("  ")
	}
	fmt.Println()

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Printf("%-*s  ", widths[i], cell)
			}
		}
		fmt.Println()
	}
}

// Success prints a success message.
func Success(format string, args ...interface{}) {
	fmt.Printf("✓ "+format+"\n", args...)
}

// Error prints an error message to stderr.
func Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
}

// Info prints an info message.
func Info(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// List outputs items in a simple list format.
func List(items []string, prefix string) {
	if prefix == "" {
		prefix = "  "
	}
	for _, item := range items {
		fmt.Printf("%s%s\n", prefix, item)
	}
}
