package heartbeat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"prompt-manager/internal/paths"
	"prompt-manager/internal/store"

	"github.com/gorilla/mux"
)

func TestStandingSupervisorWiringUsesExistingHeartbeatHistory(t *testing.T) {
	ctx := context.Background()
	fileStore := newFileStore(t, paths.RootsForTest(t))
	teams := fileStore.Teams().(*store.FileTeamStore)
	agents := fileStore.Agents().(*store.FileAgentStore)
	team := newLeaderLedSingleProcessTestTeam("team", "Team", "lead")
	if err := teams.Create(ctx, team); err != nil {
		t.Fatal(err)
	}
	if err := agents.Create(ctx, &store.Agent{ID: "lead", DisplayName: "Leader"}); err != nil {
		t.Fatal(err)
	}
	if err := fileStore.Relations().SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "team", AgentID: "lead", Status: store.MemberStatusActive}); err != nil {
		t.Fatal(err)
	}
	f := newSupervisionFixture(t)
	if err := teams.SetHeartbeatConfig(ctx, "team", "lead", f.cfg); err != nil {
		t.Fatal(err)
	}
	registry := NewRunRegistry(t.TempDir())
	executor := newTestExecutor(t, teams, agents, f.agent, t.TempDir(), registry, nil)
	queue := NewTeamExecutionStore(teams, executor, t.TempDir(), f.agent)
	executor.SetTeamExecStore(queue)
	scheduler := NewScheduler(executor, f.agent, teams, queue)
	s := WireStandingSupervisor(f.owner, executor, queue, scheduler, nil, t.TempDir())
	if _, err := s.Tick(ctx, "team", "lead"); err != nil {
		t.Fatal(err)
	}
	if len(f.agent.createRunCalls) != 0 {
		t.Fatal("wiring launched an empty heartbeat")
	}
	wake := &SupervisionWake{ID: "wake", TaskID: "task", ProfileKey: "qualified", CreatedAt: time.Now().UTC()}
	run := &Run{ID: "actual-run", Status: "running"}
	if err := executor.recordSupervisionExecution(ctx, "team", "lead", wake, run); err != nil {
		t.Fatal(err)
	}
	cfg, err := teams.GetHeartbeatConfig(ctx, "team", "lead")
	if err != nil || cfg.LastExecution == nil || cfg.LastExecution.RunID != "actual-run" || registry.Count() != 1 {
		t.Fatal("owner run was not visible in heartbeat history")
	}
	run.Status, run.Error = "failed", "owner terminal failure"
	for i := 0; i < 2; i++ {
		if err := executor.recordSupervisionExecution(ctx, "team", "lead", wake, run); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err = teams.GetHeartbeatConfig(ctx, "team", "lead")
	if err != nil || cfg.ConsecutiveFailures != 1 || cfg.LastExecution.Status != "failed" || registry.Count() != 0 {
		t.Fatal("terminal owner observation was lost or double-counted")
	}
	if _, err := s.Tick(ctx, "team", "other-agent"); err == nil {
		t.Fatal("nonleader admitted to standing service")
	}
	state, err := s.State.Load("team", "lead")
	if err != nil {
		t.Fatal(err)
	}
	state.Pending = &SupervisionWake{ID: "uncertain-original", DispatchStarted: true}
	if err := s.State.Save("team", "lead", state); err != nil {
		t.Fatal(err)
	}
	h := NewHandlers(HandlersDeps{TeamStore: teams, AgentStore: agents, RelationStore: fileStore.Relations(), Executor: executor})
	req := mux.SetURLVars(httptest.NewRequest(http.MethodDelete, "/teams/team/heartbeats/lead", nil), map[string]string{"id": "team", "agentId": "lead"})
	w := httptest.NewRecorder()
	h.DeleteHeartbeat(w, req)
	if w.Code != http.StatusConflict {
		t.Fatal("deletion bypassed unresolved wake fence")
	}
	if cfg, _ := teams.GetHeartbeatConfig(ctx, "team", "lead"); cfg == nil {
		t.Fatal("fenced config was deleted")
	}
	state.Pending = nil
	if err := s.State.Save("team", "lead", state); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.DeleteHeartbeat(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("resolved config cannot be deleted: %s", w.Body.String())
	}
}
