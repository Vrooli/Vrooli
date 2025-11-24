package browserless

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// ResolveURL normalizes browserless configuration across the API.
// Priority: BROWSERLESS_URL → BROWSERLESS_BASE_URL → host/port/scheme.
// When allowDefault is true, a missing port falls back to 127.0.0.1:4110.
func ResolveURL(log *logrus.Logger, allowDefault bool) (string, error) {
	candidates := []string{
		os.Getenv("BROWSERLESS_URL"),
		os.Getenv("BROWSERLESS_BASE_URL"),
	}
	for _, candidate := range candidates {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return strings.TrimRight(trimmed, "/"), nil
		}
	}

	port := strings.TrimSpace(os.Getenv("BROWSERLESS_PORT"))
	host := strings.TrimSpace(os.Getenv("BROWSERLESS_HOST"))
	scheme := strings.TrimSpace(os.Getenv("BROWSERLESS_SCHEME"))

	if port == "" && !allowDefault {
		if log != nil {
			log.Warn("browserless URL not configured (set BROWSERLESS_URL or BROWSERLESS_PORT)")
		}
		return "", fmt.Errorf("browserless url not configured")
	}

	if port == "" {
		port = "4110"
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if scheme == "" {
		scheme = "http"
	}

	url := fmt.Sprintf("%s://%s:%s", scheme, host, port)
	return strings.TrimRight(url, "/"), nil
}
