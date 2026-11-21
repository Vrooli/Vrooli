package deployment

import (
	"testing"

	"scenario-to-cloud/domain"
)

func TestValidateHandoffRejectsNonLocalProfilePath(t *testing.T) {
	err := validateHandoff(domain.LocalCredentialHandoff{
		ConsumerPath: "/api/v1/admin/remote-profiles/../../secrets",
		APIBase:      "https://example.test/api/v1",
	})
	if err == nil {
		t.Fatal("expected non-profile path to be rejected")
	}
}

func TestHandoffSecretResolvesDescriptorField(t *testing.T) {
	manifest := domain.CloudManifest{Secrets: &domain.ManifestSecrets{BundleSecrets: []domain.BundleSecretPlan{{
		ID: "generated-plan-id", Target: domain.BundleSecretTarget{Name: "LPBS_SERVICE_SECRET"},
		Descriptor: &domain.DescriptorAddress{LogicalID: "vrooli/app", Field: "service-secret"},
	}}}}
	if got := resolveHandoffSecret(manifest, "service-secret", map[string]string{"generated-plan-id": "in-memory-value"}); got != "in-memory-value" {
		t.Fatalf("resolveHandoffSecret() = %q, want generated value", got)
	}
}
