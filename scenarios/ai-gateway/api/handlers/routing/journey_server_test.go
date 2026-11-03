package routing

import (
	internalrouting "ai-gateway/internal/routing"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/vrooli/ai-provider-go/policy"
	_ "modernc.org/sqlite"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This helper serves the production Gateway router, SQL plan/receipt stores
// and executor. The router upstream and labelled approval are test fixtures.
func TestJourneyGatewayServer(t *testing.T) {
	upstream := os.Getenv("VROOLI_JOURNEY_UPSTREAM")
	if upstream == "" {
		t.Skip("helper invoked by LPBS journey runner")
	}
	db, err := sql.Open("sqlite", os.Getenv("VROOLI_JOURNEY_GATEWAY_DB"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(internalrouting.Schema()); err != nil {
		t.Fatal(err)
	}
	// J12 must cross the production Hub boundary. A missing human decision
	// remains a refusal; this mode never constructs an approval fixture.
	if os.Getenv("VROOLI_JOURNEY_REAL_DECISIONS") == "1" {
		root, err := filepath.Abs("../../../../..")
		if err != nil {
			t.Fatal(err)
		}
		source := internalrouting.ResourcePolicySource{Root: root, Hub: internalrouting.NewDefaultProviderDecisionHubClient()}
		serveJourneyGateway(t, db, source)
		return
	}
	supplier := "fixture-router"
	if value := os.Getenv("VROOLI_JOURNEY_SUPPLIER"); value != "" {
		supplier = value
	}
	terms := policy.ProviderTerms{PlanRules: []policy.PlanRule{{Plan: "fixture", AccessTier: policy.AccessTierGrant, UseClasses: map[policy.UseClass]policy.Permission{policy.UseBoundedFeature: policy.PermissionAllowed, policy.UseInApp: policy.PermissionAllowed, policy.UseHarnessAccess: policy.PermissionAllowed}}}, Evidence: policy.Evidence{URL: "https://fixture.invalid/terms"}, Review: policy.Review{Status: policy.ReviewApproved, DecisionRef: "fixture-human-approved"}, EffectiveFrom: "2026-01-01", ExpiresAt: "2030-01-01", OnExpiry: "deny", Downstream: policy.DownstreamRules{DataCollection: "deny", PinnedHosts: []string{"fixture-host"}}}
	if os.Getenv("VROOLI_JOURNEY_RULE_STATUS") == "proposed" {
		terms.Review.Status = policy.ReviewProposed
	}
	allowed := true
	agreement, err := policy.NewVerifiedAgreement(policy.Agreement{Supplier: supplier, Models: []string{"fixture-model"}, UseClasses: []policy.UseClass{policy.UseHarnessAccess}, DecisionRef: "fixture-human-approved", Trust: policy.TrustDecision{CustomerCodeAllowed: &allowed, DecisionRef: "fixture-human-approved"}}, func(ref string) bool { return ref == "fixture-human-approved" })
	if err != nil {
		t.Fatal(err)
	}
	routes := []policy.Route{{ID: "fixture-route", Supplier: supplier, Plan: "fixture", Model: "fixture-model", Endpoint: "chat", Profile: "balanced", QuoteRef: "fixture-price", QuotedCredits: 1000, StreamQuote: policy.StreamQuote{MaxInputBytes: 1 << 20, MaxOutputTokens: 65536, MaxCostMicros: 2, TokenLimitField: "max_tokens"}}}
	for _, harness := range []string{"opencode", "codex", "claude-code"} {
		route := routes[0]
		route.ID += "-" + harness
		route.Harness = harness
		routes = append(routes, route)
	}
	root := t.TempDir()
	dir := filepath.Join(root, "resources", supplier)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	models := map[string]policy.Model{}
	modelRules := map[string]policy.ModelRule{}
	if os.Getenv("VROOLI_JOURNEY_DENY_MODEL_LICENCE") == "1" {
		commercial := false
		models["fixture-model"] = policy.Model{Licence: &policy.ModelLicence{ID: "fixture-restricted", CommercialUse: &commercial}}
		modelRules["fixture-model"] = policy.ModelRule{LicenceCommercialUse: &commercial}
	}
	ruleDigest, err := policy.ProviderRuleDigest(supplier, terms, modelRules)
	if err != nil {
		t.Fatal(err)
	}
	agreementDigest, err := policy.ProviderAgreementDigest(agreement)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	identityKind := "verified_human"
	if os.Getenv("VROOLI_JOURNEY_DECISION_ACTOR") == "agent" {
		identityKind = "agent"
	}
	decisions := map[string]policy.ProviderDecisionReceipt{}
	addDecision := func(ref, digest, scope string) {
		decisions[scope+"|"+ref+"|"+digest] = policy.ProviderDecisionReceipt{Source: "ai-gateway", EventType: policy.ProviderDecisionEventType, Requester: ref, EventID: ref, CanonicalTermsSHA256: digest, Scope: scope, State: "answered", Answer: "approve", AnsweredByIdentityKind: identityKind, AnsweredByIdentitySource: "immutable-fixture", CreatedAt: now.Add(-time.Hour).Format(time.RFC3339Nano), AnsweredAt: now.Add(-time.Minute).Format(time.RFC3339Nano), Deadline: now.Add(time.Hour).Format(time.RFC3339Nano)}
	}
	for scope, digest := range map[string]string{policy.ProviderDecisionScopeRule: ruleDigest, policy.ProviderDecisionScopeAgreement: agreementDigest} {
		addDecision("fixture-human-approved", digest, scope)
	}
	for _, harness := range []string{"opencode", "codex", "claude-code"} {
		launcherAllowed := harness != "claude-code"
		source := policy.HarnessBillingSource{ID: "vrooli-credits", LauncherConfigAllowed: &launcherAllowed, CustomerConfigAllowed: policy.PermissionAllowed, Review: policy.Review{Status: policy.ReviewApproved, DecisionRef: "fixture-harness-" + harness}}
		digest, err := policy.HarnessBillingSourceDigest(harness, source)
		if err != nil {
			t.Fatal(err)
		}
		addDecision(source.Review.DecisionRef, digest, policy.ProviderDecisionScopeRule)
		if os.Getenv("VROOLI_JOURNEY_HARNESS_DENIED") == harness {
			source.Review.Status = policy.ReviewPending
		}
		path := filepath.Join(root, "resources", harness)
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(map[string]any{"billing": map[string]any{"sources": []policy.HarnessBillingSource{source}}})
		if err := os.WriteFile(filepath.Join(path, "model-policy.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if os.Getenv("VROOLI_JOURNEY_CHANGED_CLAUSE") == "1" {
		terms.PlanRules[0].Clause = "changed after approval"
	}
	catalog := map[string]any{"provider": map[string]any{"id": supplier, "terms": terms, "hosted_routes": routes, "agreements": []policy.Agreement{agreement}}, "models": models}
	encoded, _ := json.Marshal(catalog)
	if err := os.WriteFile(filepath.Join(dir, "model-policy.json"), encoded, 0600); err != nil {
		t.Fatal(err)
	}
	encoded, _ = json.Marshal(map[string]any{"endpoint": upstream + "/models"})
	if err := os.WriteFile(filepath.Join(dir, "resource.json"), encoded, 0600); err != nil {
		t.Fatal(err)
	}
	source := internalrouting.ResourcePolicySource{Root: root, VerifyDecision: func(_ context.Context, ref, digest, scope string) bool {
		expected := policy.ProviderDecisionReceipt{Source: "ai-gateway", EventType: policy.ProviderDecisionEventType, Requester: ref, CanonicalTermsSHA256: digest, Scope: scope}
		return policy.VerifyProviderDecisionReceipt(expected, decisions[scope+"|"+ref+"|"+digest], time.Now()) == nil
	}}
	serveJourneyGateway(t, db, source)
}

func serveJourneyGateway(t *testing.T, db *sql.DB, source internalrouting.ResourcePolicySource) {
	var registry struct {
		Features []struct {
			Feature internalrouting.PrivateFeature `json:"feature"`
		} `json:"features"`
	}
	data, err := os.ReadFile("../../../../../packages/monetization-go/ai-features.generated.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		t.Fatal(err)
	}
	features := []internalrouting.PrivateFeature{{ID: "agent.session", Role: "chat", Profile: "balanced", MaxInputBytes: 1 << 20, MaxOutputBytes: 1 << 20, MaxUserTurns: 1}}
	for _, entry := range registry.Features {
		features = append(features, entry.Feature)
	}
	service := internalrouting.NewPrivatePolicyService(features, source.Snapshot, internalrouting.NewSQLPrivatePlanRepository(db), internalrouting.NewSQLPrivateReceiptRepository(db))
	service.SetStreamExecutor(&internalrouting.HTTPStreamExecutor{DB: db, ResolveSupplier: source.Endpoint})
	if os.Getenv("VROOLI_JOURNEY_RECEIPT_WRITE_FAILURE") == "1" {
		if _, err := db.Exec(`CREATE TRIGGER journey_receipt_write_failure BEFORE UPDATE ON private_provider_receipts BEGIN SELECT RAISE(ABORT,'fixture receipt persistence failure'); END`); err != nil {
			t.Fatal(err)
		}
	}

	router := mux.NewRouter()
	Module(Deps{DB: db, PrivatePolicy: service}).Mount(router)
	done := make(chan struct{})
	router.HandleFunc("/__journey/stop", func(w http.ResponseWriter, r *http.Request) { close(done) })
	router.HandleFunc("/__journey/legacy-receipt/{operation}", func(w http.ResponseWriter, r *http.Request) {
		_, err := db.Exec(`UPDATE private_provider_receipts SET receipt_json=json_remove(receipt_json,'$.feature_id') WHERE operation_id=?`, mux.Vars(r)["operation"])
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(204)
	}).Methods("POST")
	server := httptest.NewUnstartedServer(router)
	server.Config.WriteTimeout = 2 * time.Minute
	server.Start()
	defer server.Close()
	fmt.Println("GATEWAY_URL=" + server.URL)
	<-done
}
