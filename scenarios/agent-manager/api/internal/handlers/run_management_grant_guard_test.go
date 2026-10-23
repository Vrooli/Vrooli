package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

const (
	managementBaselineToken  = "a99-baseline-token"
	managementOperationToken = "a99-operation-token"
)

type managementGrantVerifier struct {
	baseline  *cliutil.VerifyResult
	operation *cliutil.VerifyResult
}

func (v managementGrantVerifier) Verify(token string) (*cliutil.VerifyResult, error) {
	if token != managementBaselineToken {
		return nil, nil
	}
	return v.baseline, nil
}

func (v managementGrantVerifier) VerifyPair(_ context.Context, baseline, operation string) (*cliutil.VerifyPairResult, error) {
	if baseline != managementBaselineToken || operation != managementOperationToken {
		return nil, nil
	}
	return &cliutil.VerifyPairResult{Valid: true, Operation: v.operation}, nil
}

func managementGrantFixture(now time.Time, rules ...coreidentity.OperationRule) *coreidentity.OperationGrant {
	return &coreidentity.OperationGrant{
		ID: "approved-55add", Policy: "agent-management+run-creation+effort-metadata+team-preparation+swarm-bookkeeping",
		Approver: "verified-owner", Operations: []string{"agent.investigate", "agent.manage", "run.create", "effort.reconcile", "team.prepare", "swarm.bookkeeping"},
		Targets: []string{"agent-manager:*", "prompt-manager:*", "swarm-manager:*"}, Rules: rules, ExpiresAt: now.Add(time.Hour),
	}
}

func managementRuleFixture(policy string, ops, targets []string) coreidentity.OperationRule {
	return coreidentity.OperationRule{Policy: policy, Operations: ops, Targets: targets}
}

func managementClaimsFixture(t *testing.T, runID uuid.UUID, scopes []string, grant *coreidentity.OperationGrant, expires time.Time) *cliutil.VerifyResult {
	t.Helper()
	meta := map[string]string{}
	if grant != nil {
		raw, err := json.Marshal(grant)
		if err != nil {
			t.Fatal(err)
		}
		meta["operation_grant"] = string(raw)
	}
	return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: runID.String(), TaskID: uuid.NewString(), Subject: "verified-owner", Scopes: scopes, ExpiresAt: expires.Unix(), Meta: meta}}
}

