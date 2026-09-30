package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/cli-core/cliutil"
)

type guardIdentityService struct {
	result *orchestration.IdentityVerifyResult
}

func TestRunIdentityGuardProtectsAllOperatorLifecycleHandlers(t *testing.T) {
	h := New(orchestration.HandlerServices{IdentityService: guardIdentityService{
		result: &orchestration.IdentityVerifyResult{Valid: true, Claims: &identity.Claims{RunID: uuid.New()}},
	}})

	handlers := map[string]http.HandlerFunc{
		"delete":          h.DeleteRun,
		"continue":        h.ContinueRun,
		"wake":            h.WakeRun,
		"recover":         h.RecoverRun,
		"delete-message":  h.DeleteRunMessage,
		"stop-by-tag":     h.StopRunByTag,
		"approve":         h.ApproveRun,
		"reject":          h.RejectRun,
		"partial-approve": h.PartialApproveRun,
		"sandbox-sync":    h.SyncRunFromSandbox,
	}
	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+uuid.NewString(), nil)
			req.Header.Set(cliutil.HeaderAgentIdentityToken, "run-token")
			req = mux.SetURLVars(req, map[string]string{"id": uuid.NewString(), "tag": "tag", "event_id": uuid.NewString()})
			rr := httptest.NewRecorder()
			handler(rr, req)
			if rr.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
			}
		})
	}
}

func (s guardIdentityService) VerifyIdentityToken(context.Context, string) (*orchestration.IdentityVerifyResult, error) {
	return s.result, nil
}

