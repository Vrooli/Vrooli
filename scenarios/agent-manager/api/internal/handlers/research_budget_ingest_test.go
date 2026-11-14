package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/researchbudget"
	"agent-manager/internal/runmodel"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	_ "modernc.org/sqlite"
)

func TestResearchBudgetIngressRequiresTreasuryServiceAndExactRunGrant(t *testing.T) {
	db, runID := researchBudgetIngressDB(t)
	router := mux.NewRouter()
	RegisterResearchBudgetIngestion(router, researchbudget.Ledger{DB: db})
	h := AgentOperationMiddleware(router)
	body := `{"treasury_budget_id":"budget-1","mandate_id":"mandate-1","agent_subject":"research-agent","cap_minor":500,"currency":"USDC","unit_code":"USDC_ATOMS","policy_revision":"policy-1","created_at":"2026-11-13T12:00:00Z"}`
	path := "/api/v1/research-budgets/" + runID.String() + "/snapshot"
	tests := []struct {
		name      string
		principal identity.Principal
		authority authn.RequestOperationAuthority
		want      int
	}{
		{name: "agent", principal: identity.Principal{Kind: identity.ActorAgent, Subject: "agent-run", Verified: true, Scopes: []string{"agent-manager:write"}, Grant: researchBudgetIngestGrant(runID)}, want: http.StatusForbidden},
		{name: "operator", principal: identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"agent-manager:write"}, Grant: researchBudgetIngestGrant(runID)}, want: http.StatusForbidden},
		{name: "other service", principal: identity.Principal{Kind: identity.ActorService, Subject: "other-service", Verified: true, Scopes: []string{"agent-manager:write"}, Grant: researchBudgetIngestGrant(runID)}, want: http.StatusForbidden},
		// Treasury holds one principal per peer: its name before the split, and its
		// principal for another peer, carry a grant and are still not accepted here.
		{name: "Treasury before the principal split", principal: identity.Principal{Kind: identity.ActorService, Subject: "treasury", Verified: true, Scopes: []string{"agent-manager:write"}, Grant: researchBudgetIngestGrant(runID)}, want: http.StatusForbidden},
		{name: "Treasury principal for another peer", principal: identity.Principal{Kind: identity.ActorService, Subject: "treasury-notification-hub", Verified: true, Scopes: []string{"agent-manager:write"}, Grant: researchBudgetIngestGrant(runID)}, want: http.StatusForbidden},
		{name: "Treasury missing grant", principal: identity.Principal{Kind: identity.ActorService, Subject: "treasury-agent-manager", Verified: true, Scopes: []string{"agent-manager:write"}}, want: http.StatusForbidden},
		{name: "Treasury wrong target", principal: identity.Principal{Kind: identity.ActorService, Subject: "treasury-agent-manager", Verified: true, Scopes: []string{"agent-manager:write"}}, authority: treasuryIngestAuthority{runID: uuid.New()}, want: http.StatusForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
			ctx := identity.WithPrincipal(req.Context(), tc.principal)
			if tc.authority != nil {
				ctx = authn.WithRequestOperationAuthority(ctx, tc.authority)
			}
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status=%d want=%d body=%s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}

