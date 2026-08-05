package bundles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBundleSchemaSamplesValidate(t *testing.T) {
	samples := []string{
		"desktop-happy.json",
		"desktop-playwright.json",
	}
	for _, sample := range samples {
		t.Run(sample, func(t *testing.T) {
			path := filepath.Join("..", "..", "docs", "examples", "manifests", sample)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read sample %s: %v", sample, err)
			}
			if err := ValidateManifestBytes(data); err != nil {
				t.Fatalf("sample %s did not validate: %v", sample, err)
			}
		})
	}
}

func TestBundleSchemaRejectsInvalidManifest(t *testing.T) {
	invalid := []byte(`{"schema_version":"v0.1","target":"desktop","services":[]}`)
	if err := ValidateManifestBytes(invalid); err == nil {
		t.Fatalf("expected validation error for incomplete manifest")
	}
}

func TestBundleValidationCoversSecretAndServiceRules(t *testing.T) {
	validSecret := ManifestSecret{ID: "token", Class: "user_prompt", Target: SecretTarget{Type: "env", Name: "TOKEN"}}
	if err := validateSecret(validSecret); err != nil {
		t.Fatal(err)
	}
	for _, secret := range []ManifestSecret{
		{ID: "bad", Class: "unknown", Target: SecretTarget{Type: "env", Name: "TOKEN"}},
		{ID: "missing-target", Class: "user_prompt"},
		{ID: "bad-target", Class: "user_prompt", Target: SecretTarget{Type: "socket", Name: "x"}},
	} {
		if err := validateSecret(secret); err == nil {
			t.Errorf("expected secret validation failure for %+v", secret)
		}
	}
	base := ServiceEntry{
		ID: "api", Type: "api-binary", Binaries: map[string]ServiceBinary{"linux": {Path: "api"}},
		Health: HealthCheck{Type: "http"}, Readiness: ReadinessCheck{Type: "health_success"},
	}
	if err := validateService(base); err != nil {
		t.Fatal(err)
	}
	for _, svc := range []ServiceEntry{
		{},
		{ID: "bad-type", Type: "mystery", Health: base.Health, Readiness: base.Readiness},
		{ID: "no-binary", Type: "api-binary", Health: base.Health, Readiness: base.Readiness},
		{ID: "bad-binary", Type: "api-binary", Binaries: map[string]ServiceBinary{"linux": {}}, Health: base.Health, Readiness: base.Readiness},
		{ID: "no-health", Type: "api-binary", Binaries: base.Binaries, Readiness: base.Readiness},
		{ID: "no-readiness", Type: "api-binary", Binaries: base.Binaries, Health: base.Health},
	} {
		if err := validateService(svc); err == nil {
			t.Errorf("expected service validation failure for %+v", svc)
		}
	}
	if err := validateService(ServiceEntry{ID: "storage", Type: "embedded-storage"}); err == nil {
		// The service still needs health/readiness even when no binary is required.
		t.Fatal("embedded storage without health/readiness should fail")
	}
}
