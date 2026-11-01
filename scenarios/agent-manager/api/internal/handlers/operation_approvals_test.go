package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/config"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

type operationSettingsFixture struct {
	settings config.OrchestrationSettings
	runs     map[uuid.UUID]*domain.Run
	tasks    map[uuid.UUID]*domain.Task
}

func (f operationSettingsFixture) GetOrchestrationSettings(context.Context) (*config.OrchestrationSettings, error) {
	return &f.settings, nil
}

func (f operationSettingsFixture) GetRun(_ context.Context, id uuid.UUID) (*domain.Run, error) {
	return f.runs[id], nil
}
func (f operationSettingsFixture) GetTask(_ context.Context, id uuid.UUID) (*domain.Task, error) {
	return f.tasks[id], nil
}

func TestOperationApprovalsApplyOnlyToConfiguredVerifiedAgents(t *testing.T) {
	for _, tc := range []struct {
		name              string
		principal         identity.Principal
		required          bool
		operation, target string
		allowed           bool
	}{
		{"disabled", identity.Principal{Kind: identity.ActorAgent, Subject: "run-1", Owner: "owner-1", Verified: true, Scopes: []string{"agent-manager:write"}}, false, "run.create", "agent-manager:*", true},
		{"enabled", identity.Principal{Kind: identity.ActorAgent, Subject: "run-1", Owner: "owner-1", Verified: true, Scopes: []string{"agent-manager:write"}}, true, "run.create", "agent-manager:*", false},
		{"unverified", identity.Principal{Kind: identity.ActorAgent, Scopes: []string{"agent-manager:write"}}, false, "run.create", "agent-manager:*", false},
		{"expired", identity.Principal{Kind: identity.ActorAgent, Subject: "run-1", Owner: "owner-1", Verified: true, Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(-time.Minute)}, false, "run.create", "agent-manager:*", false},
		{"no scope", identity.Principal{Kind: identity.ActorAgent, Subject: "run-1", Owner: "owner-1", Verified: true}, false, "run.create", "agent-manager:*", false},
		{"other domain", identity.Principal{Kind: identity.ActorAgent, Subject: "run-1", Owner: "owner-1", Verified: true, Scopes: []string{"agent-manager:write"}}, false, "run.create", "other:*", false},
		{"unknown operation", identity.Principal{Kind: identity.ActorAgent, Subject: "run-1", Owner: "owner-1", Verified: true, Scopes: []string{"agent-manager:write"}}, false, "unknown", "agent-manager:*", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings := config.DefaultOrchestrationSettings()
			settings.OperationApprovals.RunCreation = tc.required
			var got error
			h := OperationApprovalMiddleware(operationSettingsFixture{settings: settings})(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				_, got = authn.RequireOperation(r.Context(), "agent-manager:write", tc.operation, tc.target)
			}))
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req = req.WithContext(identity.WithPrincipal(req.Context(), tc.principal))
			h.ServeHTTP(httptest.NewRecorder(), req)
			if (got == nil) != tc.allowed {
				t.Fatalf("allowed = %v; error = %v", got == nil, got)
			}
		})
	}
}

func TestOperationApprovalSettingControlsTaskCreation(t *testing.T) {
	_, router := setupTestHandler(t)
	principal := identity.Principal{Kind: identity.ActorAgent, Subject: "agent-1", Owner: "owner-1", Verified: true, Scopes: []string{"agent-manager:write"}}
	for _, tc := range []struct {
		required bool
		status   int
	}{{false, http.StatusCreated}, {true, http.StatusForbidden}} {
		settings := config.DefaultOrchestrationSettings()
		settings.OperationApprovals.RunCreation = tc.required
		h := OperationApprovalMiddleware(operationSettingsFixture{settings: settings})(AgentOperationMiddleware(router))
		r := taskCreateRequest(t, principal, "")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("required=%v: status %d, body %s", tc.required, w.Code, w.Body.String())
		}
	}
}

func TestOperationApprovalManagementRespectsOwner(t *testing.T) {
	runID := uuid.New()
	p := identity.Principal{Kind: identity.ActorAgent, Subject: "agent-1", Owner: "owner-1", Verified: true, Scopes: []string{"agent-manager:write"}}
	for _, tc := range []struct {
		owner    string
		required bool
		allowed  bool
	}{
		{"owner-1", false, true}, {"owner-2", false, false}, {"owner-1", true, false},
	} {
		settings := config.DefaultOrchestrationSettings()
		settings.OperationApprovals.RunManagement = tc.required
		f := operationSettingsFixture{settings: settings, runs: map[uuid.UUID]*domain.Run{runID: {ID: runID, OwnerSubject: tc.owner}}}
		var got error
		h := OperationApprovalMiddleware(f)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			_, got = authn.RequireOperation(r.Context(), "agent-manager:write", "agent.manage", "agent-manager:run/"+runID.String())
		}))
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		h.ServeHTTP(httptest.NewRecorder(), r.WithContext(identity.WithPrincipal(r.Context(), p)))
		if (got == nil) != tc.allowed {
			t.Fatalf("owner=%s required=%v: %v", tc.owner, tc.required, got)
		}
	}
}

