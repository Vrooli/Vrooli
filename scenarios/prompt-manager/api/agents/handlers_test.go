package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	httpx "github.com/vrooli/api-core/servertest"
	"prompt-manager/store"
)

// MockAgentStore implements store.AgentStore for testing
type MockAgentStore struct {
	agents map[string]*store.Agent
}

func NewMockAgentStore() *MockAgentStore {
	return &MockAgentStore{
		agents: make(map[string]*store.Agent),
	}
}

func (m *MockAgentStore) List(ctx context.Context) ([]store.Agent, error) {
	result := make([]store.Agent, 0, len(m.agents))
	for _, a := range m.agents {
		result = append(result, *a)
	}
	return result, nil
}

func (m *MockAgentStore) Get(ctx context.Context, id string) (*store.Agent, error) {
	if a, ok := m.agents[id]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("agent not found: %s", id)
}

func (m *MockAgentStore) Create(ctx context.Context, agent *store.Agent) error {
	m.agents[agent.ID] = agent
	return nil
}

func (m *MockAgentStore) Update(ctx context.Context, id string, agent *store.Agent) error {
	if existing, ok := m.agents[id]; ok {
		if agent.DisplayName != "" {
			existing.DisplayName = agent.DisplayName
		}
		if agent.Appearance != nil {
			existing.Appearance = agent.Appearance
		}
		if agent.Status != "" {
			existing.Status = agent.Status
		}
	}
	return nil
}

func (m *MockAgentStore) Delete(ctx context.Context, id string) error {
	delete(m.agents, id)
	return nil
}

// MockIndexStore implements store.IndexStore for testing
type MockIndexStore struct{}

func (m *MockIndexStore) RegenerateAll(ctx context.Context) error    { return nil }
func (m *MockIndexStore) RegenerateSkills(ctx context.Context) error { return nil }
func (m *MockIndexStore) RegenerateAgents(ctx context.Context) error { return nil }
func (m *MockIndexStore) RegenerateTeams(ctx context.Context) error  { return nil }
func (m *MockIndexStore) GetSkillsIndex(ctx context.Context) (*store.SkillsIndex, error) {
	return nil, nil
}

func (m *MockIndexStore) GetAgentsIndex(ctx context.Context) (*store.AgentsIndex, error) {
	return nil, nil
}

func (m *MockIndexStore) GetTeamsIndex(ctx context.Context) (*store.TeamsIndex, error) {
	return nil, nil
}

func (m *MockIndexStore) RegenerateTopics(ctx context.Context) error { return nil }
func (m *MockIndexStore) GetTopicsIndex(ctx context.Context) (*store.TopicsIndex, error) {
	return nil, nil
}

// MockRelationStore implements store.RelationStore for testing
type MockRelationStore struct {
	relations []store.TeamMemberRelation
}

func (m *MockRelationStore) GetTeamMember(ctx context.Context, teamID, agentID string) (*store.TeamMemberRelation, error) {
	return nil, fmt.Errorf("not found")
}

func (m *MockRelationStore) SetTeamMember(ctx context.Context, rel *store.TeamMemberRelation) error {
	return nil
}

func (m *MockRelationStore) DeleteTeamMember(ctx context.Context, teamID, agentID string) error {
	return nil
}

func (m *MockRelationStore) ListTeamMembers(ctx context.Context, teamID string) ([]store.TeamMemberRelation, error) {
	return nil, nil
}

func (m *MockRelationStore) ListAgentTeams(ctx context.Context, agentID string) ([]store.TeamMemberRelation, error) {
	var result []store.TeamMemberRelation
	for _, rel := range m.relations {
		if rel.AgentID == agentID {
			result = append(result, rel)
		}
	}
	return result, nil
}

// MockTeamStore implements store.TeamStore for testing
type MockTeamStore struct {
	teams map[string]*store.Team
}

func NewMockTeamStore() *MockTeamStore {
	return &MockTeamStore{teams: make(map[string]*store.Team)}
}

func (m *MockTeamStore) List(ctx context.Context) ([]store.Team, error) { return nil, nil }
func (m *MockTeamStore) Get(ctx context.Context, id string) (*store.Team, error) {
	if t, ok := m.teams[id]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("team not found: %s", id)
}

