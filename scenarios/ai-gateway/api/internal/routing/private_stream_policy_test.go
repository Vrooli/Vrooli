package routing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/ai-provider-go/policy"
	_ "modernc.org/sqlite"
)

// Exercise the actual Stream/SQL/catalog/Hub HTTP composition. The first
// snapshot cannot authorize a later record whose exact human authority changed.
func TestJ3_StreamRevalidatesFinalPolicySnapshot(t *testing.T) {
	for _, tc := range []struct {
		name     string
		changeAt int
		change   string
		allow    bool
	}{
		{"unchanged_approval_dispatches", 0, "", true},
		{"changed_hosts_before_first_stream_read", 2, "hosts", false},
		{"changed_hosts_on_final_stream_read", 3, "hosts", false},
		{"agent_answer_on_final_stream_read", 3, "agent", false},
		{"expired_receipt_on_final_stream_read", 3, "expired_receipt", false},
		{"reopened_catalog_on_final_stream_read", 3, "reopened", false},
		{"changed_quote_on_final_stream_read", 3, "quote", false},
		{"changed_route_model_on_final_stream_read", 3, "model", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
			terms := policy.ProviderTerms{
				PlanRules:     []policy.PlanRule{{Plan: "fixture", AccessTier: policy.AccessTierGrant, UseClasses: map[policy.UseClass]policy.Permission{policy.UseInApp: policy.PermissionAllowed}}},
				Evidence:      policy.Evidence{URL: "https://fixture.invalid/terms"},
				Review:        policy.Review{Status: policy.ReviewApproved, DecisionRef: "isolated-j3-decision"},
				EffectiveFrom: "2026-01-01", OnExpiry: "deny", Downstream: policy.DownstreamRules{DataCollection: "deny", PinnedHosts: []string{"approved-host"}},
			}
			digest, err := policy.ProviderRuleDigest("j3-fixture", terms, nil)
			if err != nil {
				t.Fatal(err)
			}
			receipt := policy.ProviderDecisionReceipt{Source: "ai-gateway", EventType: policy.ProviderDecisionEventType, Requester: terms.Review.DecisionRef, EventID: terms.Review.DecisionRef, CanonicalTermsSHA256: digest, Scope: policy.ProviderDecisionScopeRule, State: "answered", Answer: "approve", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "isolated-fixture", CreatedAt: now.Add(-time.Hour).Format(time.RFC3339Nano), AnsweredAt: now.Add(-time.Minute).Format(time.RFC3339Nano), Deadline: now.Add(time.Hour).Format(time.RFC3339Nano)}
			var hubMu sync.Mutex
			var hubReads atomic.Int64
			hubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hubReads.Add(1)
				if r.URL.Path != providerDecisionLookupPath || r.Header.Get("Authorization") != "Bearer isolated-j3-token" {
					t.Error("unexpected Hub boundary")
					w.WriteHeader(403)
					return
				}
				var input map[string]string
				if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
					t.Error(err)
					w.WriteHeader(400)
					return
				}
				if input["requester_ref"] != "isolated-j3-decision" || input["scope"] != policy.ProviderDecisionScopeRule || input["canonical_terms_sha256"] == "" {
					t.Error("incomplete decision lookup")
				}
				hubMu.Lock()
				value := receipt
				hubMu.Unlock()
				if err := json.NewEncoder(w).Encode(value); err != nil {
					t.Error(err)
				}
			}))
			defer hubServer.Close()
			identity := NewProviderDecisionIdentity(func() (string, error) { return "isolated-credential", nil }, func(context.Context, string) (ExchangedServiceIdentity, error) {
				return ExchangedServiceIdentity{AccessToken: "isolated-j3-token", ExpiresAt: now.Add(time.Hour), Subject: providerDecisionPrincipal, Audience: providerDecisionAudience, Scopes: []string{providerDecisionScope}}, nil
			}, func() time.Time { return now })
			hub := NewProviderDecisionHubClient(identity, hubServer.Client())
			hub.baseURL = func(context.Context) (string, error) { return hubServer.URL, nil }
			root := t.TempDir()
			dir := filepath.Join(root, "resources", "j3-fixture")
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			route := policy.Route{ID: "fixture-route", Supplier: "j3-fixture", Plan: "fixture", Model: "fixture-model", Endpoint: "chat", QuoteRef: "fixture-price", QuotedCredits: 1, StreamQuote: policy.StreamQuote{MaxInputBytes: 4096, MaxOutputTokens: 20, MaxCostMicros: 1, TokenLimitField: "max_tokens"}}
			writeCatalog := func() {
				t.Helper()
				data, err := json.Marshal(map[string]any{"provider": map[string]any{"id": "j3-fixture", "terms": terms, "hosted_routes": []policy.Route{route}}, "models": map[string]any{}})
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "model-policy.json"), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			writeCatalog()
			source := ResourcePolicySource{Root: root, Hub: hub, clock: func() time.Time { return now }}
			reads := 0
			snapshot := func(ctx context.Context, feature string) (PrivatePolicySnapshot, error) {
				reads++
				if reads == tc.changeAt {
					hubMu.Lock()
					switch tc.change {
					case "hosts":
						terms.Downstream.PinnedHosts = []string{"unapproved-host"}
					case "agent":
						receipt.AnsweredByIdentityKind = "agent"
					case "expired_receipt":
						receipt.Deadline = now.Add(-time.Second).Format(time.RFC3339Nano)
					case "reopened":
						terms.Review.Status = policy.ReviewReopened
					case "quote":
						route.QuotedCredits++
					case "model":
						route.Model = "different-model"
					default:
						t.Fatalf("unknown test change %q", tc.change)
					}
					hubMu.Unlock()
					writeCatalog()
				}
				return source.Snapshot(ctx, feature)
			}
			db, err := sql.Open("sqlite", filepath.Join(root, "gateway.sqlite"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			db.SetMaxOpenConns(1)
			if _, err := db.Exec(Schema()); err != nil {
				t.Fatal(err)
			}
			feature := PrivateFeature{ID: "agent.session", Role: "chat", MaxInputBytes: 4096, MaxOutputBytes: 4096, MaxUserTurns: 1}
			service := NewPrivatePolicyService([]PrivateFeature{feature}, snapshot, NewSQLPrivatePlanRepository(db), NewSQLPrivateReceiptRepository(db))
			service.clock = func() time.Time { return now }
			var upstreamCalls atomic.Int64
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upstreamCalls.Add(1)
				var payload struct {
					Provider struct {
						Only           []string `json:"only"`
						DataCollection string   `json:"data_collection"`
						AllowFallbacks bool     `json:"allow_fallbacks"`
					} `json:"provider"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				if len(payload.Provider.Only) != 1 || payload.Provider.Only[0] != "approved-host" || payload.Provider.DataCollection != "deny" || payload.Provider.AllowFallbacks {
					t.Errorf("unapproved downstream envelope: %+v", payload.Provider)
				}
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, "data: {\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":1,\"cost\":0.000001}}\n\ndata: [DONE]\n\n")
			}))
			defer upstream.Close()
			service.SetStreamExecutor(&HTTPStreamExecutor{DB: db, Suppliers: map[string]SupplierEndpoint{"j3-fixture": {URL: upstream.URL}}, Client: upstream.Client()})
			plan, err := service.Plan(context.Background(), PrivatePlanRequest{OperationID: "isolated-j3-stream", FeatureID: feature.ID, Model: "fixture-model", BudgetMicros: 1, CreditMultiplier: 10000})
			if err != nil || !plan.Eligible {
				t.Fatalf("initial plan: %+v, %v", plan, err)
			}
			result, err := service.Stream(context.Background(), PrivateStreamRequest{OperationID: plan.OperationID, Plan: plan, Chat: json.RawMessage(`{"model":"fixture-model","messages":[{"role":"user","content":"hello"}]}`)}, httptest.NewRecorder())
			var claims int
			if queryErr := db.QueryRow(`SELECT COUNT(*) FROM private_provider_receipts`).Scan(&claims); queryErr != nil {
				t.Fatal(queryErr)
			}
			if tc.allow {
				if err != nil || result.State != "completed" || upstreamCalls.Load() != 1 || claims != 1 {
					t.Fatalf("valid approval failed: result=%+v err=%v calls=%d claims=%d", result, err, upstreamCalls.Load(), claims)
				}
			} else {
				if !errors.Is(err, ErrPrivatePlanNotEligible) || upstreamCalls.Load() != 0 || claims != 0 {
					t.Fatalf("unapproved final snapshot reached dispatch: err=%v calls=%d claims=%d", err, upstreamCalls.Load(), claims)
				}
			}
			wantReads := 3
			if tc.changeAt == 2 {
				wantReads = 2
			}
			if reads != wantReads {
				t.Fatalf("snapshot reads=%d want=%d", reads, wantReads)
			}
			t.Logf("snapshots=%d Hub reads=%d upstream calls=%d dispatch claims=%d", reads, hubReads.Load(), upstreamCalls.Load(), claims)
		})
	}
}
