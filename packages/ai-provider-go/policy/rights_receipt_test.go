package policy

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRightsDecisionReceiptPersistsAndRechecksExactInputs(t *testing.T) {
	expectation := receiptExpectationFixture()
	decisionAt := time.Date(2026, 11, 11, 21, 0, 0, 0, time.UTC)
	receipt := NewRightsDecisionReceipt(expectation, decisionAt)
	if receipt.Eligibility != RightsAllowed || receipt.Decision.Status != RightsAllowed {
		t.Fatalf("receipt=%+v; want an allowed test fixture", receipt)
	}
	if err := VerifyRightsDecisionReceipt(receipt, expectation); err != nil {
		t.Fatalf("verify current inputs: %v", err)
	}
	if err := RequireAllowedRightsDecisionReceipt(receipt); err != nil {
		t.Fatalf("require allowed: %v", err)
	}

	encoded, err := receipt.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseRightsDecisionReceipt(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.ReceiptSHA256 != receipt.ReceiptSHA256 || !bytes.Contains(encoded, []byte(`"schema_version":"rights-decision/v1"`)) {
		t.Fatalf("canonical roundtrip mismatch: %s", encoded)
	}

	changedContext := expectation
	changedContext.Context.Version = "operator-context-v2"
	if err := VerifyRightsDecisionReceipt(parsed, changedContext); err == nil {
		t.Fatal("changed trusted context must invalidate the persisted receipt")
	}
	changedPolicy := expectation
	changedPolicy.ArtifactRights = cloneArtifactRights(expectation.ArtifactRights)
	changedPolicy.ArtifactRights.Obligations = append(changedPolicy.ArtifactRights.Obligations, "attribution")
	if err := VerifyRightsDecisionReceipt(parsed, changedPolicy); err == nil {
		t.Fatal("changed policy rights facts must invalidate the persisted receipt")
	}
	changedRuntime := expectation
	changedRuntime.RuntimeIdentity = cloneRuntimeIdentity(expectation.RuntimeIdentity)
	changedRuntime.RuntimeIdentity.Digest = strings.Repeat("f", 64)
	if err := VerifyRightsDecisionReceipt(parsed, changedRuntime); err == nil {
		t.Fatal("changed runtime artifact identity must invalidate the persisted receipt")
	}
}

func TestRightsDecisionReceiptPreservesUnknownAndRefusesExecution(t *testing.T) {
	expectation := receiptExpectationFixture()
	expectation.Context.EntityScope = ""
	receipt := NewRightsDecisionReceipt(expectation, time.Date(2026, 11, 11, 21, 0, 0, 0, time.UTC))
	if receipt.Eligibility != RightsUnknown || receipt.Decision.Status != RightsUnknown {
		t.Fatalf("receipt=%+v; absent entity_scope must remain unknown", receipt)
	}
	if err := VerifyRightsDecisionReceipt(receipt, expectation); err != nil {
		t.Fatalf("unknown receipt remains internally/current-input verifiable: %v", err)
	}
	if err := RequireAllowedRightsDecisionReceipt(receipt); err == nil {
		t.Fatal("unknown receipt must not authorize inference")
	}
}

func TestRightsDecisionReceiptTamperingIsDetected(t *testing.T) {
	receipt := NewRightsDecisionReceipt(receiptExpectationFixture(), time.Date(2026, 11, 11, 21, 0, 0, 0, time.UTC))
	receipt.Context.Version = "tampered"
	if err := VerifyRightsDecisionReceiptSelf(receipt); err == nil {
		t.Fatal("self-hash must detect receipt mutation")
	}
}

func receiptExpectationFixture() RightsDecisionReceiptExpectation {
	digest := strings.Repeat("b", 64)
	artifact := &ArtifactRights{
		ArtifactID: "fixture/model", Revision: "rev-1", SHA256: digest,
		LicenseID: "LicenseRef-TestFixture", Classification: RightsUnrestricted,
		Evidence: RightsEvidence{URL: "https://example.invalid/license", Revision: "license-v1", SHA256: strings.Repeat("a", 64), ObservedAt: "2026-11-11"},
	}
	identity := &RuntimeArtifactIdentity{
		Provider: RuntimeProviderOllaya, Model: "fixture:model", ArtifactID: artifact.ArtifactID,
		ArtifactRevision: artifact.Revision, DigestKind: RuntimeDigestOllayaManifest,
		Digest: digest, Source: RuntimeSourceOllayaTags, ObservedAt: "2026-11-11",
	}
	return RightsDecisionReceiptExpectation{
		Role: "decision.fixture", Provider: RuntimeProviderOllaya, Model: "fixture:model",
		Use: RightsUseLocalInference, PolicySchemaVersion: "fixture-policy/v1", ArtifactRights: artifact, RuntimeIdentity: identity,
		Context: RightsContext{ID: "fixture-context", Version: "v1", EntityScope: "fixture-entity", RevenueStatus: "pre_revenue", AsOf: "2026-11-11", Source: "test_fixture"},
	}
}
