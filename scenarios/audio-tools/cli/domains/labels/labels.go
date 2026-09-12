// Package labels contains the shared formatting rules for generated enum
// values rendered by audio-tools CLI commands.
package labels

import "strings"

// Enum removes a generated enum prefix and applies the requested formatter.
// Unknown or unspecified values use fallback so CLI output remains explicit.
func Enum[T interface{ String() string }](value T, prefix, fallback string, format func(string) string) string {
	raw := value.String()
	if !strings.HasPrefix(raw, prefix) {
		return fallback
	}
	raw = strings.TrimPrefix(raw, prefix)
	if raw == "" || strings.HasSuffix(raw, "UNSPECIFIED") {
		return fallback
	}
	return format(raw)
}

func LowerWords(raw string) string {
	return strings.ToLower(strings.ReplaceAll(raw, "_", "-"))
}
