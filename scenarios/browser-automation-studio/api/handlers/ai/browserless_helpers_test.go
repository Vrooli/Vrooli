package ai

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveBrowserlessURL(t *testing.T) {
	// Save original env vars
	originalURL := os.Getenv("BROWSERLESS_URL")
	originalBaseURL := os.Getenv("BROWSERLESS_BASE_URL")
	originalHost := os.Getenv("BROWSERLESS_HOST")
	originalPort := os.Getenv("BROWSERLESS_PORT")
	originalScheme := os.Getenv("BROWSERLESS_SCHEME")

	// Clean up after test
	defer func() {
		os.Setenv("BROWSERLESS_URL", originalURL)
		os.Setenv("BROWSERLESS_BASE_URL", originalBaseURL)
		os.Setenv("BROWSERLESS_HOST", originalHost)
		os.Setenv("BROWSERLESS_PORT", originalPort)
		os.Setenv("BROWSERLESS_SCHEME", originalScheme)
	}()

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] uses BROWSERLESS_URL when set", func(t *testing.T) {
		os.Setenv("BROWSERLESS_URL", "http://custom-browserless:3000")
		os.Unsetenv("BROWSERLESS_BASE_URL")
		os.Unsetenv("BROWSERLESS_HOST")
		os.Unsetenv("BROWSERLESS_PORT")
		os.Unsetenv("BROWSERLESS_SCHEME")

		url := resolveBrowserlessURL()
		assert.Equal(t, "http://custom-browserless:3000", url)
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] trims trailing slash from BROWSERLESS_URL", func(t *testing.T) {
		os.Setenv("BROWSERLESS_URL", "http://custom-browserless:3000/")
		os.Unsetenv("BROWSERLESS_BASE_URL")

		url := resolveBrowserlessURL()
		assert.Equal(t, "http://custom-browserless:3000", url)
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] uses BROWSERLESS_BASE_URL as fallback", func(t *testing.T) {
		os.Unsetenv("BROWSERLESS_URL")
		os.Setenv("BROWSERLESS_BASE_URL", "http://base-browserless:4000")
		os.Unsetenv("BROWSERLESS_HOST")
		os.Unsetenv("BROWSERLESS_PORT")

		url := resolveBrowserlessURL()
		assert.Equal(t, "http://base-browserless:4000", url)
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] builds URL from components when no URL set", func(t *testing.T) {
		os.Unsetenv("BROWSERLESS_URL")
		os.Unsetenv("BROWSERLESS_BASE_URL")
		os.Setenv("BROWSERLESS_HOST", "192.168.1.100")
		os.Setenv("BROWSERLESS_PORT", "5000")
		os.Setenv("BROWSERLESS_SCHEME", "https")

		url := resolveBrowserlessURL()
		assert.Equal(t, "https://192.168.1.100:5000", url)
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] uses default values when no env vars set", func(t *testing.T) {
		os.Unsetenv("BROWSERLESS_URL")
		os.Unsetenv("BROWSERLESS_BASE_URL")
		os.Unsetenv("BROWSERLESS_HOST")
		os.Unsetenv("BROWSERLESS_PORT")
		os.Unsetenv("BROWSERLESS_SCHEME")

		url := resolveBrowserlessURL()
		assert.Equal(t, "http://127.0.0.1:4110", url)
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] uses custom port with default host", func(t *testing.T) {
		os.Unsetenv("BROWSERLESS_URL")
		os.Unsetenv("BROWSERLESS_BASE_URL")
		os.Unsetenv("BROWSERLESS_HOST")
		os.Setenv("BROWSERLESS_PORT", "9000")
		os.Unsetenv("BROWSERLESS_SCHEME")

		url := resolveBrowserlessURL()
		assert.Equal(t, "http://127.0.0.1:9000", url)
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] trims whitespace from all components", func(t *testing.T) {
		os.Unsetenv("BROWSERLESS_URL")
		os.Unsetenv("BROWSERLESS_BASE_URL")
		os.Setenv("BROWSERLESS_HOST", "  localhost  ")
		os.Setenv("BROWSERLESS_PORT", "  8080  ")
		os.Setenv("BROWSERLESS_SCHEME", "  http  ")

		url := resolveBrowserlessURL()
		assert.Equal(t, "http://localhost:8080", url)
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] ignores empty string env vars", func(t *testing.T) {
		os.Setenv("BROWSERLESS_URL", "")
		os.Setenv("BROWSERLESS_BASE_URL", "")
		os.Setenv("BROWSERLESS_HOST", "custom-host")
		os.Setenv("BROWSERLESS_PORT", "3000")

		url := resolveBrowserlessURL()
		assert.Equal(t, "http://custom-host:3000", url)
	})
}
