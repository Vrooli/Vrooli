package policy

import (
	"strings"
	"testing"
	"time"
)

func rulesFixture(status ReviewStatus) (Request, Account, []RuleRecord, []Agreement, []Route) {
	digest := strings.Repeat("a", 64)
	contract := FeatureContract{ID: "audio.summarize", MaxInputBytes: 1024, MaxOutputBytes: 256, MaxUserTurns: 1, PinnedInstructionDigest: digest}
	invocation := FeatureInvocation{InstructionHash: digest, Endpoint: "chat", RequiredCapabilities: []string{"text"}, UserTurns: 1, SchemaValid: true, InputBytes: 40, OutputBytes: 80}
	check, err := CheckFeatureInvocation(contract, invocation)
	if err != nil {
		panic(err)
	}
	request, err := NewEligibleRequest(check, "model-a", "balanced", "private", "", 100)
	if err != nil {
		panic(err)
	}
	terms := ProviderTerms{
		PlanRules: []PlanRule{{Plan: "pay_as_you_go", AccessTier: AccessTierGrant, UseClasses: map[UseClass]Permission{UseBoundedFeature: PermissionAllowed, UseInApp: PermissionNeedsWrittenApproval}}},
		Evidence:  Evidence{URL: "https://example.invalid/terms"}, Review: Review{Status: status, DecisionRef: decisionFor(status)}, EffectiveFrom: "2026-01-01", ExpiresAt: "2027-01-01", OnExpiry: "deny", Downstream: DownstreamRules{DataCollection: "deny"},
	}
	record, err := NewVerifiedRuleRecord("supplier-a", terms, nil, testDecisionVerifier)
	if err != nil {
		panic(err)
	}
	return request, Account{Plan: "pay_as_you_go", BalanceStanding: true}, []RuleRecord{record}, nil, []Route{{ID: "route-a", Supplier: "supplier-a", Plan: "pay_as_you_go", Model: "model-a", Endpoint: "chat", Capabilities: []string{"text"}, Profile: "balanced", QuoteRef: "quote-fixture", QuotedCredits: 25}}
}

func decisionFor(status ReviewStatus) string {
	if status == ReviewApproved {
		return "notification-hub-fixture-decision"
	}
	return ""
}
func testDecisionVerifier(reference string) bool {
	return reference == "notification-hub-fixture-decision" || reference == "trust-decision"
}