func TestResearchBudgetIngressRequestsHumanApprovalForOneExactRun(t *testing.T) {
	db, runID := researchBudgetIngressDB(t)
	router := mux.NewRouter()
	RegisterResearchBudgetIngestion(router, researchbudget.Ledger{DB: db})
	raiser := &testOnlyResearchBudgetRaiser{fakeRaiser: &fakeRaiser{}, testOnly: true}
	authority := NewStandingGrantAuthority(&memStandingGrants{}, nil, nil)
	h := StandingGrantMiddleware(authority, raiser)(AgentOperationMiddleware(router))
	principal := identity.Principal{Kind: identity.ActorService, Subject: "treasury-agent-manager", Verified: true, Scopes: []string{"agent-manager:write"}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/research-budgets/"+runID.String()+"/snapshot", bytes.NewBufferString(`{}`))
	req = req.WithContext(identity.WithPrincipal(req.Context(), principal))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("status=%d want=202 body=%s", w.Code, w.Body.String())
	}
	if len(raiser.raised) != 1 {
		t.Fatalf("expected one human approval request, got %+v", raiser.raised)
	}
	got := raiser.raised[0]
	if got.Subject != "treasury-agent-manager" || got.Operation != researchBudgetOperation || got.Target != "agent-manager:run/"+runID.String() || got.Kind != domain.PendingAuthorizationKindOneTime {
		t.Fatalf("request was not narrowly bound to this run: %+v", got)
	}
	if !strings.Contains(w.Body.String(), `"status":"approval_pending"`) || !strings.Contains(w.Body.String(), got.Target) {
		t.Fatalf("response should name the pending exact-run decision: %s", w.Body.String())
	}
}

