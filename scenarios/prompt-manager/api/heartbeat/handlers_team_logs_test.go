package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"prompt-manager/internal/paths"
	"prompt-manager/store"
	"testing"

	"github.com/gorilla/mux"
)

func setupTeamLogsTestHandlers(t *testing.T) (*Handlers, *store.FileTeamStore, *store.FileAgentStore, store.RelationStore, string) {
	t.Helper()
	roots := paths.RootsForTest(t)
	fileStore := store.NewFileStore(roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)
	return handlers, teamStore, agentStore, relationStore, roots.RuntimeData
}

func createLogFiles(t *testing.T, runtimeDataRoot, teamID, agentID string, filenames []string) {
	t.Helper()
	logsDir := filepath.Join(runtimeDataRoot, "teams", teamID, "members", agentID, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatalf("create logs dir: %v", err)
	}
	for _, name := range filenames {
		if err := os.WriteFile(filepath.Join(logsDir, name), []byte("log content"), 0o644); err != nil {
			t.Fatalf("write log file %s: %v", name, err)
		}
	}
}

func TestListTeamLogs_TeamNotFound(t *testing.T) {
	handlers, _, _, _, _ := setupTeamLogsTestHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/teams/nonexistent/heartbeats/logs", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.ListTeamLogs(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListTeamLogs_Aggregation(t *testing.T) {
	handlers, teamStore, agentStore, relationStore, runtimeDataRoot := setupTeamLogsTestHandlers(t)
	ctx := context.Background()

	// Create team
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Test Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}

	// Create agents
	for _, a := range []store.Agent{
		{ID: "agent-1", DisplayName: "Alice"},
		{ID: "agent-2", DisplayName: "Bob"},
	} {
		if err := agentStore.Create(ctx, &a); err != nil {
			t.Fatalf("create agent %s: %v", a.ID, err)
		}
	}

	// Add members
	for _, agentID := range []string{"agent-1", "agent-2"} {
		if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
			TeamID:  "team-1",
			AgentID: agentID,
			Status:  "active",
		}); err != nil {
			t.Fatalf("add member %s: %v", agentID, err)
		}
	}

	// Create log files
	createLogFiles(t, runtimeDataRoot, "team-1", "agent-1", []string{
		"2025-01-01T10-00-00Z.log",
		"2025-01-01T12-00-00Z.log",
	})
	createLogFiles(t, runtimeDataRoot, "team-1", "agent-2", []string{
		"2025-01-01T11-00-00Z.log",
	})

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/heartbeats/logs", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.ListTeamLogs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TeamLogListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Total != 3 {
		t.Fatalf("expected total=3, got %d", resp.Total)
	}
	if len(resp.Logs) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(resp.Logs))
	}

	// Verify sorted by timestamp descending
	if resp.Logs[0].Timestamp != "2025-01-01T12-00-00Z" {
		t.Errorf("expected first log timestamp 2025-01-01T12-00-00Z, got %s", resp.Logs[0].Timestamp)
	}
	if resp.Logs[0].AgentDisplayName != "Alice" {
		t.Errorf("expected first log agent name Alice, got %s", resp.Logs[0].AgentDisplayName)
	}
	if resp.Logs[1].Timestamp != "2025-01-01T11-00-00Z" {
		t.Errorf("expected second log timestamp 2025-01-01T11-00-00Z, got %s", resp.Logs[1].Timestamp)
	}
	if resp.Logs[1].AgentDisplayName != "Bob" {
		t.Errorf("expected second log agent name Bob, got %s", resp.Logs[1].AgentDisplayName)
	}

	if resp.HasMore {
		t.Errorf("expected hasMore=false")
	}
}

