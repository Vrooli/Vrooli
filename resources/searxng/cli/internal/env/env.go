// Package env owns derived environment export helpers for SearXNG integration
// points.
package env

import (
	"os"
	"strings"
)

// DefaultBaseURL is the conventional local SearXNG endpoint (host port 8280
// from the resource port registry).
const DefaultBaseURL = "http://localhost:8280"

// ResolveBaseURL picks the SearXNG base URL from an explicit override, the
// exported resource environment, or the local default — in that order.
func ResolveBaseURL(override string) string {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return trimmed
	}
	for _, key := range []string{"SEARXNG_URL", "SEARXNG_BASE_URL"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return DefaultBaseURL
}
