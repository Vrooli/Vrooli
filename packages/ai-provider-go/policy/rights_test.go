package policy

import (
	"math"
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

func TestBehavioralRestrictionsPermitLocalUseWithObligationsOnly(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	artifact := testConditionalRights()
	artifact.Classification = RightsBehavioralRestrictions
	artifact.Conditions = nil
	artifact.ConditionalGrants = nil
	artifact.Obligations = []string{"comply_with_publisher_use_restrictions", "retain_license_notice_if_distributing"}
	ctx := testRightsContext("pre_revenue", "context-v1")

	local := EvaluateRights(&artifact, RightsUseLocalInference, ctx, now)
	if local.Status != RightsAllowed || local.Reason != "behavioral_use_restrictions_apply" || len(local.Obligations) != 2 {
		t.Fatalf("local use did not carry behavioral obligations: %+v", local)
	}
	if err := VerifyRightsDecision(local); err != nil {
		t.Fatalf("behaviorally restricted decision failed verification: %v", err)
	}
	if hosted := EvaluateRights(&artifact, RightsUseHostedInference, ctx, now); hosted.Status != RightsUnknown {
		t.Fatalf("hosted use bypassed required downstream enforcement: %+v", hosted)
	}
	if distributed := EvaluateRights(&artifact, RightsUseRedistributeWeights, ctx, now); distributed.Status != RightsUnknown || distributed.Reason != "distribution_obligations_require_owner_attestation" {
		t.Fatalf("weight distribution bypassed owner attestation: %+v", distributed)
	}
	noObligations := artifact
	noObligations.Obligations = nil
	if got := EvaluateRights(&noObligations, RightsUseLocalInference, ctx, now); got.Status != RightsUnknown {
		t.Fatalf("behavioral license without its obligations was allowed: %+v", got)
	}
}

func TestConditionalGrantsScopeUsesAndCarrySourceReviewTriggers(t *testing.T) {
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	artifact := testConditionalRights()
	artifact.Conditions = nil
	artifact.ConditionalGrants = []ConditionalGrant{{
		ID:   "local-under-threshold-v1",
		Uses: []RightsUse{RightsUseLocalInference},
		Conditions: []RightsCondition{{
			ID: "revenue-v1", Kind: "annual_revenue", Comparator: "lt", Threshold: 10000000,
			Unit: "USD", EntityScope: "operator-group", Interpretation: "re-evaluate when the operator context reaches $10M",
		}},
		Obligations: []string{"retain_license_notice"},
	}}
	artifact.ReviewTriggers = []string{"revenue_context_changed", "license_source_revision_changed"}
	preRevenue := testRightsContext("pre_revenue", "context-v1")
	allowed := EvaluateRights(&artifact, RightsUseLocalInference, preRevenue, now)
	if allowed.Status != RightsAllowed || allowed.Obligations[len(allowed.Obligations)-1] != "retain_license_notice" ||
		len(allowed.ConditionalGrants) != 1 || len(allowed.ReviewTriggers) != 2 {
		t.Fatalf("conditional grant decision lost scoped source facts: %+v", allowed)
	}
	if err := VerifyRightsDecision(allowed); err != nil {
		t.Fatalf("conditional grant decision failed self-verification: %v", err)
	}
	if got := EvaluateRights(&artifact, RightsUseHostedInference, preRevenue, now); got.Status != RightsDenied || got.Reason != "conditional_grant_does_not_cover_use" {
		t.Fatalf("hosted inference inherited the local-only grant: %+v", got)
	}
	overThreshold := preRevenue
	overThreshold.Version = "context-v2"
	overThreshold.RevenueStatus = "at_or_above_threshold"
	if got := EvaluateRights(&artifact, RightsUseLocalInference, overThreshold, now); got.Status != RightsDenied || got.Reason != "revenue_condition_not_satisfied" {
		t.Fatalf("over-threshold context retained grant: %+v", got)
	}
	restored := EvaluateRights(&artifact, RightsUseLocalInference, preRevenue, now)
	if restored.Status != RightsAllowed || restored.DecisionSHA256 != allowed.DecisionSHA256 {
		t.Fatalf("restored pre-revenue context did not restore the same eligible rights decision: %+v", restored)
	}
}

func TestMonthlyActiveUserGrantUsesExactHistoricalPeriodAndFailsClosedWithoutFact(t *testing.T) {
	now := time.Date(2026, 11, 14, 18, 0, 0, 0, time.UTC)
	artifact := testConditionalRights()
	condition := RightsCondition{
		ID: "meta-llama-monthly-active-users-at-most-700m", Kind: "monthly_active_users", Comparator: "lte",
		Threshold: 700_000_000, Unit: "users", EntityScope: "operator-group", Period: "2024-08",
		Interpretation: "License terms use the preceding calendar month's MAU as of the model release date.",
	}
	artifact.Conditions = []RightsCondition{condition}
	artifact.ConditionalGrants = []ConditionalGrant{{
		ID: "llama-use-below-mau-threshold", Uses: []RightsUse{RightsUseLocalInference}, Conditions: []RightsCondition{condition},
	}}
	ctx := testRightsContext("pre_revenue", "context-v1")
	unknown := EvaluateRights(&artifact, RightsUseLocalInference, ctx, now)
	if unknown.Status != RightsUnknown || unknown.Reason != "conditional_grant_condition_unknown" {
		t.Fatalf("missing historical MAU fact = %+v", unknown)
	}
	below := ctx
	below.MonthlyActiveUsers = map[string]string{"2024-08": "at_or_below_threshold", "2024-06": "above_threshold"}
	allowed := EvaluateRights(&artifact, RightsUseLocalInference, below, now)
	if allowed.Status != RightsAllowed || allowed.Reason != "condition_satisfied_by_current_stated_context" {
		t.Fatalf("matching historical MAU fact = %+v", allowed)
	}
	exact := ctx
	exact.MonthlyActiveUsers = map[string]string{"2024-08": "0"}
	exact.MonthlyActiveUsersEvidence = map[string]string{"2024-08": "operator answer LM5-RIGHTS-01"}
	if got := EvaluateRights(&artifact, RightsUseLocalInference, exact, now); got.Status != RightsAllowed {
		t.Fatalf("exact zero historical MAU fact = %+v", got)
	}
	exactAtThreshold := exact
	exactAtThreshold.MonthlyActiveUsers = map[string]string{"2024-08": "700000000"}
	if got := EvaluateRights(&artifact, RightsUseLocalInference, exactAtThreshold, now); got.Status != RightsAllowed {
		t.Fatalf("exact threshold MAU fact must satisfy the inclusive license condition: %+v", got)
	}
	noEvidence := exact
	noEvidence.MonthlyActiveUsersEvidence = nil
	if got := EvaluateRights(&artifact, RightsUseLocalInference, noEvidence, now); got.Status != RightsUnknown {
		t.Fatalf("numeric MAU without evidence must remain unknown: %+v", got)
	}
	exactAbove := exact
	exactAbove.MonthlyActiveUsers = map[string]string{"2024-08": "700000001"}
	if got := EvaluateRights(&artifact, RightsUseLocalInference, exactAbove, now); got.Status != RightsDenied {
		t.Fatalf("exact above-threshold MAU fact = %+v", got)
	}
	above := ctx
	above.MonthlyActiveUsers = map[string]string{"2024-08": "above_threshold", "2024-06": "at_or_below_threshold"}
	denied := EvaluateRights(&artifact, RightsUseLocalInference, above, now)
	if denied.Status != RightsDenied || denied.Reason != "monthly_active_users_condition_not_satisfied" {
		t.Fatalf("above-threshold historical MAU fact = %+v", denied)
	}
	wrongPeriod := ctx
	wrongPeriod.MonthlyActiveUsers = map[string]string{"2024-06": "at_or_below_threshold"}
	if got := EvaluateRights(&artifact, RightsUseLocalInference, wrongPeriod, now); got.Status != RightsUnknown {
		t.Fatalf("fact for another release period was reused: %+v", got)
	}
}

func TestRightsUnknownCanRetainExplicitEvidenceGapWithoutInventedDigest(t *testing.T) {
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	artifact := ArtifactRights{
		ArtifactID:       "image-tools/custom-model/fixture",
		Classification:   RightsUnverified,
		UnverifiedReason: "registry row lacks immutable artifact and license-source revision; commercial-use label is not source proof",
		ReviewTriggers:   []string{"license_source_revision_added", "artifact_digest_captured"},
	}
	decision := EvaluateRights(&artifact, RightsUseLocalInference, testRightsContext("pre_revenue", "context-v1"), now)
	if decision.Status != RightsUnknown || decision.Reason != "rights_evidence_unverified" || decision.UnverifiedReason != artifact.UnverifiedReason {
		t.Fatalf("explicit evidence gap was not preserved as unknown: %+v", decision)
	}
	if err := VerifyRightsDecision(decision); err != nil {
		t.Fatalf("explicit unknown decision failed self-verification: %v", err)
	}
}

func TestConditionalGrantAcceptsAnImmutableLicenseSourceRevisionWithoutDuplicateHash(t *testing.T) {
	now := time.Date(2026, 11, 14, 17, 0, 0, 0, time.UTC)
	artifact := testConditionalRights()
	artifact.Evidence.URL = "https://huggingface.co/MiniMaxAI/MiniMax-Music3/blob/c2509fd6b60d1ae169cd1df27f78a53174ba17e8/LICENSE"
	artifact.Evidence.Revision = "c2509fd6b60d1ae169cd1df27f78a53174ba17e8"
	artifact.Evidence.SHA256 = ""
	artifact.ConditionalGrants = []ConditionalGrant{{
		ID:   "commercial-use-below-threshold",
		Uses: []RightsUse{RightsUseLocalInference},
		Conditions: []RightsCondition{{
			ID: "annual-revenue-v1", Kind: "annual_revenue", Comparator: "lt", Threshold: 20000000,
			Unit: "USD", EntityScope: "operator-group", Interpretation: "prior written authorization is required when yearly product revenue exceeds USD 20 million; re-evaluate conservatively at the threshold",
		}},
	}}
	decision := EvaluateRights(&artifact, RightsUseLocalInference, testRightsContext("pre_revenue", "context-v1"), now)
	if decision.Status != RightsAllowed {
		t.Fatalf("revision-pinned source evidence was not accepted: %+v", decision)
	}
}

func TestAnnualRevenueContextUsesEachLicenseThresholdPrecisely(t *testing.T) {
	now := time.Date(2026, 11, 14, 17, 0, 0, 0, time.UTC)
	artifact := testConditionalRights()
	artifact.ConditionalGrants = []ConditionalGrant{{
		ID:   "commercial-use-below-20m",
		Uses: []RightsUse{RightsUseLocalInference},
		Conditions: []RightsCondition{{
			ID: "annual-revenue-v1", Kind: "annual_revenue", Comparator: "lt", Threshold: 20000000,
			Unit: "USD", EntityScope: "operator-group", Interpretation: "Separate written authorization is required at the USD 20 million boundary.",
		}},
	}}
	for _, tc := range []struct {
		name   string
		amount float64
		want   RightsStatus
	}{
		{name: "below first threshold", amount: 999999, want: RightsAllowed},
		{name: "above one million but below twenty million", amount: 1000001, want: RightsAllowed},
		{name: "at conservative twenty million boundary", amount: 20000000, want: RightsDenied},
		{name: "above twenty million", amount: 20000001, want: RightsDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testRightsContext("pre_revenue", "context-v1")
			ctx.RevenueStatus = "exact_value"
			ctx.AnnualRevenueUSD = &tc.amount
			got := EvaluateRights(&artifact, RightsUseLocalInference, ctx, now)
			if got.Status != tc.want {
				t.Fatalf("annual revenue %.2f evaluated as %s (%s), want %s", tc.amount, got.Status, got.Reason, tc.want)
			}
		})
	}
}

func TestAnnualRevenueContextRejectsNegativeAndNonFiniteAmounts(t *testing.T) {
	now := time.Date(2026, 11, 14, 17, 0, 0, 0, time.UTC)
	artifact := testConditionalRights()
	for _, amount := range []float64{-1, math.NaN(), math.Inf(1)} {
		ctx := testRightsContext("pre_revenue", "context-v1")
		ctx.RevenueStatus = "exact_value"
		ctx.AnnualRevenueUSD = &amount
		if got := EvaluateRights(&artifact, RightsUseLocalInference, ctx, now); got.Status != RightsUnknown {
			t.Errorf("invalid annual revenue %v evaluated as %s (%s), want unknown", amount, got.Status, got.Reason)
		}
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