func TestEligibleDeniesEverythingWithZeroApprovedRecords(t *testing.T) {
	request, account, records, agreements, routes := rulesFixture(ReviewPending)
	decision := Eligible(request, account, records, agreements, routes, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if len(decision.Routes) != 0 {
		t.Fatalf("pending records returned routes: %+v", decision.Routes)
	}
	if len(decision.Refusals) != 1 || decision.Refusals[0].Reason != "unapproved" {
		t.Fatalf("unexpected refusals: %+v", decision.Refusals)
	}
}

func TestEligibleAllowsOnlyApprovedCurrentPlanClass(t *testing.T) {
	request, account, records, agreements, routes := rulesFixture(ReviewApproved)
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	decision := Eligible(request, account, records, agreements, routes, now)
	if len(decision.Routes) != 1 || decision.Routes[0].ID != "route-a" {
		t.Fatalf("expected one eligible route, got %+v", decision)
	}
	request.useClass = UseInApp
	decision = Eligible(request, account, records, agreements, routes, now)
	if len(decision.Routes) != 0 || decision.Refusals[0].Reason != "permission_needs_written_approval" {
		t.Fatalf("in-app request bypassed plan permission: %+v", decision)
	}
}

func TestAllFourUseClassesAreDistinctAndExplicit(t *testing.T) {
	request, account, records, agreements, routes := rulesFixture(ReviewApproved)
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	classes := []UseClass{UseOwnAccount, UseBoundedFeature, UseInApp, UseHarnessAccess}
	for _, class := range classes {
		t.Run(string(class), func(t *testing.T) {
			request.useClass = class
			records[0].Terms.PlanRules[0].UseClasses[class] = PermissionAllowed
			if class == UseOwnAccount {
				account.ConnectedAccounts = []string{"account-a"}
				routes[0].AccountID = "account-a"
			}
			if class == UseHarnessAccess {
				request.harness = "opencode"
				sourceAllowed := true
				routes[0].Harness = "opencode"
				routes[0].HarnessSource = verifiedHarnessSourceFixture(t, &sourceAllowed)
				records[0].Terms.PlanRules[0].UseClasses[UseBoundedFeature] = PermissionAllowed
				trust := true
				agreement, err := NewVerifiedAgreement(Agreement{Supplier: "supplier-a", Trust: TrustDecision{CustomerCodeAllowed: &trust, DecisionRef: "trust-decision"}, UseClasses: []UseClass{UseHarnessAccess}, Models: []string{"model-a"}, DecisionRef: "trust-decision"}, testDecisionVerifier)
				if err != nil {
					t.Fatal(err)
				}
				agreements = []Agreement{agreement}
			}
			got := Eligible(request, account, records, agreements, routes, now)
			if len(got.Routes) != 1 {
				t.Fatalf("explicit permission should allow %s: %+v", class, got)
			}
		})
	}
}

func TestHarnessTrustCannotEscapeApprovedModelAndUseClass(t *testing.T) {
	request, account, records, _, routes := rulesFixture(ReviewApproved)
	request.useClass, request.harness = UseHarnessAccess, "opencode"
	records[0].Terms.PlanRules[0].UseClasses[UseHarnessAccess] = PermissionAllowed
	allowed := true
	routes[0].Harness = "opencode"
	routes[0].HarnessSource = verifiedHarnessSourceFixture(t, &allowed)
	for _, scope := range []struct {
		name    string
		models  []string
		classes []UseClass
		want    bool
	}{
		{"approved scope", []string{"model-a"}, []UseClass{UseHarnessAccess}, true},
		{"another model", []string{"model-b"}, []UseClass{UseHarnessAccess}, false},
		{"another use class", []string{"model-a"}, []UseClass{UseInApp}, false},
	} {
		t.Run(scope.name, func(t *testing.T) {
			agreement, err := NewVerifiedAgreement(Agreement{Supplier: "supplier-a", Models: scope.models, UseClasses: scope.classes, DecisionRef: "trust-decision", Trust: TrustDecision{CustomerCodeAllowed: &allowed, DecisionRef: "trust-decision"}}, testDecisionVerifier)
			if err != nil {
				t.Fatal(err)
			}
			got := Eligible(request, account, records, []Agreement{agreement}, routes, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
			if (len(got.Routes) == 1) != scope.want {
				t.Fatalf("trust escaped scope: %+v", got)
			}
		})
	}
}

func TestFeatureUseClassIsDerivedAndPinnedContractIsValidated(t *testing.T) {
	contract := FeatureContract{ID: "web-console.shell_command", MaxInputBytes: 1024, MaxOutputBytes: 512, MaxUserTurns: 1}
	invocation := FeatureInvocation{Endpoint: "chat", UserTurns: 1, SchemaValid: true, InputBytes: 50, OutputBytes: 100}
	unpinned, err := CheckFeatureInvocation(contract, invocation)
	if err != nil {
		t.Fatal(err)
	}
	request, err := NewEligibleRequest(unpinned, "model-a", "balanced", "private", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if request.useClass != UseInApp {
		t.Fatalf("unpinned feature use class = %q, want in_app", request.useClass)
	}
	contract.PinnedInstructionDigest = strings.Repeat("b", 64)
	invocation.InstructionHash = strings.Repeat("c", 64)
	if _, err := CheckFeatureInvocation(contract, invocation); err == nil {
		t.Fatal("pinned digest mismatch was accepted")
	}
	if _, err := NewEligibleRequest(FeatureCheck{}, "model-a", "balanced", "private", "", 0); err == nil {
		t.Fatal("unvalidated feature check was accepted")
	}
	contract.PinnedInstructionDigest = "not-a-sha256"
	if _, err := CheckFeatureInvocation(contract, invocation); err == nil {
		t.Fatal("malformed pinned digest was accepted")
	}
	badInvocation := invocation
	badInvocation.ToolCount = 1
	if _, err := CheckFeatureInvocation(FeatureContract{ID: "web-console.shell_command", MaxInputBytes: 1024, MaxOutputBytes: 512, MaxUserTurns: 1}, badInvocation); err == nil {
		t.Fatal("tool-bearing request was accepted")
	}
}

func TestServerCallerSelectsOwnAccountAndHarnessUseClasses(t *testing.T) {
	contract := FeatureContract{ID: "agent.session", MaxInputBytes: 1024, MaxOutputBytes: 512, MaxUserTurns: 1}
	check, err := CheckFeatureInvocation(contract, FeatureInvocation{Endpoint: "chat", UserTurns: 1, SchemaValid: true, InputBytes: 1, OutputBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	ownAccount, err := NewEligibleRequestForUse(check, UseOwnAccount, "model-a", "balanced", "private", "", 0)
	if err != nil || ownAccount.useClass != UseOwnAccount {
		t.Fatalf("trusted own-account caller was not represented: request=%+v err=%v", ownAccount, err)
	}
	harness, err := NewEligibleRequestForUse(check, UseHarnessAccess, "model-a", "balanced", "private", "claude-code", 0)
	if err != nil || harness.useClass != UseHarnessAccess {
		t.Fatalf("server-resolved harness caller was not represented: request=%+v err=%v", harness, err)
	}
	if _, err := NewEligibleRequestForUse(check, UseHarnessAccess, "model-a", "balanced", "private", "", 0); err == nil {
		t.Fatal("harness access without a server-resolved harness was accepted")
	}
	check.hostedService = true
	if _, err := NewEligibleRequestForUse(check, UseOwnAccount, "model-a", "balanced", "private", "", 0); err == nil {
		t.Fatal("hosted feature was reclassified as own_account")
	}
}

func TestOwnAccountRouteMustBelongToVerifiedConnectedAccount(t *testing.T) {
	request, account, records, agreements, routes := rulesFixture(ReviewApproved)
	request.useClass = UseOwnAccount
	records[0].Terms.PlanRules[0].UseClasses[UseOwnAccount] = PermissionAllowed
	routes[0].AccountID = "account-a"
	got := Eligible(request, account, records, agreements, routes, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if len(got.Routes) != 0 || got.Refusals[0].Reason != "own_account_not_connected" {
		t.Fatalf("unconnected own-account route was eligible: %+v", got)
	}
	account.ConnectedAccounts = []string{"account-a"}
	got = Eligible(request, account, records, agreements, routes, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if len(got.Routes) != 1 {
		t.Fatalf("connected own-account route was denied: %+v", got)
	}
}

func TestEligibilityHonorsPrivacyBudgetAndSortsQuotesByCost(t *testing.T) {
	request, account, records, agreements, routes := rulesFixture(ReviewApproved)
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	moreExpensive := routes[0]
	moreExpensive.ID = "route-b"
	moreExpensive.QuotedCredits = 75
	cheaper := routes[0]
	cheaper.ID = "route-c"
	cheaper.QuotedCredits = 10
	got := Eligible(request, account, records, agreements, []Route{moreExpensive, routes[0], cheaper}, now)
	if len(got.Routes) != 3 || got.Routes[0].ID != "route-c" || got.Routes[1].ID != "route-a" {
		t.Fatalf("eligible routes are not ordered by quoted cost: %+v", got.Routes)
	}
	request.budget = 20
	got = Eligible(request, account, records, agreements, []Route{routes[0]}, now)
	if len(got.Routes) != 0 || got.Refusals[0].Reason != "budget_exceeded" {
		t.Fatalf("route above budget was eligible: %+v", got)
	}
	records[0].Terms.Downstream.DataCollection = "allow"
	request.budget = 100
	got = Eligible(request, account, records, agreements, routes, now)
	if len(got.Routes) != 0 || got.Refusals[0].Reason != "privacy_mismatch" {
		t.Fatalf("private request accepted a data-collecting route: %+v", got)
	}
}

func TestEligibilityRequiresRoleEndpointAndCapabilities(t *testing.T) {
	request, account, records, agreements, routes := rulesFixture(ReviewApproved)
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	request.endpoint = "embeddings"
	got := Eligible(request, account, records, agreements, routes, now)
	if len(got.Routes) != 0 || got.Refusals[0].Reason != "endpoint_mismatch" {
		t.Fatalf("route with the wrong endpoint was eligible: %+v", got)
	}
	request.endpoint = "chat"
	routes[0].Capabilities = nil
	got = Eligible(request, account, records, agreements, routes, now)
	if len(got.Routes) != 0 || got.Refusals[0].Reason != "capability_mismatch" {
		t.Fatalf("route missing a required capability was eligible: %+v", got)
	}
}

func TestEligibilityRejectsExpiredAndInvalidReviewRecords(t *testing.T) {
	request, account, records, agreements, routes := rulesFixture(ReviewApproved)
	records[0].Terms.ExpiresAt = "2026-10-31"
	got := Eligible(request, account, records, agreements, routes, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if len(got.Routes) != 0 || got.Refusals[0].Reason != "expired" {
		t.Fatalf("expired record accepted: %+v", got)
	}
	records[0].Terms.ExpiresAt = "2027-01-01"
	records[0].Terms.Review.DecisionRef = ""
	got = Eligible(request, account, records, agreements, routes, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if len(got.Routes) != 0 || got.Refusals[0].Reason != "unapproved" {
		t.Fatalf("approval without decision accepted: %+v", got)
	}
}

func TestEligibilityDeniesTermsPastNextReviewDate(t *testing.T) {
	request, account, records, agreements, routes := rulesFixture(ReviewApproved)
	records[0].Terms.NextReviewAt = "2026-10-31"
	got := Eligible(request, account, records, agreements, routes, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if len(got.Routes) != 0 || got.Refusals[0].Reason != "review_due" {
		t.Fatalf("terms past next review date were eligible: %+v", got)
	}
}

func TestEligibleRequiresNotificationTrustForHarnessAccess(t *testing.T) {
	request, account, records, agreements, routes := rulesFixture(ReviewApproved)
	contract := FeatureContract{ID: "agent.session", MaxInputBytes: 1024, MaxOutputBytes: 512, MaxUserTurns: 1}
	check, err := CheckFeatureInvocation(contract, FeatureInvocation{Endpoint: "chat", UserTurns: 1, SchemaValid: true, InputBytes: 1, OutputBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	request, err = NewEligibleRequestForUse(check, UseHarnessAccess, "model-a", "balanced", "private", "claude-code", 100)
	if err != nil {
		t.Fatal(err)
	}
	records[0].Terms.PlanRules[0].UseClasses[UseHarnessAccess] = PermissionAllowed
	trust := true
	agreement, err := NewVerifiedAgreement(Agreement{Supplier: "supplier-a", Trust: TrustDecision{CustomerCodeAllowed: &trust, DecisionRef: "trust-decision"}, UseClasses: []UseClass{UseHarnessAccess}, Models: []string{"model-a"}, DecisionRef: "trust-decision"}, testDecisionVerifier)
	if err != nil {
		t.Fatal(err)
	}
	agreements = []Agreement{agreement}
	routes[0].Harness = "claude-code"
	routes[0].HarnessSource = verifiedHarnessSourceFixture(t, &trust)
	got := Eligible(request, account, records, agreements, routes, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if len(got.Routes) != 1 {
		t.Fatalf("Notification Hub trust decision should allow the scoped harness route: %+v", got)
	}
	launcherAllowed := false
	routes[0].HarnessSource = &HarnessBillingSource{ID: "vrooli-credits", LauncherConfigAllowed: &launcherAllowed, CustomerConfigAllowed: PermissionSilent, Evidence: Evidence{URL: "https://example.invalid/claude-terms"}, Review: Review{Status: ReviewPending}}
	got = Eligible(request, account, records, agreements, routes, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if len(got.Routes) != 0 || got.Refusals[0].Reason != "harness_billing_source_denied" {
		t.Fatalf("Claude Code Vrooli-credit route unexpectedly allowed: %+v", got)
	}
}

func TestWiderUseClassNeedsAReason(t *testing.T) {
	rule := PlanRule{Plan: "subscription", UseClasses: map[UseClass]Permission{UseHarnessAccess: PermissionAllowed, UseInApp: PermissionForbidden}}
	if got := WiderPermissionReason(rule); got == "" {
		t.Fatal("wider allowed class without reason was not detected")
	}
	rule.Reason = "supplier expressly allows resale only through our controlled harness"
	if got := WiderPermissionReason(rule); got == "" { // reason is consumed by validator; detector remains descriptive
		t.Fatal("expected lint condition to remain detectable")
	}
}

func TestWiderUseClassLintTreatsOmittedNarrowerClassAsDefaultDeny(t *testing.T) {
	rule := PlanRule{Plan: "subscription", AccessTier: AccessTierGrant, UseClasses: map[UseClass]Permission{
		UseHarnessAccess: PermissionAllowed,
		UseInApp:         PermissionAllowed,
		// bounded_feature is omitted and therefore denied by default.
	}}
	if got := WiderPermissionReason(rule); got == "" || !strings.Contains(got, string(UseBoundedFeature)) {
		t.Fatalf("omitted narrower class was not detected as default-deny widening: %q", got)
	}
	var errs []error
	validatePlanRule(&errs, "terms.plan_rules[0]", rule, map[string]struct{}{})
	if len(errs) == 0 {
		t.Fatal("omitted narrower class did not require a reason")
	}
}

func TestAgreementCannotOverrideModelDenials(t *testing.T) {
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	trust := true
	agreement, err := NewVerifiedAgreement(Agreement{
		Supplier:    "supplier-a",
		Trust:       TrustDecision{CustomerCodeAllowed: &trust, DecisionRef: "trust-decision"},
		UseClasses:  []UseClass{UseBoundedFeature},
		Models:      []string{"model-a"},
		DecisionRef: "trust-decision",
	}, testDecisionVerifier)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("terms override forbidden wins", func(t *testing.T) {
		request, account, records, _, routes := rulesFixture(ReviewApproved)
		request.useClass = UseBoundedFeature
		records[0].Models = map[string]ModelRule{"model-a": {TermsOverride: &TermsOverride{
			UseClasses: map[UseClass]Permission{UseBoundedFeature: PermissionForbidden},
		}}}
		got := Eligible(request, account, records, []Agreement{agreement}, routes, now)
		if len(got.Routes) != 0 || len(got.Refusals) != 1 || got.Refusals[0].Reason != "model_terms_forbidden" {
			t.Fatalf("agreement bypassed model terms denial: %+v", got)
		}
	})

	t.Run("commercial licence denial wins", func(t *testing.T) {
		request, account, records, _, routes := rulesFixture(ReviewApproved)
		request.useClass = UseBoundedFeature
		denied := false
		records[0].Models = map[string]ModelRule{"model-a": {LicenceCommercialUse: &denied}}
		got := Eligible(request, account, records, []Agreement{agreement}, routes, now)
		if len(got.Routes) != 0 || len(got.Refusals) != 1 || got.Refusals[0].Reason != "model_licence_denies" {
			t.Fatalf("agreement bypassed model licence denial: %+v", got)
		}
	})
}

func TestVerifiedAgreementMustNameModels(t *testing.T) {
	trust := true
	_, err := NewVerifiedAgreement(Agreement{
		Supplier:    "supplier-a",
		Trust:       TrustDecision{CustomerCodeAllowed: &trust, DecisionRef: "trust-decision"},
		UseClasses:  []UseClass{UseBoundedFeature},
		DecisionRef: "trust-decision",
	}, testDecisionVerifier)
	if err == nil {
		t.Fatal("agreement with an empty model scope was accepted")
	}

	legacy := Agreement{Supplier: "supplier-a", UseClasses: []UseClass{UseBoundedFeature}, decisionVerified: true}
	if agreementGrantsRoute(legacy, Route{Supplier: "supplier-a", Model: "model-a"}, UseBoundedFeature, time.Now()) {
		t.Fatal("empty-model agreement acted as a wildcard")
	}
}

func TestPickerAndExecutionUseSameEligibilityFunction(t *testing.T) {
	request, account, records, agreements, routes := rulesFixture(ReviewApproved)
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	picked := PickRoutes(request, account, records, agreements, routes, now)
	if len(picked.Routes) != 1 {
		t.Fatalf("picker returned %+v", picked)
	}
	selected, err := AuthorizeExecution(request, account, records, agreements, routes, now, picked.Routes[0].ID)
	if err != nil || selected.ID != picked.Routes[0].ID {
		t.Fatalf("picker/execution disagreement: selected=%+v err=%v", selected, err)
	}
	request.useClass = UseInApp
	if _, err := AuthorizeExecution(request, account, records, agreements, routes, now, "route-a"); err == nil {
		t.Fatal("execution trusted stale picker selection")
	}
}

func TestProviderTermsValidatorRequiresDecisionEvidenceAndLintReason(t *testing.T) {
	terms := ProviderTerms{PlanRules: []PlanRule{{Plan: "subscription", AccessTier: AccessTierGrant, UseClasses: map[UseClass]Permission{UseHarnessAccess: PermissionAllowed, UseInApp: PermissionForbidden}}}, Evidence: Evidence{URL: "https://example.invalid"}, Review: Review{Status: ReviewApproved}, EffectiveFrom: "2026-01-01", OnExpiry: "deny"}
	var errs []error
	validateProviderTerms(&errs, "provider.terms", terms)
	joined := ""
	for _, err := range errs {
		joined += err.Error() + "\n"
	}
	if !strings.Contains(joined, "decision_ref is required") || !strings.Contains(joined, "reason is required") {
		t.Fatalf("missing required lint errors: %s", joined)
	}
}

func TestBarredPlanAllowsEmptyUseClassesAndNeverProducesRoute(t *testing.T) {
	terms := ProviderTerms{PlanRules: []PlanRule{{Plan: "subscription", AccessTier: AccessTierBarred, UseClasses: map[UseClass]Permission{}}}, Evidence: Evidence{URL: "https://example.invalid"}, Review: Review{Status: ReviewPending}, EffectiveFrom: "2026-01-01", OnExpiry: "deny"}
	var errs []error
	validateProviderTerms(&errs, "provider.terms", terms)
	for _, err := range errs {
		if strings.Contains(err.Error(), "use_classes must not be empty") {
			t.Fatalf("barred plan should not require permission rows: %v", errs)
		}
	}
	request, account, records, agreements, routes := rulesFixture(ReviewApproved)
	records[0].Terms.PlanRules[0].AccessTier = AccessTierBarred
	records[0].Terms.PlanRules[0].UseClasses = map[UseClass]Permission{UseBoundedFeature: PermissionAllowed}
	got := Eligible(request, account, records, agreements, routes, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if len(got.Routes) != 0 || got.Refusals[0].Reason != "access_tier_barred" {
		t.Fatalf("barred plan produced a route: %+v", got)
	}
}

func TestModelTermsOverrideCanOnlyNarrow(t *testing.T) {
	terms := &ProviderTerms{PlanRules: []PlanRule{{Plan: "p", UseClasses: map[UseClass]Permission{UseInApp: PermissionAllowed}}}}
	var errs []error
	validateOverrideNarrows(&errs, "model.terms_override", terms, TermsOverride{UseClasses: map[UseClass]Permission{UseInApp: PermissionForbidden}})
	if len(errs) != 0 {
		t.Fatalf("narrow override rejected: %v", errs)
	}
	validateOverrideNarrows(&errs, "model.terms_override", terms, TermsOverride{UseClasses: map[UseClass]Permission{UseInApp: PermissionAllowed, UseHarnessAccess: PermissionAllowed}})
	if len(errs) == 0 {
		t.Fatal("model override widened permission without agreement")
	}
}

func TestEligibleBindsRequestedModelAndHarness(t *testing.T) {
	req, account, records, agreements, routes := rulesFixture(ReviewApproved)
	req.model = "another-model"
	if got := Eligible(req, account, records, agreements, routes, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)); len(got.Routes) != 0 {
		t.Fatal("route for another model authorized")
	}
	req.model = "model-a"
	req.useClass = UseHarnessAccess
	req.harness = "claude-code"
	records[0].Terms.PlanRules[0].UseClasses[UseHarnessAccess] = PermissionAllowed
	allowed := true
	routes[0].Harness = "opencode"
	routes[0].HarnessSource = verifiedHarnessSourceFixture(t, &allowed)
	if got := Eligible(req, account, records, agreements, routes, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)); len(got.Routes) != 0 || got.Refusals[0].Reason != "harness_billing_source_denied" {
		t.Fatalf("wrong harness route: %+v", got)
	}
}

func verifiedHarnessSourceFixture(t *testing.T, allowed *bool) *HarnessBillingSource {
	t.Helper()
	source, err := NewVerifiedHarnessBillingSource(HarnessBillingSource{ID: "vrooli-credits", LauncherConfigAllowed: allowed, CustomerConfigAllowed: PermissionAllowed, Review: Review{Status: ReviewApproved, DecisionRef: "trust-decision"}}, testDecisionVerifier)
	if err != nil {
		t.Fatal(err)
	}
	return &source
}
