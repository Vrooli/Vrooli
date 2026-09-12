package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"

	"github.com/gorilla/mux"
)

// MockTeamStore implements store.TeamStore for testing.
// Also implements teamDocReader for export handler tests.
type MockTeamStore struct {
	teams                 map[string]*store.Team
	roles                 map[string]*store.TeamRoles
	orgChart              map[string]*store.OrgChart
	inboxes               map[string]map[string]*store.TeamInbox
	responsibilities      map[string]map[string]string // teamID -> agentID -> content
	heartbeatInstructions map[string]map[string]string // teamID -> agentID -> content
	err                   error                        // Inject errors for testing failure paths
}

func NewMockTeamStore() *MockTeamStore {
	return &MockTeamStore{
		teams:                 make(map[string]*store.Team),
		roles:                 make(map[string]*store.TeamRoles),
		orgChart:              make(map[string]*store.OrgChart),
		inboxes:               make(map[string]map[string]*store.TeamInbox),
		responsibilities:      make(map[string]map[string]string),
		heartbeatInstructions: make(map[string]map[string]string),
	}
}

// GetResponsibilities implements teamDocReader.
func (m *MockTeamStore) GetResponsibilities(_ context.Context, teamID, agentID string) (string, error) {
	if agents, ok := m.responsibilities[teamID]; ok {
		if content, ok := agents[agentID]; ok {
			return content, nil
		}
	}
	return "", fmt.Errorf("not found")
}

