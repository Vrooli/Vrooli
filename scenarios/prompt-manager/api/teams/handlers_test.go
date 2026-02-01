package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/store"

	"github.com/gorilla/mux"
)

// MockTeamStore implements store.TeamStore for testing
type MockTeamStore struct {
	teams    map[string]*store.Team
	roles    map[string]*store.TeamRoles
	orgChart map[string]*store.OrgChart
	err      error // Inject errors for testing failure paths
}

func NewMockTeamStore() *MockTeamStore {
	return &MockTeamStore{
		teams:    make(map[string]*store.Team),
		roles:    make(map[string]*store.TeamRoles),
		orgChart: make(map[string]*store.OrgChart),
	}
}

func (m *MockTeamStore) List(ctx context.Context) ([]store.Team, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([]store.Team, 0, len(m.teams))
	for _, t := range m.teams {
		result = append(result, *t)
	}
	return result, nil
}

func (m *MockTeamStore) Get(ctx context.Context, id string) (*store.Team, error) {
	if m.err != nil {
		return nil, m.err
	}
	if t, ok := m.teams[id]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("team not found: %s", id)
}

func (m *MockTeamStore) Create(ctx context.Context, team *store.Team) error {
	if m.err != nil {
		return m.err
	}
	if _, exists := m.teams[team.ID]; exists {
		return fmt.Errorf("team already exists: %s", team.ID)
	}
	m.teams[team.ID] = team
	// Initialize default roles and org chart
	m.roles[team.ID] = &store.TeamRoles{TeamID: team.ID, Roles: []store.Role{}}
	m.orgChart[team.ID] = &store.OrgChart{TeamID: team.ID, Edges: []store.OrgEdge{}}
	return nil
}

func (m *MockTeamStore) Update(ctx context.Context, id string, updates *store.Team) error {
	if m.err != nil {
		return m.err
	}
	existing, ok := m.teams[id]
	if !ok {
		return fmt.Errorf("team not found: %s", id)
	}
	if updates.DisplayName != "" {
		existing.DisplayName = updates.DisplayName
	}
	if updates.Mission != "" {
		existing.Mission = updates.Mission
	}
	if updates.Defaults != nil {
		existing.Defaults = updates.Defaults
	}
	return nil
}

func (m *MockTeamStore) Delete(ctx context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.teams[id]; !ok {
		return fmt.Errorf("team not found: %s", id)
	}
	delete(m.teams, id)
	delete(m.roles, id)
	delete(m.orgChart, id)
	return nil
}

func (m *MockTeamStore) GetRoles(ctx context.Context, teamID string) (*store.TeamRoles, error) {
	if m.err != nil {
		return nil, m.err
	}
	if roles, ok := m.roles[teamID]; ok {
		return roles, nil
	}
	return nil, fmt.Errorf("team not found: %s", teamID)
}

func (m *MockTeamStore) GetOrgChart(ctx context.Context, teamID string) (*store.OrgChart, error) {
	if m.err != nil {
		return nil, m.err
	}
	if org, ok := m.orgChart[teamID]; ok {
		return org, nil
	}
	return nil, fmt.Errorf("team not found: %s", teamID)
}

func (m *MockTeamStore) GetMembers(ctx context.Context, teamID string) ([]store.TeamMemberRelation, error) {
	return nil, nil
}

// SetRoles sets team roles (mock implementation)
func (m *MockTeamStore) SetRoles(ctx context.Context, teamID string, roles *store.TeamRoles) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.teams[teamID]; !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}
	m.roles[teamID] = roles
	return nil
}

// SetOrgChart sets org chart (mock implementation)
func (m *MockTeamStore) SetOrgChart(ctx context.Context, teamID string, org *store.OrgChart) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.teams[teamID]; !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}
	m.orgChart[teamID] = org
	return nil
}

// MockAgentStore implements store.AgentStore for testing
type MockAgentStore struct {
	agents map[string]*store.Agent
	err    error
}

func NewMockAgentStore() *MockAgentStore {
	return &MockAgentStore{
		agents: make(map[string]*store.Agent),
	}
}

func (m *MockAgentStore) List(ctx context.Context) ([]store.Agent, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([]store.Agent, 0, len(m.agents))
	for _, a := range m.agents {
		result = append(result, *a)
	}
	return result, nil
}

func (m *MockAgentStore) Get(ctx context.Context, id string) (*store.Agent, error) {
	if m.err != nil {
		return nil, m.err
	}
	if a, ok := m.agents[id]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("agent not found: %s", id)
}

