// Package integration contains shared gates for optional integration tests.
//
// These helpers keep skip messages consistent and searchable when a test needs
// local services or binaries such as the Playwright driver, Ollama, or FFmpeg.
package integration

import (
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// SkipShort skips the current test in short mode with a consistent reason.
func SkipShort(t testing.TB, reason string) {
	t.Helper()

	if testing.Short() {
		t.Skipf("integration dependency unavailable: short mode enabled (%s)", reason)
	}
}

// RequireEnv returns the configured environment value or skips the test.
func RequireEnv(t testing.TB, name string, reason string) string {
	t.Helper()

	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		t.Skipf("integration dependency unavailable: %s not set (%s)", name, reason)
	}
	return value
}

// RequireAnyEnv returns the first configured environment value or skips the
// test after reporting the accepted variable names.
func RequireAnyEnv(t testing.TB, names []string, reason string) string {
	t.Helper()

	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	t.Skipf("integration dependency unavailable: none of %s set (%s)", strings.Join(names, ", "), reason)
	return ""
}

// RequireCommand skips the test when the named executable is not on PATH.
func RequireCommand(t testing.TB, name string, reason string) {
	t.Helper()

	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("integration dependency unavailable: %s not found on PATH (%s)", name, reason)
	}
}

// RequireCommands skips the test when any named executable is not on PATH.
func RequireCommands(t testing.TB, names []string, reason string) {
	t.Helper()

	for _, name := range names {
		RequireCommand(t, name, reason)
	}
}

// RequireHTTPStatusOK skips the test unless a GET request to url returns 2xx.
func RequireHTTPStatusOK(t testing.TB, client *http.Client, url string, reason string) {
	t.Helper()

	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Get(url)
	if err != nil {
		t.Skipf("integration dependency unavailable: %s GET failed: %v (%s)", url, err, reason)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		t.Skipf(
			"integration dependency unavailable: %s returned %s (%s)",
			url,
			resp.Status,
			reason,
		)
	}
}
