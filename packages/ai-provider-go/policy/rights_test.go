package policy

import (
	"strings"
	"testing"
	"time"
)

func TestEvaluateRightsBindsConditionalGrantToArtifactUseAndContext(t *testing.T) {
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	artifact := testConditionalRights()
	context := testRightsContext("pre_revenue", "context-v1")
	allowed := EvaluateRights(&artifact, RightsUseLocalInference, context, now)
	if allowed.Status != RightsAllowed || allowed.Reason != "condition_satisfied_by_current_stated_context" {
		t.Fatalf("pre-revenue local use = %+v", allowed)
	}
	if err := VerifyRightsDecision(allowed); err != nil {
		t.Fatalf("valid decision failed verification: %v", err)
	}
	if len(allowed.Obligations) != 2 || allowed.ArtifactSHA256 != artifact.SHA256 || allowed.ContextVersion != context.Version {
		t.Fatalf("decision did not retain its evidence and obligations: %+v", allowed)
	}

	changed := context
	changed.Version = "context-v2"
	changed.RevenueStatus = "at_or_above_threshold"
	denied := EvaluateRights(&artifact, RightsUseLocalInference, changed, now)
	if denied.Status != RightsDenied || denied.Reason != "revenue_condition_not_satisfied" {
		t.Fatalf("changed context = %+v", denied)
	}
	if denied.DecisionSHA256 == allowed.DecisionSHA256 {
		t.Fatal("changed context retained the previous decision identity")
	}
	changedSameResult := context
	changedSameResult.Version = "context-v1-refresh"
	sameStatus := EvaluateRights(&artifact, RightsUseLocalInference, changedSameResult, now)
	if sameStatus.Status != allowed.Status || sameStatus.DecisionSHA256 == allowed.DecisionSHA256 {
		t.Fatal("context facts/version were not bound when the verdict stayed the same")
	}
	changedArtifact := artifact
	changedArtifact.Obligations = append(changedArtifact.Obligations, "preserve_derivative_lineage")
	artifactChanged := EvaluateRights(&changedArtifact, RightsUseLocalInference, context, now)
	if artifactChanged.DecisionSHA256 == allowed.DecisionSHA256 {
		t.Fatal("rights facts were not bound to the decision")
	}
	allowed.Reason = "edited after evaluation"
	if err := VerifyRightsDecision(allowed); err == nil {
		t.Fatal("modified decision was not detected")
	}
}

func TestEvaluateRightsUnknownAndDistributionObligationsFailClosed(t *testing.T) {
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	artifact := testConditionalRights()
	ctx := testRightsContext("pre_revenue", "context-v1")

	unknown := EvaluateRights(&artifact, RightsUseLocalInference, RightsContext{}, now)
	if unknown.Status != RightsUnknown || unknown.Reason != "rights_facts_incomplete" {
		t.Fatalf("missing context = %+v", unknown)
	}
	unknownRights := artifact
	unknownRights.SHA256 = ""
	if got := EvaluateRights(&unknownRights, RightsUseLocalInference, ctx, now); got.Status != RightsUnknown {
		t.Fatalf("artifact without exact digest = %+v", got)
	}
	if got := EvaluateRights(&artifact, RightsUseRedistributeWeights, ctx, now); got.Status != RightsUnknown || got.Reason != "distribution_obligations_require_owner_attestation" {
		t.Fatalf("unattested distribution = %+v", got)
	}
}

func TestEvaluateRightsDeniesProhibitedAndUnknownUse(t *testing.T) {
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	artifact := testConditionalRights()
	ctx := testRightsContext("pre_revenue", "context-v1")
	prohibited := artifact
	prohibited.Classification = RightsProhibited
	if got := EvaluateRights(&prohibited, RightsUseLocalInference, ctx, now); got.Status != RightsDenied {
		t.Fatalf("prohibited grant = %+v", got)
	}
	if got := EvaluateRights(&artifact, RightsUse("invented"), ctx, now); got.Status != RightsUnknown {
		t.Fatalf("unknown use = %+v", got)
	}
}

func TestRequireCurrentRightsDecisionRejectsChangedArtifactContextAndPolicyRevision(t *testing.T) {
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	artifact := testConditionalRights()
	ctx := testRightsContext("pre_revenue", "context-v1")
	decision := EvaluateRights(&artifact, RightsUseLocalInference, ctx, now)
	if err := RequireCurrentRightsDecision(decision, &artifact, RightsUseLocalInference, ctx); err != nil {
		t.Fatalf("current allowed decision rejected: %v", err)
	}

	changedContext := ctx
	changedContext.Version = "context-v2"
	if err := RequireCurrentRightsDecision(decision, &artifact, RightsUseLocalInference, changedContext); err == nil {
		t.Fatal("previously allowed decision survived a context-version change with unchanged outcome")
	}
	changedArtifact := artifact
	changedArtifact.Revision = "weights-rev-2"
	if err := RequireCurrentRightsDecision(decision, &changedArtifact, RightsUseLocalInference, ctx); err == nil {
		t.Fatal("previously allowed decision survived an artifact policy revision change")
	}
	changedUse := RightsUseRedistributeWeights
	if err := RequireCurrentRightsDecision(decision, &artifact, changedUse, ctx); err == nil {
		t.Fatal("local-inference decision was reused for distribution")
	}
	if err := RequireCurrentRightsDecision(decision, &artifact, RightsUseLocalInference, RightsContext{}); err == nil {
		t.Fatal("previously allowed decision survived missing trusted context")
	}
	deniedContext := ctx
	deniedContext.RevenueStatus = "at_or_above_threshold"
	denied := EvaluateRights(&artifact, RightsUseLocalInference, deniedContext, now)
	if err := RequireCurrentRightsDecision(denied, &artifact, RightsUseLocalInference, deniedContext); err == nil {
		t.Fatal("current denied decision was admitted")
	}
	unknown := EvaluateRights(&artifact, RightsUseLocalInference, RightsContext{}, now)
	if err := RequireCurrentRightsDecision(unknown, &artifact, RightsUseLocalInference, RightsContext{}); err == nil {
		t.Fatal("current unknown decision was admitted")
	}
}

