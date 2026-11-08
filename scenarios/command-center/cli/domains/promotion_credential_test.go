package domains

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

func useAuthority(t *testing.T, authority *credentialauthority.Authority) {
	t.Helper()
	previous := promotionCredentialAuthority
	promotionCredentialAuthority = func() (*credentialauthority.Authority, error) { return authority, nil }
	t.Cleanup(func() { promotionCredentialAuthority = previous })
	t.Setenv("COMMAND_CENTER_LPBS_READER_TOKEN", "")
	t.Setenv("COMMAND_CENTER_LPBS_ORIGIN", "")
}

func scopedAuthority(t *testing.T) *credentialauthority.Authority {
	t.Helper()
	authority, err := credentialauthority.OpenEncryptedFile(filepath.Join(t.TempDir(), "credentials.enc"), "test-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	return authority
}

// The reader token must keep the authority's failure taxonomy: an absent
// provider and an unconfigured credential are different remediations.
func TestPromotionReaderTokenPreservesCredentialTaxonomy(t *testing.T) {
	absent, err := credentialauthority.Unavailable("no store on this host")
	if err != nil {
		t.Fatal(err)
	}
	useAuthority(t, absent)
	if _, err := resolvePromotionReaderToken(); !errors.Is(err, credentialauthority.ErrProviderAbsent) {
		t.Fatalf("absent provider error = %v, want ErrProviderAbsent", err)
	}

	useAuthority(t, scopedAuthority(t))
	_, err = resolvePromotionReaderToken()
	if !errors.Is(err, credentialauthority.ErrUnconfigured) || !strings.Contains(err.Error(), "metrics-reader-token") {
		t.Fatalf("unconfigured error = %v, want ErrUnconfigured naming the field", err)
	}
}

func TestPromotionReaderTokenResolvesTheOriginField(t *testing.T) {
	authority := scopedAuthority(t)
	identity := credentialauthority.Identity("vrooli/landing-page-business-suite")
	if err := authority.Put(identity, "metrics-reader-token", " reader "); err != nil {
		t.Fatal(err)
	}
	if err := authority.Put(identity, "service-secret", "service"); err != nil {
		t.Fatal(err)
	}
	useAuthority(t, authority)
	if token, err := resolvePromotionReaderToken(); err != nil || token != "reader" {
		t.Fatalf("default origin token = %q, %v; want trimmed reader token", token, err)
	}
	t.Setenv("COMMAND_CENTER_LPBS_ORIGIN", "production")
	if token, err := resolvePromotionReaderToken(); err != nil || token != "service" {
		t.Fatalf("production origin token = %q, %v; want service secret", token, err)
	}
}
