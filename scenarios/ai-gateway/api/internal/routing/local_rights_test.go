package routing

import (
	"strings"
	"testing"
	"time"

	"github.com/vrooli/ai-provider-go/policy"
)

func TestGatewayLocalRightsAdmission(t *testing.T) {
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	model := policy.Model{Rights: &policy.ArtifactRights{
		// The test pins a synthetic weight artifact; research has a pinned
		// license source but no acquired LFM weight digest in this checkout.
		ArtifactID:     "fixture/liquidai-lfm3-candidate",
		Revision:       "fixture-weights-revision",
		SHA256:         strings.Repeat("a", 64),
		LicenseID:      "LicenseRef-LFM-Open-1.0",
		Classification: policy.RightsConditional,
		Evidence:       policy.RightsEvidence{URL: "https://huggingface.co/LiquidAI/LFM3-1.7B/blob/1da06bec6a819da10b54c67e5c90972b46d81871/LICENSE", Revision: "1da06bec6a819da10b54c67e5c90972b46d81871", SHA256: "4d28ca14dedc0b3d0fcc2b3339f0e79931faa33874f3d24f522183a8fc70068c", ObservedAt: "2026-11-10"},
		Conditions:     []policy.RightsCondition{{ID: "lfm-commercial-revenue-v1", Kind: "annual_revenue", Comparator: "lt", Threshold: 10000000, Unit: "USD", EntityScope: "operator-group", Interpretation: "re-evaluate conservatively at $10M due to source boundary ambiguity"}},
		Obligations:    []string{"retain_license_and_notices", "mark_modifications", "include_NOTICE_attribution_when_applicable"},
	}}
	ctx := policy.RightsContext{ID: "operator-business-context/current", Version: "2026-11-10", EntityScope: "operator-group", RevenueStatus: "pre_revenue", AsOf: "2026-11-10", Source: "operator_statement"}
	decision := EvaluateLocalModelRights(model, policy.RightsUseLocalInference, ctx, now)
	if decision.Status != policy.RightsAllowed {
		t.Fatalf("stated current local use decision = %+v", decision)
	}
	if err := RequireLocalRight(decision, model, policy.RightsUseLocalInference, ctx); err != nil {
		t.Fatalf("allowed local decision rejected: %v", err)
	}

	staleContext := ctx
	ctx.Version = "2026-11-11"
	ctx.RevenueStatus = "at_or_above_threshold"
	if err := RequireLocalRight(decision, model, policy.RightsUseLocalInference, ctx); err == nil {
		t.Fatal("previously allowed decision survived changed current context")
	}
	denied := EvaluateLocalModelRights(model, policy.RightsUseLocalInference, ctx, now)
	if denied.Status != policy.RightsDenied {
		t.Fatalf("threshold context decision = %+v", denied)
	}
	if err := RequireLocalRight(denied, model, policy.RightsUseLocalInference, ctx); err == nil {
		t.Fatal("denied local decision admitted")
	}
	if err := RequireLocalRight(decision, model, policy.RightsUseLocalInference, staleContext); err != nil {
		t.Fatalf("decision failed against its original context: %v", err)
	}
	sameStatusChangedContext := staleContext
	sameStatusChangedContext.Version = "new-version-same-status"
	if err := RequireLocalRight(decision, model, policy.RightsUseLocalInference, sameStatusChangedContext); err == nil {
		t.Fatal("decision survived changed context version with unchanged verdict")
	}
	if err := RequireLocalRight(decision, model, policy.RightsUseHostedInference, staleContext); err == nil {
		t.Fatal("decision evaluated for a different use was accepted")
	}
	if err := RequireLocalRight(decision, model, policy.RightsUseLocalInference, policy.RightsContext{}); err == nil {
		t.Fatal("decision was accepted with missing current organization context")
	}

	changedArtifact := model
	changedArtifact.Rights = &policy.ArtifactRights{ArtifactID: "fixture/replaced-artifact"}
	if err := RequireLocalRight(decision, changedArtifact, policy.RightsUseLocalInference, staleContext); err == nil {
		t.Fatal("previously allowed decision survived changed current artifact")
	}
	unknown := EvaluateLocalModelRights(policy.Model{}, policy.RightsUseLocalInference, ctx, now)
	if unknown.Status != policy.RightsUnknown {
		t.Fatalf("model without rights = %+v", unknown)
	}
	if err := RequireLocalRight(unknown, policy.Model{}, policy.RightsUseLocalInference, ctx); err == nil {
		t.Fatal("unknown local decision admitted")
	}
}
