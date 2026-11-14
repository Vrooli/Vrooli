package httputil

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateServiceBaseURL validates a URL obtained from operator configuration
// or Vrooli service discovery before it becomes an HTTP request target. It is
// deliberately a trust-boundary check, not a DNS/IP allowlist: service
// discovery and explicit environment overrides are the authority for which
// local service is being addressed.
func ValidateServiceBaseURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("service URL is empty")
	}
	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid service URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("service URL scheme %q is not supported", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("service URL must include a host")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("service URL must not include user information")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("service URL must not include a query or fragment")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}
