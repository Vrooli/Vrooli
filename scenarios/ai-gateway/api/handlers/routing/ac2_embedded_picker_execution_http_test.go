package routing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/ai-provider-go/policy"
	"github.com/vrooli/api-core/consumeridentity"
	_ "modernc.org/sqlite"

	"ai-gateway/internal/catalog"
	internalrouting "ai-gateway/internal/routing"
)

type ac2FixtureCreditAssertionVerifier struct{}

func (ac2FixtureCreditAssertionVerifier) VerifyCreditAssertion(_ *http.Request, token string, expected consumeridentity.CreditAssertionExpectation) error {
	if token != "ac2-local-fixture-assertion" || expected.BusinessAccountID == "" || expected.ActorID == "" || expected.AppKey == "" || expected.FeatureID == "" || expected.OperationID == "" || expected.MaxCredits <= 0 || expected.Audience != consumeridentity.CreditGatewayAudience || expected.Source != consumeridentity.CreditAssertionSource {
		return fmt.Errorf("invalid local AC2 fixture credit assertion")
	}
	return nil
}

// This runs the embedded catalog through the same private plan/stream HTTP
// handlers used by LPBS. Synthetic in-memory approvals and local endpoints
// make each catalog supplier the cheapest in turn; no source record or
// provider decision is changed on disk.
func TestAC2EmbeddedCatalogPickerAndExecutionAgreeAcrossAllCandidatesHTTP(t *testing.T) {
	bundle, err := catalog.Embedded()
	if err != nil {
		t.Fatalf("load embedded provider catalog: %v", err)
	}
	candidateIDs := journeyAC2CandidateIDs(t)
	if len(candidateIDs) != 19 {
		t.Fatalf("embedded candidate fixture count = %d, want 19", len(candidateIDs))
	}

	var upstreamCalls = make(map[string]*atomic.Int64, len(candidateIDs))
	var upstreamServers = make(map[string]*httptest.Server, len(candidateIDs))
	for _, id := range candidateIDs {
		id := id
		calls := &atomic.Int64{}
		upstreamCalls[id] = calls
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
				http.NotFound(w, r)
				return
			}
			var body struct {
				Model    string `json:"model"`
				Provider struct {
					DataCollection string   `json:"data_collection"`
					Only           []string `json:"only"`
					AllowFallbacks bool     `json:"allow_fallbacks"`
				} `json:"provider"`
			}
			if json.NewDecoder(r.Body).Decode(&body) != nil || body.Model != "ac2-recorded-model" || body.Provider.DataCollection != "deny" || len(body.Provider.Only) != 1 || body.Provider.Only[0] != "ac2-local-fixture" || body.Provider.AllowFallbacks {
				http.Error(w, "unbounded fixture dispatch", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"fixture\"}}]}\n\n")
			_, _ = io.WriteString(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":3,\"cost\":0.000002}}\n\n")
			_, _ = io.WriteString(w, "data: [DONE]\n\n")
		}))
		upstreamServers[id] = server
		t.Cleanup(server.Close)
	}

	decisionReceipts := make(map[string]bool, len(candidateIDs)*2)
	for _, id := range candidateIDs {
		provider, ok := bundle.Providers[id]
		if !ok {
			t.Fatalf("embedded bundle omitted candidate %q", id)
		}
		terms := policy.ProviderTerms{
			PlanRules: []policy.PlanRule{{
				Plan: "ac2_fixture", AccessTier: policy.AccessTierGrant,
				UseClasses: map[policy.UseClass]policy.Permission{
					policy.UseInApp:          policy.PermissionAllowed,
					policy.UseHarnessAccess:  policy.PermissionAllowed,
					policy.UseBoundedFeature: policy.PermissionAllowed,
				},
			}},
			Evidence:      policy.Evidence{URL: "https://fixture.invalid/ac2/" + id},
			Review:        policy.Review{Status: policy.ReviewApproved, DecisionRef: "ac2-synthetic-approval-" + id},
			EffectiveFrom: "2026-01-01", ExpiresAt: "2030-01-01", OnExpiry: "deny",
			Downstream: policy.DownstreamRules{DataCollection: "deny", PinnedHosts: []string{"ac2-local-fixture"}},
		}
		agreement, err := policy.NewVerifiedAgreement(policy.Agreement{
			Supplier: id, Models: []string{"ac2-recorded-model"},
			UseClasses:  []policy.UseClass{policy.UseInApp, policy.UseHarnessAccess},
			DecisionRef: "ac2-synthetic-approval-" + id,
			Trust:       policy.TrustDecision{CustomerCodeAllowed: ac2BoolPtr(true), DecisionRef: "ac2-synthetic-approval-" + id},
		}, func(string) bool { return true })
		if err != nil {
			t.Fatalf("construct %s synthetic fixture agreement: %v", id, err)
		}
		route := policy.Route{
			ID: id + "-ac2-local-route", Supplier: id, Plan: "ac2_fixture", Model: "ac2-recorded-model",
			Endpoint: "chat", Profile: "balanced", QuoteRef: id + "-quote", QuotedCredits: 400,
			StreamQuote: policy.StreamQuote{MaxInputBytes: 1 << 20, MaxOutputTokens: 64, MaxCostMicros: 2, TokenLimitField: "max_tokens"},
		}
		fixturePolicy := struct {
			Provider struct {
				ID         string               `json:"id"`
				Terms      policy.ProviderTerms `json:"terms"`
				Routes     []policy.Route       `json:"hosted_routes"`
				Agreements []policy.Agreement   `json:"agreements"`
			} `json:"provider"`
			Models map[string]policy.Model `json:"models"`
		}{Models: map[string]policy.Model{"ac2-recorded-model": {}}}
		fixturePolicy.Provider.ID = id
		fixturePolicy.Provider.Terms = terms
		fixturePolicy.Provider.Routes = []policy.Route{route}
		fixturePolicy.Provider.Agreements = []policy.Agreement{agreement}
		encoded, err := json.Marshal(fixturePolicy)
		if err != nil {
			t.Fatal(err)
		}
		provider.Policy = encoded
		provider.Endpoint = upstreamServers[id].URL
		bundle.Providers[id] = provider

		modelRules := map[string]policy.ModelRule{"ac2-recorded-model": {}}
		ruleDigest, err := policy.ProviderRuleDigest(id, terms, modelRules)
		if err != nil {
			t.Fatal(err)
		}
		agreementDigest, err := policy.ProviderAgreementDigest(agreement)
		if err != nil {
			t.Fatal(err)
		}
		decisionReceipts[policy.ProviderDecisionScopeRule+"|"+terms.Review.DecisionRef+"|"+ruleDigest] = true
		decisionReceipts[policy.ProviderDecisionScopeAgreement+"|"+agreement.DecisionRef+"|"+agreementDigest] = true
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(internalrouting.Schema()); err != nil {
		t.Fatalf("create local route/receipt schema: %v", err)
	}
	source := internalrouting.ResourcePolicySource{
		Catalog: &bundle,
		VerifyDecision: func(_ context.Context, ref, digest, scope string) bool {
			return decisionReceipts[scope+"|"+ref+"|"+digest]
		},
	}
	service := internalrouting.NewPrivatePolicyService(
		[]internalrouting.PrivateFeature{{ID: "agent.session", Role: "chat", Profile: "balanced", MaxInputBytes: 1 << 20, MaxOutputBytes: 1 << 20, MaxUserTurns: 1}},
		source.Snapshot,
		internalrouting.NewSQLPrivatePlanRepository(db),
		internalrouting.NewSQLPrivateReceiptRepository(db),
	)
	service.SetStreamExecutor(&internalrouting.HTTPStreamExecutor{DB: db, ResolveSupplier: source.Endpoint})
	router := mux.NewRouter()
	mountPrivatePolicyRoutes(router, NewConnectHandler(Deps{PrivatePolicy: service, CreditAssertions: ac2FixtureCreditAssertionVerifier{}}))
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	for selectedIndex, selectedID := range candidateIDs {
		for _, id := range candidateIDs {
			provider := bundle.Providers[id]
			// Decode through a dedicated envelope because the bundle is the exact
			// source observed by the service on the following HTTP request.
			var policyEnvelope map[string]json.RawMessage
			if err := json.Unmarshal(provider.Policy, &policyEnvelope); err != nil {
				t.Fatalf("decode %s fixture policy: %v", id, err)
			}
			var providerEnvelope struct {
				ID         string               `json:"id"`
				Terms      policy.ProviderTerms `json:"terms"`
				Routes     []policy.Route       `json:"hosted_routes"`
				Agreements []policy.Agreement   `json:"agreements"`
			}
			if err := json.Unmarshal(policyEnvelope["provider"], &providerEnvelope); err != nil {
				t.Fatal(err)
			}
			quote := int64(400 + selectedIndex)
			if id == selectedID {
				quote = 200
			}
			providerEnvelope.Routes[0].QuotedCredits = quote
			providerEnvelope.Routes[0].QuoteRef = fmt.Sprintf("%s-quote-%d", id, selectedIndex)
			var providerDoc map[string]json.RawMessage
			encodedProvider, err := json.Marshal(providerEnvelope)
			if err != nil {
				t.Fatal(err)
			}
			providerDoc = map[string]json.RawMessage{"provider": encodedProvider, "models": policyEnvelope["models"]}
			encodedPolicy, err := json.Marshal(providerDoc)
			if err != nil {
				t.Fatal(err)
			}
			provider.Policy = encodedPolicy
			bundle.Providers[id] = provider
		}

		operationID := fmt.Sprintf("ac2-embedded-http-%02d-%s", selectedIndex, selectedID)
		// The chooser and execution path must observe the same current
		// recorded-shape policy. The catalog endpoint should offer the model
		// before the credits planner selects this iteration's cheapest supplier.
		modelsResponse, err := server.Client().Get(server.URL + "/api/v1/private/provider-routes/models")
		if err != nil {
			t.Fatalf("get chooser catalog for %q: %v", selectedID, err)
		}
		if modelsResponse.StatusCode != http.StatusOK {
			_ = modelsResponse.Body.Close()
			t.Fatalf("chooser catalog status for %q = %d", selectedID, modelsResponse.StatusCode)
		}
		if got := modelsResponse.Header.Get("Cache-Control"); got != "no-store" {
			_ = modelsResponse.Body.Close()
			t.Fatalf("chooser catalog cache policy for %q = %q, want no-store", selectedID, got)
		}
		var chooser struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
			Reason string `json:"reason"`
		}
		decodeErr := json.NewDecoder(modelsResponse.Body).Decode(&chooser)
		_ = modelsResponse.Body.Close()
		if decodeErr != nil {
			t.Fatalf("decode chooser catalog for %q: %v", selectedID, decodeErr)
		}
		chooserOffersModel := false
		for _, item := range chooser.Data {
			if item.ID == "ac2-recorded-model" {
				chooserOffersModel = true
				break
			}
		}
		if !chooserOffersModel {
			t.Fatalf("chooser omitted model before planner selected %q: %+v", selectedID, chooser)
		}
		planRequest := map[string]any{
			"operation_id": operationID, "feature_id": "agent.session", "model": "ac2-recorded-model",
			"budget_micros": 10, "credit_multiplier": 1000000,
			"business_account_id": "ac2-fixture-account", "actor_id": "ac2-fixture-actor",
			"app_key": "web-console", "max_credits": 1000,
		}
		planResponse := doAC2PrivateJSON(t, server.Client(), server.URL+"/api/v1/private/provider-routes/plan", planRequest)
		var plan internalrouting.PrivatePlan
		if err := json.Unmarshal(planResponse, &plan); err != nil {
			t.Fatalf("decode plan for %q: %v", selectedID, err)
		}
		if !plan.Eligible || plan.Supplier != selectedID || plan.RouteID != selectedID+"-ac2-local-route" || plan.QuotedCredits != 200 {
			t.Fatalf("lowest recorded route for %q = %+v; want selected supplier, pinned route, quote 100", selectedID, plan)
		}

		before := make(map[string]int64, len(candidateIDs))
		for _, id := range candidateIDs {
			before[id] = upstreamCalls[id].Load()
		}
		streamBody := map[string]any{
			"operation_id": operationID, "business_account_id": plan.BusinessAccountID,
			"actor_id": plan.ActorID, "app_key": plan.AppKey, "plan": plan,
			"chat": map[string]any{"model": plan.Model, "messages": []any{map[string]any{"role": "user", "content": "recorded fixture"}}, "max_tokens": 16},
		}
		receiptBody := doAC2PrivateJSON(t, server.Client(), server.URL+"/api/v1/private/provider-routes/stream", streamBody)
		receipt, err := parseAC2ReceiptEvent(receiptBody)
		if err != nil {
			t.Fatalf("parse stream receipt for %q: %v", selectedID, err)
		}
		if receipt.State != "completed" || receipt.Supplier != selectedID || receipt.RouteID != plan.RouteID || receipt.Credits != 200 {
			t.Fatalf("execution for picked supplier %q disagreed: %+v", selectedID, receipt)
		}
		for _, id := range candidateIDs {
			want := before[id]
			if id == selectedID {
				want++
			}
			if got := upstreamCalls[id].Load(); got != want {
				t.Errorf("HTTP execution %q dispatched to %q %d times; want %d", selectedID, id, got, want)
			}
		}
	}
}

func doAC2PrivateJSON(t *testing.T, client *http.Client, endpoint string, body any) []byte {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(encoded)))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer ac2-local-fixture-assertion")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("POST %s status=%d body=%s", endpoint, response.StatusCode, data)
	}
	return data
}

func parseAC2ReceiptEvent(body []byte) (internalrouting.PrivateReceipt, error) {
	for _, event := range strings.Split(string(body), "\n\n") {
		if !strings.HasPrefix(event, "event: receipt\n") {
			continue
		}
		data := strings.TrimPrefix(event, "event: receipt\ndata: ")
		var receipt internalrouting.PrivateReceipt
		if err := json.Unmarshal([]byte(data), &receipt); err != nil {
			return receipt, err
		}
		return receipt, nil
	}
	return internalrouting.PrivateReceipt{}, fmt.Errorf("receipt event missing from body %q", body)
}

func ac2BoolPtr(value bool) *bool { return &value }
