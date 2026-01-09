// Package report provides execution report generation and failure analysis.
package report

import (
	"io"
	"os"

	"golang.org/x/term"
)

// ANSI escape codes for terminal colors.
const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiBlue   = "\033[34m"
	ansiCyan   = "\033[36m"
	ansiBold   = "\033[1m"
)

// Color provides color formatting for terminal output.
// Colors are automatically disabled when output is not a TTY.
type Color struct {
	enabled bool
}

// NewColor creates a Color instance with automatic TTY detection.
// Colors are enabled only when w is an *os.File attached to a terminal.
func NewColor(w io.Writer) *Color {
	enabled := false
	if f, ok := w.(*os.File); ok {
		enabled = term.IsTerminal(int(f.Fd()))
	}
	// Also check NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		enabled = false
	}
	return &Color{enabled: enabled}
}

// NewColorForced creates a Color instance with explicit enable/disable.
// Useful for testing or when TTY detection should be bypassed.
func NewColorForced(enabled bool) *Color {
	return &Color{enabled: enabled}
}

// Enabled returns whether colors are enabled.
func (c *Color) Enabled() bool {
	return c.enabled
}

// Reset returns the ANSI reset code if colors are enabled.
func (c *Color) Reset() string {
	if !c.enabled {
		return ""
	}
	return ansiReset
}

// Red wraps text in red color codes.
func (c *Color) Red(s string) string {
	if !c.enabled || s == "" {
		return s
	}
	return ansiRed + s + ansiReset
}

// Green wraps text in green color codes.
func (c *Color) Green(s string) string {
	if !c.enabled || s == "" {
		return s
	}
	return ansiGreen + s + ansiReset
}

// Yellow wraps text in yellow color codes.
func (c *Color) Yellow(s string) string {
	if !c.enabled || s == "" {
		return s
	}
	return ansiYellow + s + ansiReset
}

// Blue wraps text in blue color codes.
func (c *Color) Blue(s string) string {
	if !c.enabled || s == "" {
		return s
	}
	return ansiBlue + s + ansiReset
}

// Cyan wraps text in cyan color codes.
func (c *Color) Cyan(s string) string {
	if !c.enabled || s == "" {
		return s
	}
	return ansiCyan + s + ansiReset
}

// Bold wraps text in bold codes.
func (c *Color) Bold(s string) string {
	if !c.enabled || s == "" {
		return s
	}
	return ansiBold + s + ansiReset
}

// BoldGreen wraps text in bold green codes.
func (c *Color) BoldGreen(s string) string {
	if !c.enabled || s == "" {
		return s
	}
	return ansiBold + ansiGreen + s + ansiReset
}

// BoldRed wraps text in bold red codes.
func (c *Color) BoldRed(s string) string {
	if !c.enabled || s == "" {
		return s
	}
	return ansiBold + ansiRed + s + ansiReset
}

// BoldYellow wraps text in bold yellow codes.
func (c *Color) BoldYellow(s string) string {
	if !c.enabled || s == "" {
		return s
	}
	return ansiBold + ansiYellow + s + ansiReset
}

// BoldCyan wraps text in bold cyan codes.
func (c *Color) BoldCyan(s string) string {
	if !c.enabled || s == "" {
		return s
	}
	return ansiBold + ansiCyan + s + ansiReset
}

// StatusColor returns colored text based on status.
// passed/success -> green, failed/error -> red, skipped -> yellow, else -> default.
func (c *Color) StatusColor(status, text string) string {
	if !c.enabled {
		return text
	}
	switch NormalizeName(status) {
	case "passed", "success":
		return c.Green(text)
	case "failed", "error":
		return c.Red(text)
	case "skipped":
		return c.Yellow(text)
	default:
		return text
	}
}
