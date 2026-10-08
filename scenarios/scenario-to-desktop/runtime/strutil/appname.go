package strutil

import (
	"strings"
	"unicode"
)

// SanitizeAppName is the single filesystem-safe app-name normalization used by
// the supervisor, credential namespace, and bundled CLI shim. It keeps Unicode
// letters and digits stable while mapping punctuation and whitespace to one
// separator, so every platform resolves one app-data directory.
func SanitizeAppName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "desktop-app"
	}
	var out strings.Builder
	previousDash := false
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			previousDash = false
			continue
		}
		if out.Len() > 0 && !previousDash {
			out.WriteByte('-')
			previousDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}
