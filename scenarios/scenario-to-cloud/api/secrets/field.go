package secrets

import "strings"

// CredentialField converts a bundle secret key into the durable field name the
// credential authority addresses. It matches the normalization the deploy path
// uses, so a value written here is a value the scenario can read back.
func CredentialField(key string) string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(strings.NewReplacer("_", "-", ".", "-").Replace(trimmed))
}