func managementGuardAllows(t *testing.T, baseline, operation *cliutil.VerifyResult, requestedOperation string, requestedTarget string, offerPair bool) bool {
	t.Helper()
	verifier := managementGrantVerifier{baseline: baseline, operation: operation}
	baselineClaims := baseline.Claims
	if baselineClaims == nil {
		t.Fatal("baseline verifier fixture has no claims")
	}
	baselineRunID, err := uuid.Parse(baselineClaims.RunID)
	if err != nil {
		t.Fatal(err)
	}
	h := New(orchestration.HandlerServices{IdentityService: guardIdentityService{result: &orchestration.IdentityVerifyResult{Valid: true, Claims: &identity.Claims{RunID: baselineRunID, TaskID: uuid.MustParse(baselineClaims.TaskID), Subject: baselineClaims.Subject, Scopes: append([]string(nil), baselineClaims.Scopes...), Meta: baselineClaims.Meta}}}})
	router := mux.NewRouter()
	router.Use(provenance.Middleware(verifier))
	router.Use(authn.Middleware(authn.Config{}))
	router.HandleFunc("/api/v1/runs/{id}", func(w http.ResponseWriter, r *http.Request) {
		refused := h.denyRunInitiatedLifecycleOperation(w, r, requestedOperation)
		if !refused {
			w.WriteHeader(http.StatusNoContent)
		}
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+requestedTarget, nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, managementBaselineToken)
	if offerPair {
		req.Header.Set(cliutil.HeaderAgentOperationToken, managementOperationToken)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr.Code == http.StatusNoContent
}

func TestRunLifecycleOperationGrantUsesCanonicalAuthorization(t *testing.T) {
	now := time.Now()
	runID := uuid.New()
	targetID := uuid.NewString()
	approved := managementGrantFixture(now,
		managementRuleFixture("agent-management", []string{"agent.investigate", "agent.manage"}, []string{"agent-manager:*"}),
		managementRuleFixture("run-creation", []string{"run.create"}, []string{"agent-manager:*"}),
		managementRuleFixture("effort-metadata", []string{"effort.reconcile"}, []string{"agent-manager:*"}),
		managementRuleFixture("team-preparation", []string{"team.prepare"}, []string{"prompt-manager:*"}),
		managementRuleFixture("swarm-bookkeeping", []string{"swarm.bookkeeping"}, []string{"swarm-manager:*"}),
	)
	baseline := managementClaimsFixture(t, runID, []string{"agent-manager:read"}, nil, now.Add(time.Hour))
	operation := managementClaimsFixture(t, runID, []string{"agent-manager:read", "agent-manager:write"}, approved, now.Add(time.Hour))
	for _, operationName := range []string{"stop-run", "continue-run", "wake-run", "recover-run"} {
		t.Run("paired_"+operationName, func(t *testing.T) {
			if !managementGuardAllows(t, baseline, operation, operationName, targetID, true) {
				t.Fatal("approved paired agent.manage rule was refused")
			}
		})
	}
	// Legacy grant-only credentials remain supported by the same canonical
	// operation evaluator; the A99 change does not require an operation pair.
	legacy := managementClaimsFixture(t, runID, []string{"agent-manager:write"}, approved, now.Add(time.Hour))
	if !managementGuardAllows(t, legacy, nil, "continue-run", targetID, false) {
		t.Fatal("legacy grant-only lifecycle credential was refused")
	}

	wrongTarget := managementGrantFixture(now, managementRuleFixture("agent-management", []string{"agent.manage"}, []string{"agent-manager:run/" + uuid.NewString()}))
	if managementGuardAllows(t, baseline, managementClaimsFixture(t, runID, []string{"agent-manager:write"}, wrongTarget, now.Add(time.Hour)), "continue-run", targetID, true) {
		t.Fatal("wrong object target was allowed")
	}
	wrongOperation := managementGrantFixture(now, managementRuleFixture("agent-management", []string{"agent.investigate"}, []string{"agent-manager:*"}))
	if managementGuardAllows(t, baseline, managementClaimsFixture(t, runID, []string{"agent-manager:write"}, wrongOperation, now.Add(time.Hour)), "continue-run", targetID, true) {
		t.Fatal("wrong operation was allowed")
	}
	expired := managementGrantFixture(now.Add(-2*time.Hour), managementRuleFixture("agent-management", []string{"agent.manage"}, []string{"agent-manager:*"}))
	if managementGuardAllows(t, baseline, managementClaimsFixture(t, runID, []string{"agent-manager:write"}, expired, now.Add(-time.Hour)), "continue-run", targetID, true) {
		t.Fatal("expired proof was allowed")
	}
	unsupportedBaseline := managementClaimsFixture(t, runID, []string{"agent-manager:read"}, approved, now.Add(time.Hour))
	if managementGuardAllows(t, unsupportedBaseline, nil, "continue-run", targetID, false) {
		t.Fatal("legacy grant without write capability was allowed")
	}
}

type taskCancelAuthoritySpy struct {
	orchestration.TaskService
	cancelled []uuid.UUID
}

func (s *taskCancelAuthoritySpy) CancelTask(_ context.Context, id uuid.UUID) error {
	s.cancelled = append(s.cancelled, id)
	return nil
}

func servePairedTaskCancel(t *testing.T, taskID uuid.UUID, baseline, operation *cliutil.VerifyResult, baselineToken, operationToken string) (*httptest.ResponseRecorder, int) {
	t.Helper()
	spy := &taskCancelAuthoritySpy{}
	h := New(orchestration.HandlerServices{TaskService: spy})
	verifier := managementGrantVerifier{baseline: baseline, operation: operation}
	router := mux.NewRouter()
	router.Use(provenance.Middleware(verifier))
	router.Use(authn.Middleware(authn.Config{}))
	router.Use(AgentOperationMiddleware)
	h.RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID.String()+"/cancel", nil)
	if baselineToken != "" {
		req.Header.Set(cliutil.HeaderAgentIdentityToken, baselineToken)
	}
	if operationToken != "" {
		req.Header.Set(cliutil.HeaderAgentOperationToken, operationToken)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response, len(spy.cancelled)
}

func TestTaskCancelRequiresPairedExactAgentManageGrant(t *testing.T) {
	now := time.Now()
	runID, taskID := uuid.New(), uuid.New()
	allowed := managementGrantFixture(now, managementRuleFixture("agent-management", []string{"agent.manage"}, []string{"agent-manager:*"}))
	exact := managementGrantFixture(now, managementRuleFixture("agent-management", []string{"agent.manage"}, []string{"agent-manager:task/" + taskID.String()}))
	wrongTarget := managementGrantFixture(now, managementRuleFixture("agent-management", []string{"agent.manage"}, []string{"agent-manager:task/" + uuid.NewString()}))
	wrongOperation := managementGrantFixture(now, managementRuleFixture("agent-management", []string{"agent.investigate"}, []string{"agent-manager:*"}))
	expired := managementGrantFixture(now.Add(-2*time.Hour), managementRuleFixture("agent-management", []string{"agent.manage"}, []string{"agent-manager:*"}))
	cases := []struct {
		name             string
		grant            *coreidentity.OperationGrant
		operationToken   string
		baselineToken    string
		baselineScopes   []string
		operationExpires time.Time
		wantStatus       int
		wantCalls        int
	}{
		{name: "domain-wide target is bounded to this operation", grant: allowed, operationToken: managementOperationToken, baselineToken: managementBaselineToken, baselineScopes: []string{"agent-manager:read"}, operationExpires: now.Add(time.Hour), wantStatus: http.StatusOK, wantCalls: 1},
		{name: "exact task target", grant: exact, operationToken: managementOperationToken, baselineToken: managementBaselineToken, baselineScopes: []string{"agent-manager:read"}, operationExpires: now.Add(time.Hour), wantStatus: http.StatusOK, wantCalls: 1},
		{name: "wrong task target", grant: wrongTarget, operationToken: managementOperationToken, baselineToken: managementBaselineToken, baselineScopes: []string{"agent-manager:read"}, operationExpires: now.Add(time.Hour), wantStatus: http.StatusForbidden},
		{name: "wrong operation", grant: wrongOperation, operationToken: managementOperationToken, baselineToken: managementBaselineToken, baselineScopes: []string{"agent-manager:read"}, operationExpires: now.Add(time.Hour), wantStatus: http.StatusForbidden},
		{name: "expired grant", grant: expired, operationToken: managementOperationToken, baselineToken: managementBaselineToken, baselineScopes: []string{"agent-manager:read"}, operationExpires: now.Add(-time.Hour), wantStatus: http.StatusForbidden},
		{name: "invalid pair token", grant: allowed, operationToken: "unverified-operation", baselineToken: managementBaselineToken, baselineScopes: []string{"agent-manager:read"}, operationExpires: now.Add(time.Hour), wantStatus: http.StatusForbidden},
		{name: "missing pair and no grant on baseline", baselineToken: managementBaselineToken, baselineScopes: []string{"agent-manager:read", "agent-manager:write"}, wantStatus: http.StatusForbidden},
		{name: "unverified identity token", baselineToken: "unverified-baseline", baselineScopes: []string{"agent-manager:read", "agent-manager:write"}, wantStatus: http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var operation *cliutil.VerifyResult
			if tc.grant != nil {
				operation = managementClaimsFixture(t, runID, []string{"agent-manager:read", "agent-manager:write"}, tc.grant, tc.operationExpires)
			}
			base := managementClaimsFixture(t, runID, tc.baselineScopes, nil, now.Add(time.Hour))
			response, calls := servePairedTaskCancel(t, taskID, base, operation, tc.baselineToken, tc.operationToken)
			if response.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, tc.wantStatus, response.Body.String())
			}
			if calls != tc.wantCalls {
				t.Fatalf("CancelTask calls = %d, want %d", calls, tc.wantCalls)
			}
		})
	}
}

