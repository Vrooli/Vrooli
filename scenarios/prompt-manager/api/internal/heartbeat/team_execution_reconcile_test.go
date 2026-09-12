package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// writeQueueFile writes a persistedTeamQueue to disk for Recover tests.
func writeQueueFile(t *testing.T, dir, teamID string, q persistedTeamQueue) {
	t.Helper()
	path := filepath.Join(dir, "team-queue-"+teamID+".json")
	data, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		t.Fatalf("marshal queue: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write queue: %v", err)
	}
}

func readQueueFile(t *testing.T, dir, teamID string) persistedTeamQueue {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "team-queue-"+teamID+".json"))
	if err != nil {
		t.Fatalf("read queue: %v", err)
	}
	var q persistedTeamQueue
	if err := json.Unmarshal(data, &q); err != nil {
		t.Fatalf("unmarshal queue: %v", err)
	}
	return q
}

func TestRecover_DropsRunningWithTerminalRunID(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-terminal"},
		},
	})

	client := newMockAgentClient().
		WithGetRunResponse("run-terminal", &Run{ID: "run-terminal", Status: "RUN_STATUS_COMPLETE"})

	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.RunningAgentIDs) != 0 {
		t.Fatalf("expected terminal entry dropped, got running=%v", status.RunningAgentIDs)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 0 {
		t.Fatalf("expected disk-running cleared, got %v", persisted.Running)
	}
}

func TestRecover_DropsRunningWithEmptyRunID(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: ""},
		},
	})

	client := newMockAgentClient()
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.RunningAgentIDs) != 0 {
		t.Fatalf("expected empty-RunID entry dropped, got running=%v", status.RunningAgentIDs)
	}
	if len(client.getRunCalls) != 0 {
		t.Fatalf("expected no GetRun calls for empty RunID, got %v", client.getRunCalls)
	}
}

func TestRecover_DropsRunningWithMissingRun(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-gone"},
		},
	})

	// (nil, nil) — run not found but no error
	client := newMockAgentClient().WithGetRunResponse("run-gone", nil)
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("expected missing-run entry dropped")
	}
}

func TestRecover_PreservesRunningWithActiveRun(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-active"},
		},
	})

	client := newMockAgentClient().
		WithGetRunResponse("run-active", &Run{ID: "run-active", Status: "RUN_STATUS_RUNNING"})

	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" {
		t.Fatalf("expected active entry preserved, got %v", status.RunningAgentIDs)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].RunID != "run-active" {
		t.Fatalf("expected RunID preserved on disk, got %+v", persisted.Running)
	}
}

func TestRecover_DropsRunningOnAgentManagerError(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-err"},
		},
	})

	client := newMockAgentClient()
	client.getRunErr = errors.New("connection refused")
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("expected entry dropped on GetRun error (conservative drop)")
	}
}

func TestSetRunningRunID_UpdatesEntry(t *testing.T) {
	dir := t.TempDir()
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, nil)
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	tec.SetRunningRunID("agent-1", "run-123")

	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].RunID != "run-123" {
		t.Fatalf("expected RunID persisted, got %+v", persisted.Running)
	}
}

func TestSetRunningRunID_NoopWhenAgentNotRunning(t *testing.T) {
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, t.TempDir(), nil)
	// Does not panic; just logs and returns.
	tec.SetRunningRunID("ghost-agent", "run-x")
	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("expected no running entries after no-op SetRunningRunID")
	}
}

