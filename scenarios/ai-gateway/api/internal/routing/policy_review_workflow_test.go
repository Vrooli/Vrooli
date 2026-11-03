package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/vrooli/ai-provider-go/policy"
)

type policyReviewSpec struct {
	ID          string                `json:"id"`
	Domain      string                `json:"domain"`
	Initial     policy.ReviewStatus   `json:"initial_state"`
	Terminal    []policy.ReviewStatus `json:"terminal_states"`
	Formal      string                `json:"formal_model_status"`
	Scope       string                `json:"scope"`
	States      []policy.ReviewStatus `json:"states"`
	Events      []policyReviewEvent   `json:"events"`
	Invariants  []string              `json:"invariants"`
	Transitions []struct {
		State policy.ReviewStatus
		Event policyReviewEvent
		Next  policy.ReviewStatus
	} `json:"transitions"`
	Traces []struct {
		ID     string
		Source string
		Steps  []struct {
			Catalog      policy.ReviewStatus
			Event        policyReviewEvent
			Next         policy.ReviewStatus
			Receipt      string
			ExpiresAt    string `json:"expires_at"`
			NextReviewAt string `json:"next_review_at"`
		}
	} `json:"traces"`
}

func readPolicyReviewSpec(t *testing.T) policyReviewSpec {
	t.Helper()
	data, err := os.ReadFile("policy_review_workflow.spec.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec policyReviewSpec
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&spec); err != nil {
		t.Fatal(err)
	}
	return spec
}

// Read the actual typed constant declarations: adding a production state/event
// without a corresponding matrix row must fail rather than silently narrow it.
func policyReviewConstants(t *testing.T, path, kind string) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var values []string
	ast.Inspect(file, func(node ast.Node) bool {
		decl, ok := node.(*ast.ValueSpec)
		if !ok {
			return true
		}
		typ, ok := decl.Type.(*ast.Ident)
		if !ok || typ.Name != kind {
			return true
		}
		for _, v := range decl.Values {
			lit, ok := v.(*ast.BasicLit)
			if !ok {
				t.Fatalf("nonliteral %s constant", kind)
			}
			value, err := strconv.Unquote(lit.Value)
			if err != nil {
				t.Fatal(err)
			}
			values = append(values, value)
		}
		return true
	})
	sort.Strings(values)
	return values
}

func TestPolicyReviewSpecMatrix(t *testing.T) {
	spec := readPolicyReviewSpec(t)
	if spec.ID != "routing.provider-rule-admission" || spec.Domain != "ai-gateway/routing" || spec.Initial != policy.ReviewProposed || spec.Formal != "not_adopted" || len(spec.Terminal) != 0 || len(spec.Invariants) < 5 || len(spec.Traces) < 5 {
		t.Fatal("incomplete policy review contract")
	}
	var states, events []string
	for _, s := range spec.States {
		states = append(states, string(s))
	}
	for _, e := range spec.Events {
		events = append(events, string(e))
	}
	sort.Strings(states)
	sort.Strings(events)
	if !reflect.DeepEqual(states, policyReviewConstants(t, "../../../../../packages/ai-provider-go/policy/provider_rules.go", "ReviewStatus")) {
		t.Fatal("spec does not cover exactly the production review statuses")
	}
	if !reflect.DeepEqual(events, policyReviewConstants(t, "policy_review_workflow.go", "policyReviewEvent")) {
		t.Fatal("spec does not cover exactly the production observations")
	}
	seen := map[string]bool{}
	for _, row := range spec.Transitions {
		key := string(row.State) + "/" + string(row.Event)
		if seen[key] {
			t.Fatalf("duplicate matrix pair %s", key)
		}
		seen[key] = true
		if !containsPolicyValue(states, string(row.State)) || !containsPolicyValue(events, string(row.Event)) || !containsPolicyValue(states, string(row.Next)) {
			t.Fatalf("unknown matrix value: %+v", row)
		}
		t.Run(key, func(t *testing.T) {
			next, err := transitionPolicyReview(row.State, row.Event)
			if err != nil || next != row.Next {
				t.Fatalf("next=%s err=%v want=%s", next, err, row.Next)
			}
			if err := checkPolicyReviewState(next); err != nil {
				t.Fatal(err)
			}
		})
	}
	if len(seen) != len(spec.States)*len(spec.Events) {
		t.Fatalf("incomplete matrix: %d pairs", len(seen))
	}
	for _, state := range spec.States {
		if _, err := transitionPolicyReview(state, "unknown"); err == nil {
			t.Fatal("unknown observation accepted")
		}
	}
	for _, event := range spec.Events {
		if _, err := transitionPolicyReview("unknown", event); err == nil {
			t.Fatal("unknown status accepted")
		}
	}
	t.Logf("consumed %d states × %d events = %d pairs", len(spec.States), len(spec.Events), len(seen))
}

