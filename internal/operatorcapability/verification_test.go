package operatorcapability

import (
	"context"
	"strings"
	"testing"
	"time"
)

type verifyingFixtureProvider struct {
	descriptorFixtureProvider
	receipts []EvidenceReference
}

func (p verifyingFixtureProvider) Verify(context.Context, VerificationRequest) ([]EvidenceReference, error) {
	return append([]EvidenceReference(nil), p.receipts...), nil
}

func TestRegistryVerifyBindsEvidenceToCredentialAndTarget(t *testing.T) {
	descriptor := fixtureDescriptor("fixture/verify", "Verification fixture", SensitivitySecret, nil)
	descriptor.Evidence = EvidenceContract{Kinds: []string{"fixture"}, RequiredFields: []string{"credential_ref", "target_id", "operation", "observed_at", "expires_at", "verified"}, SecretFree: true}
	now := time.Now().UTC()
	provider := verifyingFixtureProvider{
		descriptorFixtureProvider: descriptorFixtureProvider{descriptor: descriptor},
		receipts: []EvidenceReference{{
			SchemaVersion: EvidenceSchemaVersion, Kind: "fixture", ArtifactIdentity: "fixture-evidence-1", TargetID: "host-1", Operation: "readiness-check",
			CredentialRef: &CredentialEvidenceRef{LogicalID: "vrooli/test", Field: "token", Version: "authority-version-4"}, ObservedAt: now, ExpiresAt: now.Add(15 * time.Minute), Status: "exercised", Verified: true,
		}},
	}
	registry, err := NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}
	receipts, err := registry.Verify(context.Background(), VerificationRequest{
		CapabilityID: "fixture/verify", CredentialRef: provider.receipts[0].CredentialRef, TargetID: "host-1", Operation: "readiness-check",
		Effect: EffectBudget{Class: EffectReadOnly},
	})
	if err != nil || len(receipts) != 1 || receipts[0].CapabilityID != "fixture/verify" {
		t.Fatalf("Verify() = %+v, %v", receipts, err)
	}
	if !receipts[0].FreshAt(now.Add(time.Minute)) || receipts[0].FreshAt(now.Add(16*time.Minute)) {
		t.Fatalf("receipt freshness did not respect its observed/expiry window: %+v", receipts[0])
	}

	provider.receipts[0].TargetID = "other-host"
	registry, _ = NewRegistry(provider)
	if _, err := registry.Verify(context.Background(), VerificationRequest{CapabilityID: "fixture/verify", CredentialRef: provider.receipts[0].CredentialRef, TargetID: "host-1", Operation: "readiness-check", Effect: EffectBudget{Class: EffectReadOnly}}); err == nil {
		t.Fatal("verification accepted evidence for a different target")
	}
}

func TestVerificationBoundsAndProbeURLSafety(t *testing.T) {
	for _, raw := range []string{
		"http://provider.example.test/check", "https://user:password@provider.example.test/check", "https://localhost/check",
		"https://127.0.0.1/check", "https://10.0.0.3/check", "https://provider.example.test/check#fragment",
	} {
		if err := ValidateProbeURL(raw); err == nil {
			t.Fatalf("ValidateProbeURL(%q) accepted an unsafe endpoint", raw)
		}
	}
	if err := ValidateProbeURL("https://provider.example.test/check"); err != nil {
		t.Fatalf("valid provider URL rejected: %v", err)
	}
	request := VerificationRequest{CapabilityID: "fixture/check", TargetID: "host-1", Operation: "check", Context: map[string]string{"api_token": "redacted"}, Effect: EffectBudget{Class: EffectBoundedWrite, MaxOperations: 17}}
	if err := request.Validate(); err == nil || !strings.Contains(err.Error(), "secret metadata") {
		t.Fatalf("sensitive verification context was not rejected: %v", err)
	}
	request.Context = map[string]string{}
	if err := request.Validate(); err == nil || !strings.Contains(err.Error(), "effect budget") {
		t.Fatalf("unbounded effect budget was not rejected: %v", err)
	}
}
