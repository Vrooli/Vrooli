package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

type diagnosticUsage struct {
	byOwner map[string]authn.StandingGrantUsage
	err     error
	seen    []string
}

func (u *diagnosticUsage) StandingGrantUsage(_ context.Context, owner string) (authn.StandingGrantUsage, error) {
	u.seen = append(u.seen, owner)
	if u.err != nil {
		return authn.StandingGrantUsage{}, u.err
	}
	return u.byOwner[owner], nil
}

func admissionDiagnosticHandler(authority *StandingGrantAuthority, raiser ApprovalRaiser) http.Handler {
	r := mux.NewRouter()
	RegisterPromptManagerAdmissionDiagnostic(r, authority)
	return StandingGrantMiddleware(authority, raiser)(AgentOperationMiddleware(r))
}

func admissionDiagnosticPrincipal(grant *identity.OperationGrant, scopes ...string) identity.Principal {
	return identity.Principal{
		Kind: identity.ActorAgent, Subject: "investigator-run", Owner: "investigator-owner",
		Verified: true, Scopes: scopes, Grant: grant,
	}
}

func admissionDiagnosticGrant(operations, targets []string) *identity.OperationGrant {
	return &identity.OperationGrant{
		ID: "investigation", Approver: "investigator-owner", Operations: operations,
		Targets: targets, ExpiresAt: time.Now().Add(time.Hour),
	}
}

func requestAdmissionDiagnostic(h http.Handler, principal *identity.Principal) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, promptManagerAdmissionDiagnosticPath, nil)
	if principal != nil {
		r = r.WithContext(identity.WithPrincipal(r.Context(), *principal))
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func decodeAdmissionDiagnostic(t *testing.T, w *httptest.ResponseRecorder) ServiceAdmissionDiagnostic {
	t.Helper()
	var got ServiceAdmissionDiagnostic
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode diagnostic: %v; body=%s", err, w.Body.String())
	}
	return got
}

func TestPromptManagerAdmissionDiagnosticReportsFixedAggregateWithoutRaisingApproval(t *testing.T) {
	repo := &memStandingGrants{}
	if err := repo.Put(context.Background(), standingGrantRecord(
		promptManagerAdmissionDiagnosticSubject, "grant-owner", time.Now().Add(time.Hour),
		authn.StandingGrantLimits{MaxConcurrentRuns: 2},
	)); err != nil {
		t.Fatal(err)
	}
	usage := &diagnosticUsage{byOwner: map[string]authn.StandingGrantUsage{
		"grant-owner": {ConcurrentRuns: 2},
	}}
	raiser := &fakeRaiser{}
	authority := NewStandingGrantAuthority(repo, usage, nil)
	principal := admissionDiagnosticPrincipal(
		admissionDiagnosticGrant([]string{"agent.investigate"}, []string{promptManagerAdmissionDiagnosticTarget}),
		"agent-manager:write",
	)
	w := requestAdmissionDiagnostic(admissionDiagnosticHandler(authority, raiser), &principal)
	if w.Code != http.StatusOK {
		t.Fatalf("authorized read returned %d: %s", w.Code, w.Body.String())
	}
	got := decodeAdmissionDiagnostic(t, w)
	if got.Outcome != ServiceAdmissionWouldRefuseLimit || got.Reason != "grant_limit_reached" || len(got.Limits) != 1 {
		t.Fatalf("unexpected diagnostic: %+v", got)
	}
	if got.Limits[0].Dimension != string(authn.StandingGrantConcurrentRunsLimit) || got.Limits[0].Current != 2 || got.Limits[0].Bound != 2 {
		t.Fatalf("wrong aggregate limit: %+v", got.Limits[0])
	}
	if got.ObservedAt.IsZero() || len(raiser.raised) != 0 {
		t.Fatalf("diagnostic lacks observation time or raised approval: observed=%v raises=%+v", got.ObservedAt, raiser.raised)
	}
	if len(usage.seen) != 1 || usage.seen[0] != "grant-owner" {
		t.Fatalf("usage read did not use grant owner: %v", usage.seen)
	}
	for _, secret := range []string{"prompt-manager", "grant-owner", "investigator-owner", "investigator-run"} {
		if strings.Contains(w.Body.String(), secret) {
			t.Fatalf("diagnostic leaked private identity %q: %s", secret, w.Body.String())
		}
	}
}