func TestTaskCancelKeepsHumanRouteAndDeniesOtherTaskWrites(t *testing.T) {
	spy := &taskCancelAuthoritySpy{}
	h := New(orchestration.HandlerServices{TaskService: spy})
	router := mux.NewRouter()
	router.Use(AgentOperationMiddleware)
	h.RegisterRoutes(router)
	human := coreidentity.Principal{Kind: coreidentity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"agent-manager:write"}}
	id := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+id.String()+"/cancel", nil).WithContext(coreidentity.WithPrincipal(context.Background(), human))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusOK || len(spy.cancelled) != 1 {
		t.Fatalf("verified human cancellation: status=%d calls=%d body=%s", response.Code, len(spy.cancelled), response.Body.String())
	}
	grant := managementGrantFixture(time.Now(), managementRuleFixture("agent-management", []string{"agent.manage"}, []string{"agent-manager:*"}))
	agent := coreidentity.Principal{Kind: coreidentity.ActorAgent, Subject: "grant-only", Verified: true, Scopes: []string{"agent-manager:write"}, Grant: grant}
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			taskID := uuid.New()
			caller := agent
			var body io.Reader
			if method == http.MethodPut {
				// Task edits with an explicit agent.manage proof have their own
				// positive coverage; this case verifies the intrinsic path alone
				// cannot admit an unrelated agent.
				caller.Grant = nil
				encoded := encodeProtoJSON(t, &apipb.UpdateTaskRequest{
					TaskId: taskID.String(),
					Task:   &domainpb.Task{Title: "Queued brief", ScopePath: "scenarios/demo"},
				})
				body = bytes.NewReader(encoded)
			}
			req := httptest.NewRequest(method, "/api/v1/tasks/"+taskID.String(), body).WithContext(coreidentity.WithPrincipal(context.Background(), caller))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusForbidden {
				t.Fatalf("%s status = %d, want forbidden", method, w.Code)
			}
		})
	}
	if len(spy.cancelled) != 1 {
		t.Fatalf("unrelated task mutation reached TaskService: %#v", spy.cancelled)
	}
}