func TestClearRunning_TerminalRunSucceeds(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().
		WithGetRunResponse("run-done", &Run{ID: "run-done", Status: "RUN_STATUS_COMPLETE"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.SetRunningRunID("agent-1", "run-done")

	if err := tec.ClearRunning(context.Background(), "agent-1", false); err != nil {
		t.Fatalf("ClearRunning: %v", err)
	}
	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("expected entry cleared")
	}
}

func TestClearRunning_ActiveRunRefused(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().
		WithGetRunResponse("run-live", &Run{ID: "run-live", Status: "RUN_STATUS_RUNNING"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.SetRunningRunID("agent-1", "run-live")

	err := tec.ClearRunning(context.Background(), "agent-1", false)
	if err == nil {
		t.Fatalf("expected RunningStillActiveError")
	}
	if !IsRunningStillActive(err) {
		t.Fatalf("expected RunningStillActiveError, got %T: %v", err, err)
	}
	if len(tec.Status().RunningAgentIDs) != 1 {
		t.Fatalf("entry should still be present after refusal")
	}
}

func TestClearRunning_ForceBypassesActiveCheck(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().
		WithGetRunResponse("run-live", &Run{ID: "run-live", Status: "RUN_STATUS_RUNNING"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.SetRunningRunID("agent-1", "run-live")

	if err := tec.ClearRunning(context.Background(), "agent-1", true); err != nil {
		t.Fatalf("force ClearRunning: %v", err)
	}
	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("force should clear despite active run")
	}
}

func TestClearRunning_EmptyRunIDUnconditional(t *testing.T) {
	dir := t.TempDir()
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, newMockAgentClient())

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	// No SetRunningRunID — entry has empty RunID.

	if err := tec.ClearRunning(context.Background(), "agent-1", false); err != nil {
		t.Fatalf("ClearRunning: %v", err)
	}
	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("expected entry cleared")
	}
}

func TestClearRunning_NotFound(t *testing.T) {
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, t.TempDir(), nil)
	err := tec.ClearRunning(context.Background(), "ghost", false)
	if err != ErrRunningEntryNotFound {
		t.Fatalf("expected ErrRunningEntryNotFound, got %v", err)
	}
}

// fakeTeamExecStoreRegistrar captures SetRunningRunID calls for executor tests.
type fakeTeamExecStoreRegistrar struct {
	calls []struct{ TeamID, AgentID, RunID string }
}

func (f *fakeTeamExecStoreRegistrar) SetRunningRunID(teamID, agentID, runID string) {
	f.calls = append(f.calls, struct{ TeamID, AgentID, RunID string }{teamID, agentID, runID})
}

func TestExecutor_SetTeamExecStore_RegistersRunIDAfterCreateRun(t *testing.T) {
	// This exercises the Executor.SetTeamExecStore wiring. A full Execute
	// integration test would need a configured team — keep this focused on
	// the seam: the registrar is captured and invokable.
	registrar := &fakeTeamExecStoreRegistrar{}
	e := &Executor{}
	e.SetTeamExecStore(registrar)
	if e.teamExecStore == nil {
		t.Fatalf("expected teamExecStore wired")
	}
	e.teamExecStore.SetRunningRunID("t", "a", "r")
	if len(registrar.calls) != 1 || registrar.calls[0].RunID != "r" {
		t.Fatalf("expected registrar.SetRunningRunID called once, got %+v", registrar.calls)
	}
}

// --- HTTP handler tests ---

func TestClearTeamQueueRunning_OK(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().
		WithGetRunResponse("run-done", &Run{ID: "run-done", Status: "RUN_STATUS_COMPLETE"})
	teamExecStore := NewTeamExecutionStore(nil, &captureExecutor{}, dir, client)

	// Pre-populate a running entry by enqueuing then SetRunningRunID.
	tec := teamExecStore.GetOrCreate("team-x")
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.SetRunningRunID("agent-1", "run-done")

	handlers := &Handlers{teamExecStore: teamExecStore}

	req := httptest.NewRequest(http.MethodDelete, "/teams/team-x/queue/running/agent-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-x", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.ClearTeamQueueRunning(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var status TeamExecutionStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(status.RunningAgentIDs) != 0 {
		t.Fatalf("expected cleared status, got %v", status.RunningAgentIDs)
	}
}

func TestClearTeamQueueRunning_NotFound(t *testing.T) {
	dir := t.TempDir()
	teamExecStore := NewTeamExecutionStore(nil, &captureExecutor{}, dir, newMockAgentClient())
	handlers := &Handlers{teamExecStore: teamExecStore}

	req := httptest.NewRequest(http.MethodDelete, "/teams/team-x/queue/running/ghost", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-x", "agentId": "ghost"})
	w := httptest.NewRecorder()

	handlers.ClearTeamQueueRunning(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestClearTeamQueueRunning_ActiveRefused409(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().
		WithGetRunResponse("run-live", &Run{ID: "run-live", Status: "RUN_STATUS_RUNNING"})
	teamExecStore := NewTeamExecutionStore(nil, &captureExecutor{}, dir, client)
	tec := teamExecStore.GetOrCreate("team-x")
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.SetRunningRunID("agent-1", "run-live")

	handlers := &Handlers{teamExecStore: teamExecStore}
	req := httptest.NewRequest(http.MethodDelete, "/teams/team-x/queue/running/agent-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-x", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.ClearTeamQueueRunning(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "active") {
		t.Fatalf("expected error body to mention active, got %q", w.Body.String())
	}
}

func TestClearTeamQueueRunning_ForceQueryBypassesActive(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().
		WithGetRunResponse("run-live", &Run{ID: "run-live", Status: "RUN_STATUS_RUNNING"})
	teamExecStore := NewTeamExecutionStore(nil, &captureExecutor{}, dir, client)
	tec := teamExecStore.GetOrCreate("team-x")
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.SetRunningRunID("agent-1", "run-live")

	handlers := &Handlers{teamExecStore: teamExecStore}
	req := httptest.NewRequest(http.MethodDelete, "/teams/team-x/queue/running/agent-1?force=true", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-x", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.ClearTeamQueueRunning(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with force, got %d: %s", w.Code, w.Body.String())
	}
}