func TestPromptManagerAdmissionDiagnosticAcceptsApprovedAgentManagementWildcard(t *testing.T) {
	grant := admissionDiagnosticGrant(nil, nil)
	grant.Policy = "agent-management"
	grant.Rules = []identity.OperationRule{{
		Policy: "agent-management", Operations: []string{"agent.investigate", "agent.manage"},
		Targets: []string{"agent-manager:*"},
	}}
	principal := admissionDiagnosticPrincipal(grant, "agent-manager:write")
	w := requestAdmissionDiagnostic(admissionDiagnosticHandler(NewStandingGrantAuthority(&memStandingGrants{}, fakeUsage{}, nil), nil), &principal)
	if w.Code != http.StatusOK {
		t.Fatalf("approved agent-management wildcard did not cover the fixed Prompt Manager target: %d (%s)", w.Code, w.Body.String())
	}
	got := decodeAdmissionDiagnostic(t, w)
	if got.Outcome != ServiceAdmissionWouldRequireApproval || got.Reason != "no_covering_grant" {
		t.Fatalf("wildcard investigation read changed the independent service admission result: %+v", got)
	}
}

func TestPromptManagerAdmissionDiagnosticAllowsCapableHuman(t *testing.T) {
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"agent-manager:write"}}
	w := requestAdmissionDiagnostic(admissionDiagnosticHandler(NewStandingGrantAuthority(&memStandingGrants{}, fakeUsage{}, nil), nil), &principal)
	if w.Code != http.StatusOK {
		t.Fatalf("capable human was refused the read-only operation: %d (%s)", w.Code, w.Body.String())
	}
	got := decodeAdmissionDiagnostic(t, w)
	if got.Outcome != ServiceAdmissionWouldRequireApproval || got.Reason != "no_covering_grant" {
		t.Fatalf("human diagnostic changed service admission: %+v", got)
	}
}

type admissionDiagnosticPairVerifier struct {
	grant *identity.OperationGrant
}

func (v admissionDiagnosticPairVerifier) Verify(token string) (*cliutil.VerifyResult, error) {
	if token != "baseline-token" {
		return &cliutil.VerifyResult{Valid: false}, nil
	}
	return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{
		RunID: "baseline-run", Subject: "operator", ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}}, nil
}

func (v admissionDiagnosticPairVerifier) VerifyPair(_ context.Context, baselineToken, operationToken string) (*cliutil.VerifyPairResult, error) {
	if baselineToken != "baseline-token" || operationToken != "operation-token" {
		return &cliutil.VerifyPairResult{Valid: false}, nil
	}
	rawGrant, err := json.Marshal(v.grant)
	if err != nil {
		return nil, err
	}
	return &cliutil.VerifyPairResult{Valid: true, Operation: &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{
		RunID: "baseline-run", ExpiresAt: time.Now().Add(time.Hour).Unix(), Scopes: []string{"agent-manager:write"},
		Meta: map[string]string{"operation_grant": string(rawGrant)},
	}}}, nil
}

func TestPromptManagerAdmissionDiagnosticAcceptsFreshPairedOperationProof(t *testing.T) {
	grant := admissionDiagnosticGrant([]string{"agent.investigate"}, []string{"agent-manager:*"})
	principal := identity.Principal{Kind: identity.ActorAgent, Subject: "baseline-run", Owner: "operator", Verified: true}
	r := httptest.NewRequest(http.MethodGet, promptManagerAdmissionDiagnosticPath, nil)
	r.Header.Set(cliutil.HeaderAgentIdentityToken, "baseline-token")
	r.Header.Set(cliutil.HeaderAgentOperationToken, "operation-token")
	r = r.WithContext(identity.WithPrincipal(r.Context(), principal))
	h := provenance.Middleware(admissionDiagnosticPairVerifier{grant: grant})(admissionDiagnosticHandler(NewStandingGrantAuthority(&memStandingGrants{}, fakeUsage{}, nil), nil))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("fresh paired proof was refused: %d (%s)", w.Code, w.Body.String())
	}
	got := decodeAdmissionDiagnostic(t, w)
	if got.Outcome != ServiceAdmissionWouldRequireApproval || got.Reason != "no_covering_grant" {
		t.Fatalf("paired read altered the service's standing-grant state: %+v", got)
	}
}

