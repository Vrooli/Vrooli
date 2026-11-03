package policy_test

import (
	"testing"
	"time"

	"github.com/vrooli/ai-provider-go/policy"
)

func TestCanonicalProviderRuleDigestBindsTermsAndExcludesMutableApprovalState(t *testing.T) {
	terms := policy.ProviderTerms{
		PlanRules: []policy.PlanRule{
			{Plan: "plan-b", AccessTier: policy.AccessTierNoBar, UseClasses: map[policy.UseClass]policy.Permission{policy.UseInApp: policy.PermissionAllowed}, Conditions: []string{"z", "a"}},
			{Plan: "plan-a", AccessTier: policy.AccessTierGrant, UseClasses: map[policy.UseClass]policy.Permission{policy.UseOwnAccount: policy.PermissionAllowed}},
		},
		Evidence:      policy.Evidence{URL: "https://terms.example", SHA256: "evidence-digest"},
		Review:        policy.Review{Status: policy.ReviewProposed, DecisionRef: "proposal-ref"},
		EffectiveFrom: "2026-11-01T00:00:00Z", OnExpiry: "deny",
		Downstream: policy.DownstreamRules{DataCollection: "deny", PinnedHosts: []string{"b.example", "a.example"}},
	}
	models := map[string]policy.ModelRule{" model-b ": {LicenceCommercialUse: boolPtr(false)}}
	first, err := policy.ProviderRuleDigest(" supplier-a ", terms, models)
	if err != nil {
		t.Fatal(err)
	}
	if first != "71910b64c84acb80a2d14698e3cc1b8f543b9c75ed6d8c5ba02d13c02cf3c8a0" {
		t.Fatalf("provider rule golden digest = %s", first)
	}

	terms.Review = policy.Review{Status: policy.ReviewApproved, DecisionRef: "later-human-decision"}
	terms.PlanRules[0].Conditions = []string{"a", "z"}
	terms.PlanRules[0], terms.PlanRules[1] = terms.PlanRules[1], terms.PlanRules[0]
	terms.Downstream.PinnedHosts = []string{"a.example", "b.example"}
	second, err := policy.ProviderRuleDigest("supplier-a", terms, map[string]policy.ModelRule{"model-b": {LicenceCommercialUse: boolPtr(false)}})
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("digest changed for equivalent terms: %s != %s", first, second)
	}

	terms.PlanRules[0].Clause = "changed clause"
	changed, err := policy.ProviderRuleDigest("supplier-a", terms, map[string]policy.ModelRule{"model-b": {LicenceCommercialUse: boolPtr(false)}})
	if err != nil {
		t.Fatal(err)
	}
	if first == changed {
		t.Fatal("digest did not change when a clause changed")
	}
}

func TestCanonicalProviderRuleDigestSortsRulesWithTheSamePlan(t *testing.T) {
	first := policy.ProviderTerms{EffectiveFrom: "2026-11-01", OnExpiry: "deny"}
	second := first
	first.PlanRules = []policy.PlanRule{
		{Plan: "team", AccessTier: policy.AccessTierGrant, Clause: "beta"},
		{Plan: "team", AccessTier: policy.AccessTierNoBar, Clause: "alpha"},
	}
	second.PlanRules = []policy.PlanRule{first.PlanRules[1], first.PlanRules[0]}
	left, err := policy.ProviderRuleDigest("supplier-a", first, nil)
	if err != nil {
		t.Fatal(err)
	}
	right, err := policy.ProviderRuleDigest("supplier-a", second, nil)
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatalf("same-plan rule order changed digest: %s != %s", left, right)
	}
}

func TestCanonicalProviderAgreementDigestBindsScopeButExcludesDecisionReference(t *testing.T) {
	allow := true
	agreement := policy.Agreement{
		Supplier: "supplier-a", Trust: policy.TrustDecision{CustomerCodeAllowed: &allow, Facts: []string{"z", "a"}, DecisionRef: "trust-ref"},
		UseClasses: []policy.UseClass{policy.UseInApp, policy.UseOwnAccount}, Models: []string{"model-b", "model-a"},
		EffectiveAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.FixedZone("local", -4*60*60)),
		ExpiresAt:   time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC), DecisionRef: "decision-ref",
	}
	first, err := policy.ProviderAgreementDigest(agreement)
	if err != nil {
		t.Fatal(err)
	}
	if first != "57c11bcfea4df41f3943eb6f7a9c3b46ddb397e30caf57962081b7df834aa70d" {
		t.Fatalf("provider agreement golden digest = %s", first)
	}
	agreement.DecisionRef = "different-decision"
	agreement.Trust.DecisionRef = "different-trust-decision"
	agreement.Trust.Facts = []string{"a", "z"}
	agreement.UseClasses = []policy.UseClass{policy.UseOwnAccount, policy.UseInApp}
	agreement.Models = []string{"model-a", "model-b"}
	second, err := policy.ProviderAgreementDigest(agreement)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("digest changed for equivalent agreements: %s != %s", first, second)
	}
	agreement.ExpiresAt = agreement.ExpiresAt.Add(time.Second)
	changed, err := policy.ProviderAgreementDigest(agreement)
	if err != nil {
		t.Fatal(err)
	}
	if first == changed {
		t.Fatal("digest did not change when the agreement window changed")
	}
}

func TestCanonicalProviderDecisionRejectsMissingScopeIdentity(t *testing.T) {
	_, err := policy.ProviderRuleDigest(" ", policy.ProviderTerms{}, nil)
	if err == nil {
		t.Fatal("expected missing supplier to fail")
	}
	_, err = policy.ProviderAgreementDigest(policy.Agreement{})
	if err == nil {
		t.Fatal("expected missing supplier to fail")
	}
}

func boolPtr(value bool) *bool { return &value }
