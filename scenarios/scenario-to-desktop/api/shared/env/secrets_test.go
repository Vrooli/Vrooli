package env

import "testing"

func TestResolveSecretFromEnv(t *testing.T) {
	t.Setenv("LPBS_SERVICE_SECRET", "env-secret")
	if got := ResolveSecret("LPBS_SERVICE_SECRET"); got != "env-secret" {
		t.Fatalf("expected env-secret, got %q", got)
	}
	resolved := ResolveSecretWithSource("LPBS_SERVICE_SECRET")
	if resolved.Source != "process" {
		t.Fatalf("expected source process, got %q", resolved.Source)
	}
}

func TestResolveSecretMissing(t *testing.T) {
	t.Setenv("LPBS_SERVICE_SECRET", "")

	resolved := ResolveSecretWithSource("LPBS_SERVICE_SECRET")
	if resolved.Value != "" {
		t.Fatalf("expected empty value, got %q", resolved.Value)
	}
	if resolved.Source != "missing" {
		t.Fatalf("expected source missing, got %q", resolved.Source)
	}
}