func TestPromptManagerAdmissionDiagnosticDoesNotTreatServiceRunGrantAsInvestigation(t *testing.T) {
	repo := &memStandingGrants{}
	if err := repo.Put(context.Background(), standingGrantRecord("prompt-manager", "operator", time.Now().Add(time.Hour), authn.StandingGrantLimits{})); err != nil {
		t.Fatal(err)
	}
	service := identity.Principal{Kind: identity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}}
	raiser := &fakeRaiser{}
	h := admissionDiagnosticHandler(NewStandingGrantAuthority(repo, fakeUsage{}, nil), raiser)
	w := requestAdmissionDiagnostic(h, &service)
	if w.Code != http.StatusForbidden || len(raiser.raised) != 0 {
		t.Fatalf("service run-creation authority reached diagnostic or raised ask: status=%d raises=%+v body=%s", w.Code, raiser.raised, w.Body.String())
	}
}

func TestPromptManagerAdmissionDiagnosticPreservesMixedGrantAdmissionSemantics(t *testing.T) {
	now := time.Now()
	repo := &memStandingGrants{}
	blocked := standingGrantRecord(promptManagerAdmissionDiagnosticSubject, "blocked-owner", now.Add(time.Hour), authn.StandingGrantLimits{MaxConcurrentRuns: 1})
	admitted := standingGrantRecord(promptManagerAdmissionDiagnosticSubject, "admitted-owner", now.Add(time.Hour), authn.StandingGrantLimits{MaxConcurrentRuns: 4})
	if err := repo.Put(context.Background(), blocked); err != nil {
		t.Fatal(err)
	}
	if err := repo.Put(context.Background(), admitted); err != nil {
		t.Fatal(err)
	}
	usage := &diagnosticUsage{byOwner: map[string]authn.StandingGrantUsage{
		"blocked-owner":  {ConcurrentRuns: 1},
		"admitted-owner": {ConcurrentRuns: 1},
	}}
	principal := admissionDiagnosticPrincipal(
		admissionDiagnosticGrant([]string{"agent.investigate"}, []string{promptManagerAdmissionDiagnosticTarget}),
		"agent-manager:write",
	)
	w := requestAdmissionDiagnostic(admissionDiagnosticHandler(NewStandingGrantAuthority(repo, usage, nil), nil), &principal)
	if w.Code != http.StatusOK {
		t.Fatalf("authorized read returned %d: %s", w.Code, w.Body.String())
	}
	got := decodeAdmissionDiagnostic(t, w)
	if got.Outcome != ServiceAdmissionWouldAdmit || got.Reason != "grant_covers_operation" || len(got.Limits) != 0 {
		t.Fatalf("an admissible covering grant did not win: %+v", got)
	}
	if len(usage.seen) != 2 {
		t.Fatalf("diagnostic did not evaluate both active owners: %v", usage.seen)
	}
}

