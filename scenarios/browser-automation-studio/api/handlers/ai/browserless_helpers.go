package ai

import (
	"fmt"
	"os"
	"strings"
)

// resolveBrowserlessURL determines the base URL for the Browserless service.
// It honours a variety of environment variables to support different deployment
// configurations while providing sensible defaults for local development.
func resolveBrowserlessURL() string {
	candidates := []string{
		strings.TrimSpace(os.Getenv("BROWSERLESS_URL")),
		strings.TrimSpace(os.Getenv("BROWSERLESS_BASE_URL")),
	}
	for _, candidate := range candidates {
		if candidate != "" {
			return strings.TrimRight(candidate, "/")
		}
	}

	port := strings.TrimSpace(os.Getenv("BROWSERLESS_PORT"))
	if port == "" {
		port = "4110"
	}

	host := strings.TrimSpace(os.Getenv("BROWSERLESS_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}

	scheme := strings.TrimSpace(os.Getenv("BROWSERLESS_SCHEME"))
	if scheme == "" {
		scheme = "http"
	}

	return fmt.Sprintf("%s://%s:%s", scheme, host, port)
}