func (m *MockTeamStore) Create(ctx context.Context, team *store.Team) error            { return nil }
func (m *MockTeamStore) Update(ctx context.Context, id string, team *store.Team) error { return nil }
func (m *MockTeamStore) Delete(ctx context.Context, id string) error                   { return nil }
func (m *MockTeamStore) GetRoles(ctx context.Context, teamID string) (*store.TeamRoles, error) {
	return nil, nil
}

func (m *MockTeamStore) SetRoles(ctx context.Context, teamID string, roles *store.TeamRoles) error {
	return nil
}

func (m *MockTeamStore) GetOrgChart(ctx context.Context, teamID string) (*store.OrgChart, error) {
	return nil, nil
}

func (m *MockTeamStore) SetOrgChart(ctx context.Context, teamID string, org *store.OrgChart) error {
	return nil
}

func (m *MockTeamStore) GetMembers(ctx context.Context, teamID string) ([]store.TeamMemberRelation, error) {
	return nil, nil
}

func (m *MockTeamStore) GetInbox(ctx context.Context, teamID, agentID string) (*store.TeamInbox, error) {
	return nil, nil
}

func (m *MockTeamStore) SetInbox(ctx context.Context, teamID, agentID string, inbox *store.TeamInbox) error {
	return nil
}

func TestList(t *testing.T) {
	agentStore := NewMockAgentStore()
	agentStore.agents["agent-1"] = &store.Agent{
		ID:          "agent-1",
		DisplayName: "Test Agent",
		Status:      store.AgentStatusActive,
	}

	handlers := NewHandlers(agentStore, &MockIndexStore{}, "", &MockRelationStore{}, NewMockTeamStore())

	req := httpx.Request(t, http.MethodGet, "/agents", nil, nil)
	w := httpx.Recorder()

	handlers.List(w, req)

	httpx.AssertStatus(t, w, http.StatusOK)

	var response []Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("Expected 1 agent, got %d", len(response))
	}

	if response[0].DisplayName != "Test Agent" {
		t.Errorf("Expected display name 'Test Agent', got '%s'", response[0].DisplayName)
	}
}

func TestCreate(t *testing.T) {
	agentStore := NewMockAgentStore()
	handlers := NewHandlers(agentStore, &MockIndexStore{}, "", &MockRelationStore{}, NewMockTeamStore())

	body := CreateRequest{
		DisplayName: "New Agent",
		Appearance: &AppearanceDTO{
			Body:   "#FF0000",
			Head:   "#00FF00",
			Accent: "#0000FF",
		},
	}
	req := httpx.JSONRequest(t, http.MethodPost, "/agents", body, nil)
	w := httpx.Recorder()

	handlers.Create(w, req)

	httpx.AssertStatus(t, w, http.StatusCreated)

	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.DisplayName != "New Agent" {
		t.Errorf("Expected display name 'New Agent', got '%s'", response.DisplayName)
	}

	if response.ID != "new-agent" {
		t.Errorf("Expected ID 'new-agent', got '%s'", response.ID)
	}
}

func TestCreateMissingDisplayName(t *testing.T) {
	agentStore := NewMockAgentStore()
	handlers := NewHandlers(agentStore, &MockIndexStore{}, "", &MockRelationStore{}, NewMockTeamStore())

	body := CreateRequest{}
	req := httpx.JSONRequest(t, http.MethodPost, "/agents", body, nil)
	w := httpx.Recorder()

	handlers.Create(w, req)

	httpx.AssertStatus(t, w, http.StatusBadRequest)
}

func TestCreateInvalidColor(t *testing.T) {
	agentStore := NewMockAgentStore()
	handlers := NewHandlers(agentStore, &MockIndexStore{}, "", &MockRelationStore{}, NewMockTeamStore())

	body := CreateRequest{
		DisplayName: "New Agent",
		Appearance: &AppearanceDTO{
			Body:   "not-a-color",
			Head:   "#00FF00",
			Accent: "#0000FF",
		},
	}
	req := httpx.JSONRequest(t, http.MethodPost, "/agents", body, nil)
	w := httpx.Recorder()

	handlers.Create(w, req)

	httpx.AssertStatus(t, w, http.StatusBadRequest)
}