// GetHeartbeatInstructions implements teamDocReader.
func (m *MockTeamStore) GetHeartbeatInstructions(_ context.Context, teamID, agentID string) (string, error) {
	if agents, ok := m.heartbeatInstructions[teamID]; ok {
		if content, ok := agents[agentID]; ok {
			return content, nil
		}
	}
	return "", fmt.Errorf("not found")
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
	if updates.EnabledSet {
		existing.Enabled = updates.Enabled
	}
	if updates.Runtime.Mode != "" {
		existing.Runtime = updates.Runtime
	}
	if updates.Coordination.Pattern != "" {
		existing.Coordination = updates.Coordination
	}
	if updates.Execution.QueuePolicy != "" {
		existing.Execution = updates.Execution
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
	delete(m.inboxes, id)
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

func (m *MockTeamStore) GetInbox(ctx context.Context, teamID, agentID string) (*store.TeamInbox, error) {
	if m.err != nil {
		return nil, m.err
	}
	if _, ok := m.teams[teamID]; !ok {
		return nil, fmt.Errorf("team not found: %s", teamID)
	}
	if teamInbox, ok := m.inboxes[teamID]; ok {
		if inbox, ok := teamInbox[agentID]; ok {
			return inbox, nil
		}
	}
	return &store.TeamInbox{
		TeamID:   teamID,
		AgentID:  agentID,
		Messages: []store.TeamMessage{},
	}, nil
}

func (m *MockTeamStore) SetInbox(ctx context.Context, teamID, agentID string, inbox *store.TeamInbox) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.teams[teamID]; !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}
	if m.inboxes[teamID] == nil {
		m.inboxes[teamID] = make(map[string]*store.TeamInbox)
	}
	m.inboxes[teamID][agentID] = inbox
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

func (m *MockRelationStore) ListAgentTeams(ctx context.Context, agentID string) ([]store.TeamMemberRelation, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []store.TeamMemberRelation
	for _, team := range m.teamMembers {
		for _, rel := range team {
			if rel.AgentID == agentID {
				result = append(result, *rel)
			}
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

func (m *MockIndexStore) RegenerateTopics(ctx context.Context) error { return nil }
func (m *MockIndexStore) GetTopicsIndex(ctx context.Context) (*store.TopicsIndex, error) {
	return nil, nil
}

// Test helper to create handlers with mocks
func setupTestHandlers() (*Handlers, *MockTeamStore, *MockAgentStore, *MockRelationStore) {
	teamStore := NewMockTeamStore()
	agentStore := NewMockAgentStore()
	relationStore := NewMockRelationStore()
	indexStore := &MockIndexStore{}
	handlers := NewHandlers(teamStore, agentStore, relationStore, indexStore, nil)
	return handlers, teamStore, agentStore, relationStore
}

func independentCreateRequest(displayName string) CreateRequest {
	return CreateRequest{
		DisplayName:  displayName,
		Runtime:      teamconfig.Runtime{Mode: teamconfig.RuntimeModeMultiProcess},
		Coordination: newIndependentTestTeam("tmp", displayName).Coordination,
		Execution:    teamconfig.Execution{QueuePolicy: teamconfig.QueuePolicyBoundedParallel, MaxConcurrentRuns: 2},
	}
}

func leaderLedCreateRequest(displayName, leadAgentID string) CreateRequest {
	return CreateRequest{
		DisplayName: displayName,
		Runtime:     teamconfig.Runtime{Mode: teamconfig.RuntimeModeSingleProcess},
		Coordination: teamconfig.Coordination{
			Pattern:       teamconfig.CoordinationPatternLeaderLed,
			LeadAgentID:   leadAgentID,
			ReportingMode: teamconfig.ReportingModeLeader,
			MessagingMode: teamconfig.MessagingModeInSession,
			Capabilities: teamconfig.Capabilities{
				ShowOrgContext:           true,
				InjectInbox:              false,
				AllowPeerTriggers:        false,
				ShowTaskBoardGuidance:    true,
				ShowKnowledgeLogGuidance: true,
				RequireHandoff:           true,
			},
		},
		Execution: teamconfig.Execution{QueuePolicy: teamconfig.QueuePolicySerialized, MaxConcurrentRuns: 1},
	}
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
		DisplayName:  "New Team",
		Mission:      "Build great things",
		Runtime:      independentCreateRequest("New Team").Runtime,
		Coordination: independentCreateRequest("New Team").Coordination,
		Execution:    independentCreateRequest("New Team").Execution,
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

	body := independentCreateRequest("Custom ID Team")
	body.ID = "custom-team-id"
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

func TestCreateWithRuntimePolicy(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	body := leaderLedCreateRequest("SP Team", "lead-agent")
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/teams", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	handlers.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var response TeamDetailsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Runtime.Mode != teamconfig.RuntimeModeSingleProcess {
		t.Errorf("expected runtime.mode %q, got %q", teamconfig.RuntimeModeSingleProcess, response.Runtime.Mode)
	}
	if response.Coordination.Pattern != teamconfig.CoordinationPatternLeaderLed {
		t.Errorf("expected coordination.pattern %q, got %q", teamconfig.CoordinationPatternLeaderLed, response.Coordination.Pattern)
	}

	stored := teamStore.teams["sp-team"]
	if stored == nil {
		t.Fatal("team not stored")
	}
	if stored.Runtime.Mode != teamconfig.RuntimeModeSingleProcess {
		t.Errorf("stored runtime.mode = %q, want %q", stored.Runtime.Mode, teamconfig.RuntimeModeSingleProcess)
	}
}

func TestCreateWithInvalidRuntimeMode(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	body := independentCreateRequest("Bad Mode")
	body.Runtime.Mode = "invalid-mode"
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/teams", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	handlers.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
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

	teamStore.teams["existing-team"] = newIndependentTestTeam("existing-team", "Existing Team")

	body := independentCreateRequest("New Team")
	body.ID = "existing-team"
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

	team := newIndependentTestTeam("team-1", "Original Name")
	team.Mission = "Original mission"
	teamStore.teams["team-1"] = team
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

func TestUpdateMemberRejectsDeactivatingEnabledLeader(t *testing.T) {
	handlers, teamStore, agentStore, relationStore := setupTestHandlers()

	team := newLeaderLedSingleProcessTestTeam("team-1", "Test Team", "lead-agent")
	team.Enabled = true
	teamStore.teams["team-1"] = team

	agentStore.agents["lead-agent"] = &store.Agent{
		ID:          "lead-agent",
		DisplayName: "Lead Agent",
	}

	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"lead-agent": {
			TeamID:  "team-1",
			AgentID: "lead-agent",
			Status:  store.MemberStatusActive,
		},
	}

	newStatus := store.MemberStatusInactive
	body := UpdateMemberRequest{Status: &newStatus}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/teams/team-1/members/lead-agent", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "lead-agent"})
	w := httptest.NewRecorder()

	handlers.UpdateMember(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	if got := relationStore.teamMembers["team-1"]["lead-agent"].Status; got != store.MemberStatusActive {
		t.Fatalf("lead status = %q, want %q", got, store.MemberStatusActive)
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

func TestRemoveMemberRejectsEnabledLeaderLead(t *testing.T) {
	handlers, teamStore, _, relationStore := setupTestHandlers()

	team := newLeaderLedSingleProcessTestTeam("team-1", "Test Team", "lead-agent")
	team.Enabled = true
	teamStore.teams["team-1"] = team

	relationStore.teamMembers["team-1"] = map[string]*store.TeamMemberRelation{
		"lead-agent": {
			TeamID:  "team-1",
			AgentID: "lead-agent",
			Status:  store.MemberStatusActive,
		},
	}

	req := httptest.NewRequest("DELETE", "/teams/team-1/members/lead-agent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "lead-agent"})
	w := httptest.NewRecorder()

	handlers.RemoveMember(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	if _, ok := relationStore.teamMembers["team-1"]["lead-agent"]; !ok {
		t.Fatal("expected lead membership to remain intact")
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

func TestSetRoles(t *testing.T) {
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

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response []RoleDTO
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(response) != 1 || response[0].ID != "dev" {
		t.Errorf("Expected updated roles to include dev, got %+v", response)
	}
}

// ============== OrgChart Tests ==============

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

func TestSetOrgChart(t *testing.T) {
	handlers, teamStore, _, relationStore := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "manager-1"})
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "dev-1"})

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

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetOrgChartRejectsNonMember(t *testing.T) {
	handlers, teamStore, _, relationStore := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "dev-1"})

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

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateOrgChartEdge(t *testing.T) {
	handlers, teamStore, _, relationStore := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}
	teamStore.orgChart["team-1"] = &store.OrgChart{
		TeamID: "team-1",
		Edges: []store.OrgEdge{
			{ManagerAgentID: "manager-1", ReportAgentID: "dev-1"},
		},
	}
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "manager-1"})
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "manager-2"})
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "dev-1"})

	body := UpdateOrgEdgeRequest{ManagerAgentID: "manager-2"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/teams/team-1/org/edges/dev-1", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "reportId": "dev-1"})
	w := httptest.NewRecorder()

	handlers.UpdateOrgChartEdge(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	org, _ := teamStore.GetOrgChart(context.Background(), "team-1")
	if len(org.Edges) != 1 || org.Edges[0].ManagerAgentID != "manager-2" {
		t.Errorf("Expected updated edge to manager-2, got %+v", org.Edges)
	}
}

func TestDeleteOrgChartEdge(t *testing.T) {
	handlers, teamStore, _, relationStore := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}
	teamStore.orgChart["team-1"] = &store.OrgChart{
		TeamID: "team-1",
		Edges: []store.OrgEdge{
			{ManagerAgentID: "manager-1", ReportAgentID: "dev-1"},
		},
	}
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "manager-1"})
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "dev-1"})

	req := httptest.NewRequest("DELETE", "/teams/team-1/org/edges/dev-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "reportId": "dev-1"})
	w := httptest.NewRecorder()

	handlers.DeleteOrgChartEdge(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d: %s", w.Code, w.Body.String())
	}

	org, _ := teamStore.GetOrgChart(context.Background(), "team-1")
	if len(org.Edges) != 0 {
		t.Errorf("Expected org chart to be empty, got %+v", org.Edges)
	}
}