func TestListTeamLogs_AgentFilter(t *testing.T) {
	handlers, teamStore, agentStore, relationStore, runtimeDataRoot := setupTeamLogsTestHandlers(t)
	ctx := context.Background()

	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Test Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}

	for _, a := range []store.Agent{
		{ID: "agent-1", DisplayName: "Alice"},
		{ID: "agent-2", DisplayName: "Bob"},
	} {
		if err := agentStore.Create(ctx, &a); err != nil {
			t.Fatalf("create agent %s: %v", a.ID, err)
		}
	}

	for _, agentID := range []string{"agent-1", "agent-2"} {
		if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
			TeamID:  "team-1",
			AgentID: agentID,
			Status:  "active",
		}); err != nil {
			t.Fatalf("add member %s: %v", agentID, err)
		}
	}

	createLogFiles(t, runtimeDataRoot, "team-1", "agent-1", []string{
		"2025-01-01T10-00-00Z.log",
		"2025-01-01T12-00-00Z.log",
	})
	createLogFiles(t, runtimeDataRoot, "team-1", "agent-2", []string{
		"2025-01-01T11-00-00Z.log",
	})

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/heartbeats/logs?agentId=agent-2", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.ListTeamLogs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TeamLogListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Total != 1 {
		t.Fatalf("expected total=1, got %d", resp.Total)
	}
	if len(resp.Logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(resp.Logs))
	}
	if resp.Logs[0].AgentID != "agent-2" {
		t.Errorf("expected agentId=agent-2, got %s", resp.Logs[0].AgentID)
	}
}

func TestListTeamLogs_Pagination(t *testing.T) {
	handlers, teamStore, agentStore, relationStore, runtimeDataRoot := setupTeamLogsTestHandlers(t)
	ctx := context.Background()

	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Test Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}

	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Alice"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
		TeamID:  "team-1",
		AgentID: "agent-1",
		Status:  "active",
	}); err != nil {
		t.Fatalf("add member: %v", err)
	}

	createLogFiles(t, runtimeDataRoot, "team-1", "agent-1", []string{
		"2025-01-01T01-00-00Z.log",
		"2025-01-01T02-00-00Z.log",
		"2025-01-01T03-00-00Z.log",
		"2025-01-01T04-00-00Z.log",
		"2025-01-01T05-00-00Z.log",
	})

	// Page 1: limit=2, offset=0
	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/heartbeats/logs?limit=2&offset=0", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.ListTeamLogs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TeamLogListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Total != 5 {
		t.Fatalf("expected total=5, got %d", resp.Total)
	}
	if len(resp.Logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(resp.Logs))
	}
	if !resp.HasMore {
		t.Errorf("expected hasMore=true for first page")
	}
	// Descending order: 05, 04, 03, 02, 01
	if resp.Logs[0].Timestamp != "2025-01-01T05-00-00Z" {
		t.Errorf("expected first timestamp 2025-01-01T05-00-00Z, got %s", resp.Logs[0].Timestamp)
	}
	if resp.Logs[1].Timestamp != "2025-01-01T04-00-00Z" {
		t.Errorf("expected second timestamp 2025-01-01T04-00-00Z, got %s", resp.Logs[1].Timestamp)
	}

	// Page 2: limit=2, offset=2
	req = httptest.NewRequest(http.MethodGet, "/teams/team-1/heartbeats/logs?limit=2&offset=2", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w = httptest.NewRecorder()

	handlers.ListTeamLogs(w, req)

	var resp2 TeamLogListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp2); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp2.Logs) != 2 {
		t.Fatalf("expected 2 logs on page 2, got %d", len(resp2.Logs))
	}
	if !resp2.HasMore {
		t.Errorf("expected hasMore=true for second page")
	}

	// Page 3: limit=2, offset=4
	req = httptest.NewRequest(http.MethodGet, "/teams/team-1/heartbeats/logs?limit=2&offset=4", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w = httptest.NewRecorder()

	handlers.ListTeamLogs(w, req)

	var resp3 TeamLogListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp3); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp3.Logs) != 1 {
		t.Fatalf("expected 1 log on page 3, got %d", len(resp3.Logs))
	}
	if resp3.HasMore {
		t.Errorf("expected hasMore=false for last page")
	}
}
