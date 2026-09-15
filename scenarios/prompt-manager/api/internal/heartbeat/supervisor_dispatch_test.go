package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"prompt-manager/internal/teamconfig"
)

type supervisorDispatchFake struct {
	*mockAgentClient
	requests   []*api.CreateSupervisorRunRequest
	tokens     []string
	err        error
	omitTaskID bool
}

func (f *supervisorDispatchFake) CreateSupervisorRun(_ context.Context, req *api.CreateSupervisorRunRequest, token string) (*Run, error) {
	f.requests = append(f.requests, req)
	f.tokens = append(f.tokens, token)
	if f.err != nil {
		return nil, f.err
	}
	taskID := req.TaskId
	if f.omitTaskID {
		taskID = ""
	}
	return &Run{ID: "wake-run-1", TaskID: taskID, Tag: "supervision-" + req.IdempotencyKey, Status: "running"}, nil
}

func TestStandingSupervisorDelegatedFutureWakesAndMultipleEfforts(t *testing.T) {
	f := newSupervisionFixture(t)
	agent := &supervisorDispatchFake{mockAgentClient: f.agent}
	f.s.Agent = agent
	f.s.DispatchCredential = func(context.Context) (string, error) { return "secret-dispatch-fixture", nil }
	f.cfg.Supervision.DispatchAuthorization = &teamconfig.SupervisorDispatchBinding{EffortRef: "service:standing", AuthorizationID: "owner-granted-id"}
	f.cfg.Supervision.MaxEffortsPerWake = 2
	f.owner.rows = []EffortObservation{effort("effort:first"), effort("effort:second")}
	f.tick(t)
	f.dispatch(t)
	f.receipt(t, "quiet")
	f.agent.getRuns["wake-run-1"].Status = "complete"
	f.now = f.now.Add(time.Minute)
	f.tick(t)
	f.owner.rows[0].EvidenceRevision = "future-proof"
	f.tick(t)
	f.dispatch(t)
	if len(agent.requests) != 2 || len(f.agent.createRunCalls) != 0 {
		t.Fatal("delegated wake missing or ordinary create fallback occurred")
	}
	first, next := agent.requests[0], agent.requests[1]
	if first.IdempotencyKey == next.IdempotencyKey || first.AuthorizationId != next.AuthorizationId || first.TeamId != "supervisors" || first.MemberId != "leader" || len(first.WorkReferences) != 2 {
		t.Fatal("future wakes lost stable purpose binding or multi-effort attribution")
	}
	state, err := f.s.State.Load("supervisors", "leader")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(state)
	if strings.Contains(string(body), agent.tokens[0]) {
		t.Fatal("wake state persisted dispatcher credential")
	}
}

func TestStandingSupervisorMissingCredentialFailsBeforeTaskWithoutFallback(t *testing.T) {
	f := newSupervisionFixture(t)
	f.s.Agent = &supervisorDispatchFake{mockAgentClient: f.agent}
	f.cfg.Supervision.DispatchAuthorization = &teamconfig.SupervisorDispatchBinding{EffortRef: "service:standing", AuthorizationID: "grant"}
	f.s.DispatchCredential = func(context.Context) (string, error) { return "", errors.New("private-authority-detail") }
	f.owner.rows = []EffortObservation{effort("work-reference-is-not-a-grant")}
	f.tick(t)
	_, err := f.s.Dispatch(context.Background(), "supervisors", "leader")
	if err == nil || strings.Contains(err.Error(), "private-authority-detail") || !strings.Contains(err.Error(), "issue-dispatch") {
		t.Fatal("missing actionable secret-safe authority error")
	}
	state, _ := f.s.State.Load("supervisors", "leader")
	if state.Pending.DispatchStarted || len(f.agent.createRunCalls) != 0 || len(f.agent.createTaskCalls) != 0 {
		t.Fatal("missing credential created work or widened default authority")
	}
}