func (m *MockAgentStore) Create(ctx context.Context, agent *store.Agent) error {
	if m.err != nil {
		return m.err
	}
	m.agents[agent.ID] = agent
	return nil
}

func (m *MockAgentStore) Update(ctx context.Context, id string, agent *store.Agent) error {
	if m.err != nil {
		return m.err
	}
	if existing, ok := m.agents[id]; ok {
		if agent.DisplayName != "" {
			existing.DisplayName = agent.DisplayName
		}
	}
	return nil
}

func (m *MockAgentStore) Delete(ctx context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.agents, id)
	return nil
}

func (m *MockAgentStore) GetSkills(ctx context.Context, agentID string) ([]store.AgentSkillRelation, error) {
	return nil, nil
}

func (m *MockAgentStore) GetEffectiveSkills(ctx context.Context, agentID string, teamID *string) ([]string, error) {
	return []string{}, nil
}

// MockRelationStore implements store.RelationStore for testing
type MockRelationStore struct {
	teamMembers map[string]map[string]*store.TeamMemberRelation // teamID -> agentID -> relation
	err         error
}

func NewMockRelationStore() *MockRelationStore {
	return &MockRelationStore{
		teamMembers: make(map[string]map[string]*store.TeamMemberRelation),
	}
}

func (m *MockRelationStore) GetAgentSkill(ctx context.Context, agentID, skillID string) (*store.AgentSkillRelation, error) {
	return nil, nil
}

func (m *MockRelationStore) SetAgentSkill(ctx context.Context, rel *store.AgentSkillRelation) error {
	return nil
}

func (m *MockRelationStore) DeleteAgentSkill(ctx context.Context, agentID, skillID string) error {
	return nil
}

func (m *MockRelationStore) ListAgentSkills(ctx context.Context, agentID string) ([]store.AgentSkillRelation, error) {
	return nil, nil
}

func (m *MockRelationStore) GetTeamMember(ctx context.Context, teamID, agentID string) (*store.TeamMemberRelation, error) {
	if m.err != nil {
		return nil, m.err
	}
	if team, ok := m.teamMembers[teamID]; ok {
		if rel, ok := team[agentID]; ok {
			return rel, nil
		}
	}
	return nil, fmt.Errorf("membership not found: team=%s agent=%s", teamID, agentID)
}

func (m *MockRelationStore) SetTeamMember(ctx context.Context, rel *store.TeamMemberRelation) error {
	if m.err != nil {
		return m.err
	}
	if m.teamMembers[rel.TeamID] == nil {
		m.teamMembers[rel.TeamID] = make(map[string]*store.TeamMemberRelation)
	}
	m.teamMembers[rel.TeamID][rel.AgentID] = rel
	return nil
}

func (m *MockRelationStore) DeleteTeamMember(ctx context.Context, teamID, agentID string) error {
	if m.err != nil {
		return m.err
	}
	if team, ok := m.teamMembers[teamID]; ok {
		if _, ok := team[agentID]; ok {
			delete(team, agentID)
			return nil
		}
	}
	return fmt.Errorf("membership not found: team=%s agent=%s", teamID, agentID)
}

func (m *MockRelationStore) ListTeamMembers(ctx context.Context, teamID string) ([]store.TeamMemberRelation, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []store.TeamMemberRelation
	if team, ok := m.teamMembers[teamID]; ok {
		for _, rel := range team {
			result = append(result, *rel)
		}
	}
	return result, nil
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

// Test helper to create handlers with mocks
func setupTestHandlers() (*Handlers, *MockTeamStore, *MockAgentStore, *MockRelationStore) {
	teamStore := NewMockTeamStore()
	agentStore := NewMockAgentStore()
	relationStore := NewMockRelationStore()
	indexStore := &MockIndexStore{}
	handlers := NewHandlers(teamStore, agentStore, relationStore, indexStore)
	return handlers, teamStore, agentStore, relationStore
}

// ============== List Tests ==============

func TestList(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
		Mission:     "Test mission",
	}

	req := httptest.NewRequest("GET", "/teams", nil)
	w := httptest.NewRecorder()

	handlers.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response []Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("Expected 1 team, got %d", len(response))
	}

	if response[0].DisplayName != "Test Team" {
		t.Errorf("Expected display name 'Test Team', got '%s'", response[0].DisplayName)
	}
}

func TestListEmpty(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	req := httptest.NewRequest("GET", "/teams", nil)
	w := httptest.NewRecorder()

	handlers.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response []Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response) != 0 {
		t.Errorf("Expected 0 teams, got %d", len(response))
	}
}