func TestGet(t *testing.T) {
	agentStore := NewMockAgentStore()
	agentStore.agents["agent-1"] = &store.Agent{
		ID:          "agent-1",
		DisplayName: "Test Agent",
		Status:      store.AgentStatusActive,
	}

	handlers := NewHandlers(agentStore, &MockIndexStore{}, "", &MockRelationStore{}, NewMockTeamStore())

	req := httpx.Request(t, http.MethodGet, "/agents/agent-1", nil, map[string]string{"id": "agent-1"})
	w := httpx.Recorder()

	handlers.Get(w, req)

	httpx.AssertStatus(t, w, http.StatusOK)

	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.DisplayName != "Test Agent" {
		t.Errorf("Expected display name 'Test Agent', got '%s'", response.DisplayName)
	}
}

func TestGetNotFound(t *testing.T) {
	agentStore := NewMockAgentStore()
	handlers := NewHandlers(agentStore, &MockIndexStore{}, "", &MockRelationStore{}, NewMockTeamStore())

	req := httpx.Request(t, http.MethodGet, "/agents/nonexistent", nil, map[string]string{"id": "nonexistent"})
	w := httpx.Recorder()

	handlers.Get(w, req)

	httpx.AssertStatus(t, w, http.StatusNotFound)
}

func TestDelete(t *testing.T) {
	agentStore := NewMockAgentStore()
	agentStore.agents["agent-1"] = &store.Agent{
		ID:          "agent-1",
		DisplayName: "Test Agent",
		Status:      store.AgentStatusActive,
	}

	handlers := NewHandlers(agentStore, &MockIndexStore{}, "", &MockRelationStore{}, NewMockTeamStore())

	req := httpx.Request(t, http.MethodDelete, "/agents/agent-1", nil, map[string]string{"id": "agent-1"})
	w := httpx.Recorder()

	handlers.Delete(w, req)

	httpx.AssertStatus(t, w, http.StatusNoContent)

	if _, ok := agentStore.agents["agent-1"]; ok {
		t.Error("Expected agent to be deleted")
	}
}

func TestListTeams(t *testing.T) {
	agentStore := NewMockAgentStore()
	agentStore.agents["agent-1"] = &store.Agent{
		ID:          "agent-1",
		DisplayName: "Test Agent",
		Status:      store.AgentStatusActive,
	}

	relationStore := &MockRelationStore{
		relations: []store.TeamMemberRelation{
			{TeamID: "team-alpha", AgentID: "agent-1", Roles: []string{"developer"}, Status: "active"},
			{TeamID: "team-beta", AgentID: "agent-1", Roles: []string{"lead"}, Status: "active"},
		},
	}

	teamStore := NewMockTeamStore()
	teamStore.teams["team-alpha"] = &store.Team{ID: "team-alpha", DisplayName: "Alpha Team"}
	teamStore.teams["team-beta"] = &store.Team{ID: "team-beta", DisplayName: "Beta Team"}

	handlers := NewHandlers(agentStore, &MockIndexStore{}, "", relationStore, teamStore)

	req := httpx.Request(t, http.MethodGet, "/agents/agent-1/teams", nil, map[string]string{"id": "agent-1"})
	w := httpx.Recorder()

	handlers.ListTeams(w, req)

	httpx.AssertStatus(t, w, http.StatusOK)

	var response AgentTeamsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.AgentID != "agent-1" {
		t.Errorf("Expected agentId 'agent-1', got '%s'", response.AgentID)
	}

	if len(response.Memberships) != 2 {
		t.Fatalf("Expected 2 memberships, got %d", len(response.Memberships))
	}

	// Check that display names are enriched
	found := map[string]bool{}
	for _, m := range response.Memberships {
		found[m.TeamDisplayName] = true
	}
	if !found["Alpha Team"] || !found["Beta Team"] {
		t.Errorf("Expected enriched display names, got %v", response.Memberships)
	}
}

func TestListTeamsAgentNotFound(t *testing.T) {
	agentStore := NewMockAgentStore()
	handlers := NewHandlers(agentStore, &MockIndexStore{}, "", &MockRelationStore{}, NewMockTeamStore())

	req := httpx.Request(t, http.MethodGet, "/agents/nonexistent/teams", nil, map[string]string{"id": "nonexistent"})
	w := httpx.Recorder()

	handlers.ListTeams(w, req)

	httpx.AssertStatus(t, w, http.StatusNotFound)
}
