package handlers

import (
	"context"
	"encoding/json"
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
