package delivery

import (
	"context"
	"errors"
	"strings"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

func TestValidateSettingsAcceptsConfigurationWithoutCredentialColumns(t *testing.T) {
	if err := ValidateSettings(StorageSettings{SignedURLTTLSeconds: 900, Bucket: "releases"}); err != nil {
		t.Fatalf("expected configuration-only settings to validate: %v", err)
	}
}

func TestApplySettingsUpdateNormalizesValues(t *testing.T) {
	bucket := "  releases  "
	ttl := 120
	settings := ApplySettingsUpdate(StorageSettings{}, StorageSettingsUpdate{Bucket: &bucket, SignedURLTTLSeconds: &ttl})
	if settings.Bucket != "releases" || settings.SignedURLTTLSeconds != 120 {
		t.Fatalf("unexpected settings: %#v", settings)
	}
}

func TestResolveOptionalS3CredentialTreatsUnconfiguredAsAbsent(t *testing.T) {
	value, err := resolveOptionalS3Credential(context.Background(), func(context.Context, string) (string, error) {
		return "", credentialauthority.ErrUnconfigured
	}, "delivery-s3-secret-access-key")
	if err != nil || value != "" {
		t.Fatalf("resolveOptionalS3Credential() = %q, %v; want empty, nil", value, err)
	}
}

func TestResolveOptionalS3CredentialPreservesProviderFailure(t *testing.T) {
	providerErr := errors.New("credential store unavailable")
	_, err := resolveOptionalS3Credential(context.Background(), func(context.Context, string) (string, error) {
		return "", providerErr
	}, "delivery-s3-secret-access-key")
	if !errors.Is(err, providerErr) {
		t.Fatalf("error = %v; want provider error", err)
	}
}

func TestHealthcheckObjectKeyStaysUnderInternalPrefixAndIsUnique(t *testing.T) {
	seen := make(map[string]struct{}, 64)
	for i := 0; i < 64; i++ {
		key, err := healthcheckObjectKey()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(key, ".vrooli/healthchecks/") {
			t.Fatalf("healthcheck object key = %q, want .vrooli/healthchecks/ prefix", key)
		}
		if strings.Contains(key, "..") {
			t.Fatalf("healthcheck object key = %q contains traversal", key)
		}
		if _, duplicate := seen[key]; duplicate {
			t.Fatalf("healthcheck object key %q repeated; keys must be unique", key)
		}
		seen[key] = struct{}{}
	}
}
