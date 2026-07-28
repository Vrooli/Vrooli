package teams

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"prompt-manager/internal/paths"
	"prompt-manager/store"
	"testing"

	"github.com/gorilla/mux"
)

type stubHeartbeatScheduler struct {
	unscheduled []string
}

func (s *stubHeartbeatScheduler) Schedule(teamID, agentID, schedule string) error {
	return nil
}

func (s *stubHeartbeatScheduler) Unschedule(teamID, agentID string) {
	s.unscheduled = append(s.unscheduled, teamID+"/"+agentID)
}

func TestRemoveMemberCleansDataAndUnschedules(t *testing.T) {
	ctx := context.Background()
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	indexStore := fileStore.Indexes()

	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
		TeamID:  "team-1",
		AgentID: "agent-1",
		Status:  store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("create membership: %v", err)
	}

	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", &store.HeartbeatConfig{
		TeamID:   "team-1",
		AgentID:  "agent-1",
		Enabled:  true,
		Schedule: "0 * * * *",
	}); err != nil {
		t.Fatalf("set heartbeat config: %v", err)
	}
	if err := teamStore.SetResponsibilities(ctx, "team-1", "agent-1", "Do work"); err != nil {
		t.Fatalf("set responsibilities: %v", err)
	}

	logPath := teamStore.GetMemberLogPath("team-1", "agent-1", "2026-02-03T00-00-00Z")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("create logs dir: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("log"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	scheduler := &stubHeartbeatScheduler{}
	handlers := NewHandlers(teamStore, agentStore, relationStore, indexStore, scheduler)

	req := httptest.NewRequest("DELETE", "/teams/team-1/members/agent-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.RemoveMember(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", w.Code, w.Body.String())
	}

	if len(scheduler.unscheduled) != 1 || scheduler.unscheduled[0] != "team-1/agent-1" {
		t.Fatalf("expected unschedule for team-1/agent-1, got %v", scheduler.unscheduled)
	}

	memberDir := filepath.Join(roots.Config, "teams", "team-1", "members", "agent-1")
	if _, err := os.Stat(memberDir); !os.IsNotExist(err) {
		t.Fatalf("expected member directory to be removed, got %v", err)
	}
}

func TestDeleteTeamUnschedulesHeartbeats(t *testing.T) {
	ctx := context.Background()
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	indexStore := fileStore.Indexes()

	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}

	for _, agentID := range []string{"agent-1", "agent-2"} {
		if err := agentStore.Create(ctx, &store.Agent{ID: agentID, DisplayName: agentID}); err != nil {
			t.Fatalf("create agent: %v", err)
		}
		if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
			TeamID:  "team-1",
			AgentID: agentID,
			Status:  store.MemberStatusActive,
		}); err != nil {
			t.Fatalf("create membership: %v", err)
		}
		if err := teamStore.SetHeartbeatConfig(ctx, "team-1", agentID, &store.HeartbeatConfig{
			TeamID:   "team-1",
			AgentID:  agentID,
			Enabled:  true,
			Schedule: "0 * * * *",
		}); err != nil {
			t.Fatalf("set heartbeat config: %v", err)
		}
	}

	scheduler := &stubHeartbeatScheduler{}
	handlers := NewHandlers(teamStore, agentStore, relationStore, indexStore, scheduler)

	req := httptest.NewRequest("DELETE", "/teams/team-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", w.Code, w.Body.String())
	}

	if len(scheduler.unscheduled) != 2 {
		t.Fatalf("expected two unschedule calls, got %v", scheduler.unscheduled)
	}
}