func TestTaskCancelServicePrincipalAndChildWithoutGrantAreRefused(t *testing.T) {
	spy := &taskCancelAuthoritySpy{}
	h := New(orchestration.HandlerServices{TaskService: spy})
	router := mux.NewRouter()
	router.Use(AgentOperationMiddleware)
	h.RegisterRoutes(router)
	for _, principal := range []coreidentity.Principal{
		{Kind: coreidentity.ActorAgent, Subject: "child-run", Verified: true, Scopes: []string{"agent-manager:write"}},
		{Kind: coreidentity.ActorService, Subject: "service", Verified: true, Scopes: []string{"agent-manager:write"}},
	} {
		id := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+id.String()+"/cancel", nil).WithContext(coreidentity.WithPrincipal(context.Background(), principal))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("principal kind %s status = %d, want forbidden", principal.Kind, w.Code)
		}
	}
	if len(spy.cancelled) != 0 {
		t.Fatalf("refused principal reached TaskService: %#v", spy.cancelled)
	}
}

func TestTaskCancelDirectHandlerRequiresOperationBeforeService(t *testing.T) {
	spy := &taskCancelAuthoritySpy{}
	h := New(orchestration.HandlerServices{TaskService: spy})
	id := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+id.String()+"/cancel", nil)
	req = mux.SetURLVars(req, map[string]string{"id": id.String()})
	response := httptest.NewRecorder()
	h.CancelTask(response, req)
	if response.Code != http.StatusForbidden || len(spy.cancelled) != 0 {
		t.Fatalf("direct handler bypassed authority: status=%d calls=%d", response.Code, len(spy.cancelled))
	}
}
