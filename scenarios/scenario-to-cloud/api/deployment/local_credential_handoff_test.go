package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/lpbsidentity"
	"scenario-to-cloud/bundle"
	"scenario-to-cloud/credentials"
	"scenario-to-cloud/domain"
	manifestschema "scenario-to-cloud/manifest"
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

func TestValidateHandoffSeparatesScopedAuthorizationFromLegacyBridge(t *testing.T) {
	valid := domain.LocalCredentialHandoff{
		ConsumerPath: "/api/v1/admin/remote-profiles",
		APIBase:      "https://example.test/api/v1",
		AuthDescriptor: domain.DescriptorAddress{
			LogicalID: lpbsidentity.ScenarioToCloudRemoteProfiles.CredentialIdentity("scenario-to-cloud"),
			Field:     "credential",
		},
		LegacyAuthDescriptor: &domain.DescriptorAddress{LogicalID: "vrooli/landing-page-business-suite", Field: "service-secret"},
	}
	if err := validateHandoff(valid); err != nil {
		t.Fatalf("valid scoped plus migration descriptor rejected: %v", err)
	}
	valid.AuthDescriptor = domain.DescriptorAddress{LogicalID: "vrooli/landing-page-business-suite", Field: "service-secret"}
	if err := validateHandoff(valid); err == nil {
		t.Fatal("legacy credential must not remain the primary authorization descriptor")
	}
	valid.AuthDescriptor = domain.DescriptorAddress{
		LogicalID: lpbsidentity.ScenarioToCloudRemoteProfiles.CredentialIdentity("scenario-to-cloud"),
		Field:     "credential",
	}
	valid.LegacyAuthDescriptor = &domain.DescriptorAddress{LogicalID: "vrooli/landing-page-business-suite", Field: "remote-service-secret"}
	if err := validateHandoff(valid); err == nil {
		t.Fatal("destination remote-service-secret must not be used as local auth fallback")
	}
}

func TestLPBSServiceAuthorizationPrefersScopedAndOnlyFallsBackWhenUnissued(t *testing.T) {
	sentinel := "synthetic-private-sentinel"
	t.Run("scoped role preferred", func(t *testing.T) {
		header, err := resolveLPBSServiceAuthorization(context.Background(), true, true, func(context.Context) (string, error) {
			return "Bearer scoped-lpbs-token", nil
		}, sentinel)
		if err != nil || header != "Bearer scoped-lpbs-token" {
			t.Fatalf("scoped authorization = %q, %v", header, err)
		}
	})
	t.Run("unissued scoped role permits migration fallback", func(t *testing.T) {
		header, err := resolveLPBSServiceAuthorization(context.Background(), true, false, nil, sentinel)
		if err != nil || header != "Bearer "+sentinel {
			t.Fatalf("fallback authorization = %q, %v", header, err)
		}
	})
	t.Run("authority outage does not fall back", func(t *testing.T) {
		header, err := resolveLPBSServiceAuthorization(context.Background(), false, false, nil, sentinel)
		if err == nil || header != "" || strings.Contains(err.Error(), sentinel) {
			t.Fatalf("outage result = %q, %v", header, err)
		}
	})
	t.Run("exchange failure does not fall back", func(t *testing.T) {
		header, err := resolveLPBSServiceAuthorization(context.Background(), true, true, func(context.Context) (string, error) {
			return "", errors.New(sentinel)
		}, sentinel)
		if err == nil || header != "" || strings.Contains(err.Error(), sentinel) {
			t.Fatalf("exchange failure = %q, %v", header, err)
		}
	})
}

func TestHandoffSecretResolvesDescriptorField(t *testing.T) {
	plan := domain.BundleSecretPlan{
		ID:       lpbsRemoteProfileSecretPlanID,
		Class:    domain.SecretClassPerInstallGenerated,
		Required: true,
		Target:   domain.BundleSecretTarget{Type: "env", Name: "LPBS_SERVICE_SECRET"},
		Descriptor: &domain.DescriptorAddress{
			LogicalID: "vrooli/landing-page-business-suite",
			Field:     "service-secret",
		},
		Generator: map[string]interface{}{"type": "random", "length": float64(64), "charset": "hex"},
	}
	handoff := domain.LocalCredentialHandoff{SecretID: lpbsRemoteProfileSecretPlanID}
	manifest := domain.CloudManifest{Secrets: &domain.ManifestSecrets{BundleSecrets: []domain.BundleSecretPlan{plan}}}
	if err := validateHandoffSecretPlan(manifest, handoff); err != nil {
		t.Fatalf("valid target-bound credential plan rejected: %v", err)
	}
	if got := resolveHandoffSecret(manifest, handoff.SecretID, map[string]string{handoff.SecretID: "in-memory-value"}); got != "in-memory-value" {
		t.Fatalf("resolveHandoffSecret() = %q, want generated value", got)
	}
	if got := resolveHandoffSecret(manifest, handoff.SecretID, map[string]string{"LPBS_SERVICE_SECRET": "wrong-alias"}); got != "" {
		t.Fatalf("resolver accepted an env alias for the generated handoff value: %q", got)
	}
	if got := resolveHandoffSecret(manifest, "service-secret", map[string]string{handoff.SecretID: "in-memory-value"}); got != "" {
		t.Fatalf("resolver accepted old shared-secret ID: %q", got)
	}
}