func TestResearchBudgetIngressDoesNotRaiseForNonTestRun(t *testing.T) {
	db, runID := researchBudgetIngressDB(t)
	router := mux.NewRouter()
	RegisterResearchBudgetIngestion(router, researchbudget.Ledger{DB: db})
	raiser := &testOnlyResearchBudgetRaiser{fakeRaiser: &fakeRaiser{}, testOnly: false}
	h := StandingGrantMiddleware(NewStandingGrantAuthority(&memStandingGrants{}, nil, nil), raiser)(AgentOperationMiddleware(router))
	principal := identity.Principal{Kind: identity.ActorService, Subject: "treasury-agent-manager", Verified: true, Scopes: []string{"agent-manager:write"}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/research-budgets/"+runID.String()+"/snapshot", bytes.NewBufferString(`{}`))
	req = req.WithContext(identity.WithPrincipal(req.Context(), principal))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden || len(raiser.raised) != 0 {
		t.Fatalf("non-test run must not raise an approval: status=%d requests=%+v body=%s", w.Code, raiser.raised, w.Body.String())
	}
}

type testOnlyResearchBudgetRaiser struct {
	*fakeRaiser
	testOnly bool
}

func (r *testOnlyResearchBudgetRaiser) IsCF7ResearchBudgetTestRun(context.Context, uuid.UUID) bool {
	return r.testOnly
}

func TestResearchBudgetIngressTreasurySnapshotAndSettlementAreIdempotent(t *testing.T) {
	db, runID := researchBudgetIngressDB(t)
	router := mux.NewRouter()
	RegisterResearchBudgetIngestion(router, researchbudget.Ledger{DB: db})
	h := AgentOperationMiddleware(router)
	principal := identity.Principal{Kind: identity.ActorService, Subject: "treasury-agent-manager", Verified: true, Scopes: []string{"agent-manager:write"}, Grant: researchBudgetIngestGrant(runID)}
	post := func(path, body string) int {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
		ctx := identity.WithPrincipal(req.Context(), principal)
		ctx = authn.WithRequestOperationAuthority(ctx, treasuryIngestAuthority{runID: runID})
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code
	}
	base := "/api/v1/research-budgets/" + runID.String()
	snapshot := `{"treasury_budget_id":"budget-1","mandate_id":"mandate-1","agent_subject":"research-agent","cap_minor":500,"currency":"USDC","unit_code":"USDC_ATOMS","policy_revision":"policy-1","created_at":"2026-11-13T12:00:00Z"}`
	if got := post(base+"/snapshot", snapshot); got != http.StatusNoContent {
		t.Fatalf("snapshot status=%d", got)
	}
	if got := post(base+"/snapshot", snapshot); got != http.StatusNoContent {
		t.Fatalf("snapshot replay status=%d", got)
	}
	changed := `{"treasury_budget_id":"budget-1","mandate_id":"mandate-1","agent_subject":"research-agent","cap_minor":501,"currency":"USDC","unit_code":"USDC_ATOMS","policy_revision":"policy-1","created_at":"2026-11-13T12:00:00Z"}`
	if got := post(base+"/snapshot", changed); got != http.StatusConflict {
		t.Fatalf("conflicting snapshot status=%d want 409", got)
	}
	settlement := `{"id":"spend-1","settlement_id":"settlement-1","treasury_receipt_id":"receipt-1","integration_id":"x402:local","category":"x402_call","amount_minor":125,"currency":"USDC","unit_code":"USDC_ATOMS","result_ref":"checkpoint-1","benefit_state":"measured","benefit_summary":"three fixture records accepted","settled_at":"2026-11-13T12:01:00Z"}`
	if got := post(base+"/settlements", settlement); got != http.StatusNoContent {
		t.Fatalf("settlement status=%d", got)
	}
	if got := post(base+"/settlements", settlement); got != http.StatusNoContent {
		t.Fatalf("settlement replay status=%d", got)
	}
	wrongUnit := `{"id":"spend-2","settlement_id":"settlement-2","treasury_receipt_id":"receipt-2","integration_id":"x402:local","category":"x402_call","amount_minor":10,"currency":"EUR","unit_code":"USDC_ATOMS","benefit_state":"none","settled_at":"2026-11-13T12:02:00Z"}`
	if got := post(base+"/settlements", wrongUnit); got != http.StatusConflict {
		t.Fatalf("mismatched currency status=%d want 409", got)
	}
	malformedBenefit := `{"id":"spend-3","settlement_id":"settlement-3","treasury_receipt_id":"receipt-3","integration_id":"x402:local","category":"x402_call","amount_minor":10,"currency":"USDC","unit_code":"USDC_ATOMS","benefit_state":"measured","settled_at":"2026-11-13T12:02:00Z"}`
	if got := post(base+"/settlements", malformedBenefit); got != http.StatusBadRequest {
		t.Fatalf("malformed benefit status=%d want 400", got)
	}
	duplicateReceipt := `{"id":"spend-4","settlement_id":"settlement-4","treasury_receipt_id":"receipt-1","integration_id":"x402:local","category":"x402_call","amount_minor":10,"currency":"USDC","unit_code":"USDC_ATOMS","benefit_state":"none","settled_at":"2026-11-13T12:03:00Z"}`
	if got := post(base+"/settlements", duplicateReceipt); got != http.StatusConflict {
		t.Fatalf("duplicate receipt status=%d want 409", got)
	}
	summary, err := (researchbudget.Ledger{DB: db}).Read(t.Context(), runID.String())
	if err != nil || summary.SpendCount != 1 || summary.SpendMinor != 125 {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
}

type treasuryIngestAuthority struct{ runID uuid.UUID }

func (a treasuryIngestAuthority) AuthorizeOperation(_ context.Context, p identity.Principal, capability, operation, target string) error {
	if p.Kind == identity.ActorService && p.Subject == "treasury-agent-manager" && p.IsVerified() && capability == "agent-manager:write" && operation == researchBudgetOperation && target == "agent-manager:run/"+a.runID.String() {
		return nil
	}
	return authn.ErrOperation
}

func researchBudgetIngestGrant(runID uuid.UUID) *identity.OperationGrant {
	return &identity.OperationGrant{ID: "treasury-budget-ingest", Approver: "operator", Operations: []string{researchBudgetOperation}, Targets: []string{"agent-manager:run/" + runID.String()}, ExpiresAt: time.Now().Add(time.Hour)}
}

func researchBudgetIngressDB(t *testing.T) (*sql.DB, uuid.UUID) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "budget-ingress.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err = db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(runmodel.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(researchbudget.Schema()); err != nil {
		t.Fatal(err)
	}
	runID := uuid.New()
	if _, err = db.Exec(`INSERT INTO runs(id,status) VALUES(?,?)`, runID.String(), "running"); err != nil {
		t.Fatal(err)
	}
	return db, runID
}