func TestOperationApprovalsCoverEachPolicyAndTaskOwnership(t *testing.T) {
	runID, taskID := uuid.New(), uuid.New()
	p := identity.Principal{Kind: identity.ActorAgent, Subject: runID.String(), Owner: "owner-1", Verified: true, Scopes: []string{"agent-manager:write"}}
	for _, tc := range []struct {
		name, operation, target string
		enable                  func(*config.OperationApprovalSettings)
	}{
		{"task management", "agent.manage", "agent-manager:task/" + taskID.String(), func(s *config.OperationApprovalSettings) { s.RunManagement = true }},
		{"run deletion", "run.delete", "agent-manager:run/" + runID.String(), func(s *config.OperationApprovalSettings) { s.RunDeletion = true }},
		{"investigation", "agent.investigate", "agent-manager:run/" + runID.String(), func(s *config.OperationApprovalSettings) { s.Investigation = true }},
		{"delivery", "effort.delivery", "agent-manager:effort/example", func(s *config.OperationApprovalSettings) { s.EffortDelivery = true }},
		{"metadata", "effort.reconcile", "agent-manager:effort/example", func(s *config.OperationApprovalSettings) { s.EffortMetadata = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, required := range []bool{false, true} {
				settings := config.DefaultOrchestrationSettings()
				if required {
					tc.enable(&settings.OperationApprovals)
				}
				f := operationSettingsFixture{settings: settings, runs: map[uuid.UUID]*domain.Run{runID: {ID: runID, OwnerSubject: p.Owner}}, tasks: map[uuid.UUID]*domain.Task{taskID: {ID: taskID, CreatedBy: runID.String()}}}
				var got error
				h := OperationApprovalMiddleware(f)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
					_, got = authn.RequireOperation(r.Context(), "agent-manager:write", tc.operation, tc.target)
				}))
				r := httptest.NewRequest(http.MethodPost, "/", nil)
				h.ServeHTTP(httptest.NewRecorder(), r.WithContext(identity.WithPrincipal(r.Context(), p)))
				if (got == nil) == required {
					t.Fatalf("required=%v: %v", required, got)
				}
			}
		})
	}
}

func TestFreshRecoveryChecksCreationAndSourceManagement(t *testing.T) {
	_, router := setupTestHandler(t)
	runID := uuid.New()
	p := identity.Principal{Kind: identity.ActorAgent, Subject: uuid.NewString(), Owner: "owner-1", Verified: true, Scopes: []string{"agent-manager:write"}}
	for _, tc := range []struct {
		name           string
		create, manage bool
		owner          string
		status         int
	}{
		{"allowed reaches recovery", false, false, "owner-1", http.StatusNotFound},
		{"creation gated", true, false, "owner-1", http.StatusForbidden},
		{"management gated", false, true, "owner-1", http.StatusForbidden},
		{"other owner", false, false, "owner-2", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings := config.DefaultOrchestrationSettings()
			settings.OperationApprovals.RunCreation, settings.OperationApprovals.RunManagement = tc.create, tc.manage
			f := operationSettingsFixture{settings: settings, runs: map[uuid.UUID]*domain.Run{runID: {ID: runID, OwnerSubject: tc.owner}}}
			h := OperationApprovalMiddleware(f)(AgentOperationMiddleware(router))
			r := httptest.NewRequest(http.MethodPost, "/api/v1/runs/resume-from-failed", bytes.NewBufferString(`{"runId":"`+runID.String()+`"}`))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r.WithContext(identity.WithPrincipal(r.Context(), p)))
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestStopAllUsesOwnerScopeWhenManagementApprovalIsOff(t *testing.T) {
	_, router := setupTestHandler(t)
	p := identity.Principal{Kind: identity.ActorAgent, Subject: uuid.NewString(), Owner: "owner-1", Verified: true, Scopes: []string{"agent-manager:write"}}
	for _, tc := range []struct {
		required bool
		status   int
	}{{false, http.StatusOK}, {true, http.StatusForbidden}} {
		settings := config.DefaultOrchestrationSettings()
		settings.OperationApprovals.RunManagement = tc.required
		h := OperationApprovalMiddleware(operationSettingsFixture{settings: settings})(AgentOperationMiddleware(router))
		r := httptest.NewRequest(http.MethodPost, "/api/v1/runs/stop-all", bytes.NewBufferString(`{}`))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r.WithContext(identity.WithPrincipal(r.Context(), p)))
		if w.Code != tc.status {
			t.Fatalf("required=%v status=%d body=%s", tc.required, w.Code, w.Body.String())
		}
	}
}

func TestRunMessageDeletionFollowsManagementPolicy(t *testing.T) {
	handler, _ := setupTestHandler(t)
	runID := uuid.New()
	p := identity.Principal{Kind: identity.ActorAgent, Subject: uuid.NewString(), Owner: "owner-1", Verified: true, Scopes: []string{"agent-manager:write"}}
	for _, required := range []bool{false, true} {
		settings := config.DefaultOrchestrationSettings()
		settings.OperationApprovals.RunManagement = required
		fixture := operationSettingsFixture{settings: settings, runs: map[uuid.UUID]*domain.Run{runID: {ID: runID, OwnerSubject: p.Owner}}}
		denied := false
		h := OperationApprovalMiddleware(fixture)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			denied = handler.denyRunLifecycleOnTarget(w, r, "delete-run-message", runID.String(), "")
		}))
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		h.ServeHTTP(httptest.NewRecorder(), r.WithContext(identity.WithPrincipal(r.Context(), p)))
		if denied != required {
			t.Fatalf("required=%v denied=%v", required, denied)
		}
	}
}