func containsPolicyValue(values []string, value string) bool {
	index := sort.SearchStrings(values, value)
	return index < len(values) && values[index] == value
}

func TestPolicyReviewSpecOwnerTraces(t *testing.T) {
	spec := readPolicyReviewSpec(t)
	for _, trace := range spec.Traces {
		t.Run(trace.ID, func(t *testing.T) {
			now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
			var receipt policy.ProviderDecisionReceipt
			var lookups int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				lookups++
				if r.URL.Path != providerDecisionLookupPath || r.Header.Get("Authorization") != "Bearer isolated-sm6-token" {
					t.Errorf("unexpected Hub boundary request")
					w.WriteHeader(403)
					return
				}
				var input map[string]string
				if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
					t.Error(err)
					w.WriteHeader(400)
					return
				}
				if input["requester_ref"] != "sm6-isolated-proposal" || input["scope"] != policy.ProviderDecisionScopeRule || input["canonical_terms_sha256"] == "" {
					t.Error("missing exact decision request bindings")
				}
				if receipt.State == "missing" {
					w.WriteHeader(404)
					return
				}
				if err := json.NewEncoder(w).Encode(receipt); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			identity := NewProviderDecisionIdentity(func() (string, error) { return "isolated-sm6-credential", nil }, func(context.Context, string) (ExchangedServiceIdentity, error) {
				return ExchangedServiceIdentity{AccessToken: "isolated-sm6-token", ExpiresAt: now.Add(time.Hour), Subject: providerDecisionPrincipal, Audience: providerDecisionAudience, Scopes: []string{providerDecisionScope}}, nil
			}, func() time.Time { return now })
			hub := NewProviderDecisionHubClient(identity, server.Client())
			hub.baseURL = func(context.Context) (string, error) { return server.URL, nil }
			root := t.TempDir()
			dir := filepath.Join(root, "resources", "sm6-fixture")
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			source := ResourcePolicySource{Root: root, Hub: hub, clock: func() time.Time { return now }}
			projected := spec.Initial
			for index, step := range trace.Steps {
				terms := policy.ProviderTerms{Review: policy.Review{Status: step.Catalog, DecisionRef: "sm6-isolated-proposal"}, PlanRules: []policy.PlanRule{{Plan: "fixture", AccessTier: policy.AccessTierGrant, UseClasses: map[policy.UseClass]policy.Permission{policy.UseInApp: policy.PermissionAllowed}}}, Evidence: policy.Evidence{URL: "https://fixture.invalid/terms"}, EffectiveFrom: "2026-01-01", ExpiresAt: step.ExpiresAt, NextReviewAt: step.NextReviewAt, OnExpiry: "deny"}
				digest, err := policy.ProviderRuleDigest("sm6-fixture", terms, nil)
				if err != nil {
					t.Fatal(err)
				}
				receipt = policy.ProviderDecisionReceipt{Source: "ai-gateway", EventType: policy.ProviderDecisionEventType, Requester: terms.Review.DecisionRef, EventID: terms.Review.DecisionRef, CanonicalTermsSHA256: digest, Scope: policy.ProviderDecisionScopeRule, State: "answered", Answer: "approve", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "isolated-fixture", CreatedAt: now.Add(-time.Hour).Format(time.RFC3339Nano), AnsweredAt: now.Add(-time.Minute).Format(time.RFC3339Nano), Deadline: now.Add(time.Hour).Format(time.RFC3339Nano)}
				switch step.Receipt {
				case "human":
				case "agent":
					receipt.AnsweredByIdentityKind = "agent"
				case "unknown_identity":
					receipt.AnsweredByIdentityKind = "unknown"
				case "defaulted":
					receipt.State = "defaulted"
				case "denied":
					receipt.Answer = "deny"
				case "changed_clause":
					terms.PlanRules[0].Clause = "changed after the exact human decision"
				case "expired_receipt":
					receipt.Deadline = now.Add(-time.Second).Format(time.RFC3339Nano)
				case "missing":
					receipt.State = "missing"
				case "wrong_reference":
					receipt.Requester = "different-proposal"
				case "wrong_scope":
					receipt.Scope = policy.ProviderDecisionScopeAgreement
				default:
					t.Fatalf("unknown receipt case %q", step.Receipt)
				}
				route := policy.Route{ID: "sm6-fixture-route", Supplier: "sm6-fixture", Plan: "fixture", Model: "fixture-model", Endpoint: "chat", QuoteRef: "fixture-quote", QuotedCredits: 1}
				data, err := json.Marshal(map[string]any{"provider": map[string]any{"id": "sm6-fixture", "terms": terms, "hosted_routes": []policy.Route{route}}, "models": map[string]any{}})
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(dir, "model-policy.json")
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
				before := lookups
				snapshot, err := source.Snapshot(context.Background(), "fixture")
				if err != nil {
					t.Fatal(err)
				}
				if len(snapshot.Records) != 1 {
					t.Fatalf("step %d: records=%d", index, len(snapshot.Records))
				}
				actual := snapshot.Records[0].Terms.Review.Status
				projected, err = transitionPolicyReview(projected, step.Event)
				if err != nil || projected != step.Next || actual != step.Next {
					t.Fatalf("step %d: model=%s owner=%s want=%s err=%v", index, projected, actual, step.Next, err)
				}
				if step.Catalog != policy.ReviewApproved && lookups != before {
					t.Fatalf("step %d: nonapproved catalog consulted old approval", index)
				}
				if step.Catalog == policy.ReviewApproved && lookups != before+1 {
					t.Fatalf("step %d: expected one fresh exact Hub lookup", index)
				}
				unchanged, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(data, unchanged) {
					t.Fatalf("step %d mutated catalog", index)
				}
				check, err := policy.CheckFeatureInvocation(policy.FeatureContract{ID: "fixture", MaxInputBytes: 20, MaxOutputBytes: 20, MaxUserTurns: 1}, policy.FeatureInvocation{Endpoint: "chat", UserTurns: 1, InputBytes: 1, OutputBytes: 1, SchemaValid: true})
				if err != nil {
					t.Fatal(err)
				}
				request, err := policy.NewEligibleRequest(check, "fixture-model", "", "", "", 1)
				if err != nil {
					t.Fatal(err)
				}
				eligibility := policy.Eligible(request, snapshot.Account, snapshot.Records, snapshot.Agreements, snapshot.Routes, now)
				if (len(eligibility.Routes) == 1) != (step.Next == policy.ReviewApproved) {
					t.Fatalf("step %d: projected %s diverged from actual route eligibility: %+v", index, step.Next, eligibility)
				}
			}
			t.Logf("replayed %d owner observations via isolated catalog + production Hub HTTP client; source: %s", len(trace.Steps), trace.Source)
		})
	}
}

func TestPolicyReviewDeadlineBoundary(t *testing.T) {
	end := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	if policyReviewDeadlineElapsed("2026-11-02", end.Add(-time.Nanosecond)) || !policyReviewDeadlineElapsed("2026-11-02", end) || !policyReviewDeadlineElapsed("invalid", end) || policyReviewDeadlineElapsed("", end) {
		t.Fatal("policy UTC day boundary drift")
	}
}