func TestPromptManagerAdmissionDiagnosticReportsBudgetAggregate(t *testing.T) {
	repo := &memStandingGrants{}
	if err := repo.Put(context.Background(), standingGrantRecord(
		promptManagerAdmissionDiagnosticSubject, "budget-owner", time.Now().Add(time.Hour),
		authn.StandingGrantLimits{MaxRunBudgetUSD: 50},
	)); err != nil {
		t.Fatal(err)
	}
	usage := &diagnosticUsage{byOwner: map[string]authn.StandingGrantUsage{
		"budget-owner": {CommittedBudgetUSD: 35, RequestBudgetUSD: 20},
	}}
	principal := admissionDiagnosticPrincipal(
		admissionDiagnosticGrant([]string{"agent.investigate"}, []string{promptManagerAdmissionDiagnosticTarget}),
		"agent-manager:write",
	)
	w := requestAdmissionDiagnostic(admissionDiagnosticHandler(NewStandingGrantAuthority(repo, usage, nil), nil), &principal)
	if w.Code != http.StatusOK {
		t.Fatalf("authorized read returned %d: %s", w.Code, w.Body.String())
	}
	got := decodeAdmissionDiagnostic(t, w)
	if got.Outcome != ServiceAdmissionWouldRefuseLimit || len(got.Limits) != 1 ||
		got.Limits[0].Dimension != string(authn.StandingGrantBudgetLimit) ||
		got.Limits[0].Current != 55 || got.Limits[0].Bound != 50 {
		t.Fatalf("budget refusal was not projected faithfully: %+v", got)
	}
}

func TestPromptManagerAdmissionDiagnosticCapsAndFlagsExceededDimensions(t *testing.T) {
	repo := &memStandingGrants{}
	usage := &diagnosticUsage{byOwner: make(map[string]authn.StandingGrantUsage)}
	for i := 1; i <= 12; i++ {
		grant := standingGrantRecord(
			promptManagerAdmissionDiagnosticSubject,
			fmt.Sprintf("owner-%02d", i),
			time.Now().Add(time.Hour),
			authn.StandingGrantLimits{MaxConcurrentRuns: i},
		)
		grant.ID = fmt.Sprintf("grant-%02d", i)
		grant.Grant.ID = grant.ID
		if err := repo.Put(context.Background(), grant); err != nil {
			t.Fatal(err)
		}
		usage.byOwner[grant.Grant.Approver] = authn.StandingGrantUsage{ConcurrentRuns: 100}
	}
	principal := admissionDiagnosticPrincipal(
		admissionDiagnosticGrant([]string{"agent.investigate"}, []string{promptManagerAdmissionDiagnosticTarget}),
		"agent-manager:write",
	)
	w := requestAdmissionDiagnostic(admissionDiagnosticHandler(NewStandingGrantAuthority(repo, usage, nil), nil), &principal)
	if w.Code != http.StatusOK {
		t.Fatalf("authorized read returned %d: %s", w.Code, w.Body.String())
	}
	got := decodeAdmissionDiagnostic(t, w)
	if got.Outcome != ServiceAdmissionWouldRefuseLimit || len(got.Limits) != serviceAdmissionLimitResultCap || !got.LimitsTruncated {
		t.Fatalf("exceeded dimensions were not bounded with explicit truncation: %+v", got)
	}
}

func TestPromptManagerAdmissionDiagnosticReportsAskWithoutSideEffectAndUnavailableRedacted(t *testing.T) {
	principal := admissionDiagnosticPrincipal(
		admissionDiagnosticGrant([]string{"agent.investigate"}, []string{promptManagerAdmissionDiagnosticTarget}),
		"agent-manager:write",
	)
	for _, tc := range []struct {
		name        string
		repo        *memStandingGrants
		usage       StandingGrantUsageReader
		wantStatus  int
		wantOutcome string
		wantReason  string
		wantHidden  string
	}{
		{name: "no grant", repo: &memStandingGrants{}, wantStatus: http.StatusOK, wantOutcome: ServiceAdmissionWouldRequireApproval, wantReason: "no_covering_grant"},
		{name: "usage unavailable", repo: func() *memStandingGrants {
			r := &memStandingGrants{}
			_ = r.Put(context.Background(), standingGrantRecord(promptManagerAdmissionDiagnosticSubject, "owner-secret", time.Now().Add(time.Hour), authn.StandingGrantLimits{MaxConcurrentRuns: 1}))
			return r
		}(), usage: &diagnosticUsage{err: errors.New("private database detail")}, wantStatus: http.StatusServiceUnavailable, wantOutcome: ServiceAdmissionUnavailable, wantReason: "authority_state_unavailable", wantHidden: "private database detail"},
		{name: "grant state unavailable", repo: &memStandingGrants{err: errors.New("private grant-store detail")}, wantStatus: http.StatusServiceUnavailable, wantOutcome: ServiceAdmissionUnavailable, wantReason: "authority_state_unavailable", wantHidden: "private grant-store detail"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raiser := &fakeRaiser{}
			authority := NewStandingGrantAuthority(tc.repo, tc.usage, nil)
			w := requestAdmissionDiagnostic(admissionDiagnosticHandler(authority, raiser), &principal)
			if w.Code != tc.wantStatus {
				t.Fatalf("got %d want %d: %s", w.Code, tc.wantStatus, w.Body.String())
			}
			got := decodeAdmissionDiagnostic(t, w)
			if got.Outcome != tc.wantOutcome || got.Reason != tc.wantReason {
				t.Fatalf("unexpected diagnostic: %+v", got)
			}
			if len(raiser.raised) != 0 {
				t.Fatalf("read path raised approval: %+v", raiser.raised)
			}
			if tc.wantHidden != "" && strings.Contains(w.Body.String(), tc.wantHidden) {
				t.Fatalf("response exposed internal failure: %s", w.Body.String())
			}
		})
	}
}