// ============== Get Tests ==============

func TestGet(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
		Mission:     "Test mission",
	}
	teamStore.roles["team-1"] = &store.TeamRoles{TeamID: "team-1", Roles: []store.Role{}}
	teamStore.orgChart["team-1"] = &store.OrgChart{TeamID: "team-1", Edges: []store.OrgEdge{}}

	req := httptest.NewRequest("GET", "/teams/team-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.Get(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response TeamDetailsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.DisplayName != "Test Team" {
		t.Errorf("Expected display name 'Test Team', got '%s'", response.DisplayName)
	}
}

func TestGetNotFound(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	req := httptest.NewRequest("GET", "/teams/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// ============== Create Tests ==============

func TestCreate(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	body := CreateRequest{
		DisplayName: "New Team",
		Mission:     "Build great things",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/teams", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	handlers.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var response TeamDetailsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.DisplayName != "New Team" {
		t.Errorf("Expected display name 'New Team', got '%s'", response.DisplayName)
	}

	if response.ID != "new-team" {
		t.Errorf("Expected ID 'new-team', got '%s'", response.ID)
	}

	// Verify stored
	if _, ok := teamStore.teams["new-team"]; !ok {
		t.Error("Team was not stored")
	}
}

func TestCreateWithCustomID(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	body := CreateRequest{
		ID:          "custom-team-id",
		DisplayName: "Custom ID Team",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/teams", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	handlers.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response TeamDetailsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.ID != "custom-team-id" {
		t.Errorf("Expected ID 'custom-team-id', got '%s'", response.ID)
	}

	if _, ok := teamStore.teams["custom-team-id"]; !ok {
		t.Error("Team was not stored with custom ID")
	}
}

func TestCreateMissingDisplayName(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	body := CreateRequest{}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/teams", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	handlers.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCreateConflict(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	teamStore.teams["existing-team"] = &store.Team{
		ID:          "existing-team",
		DisplayName: "Existing Team",
	}

	body := CreateRequest{
		ID:          "existing-team",
		DisplayName: "New Team",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/teams", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	handlers.Create(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

// ============== Update Tests ==============

func TestUpdate(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Original Name",
		Mission:     "Original mission",
	}
	teamStore.roles["team-1"] = &store.TeamRoles{TeamID: "team-1", Roles: []store.Role{}}
	teamStore.orgChart["team-1"] = &store.OrgChart{TeamID: "team-1", Edges: []store.OrgEdge{}}

	newName := "Updated Name"
	body := UpdateRequest{
		DisplayName: &newName,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/teams/team-1", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response TeamDetailsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.DisplayName != "Updated Name" {
		t.Errorf("Expected display name 'Updated Name', got '%s'", response.DisplayName)
	}
}

func TestUpdateNotFound(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	newName := "Updated Name"
	body := UpdateRequest{
		DisplayName: &newName,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/teams/nonexistent", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.Update(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// ============== Delete Tests ==============

func TestDelete(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}

	req := httptest.NewRequest("DELETE", "/teams/team-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	if _, ok := teamStore.teams["team-1"]; ok {
		t.Error("Expected team to be deleted")
	}
}

func TestDeleteNotFound(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	req := httptest.NewRequest("DELETE", "/teams/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.Delete(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// ============== AddMember Tests ==============

func TestAddMember(t *testing.T) {
	handlers, teamStore, agentStore, _ := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}
	teamStore.roles["team-1"] = &store.TeamRoles{TeamID: "team-1", Roles: []store.Role{}}

	agentStore.agents["agent-1"] = &store.Agent{
		ID:          "agent-1",
		DisplayName: "Test Agent",
		Status:      store.AgentStatusActive,
	}

	body := AddMemberRequest{
		AgentID: "agent-1",
		Roles:   []string{"developer"},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/teams/team-1/members", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.AddMember(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var response MemberDTO
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.AgentID != "agent-1" {
		t.Errorf("Expected agent ID 'agent-1', got '%s'", response.AgentID)
	}

	if response.Status != store.MemberStatusActive {
		t.Errorf("Expected status 'active', got '%s'", response.Status)
	}
}

func TestAddMemberMissingAgentID(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}

	body := AddMemberRequest{
		Roles: []string{"developer"},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/teams/team-1/members", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.AddMember(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAddMemberTeamNotFound(t *testing.T) {
	handlers, _, agentStore, _ := setupTestHandlers()

	agentStore.agents["agent-1"] = &store.Agent{
		ID:          "agent-1",
		DisplayName: "Test Agent",
	}

	body := AddMemberRequest{
		AgentID: "agent-1",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/teams/nonexistent/members", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.AddMember(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestAddMemberAgentNotFound(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}

	body := AddMemberRequest{
		AgentID: "nonexistent-agent",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/teams/team-1/members", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.AddMember(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// ============== UpdateMember Tests ==============

func TestUpdateMember(t *testing.T) {
	handlers, teamStore, agentStore, relationStore := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}

	agentStore.agents["agent-1"] = &store.Agent{
		ID:          "agent-1",
		DisplayName: "Test Agent",
	}

	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"developer"},
			Status:  store.MemberStatusActive,
		},
	}

	newStatus := store.MemberStatusInactive
	body := UpdateMemberRequest{
		Roles:  []string{"lead", "developer"},
		Status: &newStatus,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/teams/team-1/members/agent-1", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.UpdateMember(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response MemberDTO
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(response.Roles))
	}

	if response.Status != store.MemberStatusInactive {
		t.Errorf("Expected status 'inactive', got '%s'", response.Status)
	}
}

func TestUpdateMemberNotFound(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}

	body := UpdateMemberRequest{
		Roles: []string{"lead"},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/teams/team-1/members/nonexistent", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.UpdateMember(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// ============== RemoveMember Tests ==============

func TestRemoveMember(t *testing.T) {
	handlers, teamStore, _, relationStore := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}

	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"agent-1": {
			TeamID:  "team-1",
			AgentID: "agent-1",
			Roles:   []string{"developer"},
			Status:  store.MemberStatusActive,
		},
	}

	req := httptest.NewRequest("DELETE", "/teams/team-1/members/agent-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.RemoveMember(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	if _, ok := relationStore.teamMembers["team-1"]["agent-1"]; ok {
		t.Error("Expected member to be removed")
	}
}

func TestRemoveMemberNotFound(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}

	req := httptest.NewRequest("DELETE", "/teams/team-1/members/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.RemoveMember(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// ============== GetRoles Tests ==============

func TestGetRoles(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}
	teamStore.roles["team-1"] = &store.TeamRoles{
		TeamID: "team-1",
		Roles: []store.Role{
			{ID: "dev", Name: "Developer", Description: "Writes code"},
			{ID: "lead", Name: "Lead", Description: "Leads team"},
		},
	}

	req := httptest.NewRequest("GET", "/teams/team-1/roles", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetRoles(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response []RoleDTO
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(response))
	}
}

func TestGetRolesNotFound(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	req := httptest.NewRequest("GET", "/teams/nonexistent/roles", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.GetRoles(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// ============== SetRoles Tests ==============
// Note: SetRoles requires a type assertion to FileTeamStore, which mock doesn't satisfy.
// These tests verify the handler returns appropriate error for non-FileTeamStore.

func TestSetRolesNotSupported(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}

	body := SetRolesRequest{
		Roles: []RoleDTO{
			{ID: "dev", Name: "Developer"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/teams/team-1/roles", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.SetRoles(w, req)

	// Should return 500 because mock store doesn't implement FileTeamStore type
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 (SetRoles not supported with mock), got %d", w.Code)
	}
}

// ============== OrgChart Tests ==============
// Note: GetOrgChart and SetOrgChart handlers will be added in Phase 2.
// These tests are placeholder for the OrgChart API tests.

func TestGetOrgChart(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}
	teamStore.orgChart["team-1"] = &store.OrgChart{
		TeamID: "team-1",
		Edges: []store.OrgEdge{
			{ManagerAgentID: "manager-1", ReportAgentID: "dev-1"},
			{ManagerAgentID: "manager-1", ReportAgentID: "dev-2"},
		},
	}

	req := httptest.NewRequest("GET", "/teams/team-1/org", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetOrgChart(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response OrgChartResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.TeamID != "team-1" {
		t.Errorf("Expected team ID 'team-1', got '%s'", response.TeamID)
	}

	if len(response.Edges) != 2 {
		t.Errorf("Expected 2 edges, got %d", len(response.Edges))
	}
}

func TestGetOrgChartNotFound(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	req := httptest.NewRequest("GET", "/teams/nonexistent/org", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.GetOrgChart(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestSetOrgChartNotSupported(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}

	body := SetOrgChartRequest{
		Edges: []OrgEdgeDTO{
			{ManagerAgentID: "manager-1", ReportAgentID: "dev-1"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/teams/team-1/org", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.SetOrgChart(w, req)

	// Should return 500 because mock store doesn't implement FileTeamStore type
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 (SetOrgChart not supported with mock), got %d", w.Code)
	}
}
