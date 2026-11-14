package env

import (
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

func TestResolveSecretPrefersAuthorityOverProcessEnvironment(t *testing.T) {
	t.Setenv("S2D_TEST_CREDENTIAL_FALLBACK", "")
	t.Setenv("LPBS_SERVICE_SECRET", "process-secret")
	original := requireCredential
	requireCredential = func(identity credentialauthority.Identity, field string) (string, error) {
		if identity != "vrooli/landing-page-business-suite" || field != "service-secret" {
			t.Fatalf("unexpected authority lookup: %s:%s", identity, field)
		}
		return "authority-secret", nil
	}
	t.Cleanup(func() { requireCredential = original })

	resolved := ResolveSecretWithSource("LPBS_SERVICE_SECRET")
	if resolved.Value != "authority-secret" || resolved.Source != "authority" {
		t.Fatalf("expected authority-secret from authority, got %+v", resolved)
	}
}

func TestResolveSecretFromEnv(t *testing.T) {
	t.Setenv("S2D_TEST_CREDENTIAL_FALLBACK", "1")
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
	t.Setenv("S2D_TEST_CREDENTIAL_FALLBACK", "1")
	t.Setenv("LPBS_SERVICE_SECRET", "")

	resolved := ResolveSecretWithSource("LPBS_SERVICE_SECRET")
	if resolved.Value != "" {
		t.Fatalf("expected empty value, got %q", resolved.Value)
	}
	if resolved.Source != "missing" {
		t.Fatalf("expected source missing, got %q", resolved.Source)
	}
}

func TestResolveSecretRejectsUndeclaredProcessSecret(t *testing.T) {
	t.Setenv("S2D_TEST_CREDENTIAL_FALLBACK", "")
	t.Setenv("UNDECLARED_SECRET", "should-not-be-used")
	resolved := ResolveSecretWithSource("UNDECLARED_SECRET")
	if resolved.Value != "" || resolved.Source != "missing" {
		t.Fatalf("expected undeclared secret to be missing, got %+v", resolved)
	}
}