func TestRuntimeArtifactIdentityMatchesExactOwnerMapping(t *testing.T) {
	rights := testConditionalRights()
	digest := strings.Repeat("c", 64)
	binding := RuntimeArtifactIdentity{
		Provider: RuntimeProviderOllama, Model: "qwen.fixture", ArtifactID: rights.ArtifactID,
		ArtifactRevision: rights.Revision, DigestKind: RuntimeDigestOllamaModelContents,
		Digest: digest, Source: RuntimeSourceOllamaTags, ObservedAt: "2026-11-11",
	}
	if !RuntimeArtifactIdentityMatches(&binding, &rights, RuntimeProviderOllama, "qwen.fixture", digest) {
		t.Fatal("exact owner mapping was rejected")
	}
	if RuntimeArtifactIdentityMatches(&binding, &rights, RuntimeProviderOllama, "qwen.other", digest) {
		t.Fatal("mapping for a different model was accepted")
	}
	if RuntimeArtifactIdentityMatches(&binding, &rights, RuntimeProviderOllama, "qwen.fixture", strings.Repeat("d", 64)) {
		t.Fatal("mapping with a different observed digest was accepted")
	}
	changedRights := rights
	changedRights.Revision = "weights-rev-2"
	if RuntimeArtifactIdentityMatches(&binding, &changedRights, RuntimeProviderOllama, "qwen.fixture", digest) {
		t.Fatal("mapping for an obsolete artifact revision was accepted")
	}
}

func TestRuntimeArtifactIdentityMatchesOllayaManifestDomainOnly(t *testing.T) {
	rights := testConditionalRights()
	digest := strings.Repeat("e", 64)
	binding := RuntimeArtifactIdentity{
		Provider: RuntimeProviderOllaya, Model: "winnow:e4b", ArtifactID: rights.ArtifactID,
		ArtifactRevision: rights.Revision, DigestKind: RuntimeDigestOllayaManifest,
		Digest: digest, Source: RuntimeSourceOllayaTags, ObservedAt: "2026-11-11",
	}
	if !RuntimeArtifactIdentityMatches(&binding, &rights, RuntimeProviderOllaya, "winnow:e4b", digest) {
		t.Fatal("exact Ollaya manifest mapping was rejected")
	}
	binding.DigestKind = RuntimeDigestOllamaModelContents
	if RuntimeArtifactIdentityMatches(&binding, &rights, RuntimeProviderOllaya, "winnow:e4b", digest) {
		t.Fatal("Ollama content digest domain was accepted for Ollaya")
	}
	binding.DigestKind = RuntimeDigestOllayaManifest
	binding.Source = RuntimeSourceOllamaTags
	if RuntimeArtifactIdentityMatches(&binding, &rights, RuntimeProviderOllaya, "winnow:e4b", digest) {
		t.Fatal("Ollama tags source was accepted for Ollaya")
	}
}

func TestRightsDigestFormat(t *testing.T) {
	got := EvaluateRights(nil, RightsUseLocalInference, RightsContext{}, time.Time{}).DecisionSHA256
	if len(got) != 64 || strings.Trim(got, "0123456789abcdef") != "" {
		t.Fatalf("decision digest %q is not lowercase SHA-256", got)
	}
}

func testConditionalRights() ArtifactRights {
	return ArtifactRights{
		ArtifactID:     "fixture/model",
		Revision:       "weights-rev-1",
		SHA256:         strings.Repeat("a", 64),
		LicenseID:      "LicenseRef-test-conditional",
		Classification: RightsConditional,
		Evidence:       RightsEvidence{URL: "https://example.invalid/LICENSE", Revision: "license-rev-1", SHA256: strings.Repeat("b", 64), ObservedAt: "2026-11-10"},
		Conditions:     []RightsCondition{{ID: "revenue-v1", Kind: "annual_revenue", Comparator: "lt", Threshold: 10000000, Unit: "USD", EntityScope: "operator-group", Interpretation: "re-evaluate at threshold"}},
		Obligations:    []string{"retain_license_and_notices", "mark_modifications"},
	}
}

func testRightsContext(revenue, version string) RightsContext {
	return RightsContext{ID: "operator-business-context/current", Version: version, EntityScope: "operator-group", RevenueStatus: revenue, AsOf: "2026-11-10", Source: "operator_statement"}
}
