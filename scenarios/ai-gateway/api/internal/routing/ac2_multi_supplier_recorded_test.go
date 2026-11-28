package routing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/ai-provider-go/policy"
	_ "modernc.org/sqlite"
)

// This exercises a recorded-shape multi-supplier route set: the production
// picker chooses the lowest eligible current quote, and the production stream
// path dispatches exactly that pinned supplier and records its credit receipt.
func TestAC2RecordedMultiSupplierPickAndStreamUseSameEligibleQuote(t *testing.T) {
	var routerOneCalls, groqCalls, untrustedCalls atomic.Int64
	fixtureCalls := map[string]*atomic.Int64{}
	makeSupplier := func(calls *atomic.Int64) *httptest.Server {
		t.Helper()
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.URL.Path != "/chat/completions" || r.Method != http.MethodPost {
				http.NotFound(w, r)
				return
			}
			var envelope struct {
				Model    string `json:"model"`
				Provider struct {
					DataCollection string   `json:"data_collection"`
					Only           []string `json:"only"`
					AllowFallbacks bool     `json:"allow_fallbacks"`
				} `json:"provider"`
			}
			if json.NewDecoder(r.Body).Decode(&envelope) != nil || envelope.Model != "recorded-model" || envelope.Provider.DataCollection != "deny" || len(envelope.Provider.Only) != 1 || envelope.Provider.Only[0] != "fixture-host" || envelope.Provider.AllowFallbacks {
				t.Errorf("selected router received an unbounded/unpinned envelope: %+v", envelope)
				http.Error(w, "invalid recorded envelope", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"fixture\"}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":3,\"cost\":0.000002}}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		}))
	}
	routerOne := makeSupplier(&routerOneCalls)
	defer routerOne.Close()
	groq := makeSupplier(&groqCalls)
	defer groq.Close()
	untrusted := makeSupplier(&untrustedCalls)
	defer untrusted.Close()
	// The researched candidate set is exercised as synthetic, already-eligible
	// recorded-shape routes. This does not change the proposal status of any
	// real provider declaration or imply approval of its terms.
	fixtureIDs := ac2RecordedCandidateIDs(t)
	fixtureServers := map[string]*httptest.Server{}
	for index, id := range fixtureIDs {
		if id == "groqcloud" || id == "router-one" {
			continue
		}
		calls := &atomic.Int64{}
		fixtureCalls[id] = calls
		fixtureServers[id] = makeSupplier(calls)
		defer fixtureServers[id].Close()
		_ = index
	}

	now := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	var records []policy.RuleRecord
	var agreements []policy.Agreement
	var routes []policy.Route
	available := map[string]bool{}
	endpoints := map[string]SupplierEndpoint{}
	launcherAllowed, customerAllowed := true, true
	harnessSource, err := policy.NewVerifiedHarnessBillingSource(policy.HarnessBillingSource{
		ID: "vrooli-credits", LauncherConfigAllowed: &launcherAllowed, CustomerConfigAllowed: policy.PermissionAllowed,
		Review: policy.Review{Status: policy.ReviewApproved, DecisionRef: "recorded-harness-decision"},
	}, func(ref string) bool { return ref == "recorded-harness-decision" })
	if err != nil {
		t.Fatal(err)
	}

	addSupplier := func(id string, endpoint SupplierEndpoint, quote int64, includeAgreement bool) {
		t.Helper()
		terms := policy.ProviderTerms{
			PlanRules:     []policy.PlanRule{{Plan: "pay_as_you_go", AccessTier: policy.AccessTierGrant, UseClasses: map[policy.UseClass]policy.Permission{policy.UseHarnessAccess: policy.PermissionAllowed}}},
			Evidence:      policy.Evidence{URL: "https://fixture.invalid/terms", ReadAt: "2026-11-22"},
			Review:        policy.Review{Status: policy.ReviewApproved, DecisionRef: "decision-" + id},
			EffectiveFrom: "2026-01-01", ExpiresAt: "2030-01-01", OnExpiry: "deny",
			Downstream: policy.DownstreamRules{DataCollection: "deny", PinnedHosts: []string{"fixture-host"}},
		}
		record, err := policy.NewVerifiedRuleRecord(id, terms, nil, func(ref string) bool { return ref == "decision-"+id })
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
		if includeAgreement {
			agreement, err := policy.NewVerifiedAgreement(policy.Agreement{
				Supplier: id, Models: []string{"recorded-model"}, UseClasses: []policy.UseClass{policy.UseHarnessAccess}, DecisionRef: "agreement-" + id,
				Trust: policy.TrustDecision{CustomerCodeAllowed: &customerAllowed, DecisionRef: "agreement-" + id},
			}, func(ref string) bool { return ref == "agreement-"+id || ref == "trust-"+id })
			if err != nil {
				t.Fatal(err)
			}
			agreements = append(agreements, agreement)
		}
		routes = append(routes, policy.Route{
			ID: "route-" + id, Supplier: id, Plan: "pay_as_you_go", Model: "recorded-model", Endpoint: "chat", Profile: "balanced",
			QuoteRef: "quote-" + id, QuotedCredits: quote,
			StreamQuote: policy.StreamQuote{MaxInputBytes: 4096, MaxOutputTokens: 128, MaxCostMicros: 2, TokenLimitField: "max_tokens"},
			Harness:     "opencode", HarnessSource: &harnessSource,
		})
		endpoints[id] = endpoint
		available[id] = true
	}
	addSupplier("router-one", SupplierEndpoint{URL: routerOne.URL}, 2, true)
	addSupplier("groqcloud", SupplierEndpoint{URL: groq.URL}, 7, true)
	for index, id := range fixtureIDs {
		if id == "groqcloud" || id == "router-one" {
			continue
		}
		addSupplier(id, SupplierEndpoint{URL: fixtureServers[id].URL}, int64(10+index), true)
	}
	addSupplier("untrusted-route", SupplierEndpoint{URL: untrusted.URL}, 1, false)

	feature := PrivateFeature{ID: "ac2.recorded-harness", Role: "chat", Profile: "balanced", MaxInputBytes: 4096, MaxOutputBytes: 1024, MaxUserTurns: 1}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	snapshot := func(context.Context, string) (PrivatePolicySnapshot, error) {
		current := make([]policy.Route, 0, len(routes))
		for _, route := range routes {
			if available[route.Supplier] {
				current = append(current, route)
			}
		}
		return PrivatePolicySnapshot{Account: policy.Account{Plan: "pay_as_you_go", BalanceStanding: true}, Records: records, Agreements: agreements, Routes: current}, nil
	}
	service := NewPrivatePolicyService([]PrivateFeature{feature}, snapshot, NewSQLPrivatePlanRepository(db), NewSQLPrivateReceiptRepository(db))
	service.clock = func() time.Time { return now }
	service.SetStreamExecutor(&HTTPStreamExecutor{DB: db, Suppliers: endpoints, Client: http.DefaultClient})

	plan, err := service.Plan(context.Background(), PrivatePlanRequest{
		OperationID: "recorded-operation", FeatureID: feature.ID, Model: "recorded-model", BudgetMicros: 10,
		Harness: "opencode", CreditMultiplier: 10000, BusinessAccountID: "recorded-account", ActorID: "recorded-actor", AppKey: "web-console", MaxCredits: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Eligible || plan.RouteID != "route-router-one" || plan.Supplier != "router-one" || plan.QuotedCredits != 2 {
		t.Fatalf("picker did not choose lowest current eligible supplier quote: %+v", plan)
	}
	response := httptest.NewRecorder()
	streamRequest := func(plan PrivatePlan) PrivateStreamRequest {
		return PrivateStreamRequest{
			OperationID: plan.OperationID, BusinessAccountID: plan.BusinessAccountID, ActorID: plan.ActorID, AppKey: plan.AppKey, Plan: plan,
			Chat: json.RawMessage(`{"model":"recorded-model","messages":[{"role":"user","content":"hello"}]}`),
		}
	}
	receipt, err := service.Stream(context.Background(), streamRequest(plan), response)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != "completed" || !receipt.CostKnown || receipt.Credits != 2 || receipt.RouteID != plan.RouteID || receipt.Supplier != plan.Supplier || receipt.Model != plan.Model {
		t.Fatalf("execution/receipt diverged from selected route: plan=%+v receipt=%+v", plan, receipt)
	}
	if routerOneCalls.Load() != 1 || groqCalls.Load() != 0 || untrustedCalls.Load() != 0 {
		t.Fatalf("route dispatches = router-one:%d groq:%d untrusted:%d; want only selected router-one", routerOneCalls.Load(), groqCalls.Load(), untrustedCalls.Load())
	}
	for id, calls := range fixtureCalls {
		if calls.Load() != 0 {
			t.Errorf("unselected recorded router %q received %d calls", id, calls.Load())
		}
	}
	stored, err := service.Receipt(context.Background(), plan.OperationID)
	if err != nil || stored != receipt {
		t.Fatalf("persisted receipt = %+v, err=%v; want %+v", stored, err, receipt)
	}

	stalePlan, err := service.Plan(context.Background(), PrivatePlanRequest{
		OperationID: "recorded-drift-operation", FeatureID: feature.ID, Model: "recorded-model", BudgetMicros: 10,
		Harness: "opencode", CreditMultiplier: 10000, BusinessAccountID: "recorded-account", ActorID: "recorded-actor", AppKey: "web-console", MaxCredits: 10,
	})
	if err != nil || stalePlan.RouteID != "route-router-one" {
		t.Fatalf("initial drift fixture plan = %+v, err=%v", stalePlan, err)
	}
	for index := range routes {
		if routes[index].ID == "route-groqcloud" {
			routes[index].QuotedCredits = 1
			routes[index].QuoteRef = "quote-groqcloud-repriced"
			routes[index].StreamQuote.MaxCostMicros = 1
		}
	}
	_, err = service.Stream(context.Background(), streamRequest(stalePlan), httptest.NewRecorder())
	if !errors.Is(err, ErrPrivatePlanNotEligible) {
		t.Fatalf("execution with a changed picker result error = %v, want route denial", err)
	}
	if routerOneCalls.Load() != 1 || groqCalls.Load() != 0 || untrustedCalls.Load() != 0 {
		t.Fatalf("stale picker dispatched after quote/availability changed: router-one:%d groq:%d untrusted:%d", routerOneCalls.Load(), groqCalls.Load(), untrustedCalls.Load())
	}
	for id, calls := range fixtureCalls {
		if calls.Load() != 0 {
			t.Errorf("stale plan dispatched to recorded router %q: %d calls", id, calls.Load())
		}
	}
	if _, err := service.Receipt(context.Background(), stalePlan.OperationID); !errors.Is(err, ErrPrivateReceiptMissing) {
		t.Fatalf("stale route left a dispatch receipt: %v", err)
	}

	// A fresh plan observes both the new quote and the availability snapshot.
	// The now-unavailable lower quote is omitted, so the picker and execution
	// must move together to GroqCloud.
	available["router-one"] = false
	availablePlan, err := service.Plan(context.Background(), PrivatePlanRequest{
		OperationID: "recorded-availability-operation", FeatureID: feature.ID, Model: "recorded-model", BudgetMicros: 10,
		Harness: "opencode", CreditMultiplier: 10000, BusinessAccountID: "recorded-account", ActorID: "recorded-actor", AppKey: "web-console", MaxCredits: 10,
	})
	if err != nil || availablePlan.RouteID != "route-groqcloud" || availablePlan.QuotedCredits != 1 {
		t.Fatalf("picker ignored current availability/quote snapshot: plan=%+v, err=%v", availablePlan, err)
	}
	availableReceipt, err := service.Stream(context.Background(), streamRequest(availablePlan), httptest.NewRecorder())
	if err != nil || availableReceipt.State != "completed" || availableReceipt.Supplier != "groqcloud" || availableReceipt.RouteID != availablePlan.RouteID {
		t.Fatalf("execution did not honor refreshed available route: receipt=%+v, err=%v", availableReceipt, err)
	}
	if routerOneCalls.Load() != 1 || groqCalls.Load() != 1 || untrustedCalls.Load() != 0 {
		t.Fatalf("availability refresh dispatches = router-one:%d groq:%d untrusted:%d", routerOneCalls.Load(), groqCalls.Load(), untrustedCalls.Load())
	}
	for id, calls := range fixtureCalls {
		if calls.Load() != 0 {
			t.Errorf("availability refresh dispatched to unselected recorded router %q: %d calls", id, calls.Load())
		}
	}

	// Sweep the cheapest quote across every researched candidate identity. For
	// each one, the production picker must pin that supplier and execution must
	// dispatch only to its recorded-shape fixture endpoint.
	callsFor := func(id string) int64 {
		switch id {
		case "router-one":
			return routerOneCalls.Load()
		case "groqcloud":
			return groqCalls.Load()
		case "untrusted-route":
			return untrustedCalls.Load()
		default:
			return fixtureCalls[id].Load()
		}
	}
	for _, id := range fixtureIDs {
		before := make(map[string]int64, len(fixtureIDs))
		for _, candidateID := range fixtureIDs {
			available[candidateID] = true
			before[candidateID] = callsFor(candidateID)
		}
		for index := range routes {
			if routes[index].Supplier == id {
				routes[index].QuotedCredits = 2
				routes[index].QuoteRef = "sweep-quote-" + id
			} else {
				routes[index].QuotedCredits = int64(20 + index)
				routes[index].QuoteRef = "sweep-quote-" + routes[index].Supplier
			}
		}
		operationID := "recorded-candidate-sweep-" + id
		selected, err := service.Plan(context.Background(), PrivatePlanRequest{
			OperationID: operationID, FeatureID: feature.ID, Model: "recorded-model", BudgetMicros: 10,
			Harness: "opencode", CreditMultiplier: 10000, BusinessAccountID: "recorded-account", ActorID: "recorded-actor", AppKey: "web-console", MaxCredits: 10,
		})
		if err != nil || selected.Supplier != id || selected.RouteID != "route-"+id || selected.QuotedCredits != 2 {
			t.Fatalf("picker failed to select swept candidate %q: plan=%+v err=%v", id, selected, err)
		}
		sweptReceipt, err := service.Stream(context.Background(), streamRequest(selected), httptest.NewRecorder())
		if err != nil || sweptReceipt.State != "completed" || sweptReceipt.Supplier != id || sweptReceipt.RouteID != selected.RouteID || sweptReceipt.Credits != selected.QuotedCredits {
			t.Fatalf("execution/receipt diverged for swept candidate %q: receipt=%+v err=%v", id, sweptReceipt, err)
		}
		for _, candidateID := range fixtureIDs {
			want := before[candidateID]
			if candidateID == id {
				want++
			}
			if got := callsFor(candidateID); got != want {
				t.Errorf("candidate sweep %q call count for %q = %d, want %d", id, candidateID, got, want)
			}
		}
		if untrustedCalls.Load() != 0 {
			t.Fatalf("candidate sweep %q dispatched to untrusted route %d times", id, untrustedCalls.Load())
		}
	}

	// Make each researched candidate's route the cheapest recorded quote and
	// then mark only that route unavailable. The picker must choose the next
	// sorted eligible candidate, and execution must dispatch only there. This
	// keeps availability behavior broad without making live router calls.
	for _, unavailableID := range fixtureIDs {
		wantID := ""
		before := make(map[string]int64, len(fixtureIDs))
		for _, id := range fixtureIDs {
			available[id] = id != unavailableID
			before[id] = callsFor(id)
			if id != unavailableID && wantID == "" {
				wantID = id
			}
		}
		for index := range routes {
			if routes[index].Supplier == unavailableID {
				routes[index].QuotedCredits = 1
				routes[index].QuoteRef = "unavailable-cheapest-" + unavailableID
			} else if routes[index].Supplier != "untrusted-route" {
				routes[index].QuotedCredits = 2
				routes[index].QuoteRef = "available-fallback-" + routes[index].Supplier
			}
		}
		operationID := "recorded-unavailable-candidate-" + unavailableID
		fallback, err := service.Plan(context.Background(), PrivatePlanRequest{
			OperationID: operationID, FeatureID: feature.ID, Model: "recorded-model", BudgetMicros: 10,
			Harness: "opencode", CreditMultiplier: 10000, BusinessAccountID: "recorded-account", ActorID: "recorded-actor", AppKey: "web-console", MaxCredits: 10,
		})
		if err != nil || fallback.Supplier != wantID || fallback.RouteID != "route-"+wantID || fallback.QuotedCredits != 2 {
			t.Fatalf("unavailable cheapest candidate %q was not skipped for %q: plan=%+v err=%v", unavailableID, wantID, fallback, err)
		}
		fallbackReceipt, err := service.Stream(context.Background(), streamRequest(fallback), httptest.NewRecorder())
		if err != nil || fallbackReceipt.State != "completed" || fallbackReceipt.Supplier != wantID || fallbackReceipt.RouteID != fallback.RouteID || fallbackReceipt.Credits != 2 {
			t.Fatalf("picker/execution disagreed after %q became unavailable: receipt=%+v err=%v", unavailableID, fallbackReceipt, err)
		}
		for _, candidateID := range fixtureIDs {
			wantCalls := before[candidateID]
			if candidateID == wantID {
				wantCalls++
			}
			if got := callsFor(candidateID); got != wantCalls {
				t.Errorf("unavailable-candidate run %q dispatch count for %q = %d, want %d", unavailableID, candidateID, got, wantCalls)
			}
		}
		if untrustedCalls.Load() != 0 {
			t.Errorf("unavailable-candidate run %q dispatched to untrusted route %d times", unavailableID, untrustedCalls.Load())
		}
	}
}

func ac2RecordedCandidateIDs(t *testing.T) []string {
	t.Helper()
	declarationDir := filepath.Join("..", "..", "..", "config", "suppliers")
	declarations, err := os.ReadDir(declarationDir)
	if err != nil {
		t.Fatalf("read supplier declarations for recorded candidate fixture: %v", err)
	}
	ids := []string{"openrouter"} // Existing registry provider, not a supplier declaration.
	for _, entry := range declarations {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		ids = append(ids, entry.Name()[:len(entry.Name())-len(filepath.Ext(entry.Name()))])
	}
	sort.Strings(ids)
	if len(ids) != 19 {
		t.Fatalf("recorded candidate fixture contains %d candidates, want 19", len(ids))
	}
	return ids
}
