package teams

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"prompt-manager/store"
	"testing"
)

func TestImportClaudeCode_Success(t *testing.T) {
	handlers, teamStore, agentStore, relationStore := setupTestHandlers()

	// Inject a mock CC config reader
	handlers.readCCConfig = func(teamName string) ([]byte, error) {
		return []byte(`{
			"team_name": "my-cc-team",
			"description": "A test CC team",
			"members": [
				{"name": "lead", "agentType": "general-purpose"},
				{"name": "researcher", "agentType": "Explore"}
			]
		}`), nil
	}

	body, _ := json.Marshal(ImportCCRequest{TeamName: "my-cc-team"})
	req := httptest.NewRequest("POST", "/teams/import/claude-code", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.ImportClaudeCode(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp TeamDetailsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.ID != "my-cc-team" {
		t.Errorf("expected team ID 'my-cc-team', got %q", resp.ID)
	}

	// Verify team was created
	if _, ok := teamStore.teams["my-cc-team"]; !ok {
		t.Error("team was not stored")
	}

	// Verify agents were created
	if _, ok := agentStore.agents["lead"]; !ok {
		t.Error("lead agent was not created")
	}
	if _, ok := agentStore.agents["researcher"]; !ok {
		t.Error("researcher agent was not created")
	}

	// Verify member relations
	if _, ok := relationStore.teamMembers["my-cc-team"]["lead"]; !ok {
		t.Error("lead member relation was not created")
	}
	if _, ok := relationStore.teamMembers["my-cc-team"]["researcher"]; !ok {
		t.Error("researcher member relation was not created")
	}
}

func TestImportClaudeCode_FallbackTeamName(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	// Config has empty team_name - should fallback to request team name
	handlers.readCCConfig = func(teamName string) ([]byte, error) {
		return []byte(`{
			"description": "Fallback test",
			"members": [{"name": "agent-1", "agentType": "general-purpose"}]
		}`), nil
	}

	body, _ := json.Marshal(ImportCCRequest{TeamName: "fallback-team"})
	req := httptest.NewRequest("POST", "/teams/import/claude-code", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.ImportClaudeCode(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	if _, ok := teamStore.teams["fallback-team"]; !ok {
		t.Error("expected team stored with fallback name")
	}
}

func TestImportClaudeCode_MissingTeamName(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	body, _ := json.Marshal(ImportCCRequest{TeamName: ""})
	req := httptest.NewRequest("POST", "/teams/import/claude-code", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.ImportClaudeCode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestImportClaudeCode_TeamNotFound(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	handlers.readCCConfig = func(teamName string) ([]byte, error) {
		return nil, &os.PathError{Op: "open", Path: "/not/found", Err: os.ErrNotExist}
	}

	body, _ := json.Marshal(ImportCCRequest{TeamName: "nonexistent"})
	req := httptest.NewRequest("POST", "/teams/import/claude-code", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.ImportClaudeCode(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestImportClaudeCode_InvalidJSON(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	handlers.readCCConfig = func(teamName string) ([]byte, error) {
		return []byte(`{not valid json`), nil
	}

	body, _ := json.Marshal(ImportCCRequest{TeamName: "bad-config"})
	req := httptest.NewRequest("POST", "/teams/import/claude-code", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.ImportClaudeCode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestImportClaudeCode_Conflict(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	// Pre-create a team with the same slug
	teamStore.teams["existing-team"] = &store.Team{ID: "existing-team", DisplayName: "Existing"}

	handlers.readCCConfig = func(teamName string) ([]byte, error) {
		return []byte(`{
			"team_name": "existing-team",
			"members": [{"name": "agent", "agentType": "general-purpose"}]
		}`), nil
	}

	body, _ := json.Marshal(ImportCCRequest{TeamName: "existing-team"})
	req := httptest.NewRequest("POST", "/teams/import/claude-code", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.ImportClaudeCode(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestImportClaudeCode_SkipsExistingAgents(t *testing.T) {
	handlers, _, agentStore, _ := setupTestHandlers()

	// Pre-create an agent
	agentStore.agents["lead"] = &store.Agent{ID: "lead", DisplayName: "Existing Lead"}

	handlers.readCCConfig = func(teamName string) ([]byte, error) {
		return []byte(`{
			"team_name": "skip-test",
			"members": [
				{"name": "lead", "agentType": "general-purpose"},
				{"name": "new-agent", "agentType": "Bash"}
			]
		}`), nil
	}

	body, _ := json.Marshal(ImportCCRequest{TeamName: "skip-test"})
	req := httptest.NewRequest("POST", "/teams/import/claude-code", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.ImportClaudeCode(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Existing agent should keep its original display name
	if agentStore.agents["lead"].DisplayName != "Existing Lead" {
		t.Errorf("existing agent was overwritten, display name = %q", agentStore.agents["lead"].DisplayName)
	}

	// New agent should be created
	if _, ok := agentStore.agents["new-agent"]; !ok {
		t.Error("new agent was not created")
	}
}

func TestListAvailableCCTeams_Success(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	// Inject mock listing function
	handlers.listCCTeamDirs = func() ([]AvailableCCTeam, error) {
		return []AvailableCCTeam{
			{Name: "my-team", MemberCount: 3},
			{Name: "research-team", MemberCount: 1},
		}, nil
	}

	req := httptest.NewRequest("GET", "/teams/import/claude-code/available", nil)
	w := httptest.NewRecorder()

	handlers.ListAvailableCCTeams(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []AvailableCCTeam
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp) != 2 {
		t.Fatalf("expected 2 teams, got %d", len(resp))
	}
	if resp[0].Name != "my-team" || resp[0].MemberCount != 3 {
		t.Errorf("unexpected first team: %+v", resp[0])
	}
}

func TestListAvailableCCTeams_EmptyDir(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	handlers.listCCTeamDirs = func() ([]AvailableCCTeam, error) {
		return []AvailableCCTeam{}, nil
	}

	req := httptest.NewRequest("GET", "/teams/import/claude-code/available", nil)
	w := httptest.NewRecorder()

	handlers.ListAvailableCCTeams(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []AvailableCCTeam
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp) != 0 {
		t.Errorf("expected 0 teams, got %d", len(resp))
	}
}

func TestListAvailableCCTeams_DirNotExist(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	handlers.listCCTeamDirs = func() ([]AvailableCCTeam, error) {
		return nil, &os.PathError{Op: "open", Path: "/not/found", Err: os.ErrNotExist}
	}

	req := httptest.NewRequest("GET", "/teams/import/claude-code/available", nil)
	w := httptest.NewRecorder()

	handlers.ListAvailableCCTeams(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (empty list), got %d: %s", w.Code, w.Body.String())
	}

	var resp []AvailableCCTeam
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp) != 0 {
		t.Errorf("expected 0 teams, got %d", len(resp))
	}
}

// Suppress unused import warning by referencing store package
var _ = store.KindTeam