func TestRunIdentityGuardRefusesRunAuthorityAndUnverifiedClaims(t *testing.T) {
	h := New(orchestration.HandlerServices{IdentityService: guardIdentityService{
		result: &orchestration.IdentityVerifyResult{Valid: true, Claims: &identity.Claims{RunID: uuid.New()}},
	}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "run-token")
	rr := httptest.NewRecorder()
	if !h.denyRunInitiatedLifecycleOperation(rr, req, "create-run") {
		t.Fatal("guard did not reject a valid run identity")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}

	for _, result := range []*orchestration.IdentityVerifyResult{
		nil,
		{Valid: false},
		{Valid: true}, // A verdict without signed claims is not an identity.
	} {
		h = New(orchestration.HandlerServices{IdentityService: guardIdentityService{result: result}})
		rr = httptest.NewRecorder()
		if !h.denyRunInitiatedLifecycleOperation(rr, req, "create-run") || rr.Code != http.StatusUnauthorized {
			t.Fatalf("unverified run credential became operator authority: status=%d result=%+v", rr.Code, result)
		}
	}
	// Absence of a run credential retains the existing operator-auth path.
	// This guard does not itself authenticate that operator.
	req.Header.Del(cliutil.HeaderAgentIdentityToken)
	rr = httptest.NewRecorder()
	if h.denyRunInitiatedLifecycleOperation(rr, req, "create-run") {
		t.Fatal("run-specific guard intercepted an operator request")
	}
}

// lineageRuns and lineageProfiles implement only the reads the lineage
// exception performs; any other call panics through the nil embedded interface.
type lineageRuns struct {
	orchestration.RunService
	runs map[uuid.UUID]*domain.Run
}

func (f lineageRuns) GetRun(_ context.Context, id uuid.UUID) (*domain.Run, error) {
	if run, ok := f.runs[id]; ok {
		return run, nil
	}
	return nil, domain.NewNotFoundError("Run", id)
}

type lineageProfiles struct {
	orchestration.ProfileService
	profiles map[uuid.UUID]*domain.AgentProfile
}

func (f lineageProfiles) GetProfile(_ context.Context, id uuid.UUID) (*domain.AgentProfile, error) {
	return f.profiles[id], nil
}

func TestLineageLifecycleExceptionStaysInsideTheCallersLineage(t *testing.T) {
	orchestratorProfile, workerProfile := uuid.New(), uuid.New()
	orchestratorRun, workerRun, strangerRun := uuid.New(), uuid.New(), uuid.New()
	runs := map[uuid.UUID]*domain.Run{
		orchestratorRun: {ID: orchestratorRun, AgentProfileID: &orchestratorProfile},
		workerRun:       {ID: workerRun, AgentProfileID: &workerProfile, ParentRunID: &orchestratorRun},
		strangerRun:     {ID: strangerRun, AgentProfileID: &workerProfile},
	}
	profiles := map[uuid.UUID]*domain.AgentProfile{
		orchestratorProfile: {ID: orchestratorProfile, DeclaredScopes: []string{OrchestrateScope}},
		workerProfile:       {ID: workerProfile},
	}
	handlerFor := func(caller uuid.UUID) *Handler {
		return New(orchestration.HandlerServices{
			IdentityService: guardIdentityService{result: &orchestration.IdentityVerifyResult{Valid: true, Claims: &identity.Claims{RunID: caller}}},
			RunService:      lineageRuns{runs: runs},
			ProfileService:  lineageProfiles{profiles: profiles},
		})
	}
	request := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/x/wake", nil)
		req.Header.Set(cliutil.HeaderAgentIdentityToken, "run-token")
		return req
	}
	for _, tc := range []struct {
		name            string
		caller, target  uuid.UUID
		allowParentWake bool
		want            bool
	}{
		{"worker wakes its parked parent", workerRun, orchestratorRun, true, true},
		{"worker cannot stop its parent", workerRun, orchestratorRun, false, false},
		{"orchestrator manages its own child", orchestratorRun, workerRun, false, true},
		{"orchestrator cannot reach an unrelated run", orchestratorRun, strangerRun, false, false},
		{"worker cannot wake an unrelated run", workerRun, strangerRun, true, false},
		{"profile without the orchestrate scope cannot manage children", strangerRun, workerRun, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := handlerFor(tc.caller).lineageLifecycleAllowed(request(), tc.target, tc.allowParentWake); got != tc.want {
				t.Fatalf("lineageLifecycleAllowed = %v, want %v", got, tc.want)
			}
		})
	}

	operator := httptest.NewRequest(http.MethodPost, "/api/v1/runs/x/wake", nil)
	if handlerFor(workerRun).lineageLifecycleAllowed(operator, orchestratorRun, true) {
		t.Fatal("a request without a run identity must use the operator path, not the lineage exception")
	}
}

func TestLifecycleRefusalHealthReportsBurstsAndRecovers(t *testing.T) {
	base := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		refusals int
		spacing  time.Duration
		readAt   time.Duration
		healthy  bool
	}{
		{"expected in-run refusals", 2, time.Second, 0, true},
		{"burst below threshold", lifecycleRefusalBurstThreshold - 1, time.Second, 0, true},
		{"burst at threshold", lifecycleRefusalBurstThreshold, time.Second, 0, false},
		{"burst leaves the window", lifecycleRefusalBurstThreshold, time.Second, lifecycleRefusalBurstWindow + time.Minute, true},
		{"slow trickle never latches", 3 * lifecycleRefusalBurstThreshold, lifecycleRefusalBurstWindow, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lifecycleRefusalCounts.Lock()
			lifecycleRefusalCounts.recent = nil
			lifecycleRefusalCounts.Unlock()
			now := base
			lifecycleRefusalNow = func() time.Time { return now }
			t.Cleanup(func() { lifecycleRefusalNow = time.Now })
			for i := 0; i < tt.refusals; i++ {
				recordLifecycleRefusal("create-run")
				now = now.Add(tt.spacing)
			}
			now = now.Add(tt.readAt)
			if healthy, reason := LifecycleRefusalFunctionalStatus(); healthy != tt.healthy {
				t.Fatalf("healthy=%t (%s), want %t", healthy, reason, tt.healthy)
			}
		})
	}
}