func TestStandingSupervisorReplaysOnlyTheRetainedDelegatedBinding(t *testing.T) {
	f := newSupervisionFixture(t)
	agent := &supervisorDispatchFake{mockAgentClient: f.agent}
	f.s.Agent = agent
	f.s.DispatchCredential = func(context.Context) (string, error) { return "fresh-dispatch", nil }
	f.cfg.Supervision.DispatchAuthorization = &teamconfig.SupervisorDispatchBinding{EffortRef: "service:standing", AuthorizationID: "grant-v1"}
	wake := &SupervisionWake{ID: "wake-replay", TaskID: "task-1", ProfileKey: "qualified-role-profile", DispatchStarted: true, DispatchMode: "delegated",
		DispatchEffortRef: "service:standing", DispatchAuthorizationID: "grant-v1", Efforts: []EffortObservation{{ID: "effort:one", TargetRevision: "rev-1"}}}
	state := &SupervisionState{Version: 1, Status: "uncertain", Efforts: map[string]SupervisedCut{}, Pending: wake}
	if err := f.s.State.Save("supervisors", "leader", state); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Dispatch(context.Background(), "supervisors", "leader"); err != nil {
		t.Fatal(err)
	}
	got, err := f.s.State.Load("supervisors", "leader")
	if err != nil {
		t.Fatal(err)
	}
	if got.Pending.RunID != "wake-run-1" || got.Pending.DispatchReplayAttempts != 1 || len(agent.requests) != 1 {
		t.Fatalf("expected one exact delegated replay, got state=%+v requests=%d", got.Pending, len(agent.requests))
	}
	req := agent.requests[0]
	if req.IdempotencyKey != wake.ID || req.TaskId != wake.TaskID || req.EffortRef != wake.DispatchEffortRef || req.AuthorizationId != wake.DispatchAuthorizationID || len(req.WorkReferences) != 1 || req.WorkReferences[0].GetId() != "effort:one" {
		t.Fatalf("replay changed the durable dispatch intent: %+v", req)
	}
}

func TestStandingSupervisorReplaysLegacyOrdinaryWakeWithExactIntent(t *testing.T) {
	f := newSupervisionFixture(t)
	wakeID := "wake-legacy-replay"
	tag := "supervision-" + wakeID
	f.cfg.Supervision.DispatchRecovery = &teamconfig.SupervisorDispatchRecovery{WakeID: wakeID, Mode: "ordinary", TaskID: "task-1", ProfileKey: "qualified-role-profile", EvidenceRefs: []string{"run-report:dispatch-refused", "pm-state:pending-wake"}}
	f.agent.WithCreateRunResponse(&Run{ID: "wake-run-legacy", TaskID: "task-1", Tag: tag, Status: "running"})
	wake := &SupervisionWake{ID: wakeID, AccountingRef: "test:allowance", TaskID: "task-1", ProfileKey: "qualified-role-profile", DispatchStarted: true,
		Efforts: []EffortObservation{{ID: "effort:one", TargetRevision: "rev-1"}}}
	state := &SupervisionState{Version: 1, Status: "uncertain", WakesInWindow: 7, Efforts: map[string]SupervisedCut{}, Pending: wake}
	if err := f.s.State.Save("supervisors", "leader", state); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Dispatch(context.Background(), "supervisors", "leader"); err != nil {
		t.Fatal(err)
	}
	got, err := f.s.State.Load("supervisors", "leader")
	if err != nil {
		t.Fatal(err)
	}
	if got.Pending.RunID != "wake-run-legacy" || got.Pending.DispatchReplayAttempts != 1 || got.WakesInWindow != 7 || len(f.agent.createRunCalls) != 1 {
		t.Fatalf("expected one legacy replay, got state=%+v calls=%d", got.Pending, len(f.agent.createRunCalls))
	}
	req := f.agent.createRunCalls[0]
	if req.IdempotencyKey != tag || req.Tag == nil || *req.Tag != tag || req.TaskID != wake.TaskID || req.ProfileRef == nil || req.ProfileRef.ProfileKey != wake.ProfileKey || len(req.WorkReferences) != 1 || req.WorkReferences[0].GetId() != "effort:one" {
		t.Fatalf("legacy replay changed request identity: %+v", req)
	}
	if req.Environment["VROOLI_EFFORT_SUPERVISION_WAKE_ID"] != wakeID || req.Environment["VROOLI_EFFORT_SUPERVISION_ACCOUNTING_REF"] != wake.AccountingRef {
		t.Fatalf("legacy replay lost attribution: %+v", req.Environment)
	}
}

