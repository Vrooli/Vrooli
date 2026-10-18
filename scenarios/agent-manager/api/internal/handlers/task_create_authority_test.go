package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func taskCreateGrantFixture() *coreidentity.OperationGrant {
	return &coreidentity.OperationGrant{
		ID:         "approved-run-create",
		Policy:     "agent-management",
		Approver:   "owner-a",
		Operations: []string{"agent.investigate", "agent.manage", "run.create"},
		Targets:    []string{"agent-manager:*"},
		Rules: []coreidentity.OperationRule{
			{Policy: "agent-management", Operations: []string{"agent.investigate", "agent.manage"}, Targets: []string{"agent-manager:*"}},
			{Policy: "run-creation", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}},
		},
		ExpiresAt: time.Now().Add(time.Hour),
	}
}

func taskCreateRequest(t *testing.T, principal coreidentity.Principal, token string) *http.Request {
	t.Helper()
	body := encodeProtoJSON(t, &apipb.CreateTaskRequest{Task: &domainpb.Task{Title: "A133 task", ScopePath: "src/a133"}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set(cliutil.HeaderAgentIdentityToken, token)
	}
	return req.WithContext(coreidentity.WithPrincipal(req.Context(), principal))
}

func TestTaskCreateUsesOneRunCreateAuthority(t *testing.T) {
	_, router := setupTestHandler(t)
	serve := AgentOperationMiddleware(router)
	base := coreidentity.Principal{Kind: coreidentity.ActorAgent, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}}

	grant := taskCreateGrantFixture()
	valid := base
	valid.Grant = grant
	positive := httptest.NewRecorder()
	serve.ServeHTTP(positive, taskCreateRequest(t, valid, ""))
	if positive.Code != http.StatusCreated {
		t.Fatalf("approved multi-policy run.create task status=%d body=%s; want %d", positive.Code, positive.Body.String(), http.StatusCreated)
	}

	wrongOperation := base
	wrongOperation.Grant = &coreidentity.OperationGrant{
		ID: "agent-management-only", Approver: "owner-a", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:*"},
		Rules:     []coreidentity.OperationRule{{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:*"}}},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	wrongTarget := base
	wrongTarget.Grant = &coreidentity.OperationGrant{
		ID: "wrong-target", Approver: "owner-a", Operations: []string{"run.create"}, Targets: []string{"agent-manager:run/other"},
		Rules:     []coreidentity.OperationRule{{Policy: "run-creation", Operations: []string{"run.create"}, Targets: []string{"agent-manager:run/other"}}},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	crossedRule := base
	crossedRule.Grant = &coreidentity.OperationGrant{
		ID: "crossed-rule", Approver: "owner-a", Operations: []string{"agent.manage", "run.create"}, Targets: []string{"agent-manager:*", "agent-manager:run/other"},
		Rules: []coreidentity.OperationRule{
			{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:*"}},
			{Policy: "run-creation", Operations: []string{"run.create"}, Targets: []string{"agent-manager:run/other"}},
		},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	expired := base
	expired.Grant = taskCreateGrantFixture()
	expired.ExpiresAt = time.Now().Add(-time.Second)
	unverified := base
	unverified.Verified = false
	unverified.Grant = taskCreateGrantFixture()
	unbound := base
	unbound.Subject = ""
	unbound.Grant = taskCreateGrantFixture()
	noAuthority := base

	for _, tc := range []struct {
		name      string
		principal coreidentity.Principal
		token     string
		want      int
	}{
		{"wrong operation", wrongOperation, "", http.StatusForbidden},
		{"wrong target", wrongTarget, "", http.StatusForbidden},
		{"crossed operations and targets do not flatten", crossedRule, "", http.StatusForbidden},
		{"expired principal grant", expired, "", http.StatusForbidden},
		{"unverified credential", unverified, "unverified-run-token", http.StatusUnauthorized},
		{"unbound principal", unbound, "unbound-run-token", http.StatusUnauthorized},
		{"no ambient operation authority", noAuthority, "", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			serve.ServeHTTP(rr, taskCreateRequest(t, tc.principal, tc.token))
			if rr.Code != tc.want {
				t.Fatalf("task create status=%d body=%s; want %d", rr.Code, rr.Body.String(), tc.want)
			}
		})
	}
}

func TestTaskCreateKeepsIntrinsicOrchestratorAuthorityWithUnrelatedGrant(t *testing.T) {
	h, router := setupTestHandler(t)
	callerID, profileID := uuid.New(), uuid.New()
	h.svc.IdentityService = guardIdentityService{result: &orchestration.IdentityVerifyResult{
		Valid: true, Claims: &identity.Claims{RunID: callerID},
	}}
	h.svc.RunService = lineageRuns{runs: map[uuid.UUID]*domain.Run{
		callerID: {ID: callerID, Status: domain.RunStatusRunning, AgentProfileID: &profileID},
	}}
	h.svc.ProfileService = lineageProfiles{profiles: map[uuid.UUID]*domain.AgentProfile{
		profileID: {ID: profileID, DeclaredScopes: []string{OrchestrateScope}},
	}}

	principal := coreidentity.Principal{
		Kind: coreidentity.ActorAgent, Subject: callerID.String(), Verified: true,
		Scopes: []string{"agent-manager:write"},
		Grant: &coreidentity.OperationGrant{
			ID: "unrelated-agent-management", Approver: "owner-a", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:*"},
		},
	}
	rr := httptest.NewRecorder()
	AgentOperationMiddleware(router).ServeHTTP(rr, taskCreateRequest(t, principal, "live-run-token"))
	if rr.Code != http.StatusCreated {
		t.Fatalf("profile orchestrator with unrelated grant status=%d body=%s; want %d", rr.Code, rr.Body.String(), http.StatusCreated)
	}
}