func TestHandoffSecretPlanMustBeTargetBoundAndGenerated(t *testing.T) {
	base := domain.BundleSecretPlan{
		ID:       lpbsRemoteProfileSecretPlanID,
		Class:    domain.SecretClassPerInstallGenerated,
		Required: true,
		Target:   domain.BundleSecretTarget{Type: "env", Name: "LPBS_SERVICE_SECRET"},
		Descriptor: &domain.DescriptorAddress{
			LogicalID: "vrooli/landing-page-business-suite",
			Field:     "service-secret",
		},
		Generator: map[string]interface{}{"type": "random", "length": float64(64), "charset": "hex"},
	}
	handoff := domain.LocalCredentialHandoff{SecretID: lpbsRemoteProfileSecretPlanID}
	for _, tc := range []struct {
		name   string
		mutate func(*domain.BundleSecretPlan)
	}{
		{name: "operator prompt", mutate: func(p *domain.BundleSecretPlan) { p.Class = domain.SecretClassUserPrompt }},
		{name: "wrong target", mutate: func(p *domain.BundleSecretPlan) { p.Target.Name = "OTHER_SECRET" }},
		{name: "wrong owner", mutate: func(p *domain.BundleSecretPlan) { p.Descriptor.LogicalID = "vrooli/other" }},
		{name: "wrong field", mutate: func(p *domain.BundleSecretPlan) { p.Descriptor.Field = "remote-service-secret" }},
		{name: "weak generator", mutate: func(p *domain.BundleSecretPlan) { p.Generator["length"] = float64(16) }},
		{name: "duplicate ID", mutate: func(p *domain.BundleSecretPlan) {}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first, second := base, base
			tc.mutate(&first)
			plans := []domain.BundleSecretPlan{first}
			if tc.name == "duplicate ID" {
				plans = append(plans, second)
			}
			manifest := domain.CloudManifest{Secrets: &domain.ManifestSecrets{BundleSecrets: plans}}
			if err := validateHandoffSecretPlan(manifest, handoff); err == nil {
				t.Fatal("invalid destination credential plan accepted")
			}
		})
	}
}

func TestHandoffDestinationCredentialIsBoundToDeploymentWithoutReceiptValue(t *testing.T) {
	plan := domain.BundleSecretPlan{
		ID:       lpbsRemoteProfileSecretPlanID,
		Class:    domain.SecretClassPerInstallGenerated,
		Required: true,
		Target:   domain.BundleSecretTarget{Type: "env", Name: "LPBS_SERVICE_SECRET"},
		Descriptor: &domain.DescriptorAddress{
			LogicalID: "vrooli/landing-page-business-suite",
			Field:     "service-secret",
		},
	}
	const deploymentID = "lpbs-target-42"
	bindings, err := credentials.PlanBindings(credentials.PlanInputs{DeploymentID: deploymentID, Plans: []domain.BundleSecretPlan{plan}})
	if err != nil || len(bindings) != 1 {
		t.Fatalf("PlanBindings() = %v, %v", bindings, err)
	}
	binding := bindings[0]
	if binding.DeploymentID != deploymentID || binding.Target != plan.Target || binding.Descriptor != (domain.CredentialDescriptor{LogicalID: "vrooli/landing-page-business-suite", Field: "service-secret"}) {
		t.Fatalf("binding lost target scope: %+v", binding)
	}
	const sentinel = "synthetic-secret-must-not-enter-receipt"
	valueMap := bindingValues(deploymentID, []domain.BundleSecretPlan{plan}, map[string]string{plan.ID: sentinel})
	if valueMap[binding.ID] != sentinel {
		t.Fatal("generated value was not associated with its target binding")
	}
	receipt, err := json.Marshal(binding)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(receipt), sentinel) {
		t.Fatal("credential value leaked into binding evidence")
	}
}

func TestLPBSManifestWiresScopedAuthToDistinctTargetBoundRemoteSecretPlan(t *testing.T) {
	root, err := bundle.FindRepoRootFromCWD()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "scenarios", "landing-page-business-suite", "cloud-manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest domain.CloudManifest
	var rawManifest map[string]interface{}
	if err := json.Unmarshal(raw, &rawManifest); err != nil {
		t.Fatalf("decode LPBS cloud manifest schema input: %v", err)
	}
	if issues := manifestschema.ValidateStructure(rawManifest); len(issues) != 0 {
		t.Fatalf("LPBS cloud manifest violates scenario-to-cloud schema: %+v", issues)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode LPBS cloud manifest: %v", err)
	}
	if len(manifest.LocalCredentialHandoffs) != 1 {
		t.Fatalf("handoffs=%d, want one LPBS profile handoff", len(manifest.LocalCredentialHandoffs))
	}
	handoff := manifest.LocalCredentialHandoffs[0]
	if err := validateHandoff(handoff); err != nil {
		t.Fatalf("manifest auth descriptors invalid: %v", err)
	}
	if err := validateHandoffSecretPlan(manifest, handoff); err != nil {
		t.Fatalf("manifest destination credential plan invalid: %v", err)
	}
	if handoff.AuthDescriptor.Field == "service-secret" || handoff.LegacyAuthDescriptor == nil || handoff.SecretID == "service-secret" {
		t.Fatal("manifest conflates local auth, legacy auth, and remote destination secret")
	}
	if value := resolveHandoffSecret(manifest, handoff.SecretID, map[string]string{handoff.SecretID: "synthetic-destination-secret"}); value != "synthetic-destination-secret" {
		t.Fatal("manifest resolver did not select the exact generated destination plan ID")
	}
}