func TestStandingSupervisorKeepsLegacyOrMismatchedDelegatedWakeUncertain(t *testing.T) {
	for _, tc := range []struct {
		name string
		wake *SupervisionWake
	}{
		{name: "legacy", wake: &SupervisionWake{ID: "wake-legacy", TaskID: "task-1", DispatchStarted: true}},
		{name: "binding-mismatch", wake: &SupervisionWake{ID: "wake-mismatch", TaskID: "task-1", DispatchStarted: true, DispatchEffortRef: "service:old", DispatchAuthorizationID: "grant-old"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSupervisionFixture(t)
			agent := &supervisorDispatchFake{mockAgentClient: f.agent}
			f.s.Agent = agent
			f.s.DispatchCredential = func(context.Context) (string, error) { return "fresh-dispatch", nil }
			f.cfg.Supervision.DispatchAuthorization = &teamconfig.SupervisorDispatchBinding{EffortRef: "service:standing", AuthorizationID: "grant-v1"}
			f.cfg.Supervision.MaxEffortsPerWake = 1
			state := &SupervisionState{Version: 1, Status: "uncertain", Efforts: map[string]SupervisedCut{}, Pending: tc.wake}
			if err := f.s.State.Save("supervisors", "leader", state); err != nil {
				t.Fatal(err)
			}
			if _, err := f.s.Dispatch(context.Background(), "supervisors", "leader"); err != nil {
				t.Fatal(err)
			}
			got, err := f.s.State.Load("supervisors", "leader")
			if err != nil {
				t.Fatal(err)
			}
			if got.Pending.RunID != "" || got.Status != "uncertain" || len(agent.requests) != 0 || got.Pending.DispatchReplayError == "" {
				t.Fatalf("wake was replayed or uncertainty was released: state=%+v requests=%d", got.Pending, len(agent.requests))
			}
		})
	}
}

func TestStandingSupervisorRejectsContradictoryOrUnknownDispatchMode(t *testing.T) {
	for _, tc := range []struct {
		name string
		wake *SupervisionWake
		cfg  *teamconfig.SupervisorDispatchBinding
	}{
		{name: "ordinary-with-delegated-fields", wake: &SupervisionWake{ID: "wake-contradictory", TaskID: "task-1", ProfileKey: "qualified-role-profile", DispatchStarted: true, DispatchMode: "ordinary", DispatchEffortRef: "service:standing", DispatchAuthorizationID: "grant-v1"}, cfg: &teamconfig.SupervisorDispatchBinding{EffortRef: "service:standing", AuthorizationID: "grant-v1"}},
		{name: "unknown-mode", wake: &SupervisionWake{ID: "wake-unknown", TaskID: "task-1", ProfileKey: "qualified-role-profile", DispatchStarted: true, DispatchMode: "future-mode"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSupervisionFixture(t)
			agent := &supervisorDispatchFake{mockAgentClient: f.agent}
			f.s.Agent = agent
			f.cfg.Supervision.DispatchAuthorization = tc.cfg
			state := &SupervisionState{Version: 1, Status: "uncertain", Efforts: map[string]SupervisedCut{}, Pending: tc.wake}
			if err := f.s.State.Save("supervisors", "leader", state); err != nil {
				t.Fatal(err)
			}
			if _, err := f.s.Dispatch(context.Background(), "supervisors", "leader"); err != nil {
				t.Fatal(err)
			}
			got, err := f.s.State.Load("supervisors", "leader")
			if err != nil {
				t.Fatal(err)
			}
			if got.Pending.RunID != "" || len(agent.requests) != 0 || len(f.agent.createRunCalls) != 0 || got.Pending.DispatchReplayError == "" {
				t.Fatalf("contradictory/unknown mode escaped uncertainty: state=%+v delegated=%d ordinary=%d", got.Pending, len(agent.requests), len(f.agent.createRunCalls))
			}
		})
	}
}

