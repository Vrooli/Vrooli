package webconsole

import "testing"

func TestResolveBaseURL_EnvOverride(t *testing.T) {
	t.Setenv("WEB_CONSOLE_URL", "http://example.test:9999/")
	if got := ResolveBaseURL(); got != "http://example.test:9999" {
		t.Fatalf("ResolveBaseURL() = %q, want http://example.test:9999 (trailing slash trimmed)", got)
	}
}

func TestResolveBaseURL_PortEnv(t *testing.T) {
	t.Setenv("WEB_CONSOLE_URL", "")
	t.Setenv("WEB_CONSOLE_API_BASE", "")
	t.Setenv("WEB_CONSOLE_API_PORT", "24680")
	if got := ResolveBaseURL(); got != "http://localhost:24680" {
		t.Fatalf("ResolveBaseURL() = %q, want http://localhost:24680", got)
	}
}
