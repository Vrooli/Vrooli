package graph

import (
	"context"
	"errors"
	"prompt-manager/store"
)

// ---------------------------------------------------------------------------
// Shared test mocks for the graph package.
//
// All graph test files share a single package scope, so these types are
// available in every _test.go file without re-declaration.
// ---------------------------------------------------------------------------

// --- Scanner mocks (used by scanner_test.go) ---

type mockAgentLister struct {
	agents   []store.Agent
	listErr  error
	files    map[string][]store.AgentFileEntry // agentID → files
	contents map[string]string                 // "agentID/path" → content
	filesErr error
	readErr  error
}

func (m *mockAgentLister) List(_ context.Context) ([]store.Agent, error) {
	return m.agents, m.listErr
}

func (m *mockAgentLister) ListFiles(_ context.Context, agentID string) ([]store.AgentFileEntry, error) {
	if m.filesErr != nil {
		return nil, m.filesErr
	}
	return m.files[agentID], nil
}

func (m *mockAgentLister) ReadFile(_ context.Context, agentID, relPath string) (string, error) {
	if m.readErr != nil {
		return "", m.readErr
	}
	return m.contents[agentID+"/"+relPath], nil
}

type mockTeamLister struct {
	teams    []store.Team
	listErr  error
	files    map[string][]store.TeamFileEntry
	contents map[string]string // "teamID/path" → content
	filesErr error
	readErr  error
}

func (m *mockTeamLister) List(_ context.Context) ([]store.Team, error) {
	return m.teams, m.listErr
}

func (m *mockTeamLister) ListSharedFiles(_ context.Context, teamID string) ([]store.TeamFileEntry, error) {
	if m.filesErr != nil {
		return nil, m.filesErr
	}
	return m.files[teamID], nil
}

func (m *mockTeamLister) ReadSharedFile(_ context.Context, teamID, relPath string) (string, error) {
	if m.readErr != nil {
		return "", m.readErr
	}
	return m.contents[teamID+"/"+relPath], nil
}

type mockSkillLister struct {
	skills     []store.Skill
	listErr    error
	contentMap map[string]string // skillID → SKILL.md content
	contentErr error
}

func (m *mockSkillLister) List(_ context.Context) ([]store.Skill, error) {
	return m.skills, m.listErr
}

func (m *mockSkillLister) GetWithContent(_ context.Context, id string) (*store.Skill, string, error) {
	if m.contentErr != nil {
		return nil, "", m.contentErr
	}
	for i := range m.skills {
		if m.skills[i].ID == id {
			return &m.skills[i], m.contentMap[id], nil
		}
	}
	return nil, "", errors.New("skill not found")
}

type mockRelationStore struct {
	members    map[string][]store.TeamMemberRelation // teamID → members
	membersErr error
}

func (m *mockRelationStore) GetTeamMember(_ context.Context, _, _ string) (*store.TeamMemberRelation, error) {
	return nil, nil
}

func (m *mockRelationStore) SetTeamMember(_ context.Context, _ *store.TeamMemberRelation) error {
	return nil
}

func (m *mockRelationStore) DeleteTeamMember(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockRelationStore) ListTeamMembers(_ context.Context, teamID string) ([]store.TeamMemberRelation, error) {
	if m.membersErr != nil {
		return nil, m.membersErr
	}
	return m.members[teamID], nil
}

func (m *mockRelationStore) ListAgentTeams(_ context.Context, _ string) ([]store.TeamMemberRelation, error) {
	return nil, nil
}

// --- Builder mocks (used by builder_test.go) ---

type mockAgentNodeSource struct {
	agents []store.Agent
	err    error
}

func (m *mockAgentNodeSource) List(_ context.Context) ([]store.Agent, error) {
	return m.agents, m.err
}

type mockTeamNodeSource struct {
	teams []store.Team
	err   error
}

func (m *mockTeamNodeSource) List(_ context.Context) ([]store.Team, error) {
	return m.teams, m.err
}

type mockSkillNodeSource struct {
	skills []store.Skill
	err    error
}

func (m *mockSkillNodeSource) List(_ context.Context) ([]store.Skill, error) {
	return m.skills, m.err
}

type mockGraphScanner struct {
	edges []Edge
	err   error
}

func (m *mockGraphScanner) ScanAll(_ context.Context) ([]Edge, error) {
	return m.edges, m.err
}

// --- Handler mocks (used by handlers_test.go) ---

type mockGraphIndexProvider struct {
	idx      *GraphIndex
	getErr   error
	regenErr error
}

func (m *mockGraphIndexProvider) Get(_ context.Context) (*GraphIndex, error) {
	return m.idx, m.getErr
}

func (m *mockGraphIndexProvider) Regenerate(_ context.Context) error {
	return m.regenErr
}

type mockHealthConfigStore struct {
	cfg     HealthConfig
	getErr  error
	putErr  error
	lastPut *HealthConfig
}

func (m *mockHealthConfigStore) Get(_ context.Context) (HealthConfig, error) {
	return m.cfg, m.getErr
}

func (m *mockHealthConfigStore) Put(_ context.Context, cfg HealthConfig) error {
	if m.putErr != nil {
		return m.putErr
	}
	cloned := cfg
	m.lastPut = &cloned
	return nil
}

// testIndex creates a GraphIndex with predetermined data for handler tests.
func testIndex(nodes []Node, edges []Edge, scores []HealthScore) *GraphIndex {
	return &GraphIndex{
		GeneratedAt: "2025-01-01T00:00:00Z",
		Graph: Graph{
			Nodes:        nodes,
			Edges:        edges,
			HealthScores: scores,
		},
	}
}

// --- Index mocks (used by index_test.go) ---

type mockGraphBuilder struct {
	graph     Graph
	err       error
	callCount int
}

func (m *mockGraphBuilder) Build(_ context.Context) (Graph, error) {
	m.callCount++
	return m.graph, m.err
}

// --- Code detector mock (used by scanner_test.go) ---

// stubCodeDetector is a test double for the codeDetector interface.
// It returns a fixed set of CodeReferences for any input.
type stubCodeDetector struct {
	refs []CodeReference
}

func (s *stubCodeDetector) Detect(_ string) []CodeReference {
	return s.refs
}