// ============== Team Message Tests ==============

func TestSendTeamMessage(t *testing.T) {
	handlers, teamStore, _, relationStore := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "sender-1"})
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "recipient-1"})

	body := SendTeamMessageRequest{
		FromAgentID: "sender-1",
		Content:     "Status update",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/teams/team-1/members/recipient-1/messages", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "recipient-1"})
	w := httptest.NewRecorder()

	handlers.SendTeamMessage(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var message TeamMessageDTO
	if err := json.NewDecoder(w.Body).Decode(&message); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if message.FromAgentID != "sender-1" || message.ToAgentID != "recipient-1" {
		t.Errorf("Unexpected message payload: %+v", message)
	}
}

// ============== GetExclusiveMembers Tests ==============

func TestGetExclusiveMembers(t *testing.T) {
	handlers, teamStore, agentStore, relationStore := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}
	teamStore.teams["team-2"] = &store.Team{
		ID:          "team-2",
		DisplayName: "Other Team",
	}

	agentStore.agents["agent-1"] = &store.Agent{ID: "agent-1", DisplayName: "Exclusive Agent"}
	agentStore.agents["agent-2"] = &store.Agent{ID: "agent-2", DisplayName: "Shared Agent"}
	agentStore.agents["agent-3"] = &store.Agent{ID: "agent-3", DisplayName: "Another Exclusive"}

	// agent-1 only in team-1
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "agent-1"})
	// agent-2 in both teams
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "agent-2"})
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-2", AgentID: "agent-2"})
	// agent-3 only in team-1
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "agent-3"})

	req := httptest.NewRequest("GET", "/teams/team-1/exclusive-members", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetExclusiveMembers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response ExclusiveMembersResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.TeamID != "team-1" {
		t.Errorf("Expected teamId 'team-1', got '%s'", response.TeamID)
	}

	if len(response.Members) != 2 {
		t.Fatalf("Expected 2 exclusive members, got %d", len(response.Members))
	}

	// Verify exclusive members (order may vary)
	exclusiveIDs := make(map[string]string) // agentID -> displayName
	for _, m := range response.Members {
		exclusiveIDs[m.AgentID] = m.DisplayName
	}

	if name, ok := exclusiveIDs["agent-1"]; !ok || name != "Exclusive Agent" {
		t.Errorf("Expected agent-1 with name 'Exclusive Agent', got ok=%v name='%s'", ok, name)
	}
	if name, ok := exclusiveIDs["agent-3"]; !ok || name != "Another Exclusive" {
		t.Errorf("Expected agent-3 with name 'Another Exclusive', got ok=%v name='%s'", ok, name)
	}
	if _, ok := exclusiveIDs["agent-2"]; ok {
		t.Error("agent-2 should NOT be exclusive (belongs to 2 teams)")
	}
}

