package utils

import "regexp"

// ansiRegex is compiled once at package initialization for performance.
// It matches ANSI escape sequences commonly used for terminal colors and formatting.
//
// Pattern: \x1b\[[0-9;]*[A-Za-z]
//   - \x1b\[     - ESC[ sequence start (CSI - Control Sequence Introducer)
//   - [0-9;]*    - zero or more digits and semicolons (parameters)
//   - [A-Za-z]   - any letter (command character)
//
// This broader pattern [A-Za-z] (vs the previous [mGKHf]) catches additional sequences:
//   - m: SGR (Select Graphic Rendition) - colors, bold, italic, etc.
//   - G: CHA (Cursor Horizontal Absolute)
//   - K: EL (Erase in Line)
//   - H, f: CUP (Cursor Position)
//   - A-Z: cursor movement (CUU, CUD, CUF, CUB), erase operations (ED), etc.
//   - a-z: less common but valid ANSI commands
//
// Reference: ANSI/ECMA-48 standard (ISO 6429)
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// StripANSI removes ANSI escape sequences from the input string.
// This is used to clean agent output before storage and display.
func StripANSI(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}
