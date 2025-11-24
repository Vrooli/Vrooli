package browserless

import (
	"os"
	"testing"
)

func TestResolveURLHonorsExplicitEnv(t *testing.T) {
	t.Setenv("BROWSERLESS_URL", "http://example.com:1234/")
	url, err := ResolveURL(nil, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if url != "http://example.com:1234" {
		t.Fatalf("unexpected url: %s", url)
	}
}

func TestResolveURLAllowsDefaultWhenRequested(t *testing.T) {
	// Clear env to exercise default path.
	os.Unsetenv("BROWSERLESS_URL")
	os.Unsetenv("BROWSERLESS_BASE_URL")
	os.Unsetenv("BROWSERLESS_PORT")
	os.Unsetenv("BROWSERLESS_HOST")
	os.Unsetenv("BROWSERLESS_SCHEME")

	url, err := ResolveURL(nil, true)
	if err != nil {
		t.Fatalf("expected default to be allowed, got error %v", err)
	}
	if url != "http://127.0.0.1:4110" {
		t.Fatalf("unexpected default url: %s", url)
	}
}

func TestResolveURLErrorsWithoutConfigWhenDefaultDisallowed(t *testing.T) {
	t.Setenv("BROWSERLESS_URL", "")
	t.Setenv("BROWSERLESS_BASE_URL", "")
	t.Setenv("BROWSERLESS_PORT", "")

	if _, err := ResolveURL(nil, false); err == nil {
		t.Fatalf("expected error when browserless config missing")
	}
}