func TestGetExclusiveMembersTeamNotFound(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	req := httptest.NewRequest("GET", "/teams/nonexistent/exclusive-members", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.GetExclusiveMembers(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestGetExclusiveMembersNoMembers(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Empty Team",
	}

	req := httptest.NewRequest("GET", "/teams/team-1/exclusive-members", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetExclusiveMembers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response ExclusiveMembersResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Members) != 0 {
		t.Errorf("Expected 0 exclusive members, got %d", len(response.Members))
	}
}

func TestListTeamMessages(t *testing.T) {
	handlers, teamStore, _, relationStore := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
	}
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "sender-1"})
	_ = relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "recipient-1"})

	inbox := &store.TeamInbox{
		TeamID:  "team-1",
		AgentID: "recipient-1",
		Messages: []store.TeamMessage{
			{
				ID:          "msg-1",
				TeamID:      "team-1",
				FromAgentID: "sender-1",
				ToAgentID:   "recipient-1",
				Content:     "Hello",
				CreatedAt:   "2026-02-01T00:00:00Z",
			},
		},
	}
	_ = teamStore.SetInbox(context.Background(), "team-1", "recipient-1", inbox)

	req := httptest.NewRequest("GET", "/teams/team-1/members/recipient-1/messages", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "recipient-1"})
	w := httptest.NewRecorder()

	handlers.ListTeamMessages(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response TeamInboxResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(response.Messages) != 1 || response.Messages[0].ID != "msg-1" {
		t.Errorf("Expected 1 message, got %+v", response.Messages)
	}
}
