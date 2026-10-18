package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
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

func TestLineageLetsTheCurrentEffortOrchestratorSteerAnEarlierCheckinsWorker(t *testing.T) {
	ref := func(effort, relationship string) *eventpb.WorkReference {
		return &eventpb.WorkReference{Kind: "effort", Id: effort, Relationship: relationship, Verified: true, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE}
	}
	orchestrate, plain := uuid.New(), uuid.New()
	earlier, current, worker := uuid.New(), uuid.New(), uuid.New()
	otherEffort, child, unscoped, foreignWorker, selfDeclared, stopped := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runs := map[uuid.UUID]*domain.Run{
		// Check-in N started the worker and has ended; check-in N+1 is running.
		earlier: {ID: earlier, Status: domain.RunStatusComplete, AgentProfileID: &orchestrate, WorkReferences: []*eventpb.WorkReference{ref("effort:goal", "orchestrator")}},
		current: {ID: current, Status: domain.RunStatusRunning, AgentProfileID: &orchestrate, WorkReferences: []*eventpb.WorkReference{ref("effort:goal", "orchestrator")}},
		worker:  {ID: worker, Status: domain.RunStatusRunning, ParentRunID: &earlier, WorkReferences: []*eventpb.WorkReference{ref("effort:goal", "worker")}},
		// Refused callers and targets.
		otherEffort:   {ID: otherEffort, Status: domain.RunStatusRunning, AgentProfileID: &orchestrate, WorkReferences: []*eventpb.WorkReference{ref("effort:other", "orchestrator")}},
		child:         {ID: child, Status: domain.RunStatusRunning, ParentRunID: &earlier, AgentProfileID: &orchestrate, WorkReferences: []*eventpb.WorkReference{ref("effort:goal", "orchestrator")}},
		unscoped:      {ID: unscoped, Status: domain.RunStatusRunning, AgentProfileID: &plain, WorkReferences: []*eventpb.WorkReference{ref("effort:goal", "orchestrator")}},
		stopped:       {ID: stopped, Status: domain.RunStatusParked, AgentProfileID: &orchestrate, WorkReferences: []*eventpb.WorkReference{ref("effort:goal", "orchestrator")}},
		foreignWorker: {ID: foreignWorker, Status: domain.RunStatusRunning, ParentRunID: &otherEffort, WorkReferences: []*eventpb.WorkReference{ref("effort:goal", "worker")}},
		selfDeclared:  {ID: selfDeclared, Status: domain.RunStatusRunning, WorkReferences: []*eventpb.WorkReference{ref("effort:goal", "worker")}},
	}
	profiles := map[uuid.UUID]*domain.AgentProfile{orchestrate: {ID: orchestrate, DeclaredScopes: []string{OrchestrateScope}}, plain: {ID: plain}}
	handlerFor := func(caller uuid.UUID) *Handler {
		return New(orchestration.HandlerServices{
			IdentityService: guardIdentityService{result: &orchestration.IdentityVerifyResult{Valid: true, Claims: &identity.Claims{RunID: caller}}},
			RunService:      lineageRuns{runs: runs},
			ProfileService:  lineageProfiles{profiles: profiles},
		})
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runs/x/stop", nil)
	request.Header.Set(cliutil.HeaderAgentIdentityToken, "run-token")
	for _, tc := range []struct {
		name           string
		caller, target uuid.UUID
		want           bool
	}{
		{"current check-in steers the earlier check-in's worker", current, worker, true},
		{"an orchestrator of another effort cannot", otherEffort, worker, false},
		{"a child holding an orchestrator reference cannot", child, worker, false},
		{"a profile without the orchestrate scope cannot", unscoped, worker, false},
		{"a parked orchestrator cannot", stopped, worker, false},
		{"a worker whose parent was not the effort's orchestrator is out of reach", current, foreignWorker, false},
		{"a self-declared worker with no parent is out of reach", current, selfDeclared, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := handlerFor(tc.caller).lineageLifecycleAllowed(request, tc.target, false); got != tc.want {
				t.Fatalf("lineageLifecycleAllowed = %v, want %v", got, tc.want)
			}
		})
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

// quiesceRuns adds a recording, optionally slow QuiesceScenario to the lineage
// run fake.
type quiesceRuns struct {
	lineageRuns
	delay time.Duration
	calls *int
}

func (f quiesceRuns) QuiesceScenario(ctx context.Context, opts orchestration.QuiesceOptions) (*orchestration.QuiesceResult, error) {
	if f.calls != nil {
		*f.calls++
	}
	select {
	case <-time.After(f.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return &orchestration.QuiesceResult{Scenario: opts.Scenario, Drained: true, Reason: "drained"}, nil
}

type fakeEffortEnrollments map[string]*pb.EffortEnrollment

func (f fakeEffortEnrollments) EffortEnrollment(_ context.Context, ref string) (*pb.EffortEnrollment, error) {
	if enrollment, ok := f[ref]; ok {
		return enrollment, nil
	}
	return nil, errors.New("not found")
}

func TestQuiesceAdmitsOnlyTheTargetEffortsRunningOrchestrator(t *testing.T) {
	orchestratorProfile, workerProfile := uuid.New(), uuid.New()
	effortRef := func(id string) []*eventpb.WorkReference {
		return []*eventpb.WorkReference{{
			Kind: "effort", Id: id, Relationship: "orchestrator", Verified: true,
			State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE,
		}}
	}
	orchestrator, child, plainWorker, noEffort, parked, withdrawn := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runs := map[uuid.UUID]*domain.Run{
		orchestrator: {ID: orchestrator, Status: domain.RunStatusRunning, AgentProfileID: &orchestratorProfile, WorkReferences: effortRef("effort:bas")},
		child:        {ID: child, Status: domain.RunStatusRunning, AgentProfileID: &orchestratorProfile, ParentRunID: &orchestrator, WorkReferences: effortRef("effort:bas")},
		plainWorker:  {ID: plainWorker, Status: domain.RunStatusRunning, AgentProfileID: &workerProfile, WorkReferences: effortRef("effort:bas")},
		noEffort:     {ID: noEffort, Status: domain.RunStatusRunning, AgentProfileID: &orchestratorProfile},
		parked:       {ID: parked, Status: domain.RunStatusParked, AgentProfileID: &orchestratorProfile, WorkReferences: effortRef("effort:bas")},
		withdrawn:    {ID: withdrawn, Status: domain.RunStatusRunning, AgentProfileID: &orchestratorProfile, WorkReferences: effortRef("effort:old")},
	}
	profiles := map[uuid.UUID]*domain.AgentProfile{
		orchestratorProfile: {ID: orchestratorProfile, DeclaredScopes: []string{OrchestrateScope}},
		workerProfile:       {ID: workerProfile},
	}
	enrollments := fakeEffortEnrollments{
		"effort:bas": {EffortRef: "effort:bas", DestinationRef: "path:scenarios/browser-automation-studio/docs/internal/goal/GOAL.md"},
		"effort:old": {EffortRef: "effort:old", DestinationRef: "scenarios/browser-automation-studio/docs/GOAL.md", Withdrawn: true},
	}
	quiesce := func(result *orchestration.IdentityVerifyResult, scenario string) (*httptest.ResponseRecorder, int) {
		calls := 0
		h := New(orchestration.HandlerServices{
			IdentityService: guardIdentityService{result: result},
			RunService:      quiesceRuns{lineageRuns: lineageRuns{runs: runs}, calls: &calls},
			ProfileService:  lineageProfiles{profiles: profiles},
		}, WithEffortEnrollments(enrollments))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/quiesce", strings.NewReader(`{"scenario":"`+scenario+`"}`))
		if result != nil {
			req.Header.Set(cliutil.HeaderAgentIdentityToken, "run-token")
		}
		rr := httptest.NewRecorder()
		h.QuiesceScenario(rr, req)
		return rr, calls
	}
	caller := func(id uuid.UUID) *orchestration.IdentityVerifyResult {
		return &orchestration.IdentityVerifyResult{Valid: true, Claims: &identity.Claims{RunID: id}}
	}

	if rr, calls := quiesce(caller(orchestrator), "browser-automation-studio"); rr.Code != http.StatusOK || calls != 1 {
		t.Fatalf("effort orchestrator quiescing its target: status=%d calls=%d body=%s", rr.Code, calls, rr.Body.String())
	}
	if rr, calls := quiesce(nil, "browser-automation-studio"); rr.Code != http.StatusOK || calls != 1 {
		t.Fatalf("operator quiesce: status=%d calls=%d body=%s", rr.Code, calls, rr.Body.String())
	}
	if rr, calls := quiesce(&orchestration.IdentityVerifyResult{Valid: false}, "browser-automation-studio"); rr.Code != http.StatusUnauthorized || calls != 0 {
		t.Fatalf("unverified credential: status=%d calls=%d", rr.Code, calls)
	}

	for _, tc := range []struct {
		name     string
		caller   uuid.UUID
		scenario string
		cause    string
	}{
		{"another scenario", orchestrator, "agent-manager", "effort:bas targets browser-automation-studio"},
		{"child run", child, "browser-automation-studio", "child run"},
		{"no orchestrate scope", plainWorker, "browser-automation-studio", "does not declare " + OrchestrateScope},
		{"no effort reference", noEffort, "browser-automation-studio", "no active orchestrator effort reference"},
		{"not running", parked, "browser-automation-studio", "parked, not running"},
		{"withdrawn effort", withdrawn, "browser-automation-studio", "effort:old is withdrawn"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr, calls := quiesce(caller(tc.caller), tc.scenario)
			if rr.Code != http.StatusForbidden || calls != 0 {
				t.Fatalf("status=%d calls=%d, want 403 without draining; body=%s", rr.Code, calls, rr.Body.String())
			}
			var body struct {
				Error string `json:"error"`
				Hint  string `json:"recovery_hint"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode refusal: %v", err)
			}
			if !strings.Contains(body.Error, tc.cause) || body.Hint == "" {
				t.Fatalf("refusal must name the cause %q and give a hint, got %+v", tc.cause, body)
			}
		})
	}
}

func TestEffortDestinationScenario(t *testing.T) {
	for destination, want := range map[string]string{
		"path:scenarios/browser-automation-studio/docs/internal/goal/GOAL.md":                    "browser-automation-studio",
		"scenarios/browser-automation-studio/docs/internal/REFRACTOR_CONTRACT.json#bas-rehab-v1": "browser-automation-studio",
		"repo:scenarios/personal-planner/docs/internal/goal/GOAL.md":                             "personal-planner",
		"scenario:landing-page-business-suite":                                                   "landing-page-business-suite",
		"scenarios/x/../../scenarios-evil/y":                                                     "",
		"workspace:rcl-campaign/README.md#outcome":                                               "",
		"/home/operator/Vrooli/scenarios/x/GOAL.md":                                              "",
		"docs/agent-system/EFFORT_SUPERVISION.md":                                                "",
		"": "",
	} {
		if got := effortDestinationScenario(destination); got != want {
			t.Errorf("effortDestinationScenario(%q) = %q, want %q", destination, got, want)
		}
	}
}

func TestRunIdentityManagementGrantIsObjectBoundAndCannotApprove(t *testing.T) {
	target := uuid.NewString()
	grant := &requestidentity.OperationGrant{ID: "approval", Approver: "owner", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:run/" + target}, ExpiresAt: time.Now().Add(time.Hour)}
	raw, _ := json.Marshal(grant)
	h := New(orchestration.HandlerServices{IdentityService: guardIdentityService{result: &orchestration.IdentityVerifyResult{Valid: true, Claims: &identity.Claims{RunID: uuid.New(), Scopes: []string{"agent-manager:write"}, Meta: map[string]string{"operation_grant": string(raw)}}}}})
	for _, tc := range []struct {
		operation, target string
		denied            bool
	}{
		{"stop-run", target, false},
		{"continue-run", target, false},
		{"wake-run", target, false},
		{"recover-run", target, false},
		{"stop-run", uuid.NewString(), true},
		{"approve-run", target, true},
		{"stop-all", "", true},
		{"create-run", target, true},
	} {
		req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+tc.target, nil), map[string]string{"id": tc.target})
		req.Header.Set(cliutil.HeaderAgentIdentityToken, "approved-agent")
		if got := h.denyRunInitiatedLifecycleOperation(httptest.NewRecorder(), req, tc.operation); got != tc.denied {
			t.Fatalf("%s target=%s denied=%v", tc.operation, tc.target, got)
		}
	}
}

func TestOnlyARunningOrchestratorCreatesTasksFromARun(t *testing.T) {
	orchestratorProfile, workerProfile := uuid.New(), uuid.New()
	running, parked, worker := uuid.New(), uuid.New(), uuid.New()
	runs := map[uuid.UUID]*domain.Run{
		running: {ID: running, AgentProfileID: &orchestratorProfile, Status: domain.RunStatusRunning},
		parked:  {ID: parked, AgentProfileID: &orchestratorProfile, Status: domain.RunStatusParked},
		worker:  {ID: worker, AgentProfileID: &workerProfile, Status: domain.RunStatusRunning, ParentRunID: &running},
	}
	profiles := map[uuid.UUID]*domain.AgentProfile{
		orchestratorProfile: {ID: orchestratorProfile, DeclaredScopes: []string{OrchestrateScope}},
		workerProfile:       {ID: workerProfile},
	}
	for name, tc := range map[string]struct {
		caller uuid.UUID
		want   bool
	}{"running orchestrator": {running, true}, "parked orchestrator": {parked, false}, "worker": {worker, false}} {
		h := New(orchestration.HandlerServices{
			IdentityService: guardIdentityService{result: &orchestration.IdentityVerifyResult{Valid: true, Claims: &identity.Claims{RunID: tc.caller}}},
			RunService:      lineageRuns{runs: runs},
			ProfileService:  lineageProfiles{profiles: profiles},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
		req.Header.Set(cliutil.HeaderAgentIdentityToken, "run-token")
		if got := h.orchestratorTaskCreateAllowed(req); got != tc.want {
			t.Fatalf("%s: task create allowed = %v, want %v", name, got, tc.want)
		}
	}
}

type wakeByKeyTestRunService struct {
	orchestration.RunService
	caller *domain.Run
	parked []*domain.Run
	woken  []uuid.UUID
}

func (s *wakeByKeyTestRunService) GetRun(_ context.Context, id uuid.UUID) (*domain.Run, error) {
	if s.caller != nil && s.caller.ID == id {
		return s.caller, nil
	}
	for _, run := range s.parked {
		if run.ID == id {
			return run, nil
		}
	}
	return nil, domain.NewNotFoundError("Run", id)
}

func (s *wakeByKeyTestRunService) ParkedRunsOnHandle(context.Context, string, string) ([]*domain.Run, error) {
	return s.parked, nil
}

func (s *wakeByKeyTestRunService) WakeRun(_ context.Context, in orchestration.WakeRunInput) (*domain.Run, error) {
	s.woken = append(s.woken, in.RunID)
	return &domain.Run{ID: in.RunID, Status: domain.RunStatusRunning}, nil
}

func serveWakeByKeyTestRequest(t *testing.T, caller *domain.Run, parked []*domain.Run, grant *requestidentity.OperationGrant) (*httptest.ResponseRecorder, []uuid.UUID) {
	t.Helper()
	runs := &wakeByKeyTestRunService{caller: caller, parked: parked}
	claims := &identity.Claims{RunID: caller.ID}
	h := New(orchestration.HandlerServices{
		RunService:      runs,
		IdentityService: guardIdentityService{result: &orchestration.IdentityVerifyResult{Valid: true, Claims: claims}},
		ProfileService:  lineageProfiles{profiles: map[uuid.UUID]*domain.AgentProfile{}},
	})
	body := encodeProtoJSON(t, &pb.WakeParkedRunsRequest{Producer: "children", Key: "await-key", Result: "resolved"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/wake-by-key", strings.NewReader(string(body)))
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "verified-run-token")
	principal := requestidentity.Principal{
		Kind: requestidentity.ActorAgent, Subject: caller.ID.String(), Verified: true,
		Scopes: []string{"agent-manager:write"}, Grant: grant,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	req = req.WithContext(requestidentity.WithPrincipal(req.Context(), principal))
	rr := httptest.NewRecorder()
	h.WakeParkedRuns(rr, req)
	return rr, runs.woken
}

func wakeByKeyTestGrant(targets ...string) *requestidentity.OperationGrant {
	now := time.Now()
	return &requestidentity.OperationGrant{
		ID: "approved-55add-shaped", Policy: "agent-management+run-creation+effort-metadata+team-preparation+swarm-bookkeeping", Approver: "verified-owner",
		Operations: []string{"agent.investigate", "agent.manage", "run.create", "effort.reconcile", "team.prepare", "swarm.bookkeeping"},
		Targets:    []string{"agent-manager:*", "prompt-manager:*", "swarm-manager:*"},
		Rules: []requestidentity.OperationRule{
			{Policy: "agent-management", Operations: []string{"agent.investigate", "agent.manage"}, Targets: targets},
			{Policy: "run-creation", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}},
			{Policy: "effort-metadata", Operations: []string{"effort.reconcile"}, Targets: []string{"agent-manager:*"}},
			{Policy: "team-preparation", Operations: []string{"team.prepare"}, Targets: []string{"prompt-manager:*"}},
			{Policy: "swarm-bookkeeping", Operations: []string{"swarm.bookkeeping"}, Targets: []string{"swarm-manager:*"}},
		},
		ExpiresAt: now.Add(time.Hour),
	}
}

func TestWakeParkedRunsByKeyChecksExactManagedTargetsBeforeEffects(t *testing.T) {
	callerID := uuid.New()
	targetOne := &domain.Run{ID: uuid.New(), Status: domain.RunStatusParked}
	targetTwo := &domain.Run{ID: uuid.New(), Status: domain.RunStatusParked}
	caller := &domain.Run{ID: callerID, Status: domain.RunStatusRunning}

	t.Run("approved named-run management grant covers resolved key match", func(t *testing.T) {
		rr, woken := serveWakeByKeyTestRequest(t, caller, []*domain.Run{targetOne}, wakeByKeyTestGrant("agent-manager:*"))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", rr.Code, rr.Body.String())
		}
		if len(woken) != 1 || woken[0] != targetOne.ID {
			t.Fatalf("woken IDs = %v, want [%s]", woken, targetOne.ID)
		}
	})

	t.Run("exact target grant cannot wake a different resolved run", func(t *testing.T) {
		rr, woken := serveWakeByKeyTestRequest(t, caller, []*domain.Run{targetOne}, wakeByKeyTestGrant("agent-manager:run/"+targetTwo.ID.String()))
		if rr.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403: %s", rr.Code, rr.Body.String())
		}
		if len(woken) != 0 {
			t.Fatalf("unauthorized match was woken: %v", woken)
		}
	})

	t.Run("unrelated operation cannot wake a resolved run", func(t *testing.T) {
		now := time.Now()
		grant := &requestidentity.OperationGrant{
			ID: "agent-investigation-only", Policy: "agent-investigation", Approver: "verified-owner",
			Operations: []string{"agent.investigate"}, Targets: []string{"agent-manager:*"},
			Rules:     []requestidentity.OperationRule{{Policy: "agent-investigation", Operations: []string{"agent.investigate"}, Targets: []string{"agent-manager:*"}}},
			ExpiresAt: now.Add(time.Hour),
		}
		rr, woken := serveWakeByKeyTestRequest(t, caller, []*domain.Run{targetOne}, grant)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403: %s", rr.Code, rr.Body.String())
		}
		if len(woken) != 0 {
			t.Fatalf("run was woken using an unrelated operation grant: %v", woken)
		}
	})

	t.Run("mixed batch is fully authorized before any run wakes", func(t *testing.T) {
		rr, woken := serveWakeByKeyTestRequest(t, caller, []*domain.Run{targetOne, targetTwo}, wakeByKeyTestGrant("agent-manager:run/"+targetOne.ID.String()))
		if rr.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403: %s", rr.Code, rr.Body.String())
		}
		if len(woken) != 0 {
			t.Fatalf("batch partially woke before authorization completed: %v", woken)
		}
	})

	t.Run("existing parent-lineage wake remains allowed", func(t *testing.T) {
		lineageCaller := &domain.Run{ID: callerID, ParentRunID: &targetOne.ID, Status: domain.RunStatusRunning}
		rr, woken := serveWakeByKeyTestRequest(t, lineageCaller, []*domain.Run{targetOne}, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", rr.Code, rr.Body.String())
		}
		if len(woken) != 1 || woken[0] != targetOne.ID {
			t.Fatalf("woken IDs = %v, want [%s]", woken, targetOne.ID)
		}
	})
}
