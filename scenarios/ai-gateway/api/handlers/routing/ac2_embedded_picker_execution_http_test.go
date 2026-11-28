package routing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

// This extends supplier-wide fixture parity to each exact source-catalog chat
// identity. Synthetic approval, quote and endpoint overlays exist only in the
// in-memory embedded bundle; the checked-in candidates remain proposal-only.
func TestAC2ExactCatalogChatIdentitiesPickAndExecuteViaPrivateHTTP(t *testing.T) {
	bundle, err := catalog.Embedded()
	if err != nil {
		t.Fatalf("load embedded provider catalog: %v", err)
	}
	candidateIDs := journeyAC2CandidateIDs(t)
	if len(candidateIDs) != 19 {
		t.Fatalf("embedded candidate fixture count = %d, want 19", len(candidateIDs))
	}

	type fixtureProviderPolicy struct {
		Provider struct {
			ID         string               `json:"id"`
			Terms      policy.ProviderTerms `json:"terms"`
			Routes     []policy.Route       `json:"hosted_routes"`
			Agreements []policy.Agreement   `json:"agreements"`
		} `json:"provider"`
		Models map[string]policy.Model `json:"models"`
	}
	type modelIdentity struct {
		Supplier string
		Model    string
		RouteID  string
	}

	rows := make([]modelIdentity, 0, 78)
	nonChatOnly := map[string]bool{}
	allChat := map[string]bool{}
	totalIdentities := 0
	upstreamCalls := make(map[string]*atomic.Int64, len(candidateIDs))
	decisionReceipts := map[string]bool{}
	for _, supplierID := range candidateIDs {
		provider, ok := bundle.Providers[supplierID]
		if !ok {
			t.Fatalf("embedded bundle omitted candidate %q", supplierID)
		}
		var source fixtureProviderPolicy
		if err := json.Unmarshal(provider.Policy, &source); err != nil {
			t.Fatalf("decode %s source policy: %v", supplierID, err)
		}
		if source.Provider.ID != supplierID {
			t.Fatalf("embedded policy ID for %q = %q", supplierID, source.Provider.ID)
		}
		modelIDs := make([]string, 0, len(source.Models))
		for modelID := range source.Models {
			modelIDs = append(modelIDs, modelID)
		}
		sort.Strings(modelIDs)
		chatIDs := make([]string, 0, len(modelIDs))
		for _, modelID := range modelIDs {
			totalIdentities++
			model := source.Models[modelID]
			identity, hasIdentity := model.Provenance["identity"]
			if !hasIdentity || !strings.HasPrefix(identity.Source, "https://") || !strings.Contains(identity.Source, modelID) {
				t.Fatalf("%s/%s lacks exact HTTPS identity provenance: %+v", supplierID, modelID, identity)
			}
			if _, err := time.Parse("2006-01-02", identity.ObservedAt); err != nil {
				t.Fatalf("%s/%s identity provenance date = %q: %v", supplierID, modelID, identity.ObservedAt, err)
			}
			isChat := false
			for _, endpoint := range model.Endpoints {
				if endpoint == "chat" {
					isChat = true
					break
				}
			}
			if !isChat {
				nonChatOnly[modelID] = true
				continue
			}
			allChat[modelID] = true
			model.DefaultEligible = true // synthetic in-memory source policy only
			source.Models[modelID] = model
			chatIDs = append(chatIDs, modelID)
		}
		if len(chatIDs) == 0 {
			continue
		}

		calls := &atomic.Int64{}
		upstreamCalls[supplierID] = calls
		allowedModels := make(map[string]bool, len(chatIDs))
		for _, modelID := range chatIDs {
			allowedModels[modelID] = true
		}
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			if json.NewDecoder(r.Body).Decode(&body) != nil || !allowedModels[body.Model] || body.Provider.DataCollection != "deny" || len(body.Provider.Only) != 1 || body.Provider.Only[0] != "ac2-local-fixture" || body.Provider.AllowFallbacks {
				http.Error(w, "unbounded or mismatched recorded model dispatch", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"fixture\"}}]}\n\n")
			_, _ = io.WriteString(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":3,\"cost\":0.000002}}\n\n")
			_, _ = io.WriteString(w, "data: [DONE]\n\n")
		}))
		t.Cleanup(upstream.Close)
		provider.Endpoint = upstream.URL

		terms := policy.ProviderTerms{
			PlanRules: []policy.PlanRule{{
				Plan: "ac2_fixture", AccessTier: policy.AccessTierGrant,
				UseClasses: map[policy.UseClass]policy.Permission{
					policy.UseInApp: policy.PermissionAllowed, policy.UseHarnessAccess: policy.PermissionAllowed,
					policy.UseBoundedFeature: policy.PermissionAllowed,
				},
			}},
			Evidence:      policy.Evidence{URL: "https://fixture.invalid/ac2/" + supplierID},
			Review:        policy.Review{Status: policy.ReviewApproved, DecisionRef: "ac2-synthetic-identity-approval-" + supplierID},
			EffectiveFrom: "2026-01-01", ExpiresAt: "2030-01-01", OnExpiry: "deny",
			Downstream: policy.DownstreamRules{DataCollection: "deny", PinnedHosts: []string{"ac2-local-fixture"}},
		}
		agreement, err := policy.NewVerifiedAgreement(policy.Agreement{
			Supplier: supplierID, Models: chatIDs,
			UseClasses:  []policy.UseClass{policy.UseInApp, policy.UseHarnessAccess},
			DecisionRef: "ac2-synthetic-identity-approval-" + supplierID,
			Trust:       policy.TrustDecision{CustomerCodeAllowed: ac2BoolPtr(true), DecisionRef: "ac2-synthetic-identity-approval-" + supplierID},
		}, func(string) bool { return true })
		if err != nil {
			t.Fatalf("construct %s in-memory agreement: %v", supplierID, err)
		}
		for _, modelID := range chatIDs {
			routeID := fmt.Sprintf("%s-ac2-identity-route-%03d", supplierID, len(rows))
			rows = append(rows, modelIdentity{Supplier: supplierID, Model: modelID, RouteID: routeID})
			source.Provider.Routes = append(source.Provider.Routes, policy.Route{
				ID: routeID, Supplier: supplierID, Plan: "ac2_fixture", Model: modelID,
				Endpoint: "chat", Profile: "balanced", QuoteRef: routeID + "-quote", QuotedCredits: 900,
				StreamQuote: policy.StreamQuote{MaxInputBytes: 1 << 20, MaxOutputTokens: 64, MaxCostMicros: 2, TokenLimitField: "max_tokens"},
			})
		}
		rules := make(map[string]policy.ModelRule, len(source.Models))
		for modelID, model := range source.Models {
			rule := policy.ModelRule{TermsOverride: model.TermsOverride}
			if model.Licence != nil {
				rule.LicenceCommercialUse = model.Licence.CommercialUse
			}
			rules[modelID] = rule
		}
		source.Provider.ID = supplierID
		source.Provider.Terms = terms
		source.Provider.Agreements = []policy.Agreement{agreement}
		encoded, err := json.Marshal(source)
		if err != nil {
			t.Fatalf("encode %s in-memory policy: %v", supplierID, err)
		}
		provider.Policy = encoded
		bundle.Providers[supplierID] = provider

		ruleDigest, err := policy.ProviderRuleDigest(supplierID, terms, rules)
		if err != nil {
			t.Fatalf("digest %s in-memory rules: %v", supplierID, err)
		}
		agreementDigest, err := policy.ProviderAgreementDigest(agreement)
		if err != nil {
			t.Fatalf("digest %s in-memory agreement: %v", supplierID, err)
		}
		decisionReceipts[policy.ProviderDecisionScopeRule+"|"+terms.Review.DecisionRef+"|"+ruleDigest] = true
		decisionReceipts[policy.ProviderDecisionScopeAgreement+"|"+agreement.DecisionRef+"|"+agreementDigest] = true
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Supplier != rows[j].Supplier {
			return rows[i].Supplier < rows[j].Supplier
		}
		return rows[i].Model < rows[j].Model
	})
	if totalIdentities != 78 {
		t.Fatalf("source candidate model identities = %d, want 78", totalIdentities)
	}
	if len(rows) != 65 || len(allChat) == 0 {
		t.Fatalf("source catalog chat identities = %d records/%d IDs, want current 65 record-shapes", len(rows), len(allChat))
	}
	for id := range nonChatOnly {
		if allChat[id] {
			delete(nonChatOnly, id)
		}
	}
	if len(nonChatOnly) != 13 {
		t.Fatalf("non-chat-only source identities = %d, want 13 System One/image identities excluded from this chat journey", len(nonChatOnly))
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open local in-memory database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(internalrouting.Schema()); err != nil {
		t.Fatalf("create local receipt schema: %v", err)
	}
	source := internalrouting.ResourcePolicySource{
		Catalog: &bundle,
		VerifyDecision: func(_ context.Context, ref, digest, scope string) bool {
			return decisionReceipts[scope+"|"+ref+"|"+digest]
		},
	}
	service := internalrouting.NewPrivatePolicyService(
		[]internalrouting.PrivateFeature{{ID: "agent.session", Role: "chat", Profile: "balanced", MaxInputBytes: 1 << 20, MaxOutputBytes: 1 << 20, MaxUserTurns: 1}},
		source.Snapshot, internalrouting.NewSQLPrivatePlanRepository(db), internalrouting.NewSQLPrivateReceiptRepository(db),
	)
	service.SetStreamExecutor(&internalrouting.HTTPStreamExecutor{DB: db, ResolveSupplier: source.Endpoint})
	router := mux.NewRouter()
	mountPrivatePolicyRoutes(router, NewConnectHandler(Deps{PrivatePolicy: service, CreditAssertions: ac2FixtureCreditAssertionVerifier{}}))
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	for index, row := range rows {
		for _, supplierID := range candidateIDs {
			provider := bundle.Providers[supplierID]
			var envelope fixtureProviderPolicy
			if err := json.Unmarshal(provider.Policy, &envelope); err != nil {
				t.Fatalf("decode %s in-memory policy: %v", supplierID, err)
			}
			for routeIndex := range envelope.Provider.Routes {
				route := &envelope.Provider.Routes[routeIndex]
				if route.Model == row.Model {
					if supplierID == row.Supplier {
						route.QuotedCredits = 200
					} else {
						route.QuotedCredits = int64(300 + routeIndex)
					}
					route.QuoteRef = fmt.Sprintf("%s-quote-%02d", route.ID, index)
				}
			}
			encoded, err := json.Marshal(envelope)
			if err != nil {
				t.Fatal(err)
			}
			provider.Policy = encoded
			bundle.Providers[supplierID] = provider
		}

		modelsResponse, err := server.Client().Get(server.URL + "/api/v1/private/provider-routes/models")
		if err != nil {
			t.Fatalf("list chooser before %s/%s: %v", row.Supplier, row.Model, err)
		}
		if modelsResponse.StatusCode != http.StatusOK || modelsResponse.Header.Get("Cache-Control") != "no-store" {
			_ = modelsResponse.Body.Close()
			t.Fatalf("chooser response before %s/%s = %d / %q", row.Supplier, row.Model, modelsResponse.StatusCode, modelsResponse.Header.Get("Cache-Control"))
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
			t.Fatalf("decode chooser before %s/%s: %v", row.Supplier, row.Model, decodeErr)
		}
		listed := false
		for _, item := range chooser.Data {
			if nonChatOnly[item.ID] {
				t.Fatalf("non-chat-only identity %q leaked into the chat chooser", item.ID)
			}
			if item.ID == row.Model {
				listed = true
				break
			}
		}
		if !listed {
			t.Fatalf("chooser omitted exact source model %s before selecting supplier %s; reason=%s", row.Model, row.Supplier, chooser.Reason)
		}

		operationID := fmt.Sprintf("ac2-source-model-http-%03d", index)
		planBody := map[string]any{
			"operation_id": operationID, "feature_id": "agent.session", "model": row.Model,
			"budget_micros": 10, "credit_multiplier": 1000000,
			"business_account_id": "ac2-fixture-account", "actor_id": "ac2-fixture-actor",
			"app_key": "web-console", "max_credits": 1000,
		}
		planBytes := doAC2PrivateJSON(t, server.Client(), server.URL+"/api/v1/private/provider-routes/plan", planBody)
		var plan internalrouting.PrivatePlan
		if err := json.Unmarshal(planBytes, &plan); err != nil {
			t.Fatalf("decode plan for %s/%s: %v", row.Supplier, row.Model, err)
		}
		if !plan.Eligible || plan.Supplier != row.Supplier || plan.Model != row.Model || plan.RouteID != row.RouteID || plan.QuotedCredits != 200 {
			t.Fatalf("picker/model identity diverged for %s/%s: %+v", row.Supplier, row.Model, plan)
		}

		before := make(map[string]int64, len(upstreamCalls))
		for supplierID, calls := range upstreamCalls {
			before[supplierID] = calls.Load()
		}
		streamBody := map[string]any{
			"operation_id": operationID, "business_account_id": plan.BusinessAccountID,
			"actor_id": plan.ActorID, "app_key": plan.AppKey, "plan": plan,
			"chat": map[string]any{"model": plan.Model, "messages": []any{map[string]any{"role": "user", "content": "source-catalog fixture"}}, "max_tokens": 16},
		}
		receiptBytes := doAC2PrivateJSON(t, server.Client(), server.URL+"/api/v1/private/provider-routes/stream", streamBody)
		receipt, err := parseAC2ReceiptEvent(receiptBytes)
		if err != nil {
			t.Fatalf("parse receipt for %s/%s: %v", row.Supplier, row.Model, err)
		}
		if receipt.State != "completed" || receipt.Supplier != row.Supplier || receipt.Model != row.Model || receipt.RouteID != plan.RouteID || receipt.Credits != 200 {
			t.Fatalf("execution/receipt diverged for %s/%s: %+v", row.Supplier, row.Model, receipt)
		}
		for supplierID, calls := range upstreamCalls {
			want := before[supplierID]
			if supplierID == row.Supplier {
				want++
			}
			if got := calls.Load(); got != want {
				t.Errorf("model %s dispatched to %s %d times, want %d", row.Model, supplierID, got, want)
			}
		}
	}

	if calls := len(rows); calls != 65 {
		t.Fatalf("exact source model executions = %d, want 65", calls)
	}
}
