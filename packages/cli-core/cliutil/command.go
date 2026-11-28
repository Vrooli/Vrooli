package cliutil

import (
	"strings"
	"unicode"
)

// DisplayCommand formats argv for receipts only. Never execute its result as a shell.
func DisplayCommand(argv []string) string {
	parts := make([]string, 0, len(argv))
	for _, arg := range argv {
		parts = append(parts, displayCommandArg(arg))
	}
	return strings.Join(parts, " ")
}

func displayCommandArg(arg string) string {
	if arg != "" {
		safe := true
		for _, r := range arg {
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("_./:@+^~=,-", r)) {
				safe = false
				break
			}
		}
		if safe {
			return arg
		}
	}
	return "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
}