func TestStandingSupervisorRequiresDelegatedReplayTaskIdentity(t *testing.T) {
	f := newSupervisionFixture(t)
	agent := &supervisorDispatchFake{mockAgentClient: f.agent, omitTaskID: true}
	f.s.Agent = agent
	f.s.DispatchCredential = func(context.Context) (string, error) { return "fresh-dispatch", nil }
	f.cfg.Supervision.DispatchAuthorization = &teamconfig.SupervisorDispatchBinding{EffortRef: "service:standing", AuthorizationID: "grant-v1"}
	wake := &SupervisionWake{ID: "wake-no-task", TaskID: "task-1", ProfileKey: "qualified-role-profile", DispatchStarted: true, DispatchMode: "delegated", DispatchEffortRef: "service:standing", DispatchAuthorizationID: "grant-v1"}
	if err := f.s.State.Save("supervisors", "leader", &SupervisionState{Version: 1, Status: "uncertain", Efforts: map[string]SupervisedCut{}, Pending: wake}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Dispatch(context.Background(), "supervisors", "leader"); err != nil {
		t.Fatal(err)
	}
	got, err := f.s.State.Load("supervisors", "leader")
	if err != nil {
		t.Fatal(err)
	}
	if got.Pending != nil || len(got.UnresolvedWakes) != 1 || got.UnresolvedWakes[0].RunID != "" || got.UnresolvedWakes[0].DispatchReplayAttempts != 1 || got.UnresolvedWakes[0].DispatchReplayError == "" {
		t.Fatalf("empty delegated task identity was accepted or lost: pending=%+v unresolved=%+v", got.Pending, got.UnresolvedWakes)
	}
}

func TestStandingSupervisorBoundsDelegatedReplayAfterOwnerUncertainty(t *testing.T) {
	f := newSupervisionFixture(t)
	agent := &supervisorDispatchFake{mockAgentClient: f.agent, err: errors.New("owner unavailable")}
	f.s.Agent = agent
	f.s.DispatchCredential = func(context.Context) (string, error) { return "fresh-dispatch", nil }
	f.cfg.Supervision.DispatchAuthorization = &teamconfig.SupervisorDispatchBinding{EffortRef: "service:standing", AuthorizationID: "grant-v1"}
	state := &SupervisionState{Version: 1, Status: "uncertain", Efforts: map[string]SupervisedCut{}, Pending: &SupervisionWake{ID: "wake-bounded", TaskID: "task-1", DispatchStarted: true, DispatchMode: "delegated", DispatchEffortRef: "service:standing", DispatchAuthorizationID: "grant-v1"}}
	if err := f.s.State.Save("supervisors", "leader", state); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := f.s.Dispatch(context.Background(), "supervisors", "leader"); err != nil {
			t.Fatal(err)
		}
	}
	got, err := f.s.State.Load("supervisors", "leader")
	if err != nil {
		t.Fatal(err)
	}
	if len(agent.requests) != 1 || got.Pending != nil || len(got.UnresolvedWakes) != 1 || got.UnresolvedWakes[0].DispatchReplayAttempts != 1 || got.UnresolvedWakes[0].RunID != "" || got.Status != "degraded" {
		t.Fatalf("owner uncertainty was retried or lost: state=%+v unresolved=%+v requests=%d", got.Pending, got.UnresolvedWakes, len(agent.requests))
	}
	if got.UnresolvedWakes[0].DispatchReplayError == "" || got.UnresolvedWakes[0].Disposition != "recovery-required" {
		t.Fatal("bounded replay did not retain the owner uncertainty")
	}
}

func TestCreateSupervisorRunCredentialStaysInHeaderAndErrorsAreRedacted(t *testing.T) {
	const token = "secret-dispatch-credential"
	for _, refused := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "refused"}[refused], func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				body, _ := io.ReadAll(r.Body)
				if r.Header.Get("Authorization") != "Bearer "+token || r.Header.Get("X-Agent-Identity-Token") != "" || strings.Contains(string(body), token) || !strings.HasSuffix(r.URL.Path, "/CreateSupervisorRun") {
					t.Error("credential transport escaped dedicated bearer-only RPC")
				}
				w.Header().Set("Content-Type", "application/json")
				if refused {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"code":"permission_denied","message":"` + token + `"}`))
					return
				}
				_, _ = w.Write([]byte(`{"run":{"id":"wake-run","status":"RUN_STATUS_RUNNING"}}`))
			}))
			defer server.Close()
			run, err := newTestClient(t, server).CreateSupervisorRun(context.Background(), &api.CreateSupervisorRunRequest{AuthorizationId: "grant", IdempotencyKey: "wake"}, token)
			if refused {
				if err == nil || strings.Contains(err.Error(), token) {
					t.Fatal("refusal leaked token")
				}
			} else if err != nil || run.ID != "wake-run" {
				t.Fatalf("typed response failed: %v", err)
			}
			if calls != 1 {
				t.Fatal("client retried delegated dispatch")
			}
		})
	}
}

func TestCreateSupervisorRunNeverForwardsCredentialOnRedirect(t *testing.T) {
	forwarded := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { forwarded = true }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	_, err := newTestClient(t, server).CreateSupervisorRun(context.Background(), &api.CreateSupervisorRunRequest{AuthorizationId: "grant", IdempotencyKey: "wake"}, "secret")
	if err == nil || forwarded {
		t.Fatal("credential followed redirect")
	}
}
