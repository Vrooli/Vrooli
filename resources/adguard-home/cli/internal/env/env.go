package env

import (
	"os"
	"strings"
)

const DefaultBaseURL = "http://localhost:3000"

// ResolveBaseURL returns the operator-provided base URL, the manifest export,
// or the default AdGuard setup/admin endpoint.
func ResolveBaseURL(override string) string {
	if value := strings.TrimSpace(override); value != "" {
		return strings.TrimRight(value, "/")
	}
	for _, key := range []string{"ADGUARD_HOME_BASE_URL", "ADGUARD_HOME_URL"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return strings.TrimRight(value, "/")
		}
	}
	return DefaultBaseURL
}
