package administration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/internal/credentialleak"
)

func TestSyntheticCredentialAuthorityFamiliesDoNotEnterMetadataArtifacts(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "..", ".vrooli", "service.json")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal("credential declaration file unavailable")
	}
	families, err := credentialleak.ManifestFamilies(manifest)
	if err != nil {
		t.Fatal(err)
	}
	canaries := credentialleak.SyntheticCanaries(families)
	scanner, err := credentialleak.NewScanner(canaries)
	if err != nil {
		t.Fatal(err)
	}
	byName := make(map[string]credentialleak.Canary, len(canaries))
	for _, canary := range canaries {
		byName[canary.Family] = canary
	}
	store := &rotationAuthorityStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		family := family
		t.Run(family.Name, func(t *testing.T) {
			canary := byName[family.Name]
			identity := credentialauthority.Identity(family.Identity)
			candidate, err := authority.PutCandidate(identity, family.Field, canary.Value)
			if err != nil {
				t.Fatal("synthetic candidate creation failed")
			}
			if err := authority.ActivateCandidate(candidate); err != nil {
				t.Fatal("synthetic candidate activation failed")
			}
			binding, err := ResolveBoundCredentialFromAuthority(authority, identity, family.Field, family.Scope)
			if err != nil {
				t.Fatal("synthetic authority resolution failed")
			}
			if binding.Identity != identity || binding.Field != family.Field || binding.Scope != family.Scope || binding.Version == "" || binding.Value != canary.Value {
				t.Fatal("resolved synthetic credential binding metadata was incomplete")
			}
			metadata := struct {
				Identity string `json:"identity"`
				Field    string `json:"field"`
				Scope    string `json:"scope"`
				Version  string `json:"active_version"`
				Status   string `json:"status"`
			}{string(binding.Identity), binding.Field, binding.Scope, binding.Version, "resolved"}
			artifact, err := json.Marshal(map[string]any{
				"request_log": metadata, "response_log": metadata, "operator_receipt": metadata,
				"database_row": metadata, "test_genie_phase_result": metadata,
				"retry_error": map[string]any{"code": "credential_provider_unavailable", "credential": metadata},
			})
			if err != nil {
				t.Fatal("metadata artifact serialization failed")
			}
			findings, err := scanner.Scan(credentialleak.Observation{Channel: "synthetic-authority-metadata-artifacts", Data: artifact})
			if err != nil || len(findings) != 0 {
				t.Fatal("synthetic credential appeared in a metadata-only artifact")
			}
		})
	}
}

func TestGooglePlayReceiptCredentialResolvesWithExactDeclaredScope(t *testing.T) {
	store := &rotationAuthorityStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := authority.PutCandidate(credentialAuthorityIdentity, "google-play-oauth-token", "LPBS_SYNTHETIC_GOOGLE_PLAY_TEST_ONLY")
	if err != nil {
		t.Fatal("synthetic Google Play candidate creation failed")
	}
	if err := authority.ActivateCandidate(candidate); err != nil {
		t.Fatal("synthetic Google Play candidate activation failed")
	}
	SetCredentialAuthorityProvider(func() (*credentialauthority.Authority, error) { return authority, nil })
	t.Cleanup(func() { SetCredentialAuthorityProvider(nil) })
	t.Setenv("LPBS_TEST_CREDENTIAL_FALLBACK", "0")
	binding, err := ResolveCredentialBinding("GOOGLE_PLAY_OAUTH_TOKEN")
	if err != nil {
		t.Fatal("Google Play receipt credential did not resolve through scoped authority")
	}
	if binding.Field != "google-play-oauth-token" || binding.Scope != "google-play:receipt-verify" || binding.Version == "" {
		t.Fatal("Google Play receipt credential binding metadata was incomplete")
	}
}
