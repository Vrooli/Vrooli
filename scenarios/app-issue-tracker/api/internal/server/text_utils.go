package server

import "regexp"

// ansiRegex is compiled once at package initialization for performance.
// It matches ANSI escape sequences commonly used for terminal colors and formatting.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[mGKHf]`)

// stripANSI removes ANSI escape sequences from the input string.
// This is used to clean agent output before storage and display.
func stripANSI(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}