func TestPromptManagerAdmissionDiagnosticRejectsUnverifiedOrNoncoveringAgents(t *testing.T) {
	authority := NewStandingGrantAuthority(&memStandingGrants{}, fakeUsage{}, nil)
	valid := func(operations, targets []string) *identity.Principal {
		p := admissionDiagnosticPrincipal(admissionDiagnosticGrant(operations, targets), "agent-manager:write")
		return &p
	}
	cases := []struct {
		name      string
		principal *identity.Principal
	}{
		{name: "anonymous"},
		{name: "unverified", principal: func() *identity.Principal {
			p := *valid([]string{"agent.investigate"}, []string{promptManagerAdmissionDiagnosticTarget})
			p.Verified = false
			return &p
		}()},
		{name: "service", principal: func() *identity.Principal {
			p := *valid([]string{"agent.investigate"}, []string{promptManagerAdmissionDiagnosticTarget})
			p.Kind = identity.ActorService
			return &p
		}()},
		{name: "missing write scope", principal: func() *identity.Principal {
			p := admissionDiagnosticPrincipal(admissionDiagnosticGrant([]string{"agent.investigate"}, []string{promptManagerAdmissionDiagnosticTarget}))
			return &p
		}()},
		{name: "wrong operation", principal: valid([]string{"agent.manage"}, []string{promptManagerAdmissionDiagnosticTarget})},
		{name: "other service target", principal: valid([]string{"agent.investigate"}, []string{"agent-manager:service/other"})},
		{name: "expired exact grant", principal: func() *identity.Principal {
			grant := admissionDiagnosticGrant([]string{"agent.investigate"}, []string{promptManagerAdmissionDiagnosticTarget})
			grant.ExpiresAt = time.Now().Add(-time.Second)
			p := admissionDiagnosticPrincipal(grant, "agent-manager:write")
			return &p
		}()},
		{name: "operation and target split across rules", principal: func() *identity.Principal {
			grant := admissionDiagnosticGrant(nil, nil)
			grant.Rules = []identity.OperationRule{
				{Operations: []string{"agent.investigate"}, Targets: []string{"agent-manager:service/other"}},
				{Operations: []string{"agent.manage"}, Targets: []string{promptManagerAdmissionDiagnosticTarget}},
			}
			p := admissionDiagnosticPrincipal(grant, "agent-manager:write")
			return &p
		}()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := requestAdmissionDiagnostic(admissionDiagnosticHandler(authority, nil), tc.principal)
			if w.Code != http.StatusForbidden {
				t.Fatalf("unauthorized caller got %d: %s", w.Code, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "prompt-manager") || strings.Contains(w.Body.String(), "grant_limit") {
				t.Fatalf("refusal disclosed diagnostic data: %s", w.Body.String())
			}
		})
	}
}
