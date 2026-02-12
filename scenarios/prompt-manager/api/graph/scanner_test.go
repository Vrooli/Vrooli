package graph

import (
	"context"
	"errors"
	"testing"

	"prompt-manager/store"
)

// ---------------------------------------------------------------------------
// Mocks for Scanner interface seams
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// ScanAll tests
// ---------------------------------------------------------------------------

func TestScanAll_Empty(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{},
		&mockTeamLister{},
		&mockSkillLister{},
		nil, nil,
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 0 {
		t.Fatalf("expected 0 edges, got %d", len(edges))
	}
}

func TestScanAll_AgentSkillRef(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{
			agents: []store.Agent{{ID: "agent-1"}},
			files: map[string][]store.AgentFileEntry{
				"agent-1": {{Path: "SOUL.md"}},
			},
			contents: map[string]string{
				"agent-1/SOUL.md": "Read: prompt-manager skill read skill-a",
			},
		},
		&mockTeamLister{},
		&mockSkillLister{
			skills: []store.Skill{{ID: "skill-a"}},
		},
		nil, nil,
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].From != "agent-1" || edges[0].To != "skill-a" || edges[0].Kind != EdgeCLIRead {
		t.Errorf("unexpected edge: %+v", edges[0])
	}
	if edges[0].SourceFile != "SOUL.md" {
		t.Errorf("expected SourceFile SOUL.md, got %s", edges[0].SourceFile)
	}
}

func TestScanAll_AgentBoldRef(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{
			agents: []store.Agent{{ID: "agent-1"}},
			files: map[string][]store.AgentFileEntry{
				"agent-1": {{Path: "notes.md"}},
			},
			contents: map[string]string{
				"agent-1/notes.md": "Use **skill-b** for testing",
			},
		},
		&mockTeamLister{},
		&mockSkillLister{
			skills: []store.Skill{{ID: "skill-b"}},
		},
		nil, nil,
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].Kind != EdgeBoldListed {
		t.Errorf("expected bold-listed, got %s", edges[0].Kind)
	}
}

func TestScanAll_AgentCodeUsage(t *testing.T) {
	det := NewCLIDetector([]string{"prompt-manager"})
	s := NewScanner(
		&mockAgentLister{
			agents: []store.Agent{{ID: "agent-1"}},
			files: map[string][]store.AgentFileEntry{
				"agent-1": {{Path: "SOUL.md"}},
			},
			contents: map[string]string{
				"agent-1/SOUL.md": "Run `vrooli scenario start foo`",
			},
		},
		&mockTeamLister{},
		&mockSkillLister{},
		nil, det,
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].Kind != EdgeCodeUsage {
		t.Errorf("expected code-usage, got %s", edges[0].Kind)
	}
	if edges[0].To != "cli:vrooli" {
		t.Errorf("expected cli:vrooli, got %s", edges[0].To)
	}
}

func TestScanAll_AgentNonMdSkipped(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{
			agents: []store.Agent{{ID: "agent-1"}},
			files: map[string][]store.AgentFileEntry{
				"agent-1": {
					{Path: "config.json"},
					{Path: "data", IsDir: true},
				},
			},
			contents: map[string]string{
				"agent-1/config.json": "prompt-manager skill read skill-a",
			},
		},
		&mockTeamLister{},
		&mockSkillLister{
			skills: []store.Skill{{ID: "skill-a"}},
		},
		nil, nil,
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 0 {
		t.Fatalf("expected 0 edges (non-md skipped), got %d", len(edges))
	}
}

func TestScanAll_TeamSkillRef(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{},
		&mockTeamLister{
			teams: []store.Team{{ID: "team-1"}},
			files: map[string][]store.TeamFileEntry{
				"team-1": {{Path: "shared.md"}},
			},
			contents: map[string]string{
				"team-1/shared.md": "prompt-manager skill read skill-a",
			},
		},
		&mockSkillLister{
			skills: []store.Skill{{ID: "skill-a"}},
		},
		nil, nil,
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].From != "team-1" || edges[0].To != "skill-a" {
		t.Errorf("unexpected edge: %+v", edges[0])
	}
}

func TestScanAll_SkillCrossRef(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{},
		&mockTeamLister{},
		&mockSkillLister{
			skills: []store.Skill{
				{ID: "skill-a"},
				{ID: "skill-b"},
			},
			contentMap: map[string]string{
				"skill-a": "See **skill-b** for details",
			},
		},
		nil, nil,
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].From != "skill-a" || edges[0].To != "skill-b" {
		t.Errorf("unexpected edge: %+v", edges[0])
	}
	if edges[0].SourceFile != "SKILL.md" {
		t.Errorf("expected SourceFile SKILL.md, got %s", edges[0].SourceFile)
	}
}

func TestScanAll_SkillSelfRefIgnored(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{},
		&mockTeamLister{},
		&mockSkillLister{
			skills: []store.Skill{{ID: "skill-a"}},
			contentMap: map[string]string{
				"skill-a": "See **skill-a** for self",
			},
		},
		nil, nil,
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 0 {
		t.Fatalf("expected 0 edges (self-ref ignored), got %d", len(edges))
	}
}

func TestScanAll_SkillDefaultScope(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{},
		&mockTeamLister{},
		&mockSkillLister{
			skills: []store.Skill{
				{ID: "skill-a", DefaultScope: "skill-b"},
				{ID: "skill-b"},
			},
		},
		nil, nil,
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].Kind != EdgeDefaultScope {
		t.Errorf("expected default-scope, got %s", edges[0].Kind)
	}
	if edges[0].From != "skill-a" || edges[0].To != "skill-b" {
		t.Errorf("unexpected edge: %+v", edges[0])
	}
}

func TestScanAll_Membership(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{},
		&mockTeamLister{
			teams: []store.Team{{ID: "team-1"}},
		},
		&mockSkillLister{},
		&mockRelationStore{
			members: map[string][]store.TeamMemberRelation{
				"team-1": {{TeamID: "team-1", AgentID: "agent-1"}},
			},
		},
		nil,
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].Kind != EdgeMembership {
		t.Errorf("expected membership, got %s", edges[0].Kind)
	}
	if edges[0].From != "team-1" || edges[0].To != "agent-1" {
		t.Errorf("unexpected edge: %+v", edges[0])
	}
}

func TestScanAll_NilRelationStore(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{},
		&mockTeamLister{
			teams: []store.Team{{ID: "team-1"}},
		},
		&mockSkillLister{},
		nil, nil,
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 0 {
		t.Fatalf("expected 0 edges, got %d", len(edges))
	}
}

func TestScanAll_AgentListError(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{listErr: errors.New("agent list fail")},
		&mockTeamLister{},
		&mockSkillLister{},
		nil, nil,
	)
	_, err := s.ScanAll(context.Background())
	if err == nil || err.Error() != "agent list fail" {
		t.Fatalf("expected agent list error, got: %v", err)
	}
}

func TestScanAll_TeamListError(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{},
		&mockTeamLister{listErr: errors.New("team list fail")},
		&mockSkillLister{},
		nil, nil,
	)
	_, err := s.ScanAll(context.Background())
	if err == nil || err.Error() != "team list fail" {
		t.Fatalf("expected team list error, got: %v", err)
	}
}

func TestScanAll_SkillListError(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{},
		&mockTeamLister{},
		&mockSkillLister{listErr: errors.New("skill list fail")},
		nil, nil,
	)
	_, err := s.ScanAll(context.Background())
	if err == nil || err.Error() != "skill list fail" {
		t.Fatalf("expected skill list error, got: %v", err)
	}
}

func TestScanAll_NilCLIDetector(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{
			agents: []store.Agent{{ID: "agent-1"}},
			files: map[string][]store.AgentFileEntry{
				"agent-1": {{Path: "SOUL.md"}},
			},
			contents: map[string]string{
				"agent-1/SOUL.md": "Run `vrooli scenario start foo`",
			},
		},
		&mockTeamLister{},
		&mockSkillLister{},
		nil, nil, // nil cliDetector
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No code-usage edges since detector is nil
	if len(edges) != 0 {
		t.Fatalf("expected 0 edges (nil detector), got %d", len(edges))
	}
}

func TestScanAll_Combined(t *testing.T) {
	det := NewCLIDetector([]string{"prompt-manager"})
	s := NewScanner(
		&mockAgentLister{
			agents: []store.Agent{{ID: "agent-1"}},
			files: map[string][]store.AgentFileEntry{
				"agent-1": {{Path: "SOUL.md"}},
			},
			contents: map[string]string{
				"agent-1/SOUL.md": "prompt-manager skill read skill-a\nRun `vrooli scenario start x`",
			},
		},
		&mockTeamLister{
			teams: []store.Team{{ID: "team-1"}},
			files: map[string][]store.TeamFileEntry{
				"team-1": {{Path: "shared.md"}},
			},
			contents: map[string]string{
				"team-1/shared.md": "**skill-b**",
			},
		},
		&mockSkillLister{
			skills: []store.Skill{
				{ID: "skill-a", DefaultScope: "skill-b"},
				{ID: "skill-b"},
			},
			contentMap: map[string]string{
				"skill-a": "See **skill-b**",
			},
		},
		&mockRelationStore{
			members: map[string][]store.TeamMemberRelation{
				"team-1": {{TeamID: "team-1", AgentID: "agent-1"}},
			},
		},
		det,
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expected edges:
	// 1. agent-1 → skill-a (cli-read)
	// 2. agent-1 → cli:vrooli (code-usage)
	// 3. team-1 → skill-b (bold-listed)
	// 4. skill-a → skill-b (default-scope)
	// 5. skill-a → skill-b (bold-listed from content)
	// 6. team-1 → agent-1 (membership)
	if len(edges) < 5 {
		t.Fatalf("expected at least 5 edges, got %d: %+v", len(edges), edges)
	}

	kinds := make(map[EdgeKind]int)
	for _, e := range edges {
		kinds[e.Kind]++
	}
	if kinds[EdgeCLIRead] < 1 {
		t.Error("expected at least 1 cli-read edge")
	}
	if kinds[EdgeCodeUsage] < 1 {
		t.Error("expected at least 1 code-usage edge")
	}
	if kinds[EdgeMembership] < 1 {
		t.Error("expected at least 1 membership edge")
	}
	if kinds[EdgeDefaultScope] < 1 {
		t.Error("expected at least 1 default-scope edge")
	}
}

// testValidIDs is a set of known skill IDs for testing.
var testValidIDs = map[string]bool{
	"screaming-architecture-audit": true,
	"e2e-testing":                  true,
	"cli-steer":                    true,
	"api-steer":                    true,
	"utils-unification":            true,
	"knowledge-observatory-tools":  true,
	"feature-scope":                true,
	"platform-scope":               true,
	"some-other-skill":             true,
}

func TestExtractRefsFromContent_CLIReadSingle(t *testing.T) {
	content := "Use this: `prompt-manager skill read screaming-architecture-audit`"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].skillID != "screaming-architecture-audit" {
		t.Errorf("expected screaming-architecture-audit, got %s", refs[0].skillID)
	}
	if refs[0].edgeKind != EdgeCLIRead {
		t.Errorf("expected cli-read, got %s", refs[0].edgeKind)
	}
	if refs[0].lineNumber != 1 {
		t.Errorf("expected line 1, got %d", refs[0].lineNumber)
	}
}

func TestExtractRefsFromContent_CLIReadPlural(t *testing.T) {
	content := "`prompt-manager skills read knowledge-observatory-tools`"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].skillID != "knowledge-observatory-tools" {
		t.Errorf("expected knowledge-observatory-tools, got %s", refs[0].skillID)
	}
}

func TestExtractRefsFromContent_CLIReadMulti(t *testing.T) {
	content := "`prompt-manager skill read cli-steer api-steer utils-unification`"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 3 {
		t.Fatalf("expected 3 refs, got %d", len(refs))
	}

	ids := make(map[string]bool)
	for _, ref := range refs {
		ids[ref.skillID] = true
		if ref.edgeKind != EdgeCLIRead {
			t.Errorf("expected cli-read, got %s", ref.edgeKind)
		}
	}
	for _, expected := range []string{"cli-steer", "api-steer", "utils-unification"} {
		if !ids[expected] {
			t.Errorf("missing expected skill ID: %s", expected)
		}
	}
}

func TestExtractRefsFromContent_BoldListed(t *testing.T) {
	content := "**screaming-architecture-audit** -- Architecture alignment tool"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].edgeKind != EdgeBoldListed {
		t.Errorf("expected bold-listed, got %s", refs[0].edgeKind)
	}
}

func TestExtractRefsFromContent_BoldListedInvalid(t *testing.T) {
	content := "**Not A Skill** -- Some description"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 0 {
		t.Fatalf("expected 0 refs for invalid bold text, got %d", len(refs))
	}
}

func TestExtractRefsFromContent_TemplateExclusion(t *testing.T) {
	content := "{{SKILL}} {{TARGET}} prompt-manager skill read {{skill-id}}"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 0 {
		t.Fatalf("expected 0 refs for template vars, got %d", len(refs))
	}
}

func TestExtractRefsFromContent_PlaceholderExclusion(t *testing.T) {
	content := "prompt-manager skill read <skill-id>"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 0 {
		t.Fatalf("expected 0 refs for placeholder, got %d", len(refs))
	}
}

func TestExtractRefsFromContent_PathRefRelative(t *testing.T) {
	content := "See store/skills/packs/core/feature-scope/SKILL.md for details"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].skillID != "feature-scope" {
		t.Errorf("expected feature-scope, got %s", refs[0].skillID)
	}
	if refs[0].edgeKind != EdgePathRef {
		t.Errorf("expected path-ref, got %s", refs[0].edgeKind)
	}
}

func TestExtractRefsFromContent_PathRefDirectory(t *testing.T) {
	content := "Located at store/skills/packs/core/platform-scope/"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].skillID != "platform-scope" {
		t.Errorf("expected platform-scope, got %s", refs[0].skillID)
	}
}

func TestExtractRefsFromContent_MixedPatterns(t *testing.T) {
	content := `# Agent Skills

- **e2e-testing** -- End to end testing skill
- Architecture: ` + "`prompt-manager skill read screaming-architecture-audit`" + `
- See store/skills/packs/core/feature-scope/SKILL.md
`
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 3 {
		t.Fatalf("expected 3 refs, got %d", len(refs))
	}

	kinds := make(map[EdgeKind]int)
	ids := make(map[string]bool)
	for _, ref := range refs {
		kinds[ref.edgeKind]++
		ids[ref.skillID] = true
	}

	if !ids["e2e-testing"] {
		t.Error("missing e2e-testing")
	}
	if !ids["screaming-architecture-audit"] {
		t.Error("missing screaming-architecture-audit")
	}
	if !ids["feature-scope"] {
		t.Error("missing feature-scope")
	}
}

func TestExtractRefsFromContent_Deduplication(t *testing.T) {
	content := `prompt-manager skill read e2e-testing
Some text
prompt-manager skill read e2e-testing`
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 1 {
		t.Fatalf("expected 1 ref (deduplicated), got %d", len(refs))
	}
	if refs[0].lineNumber != 1 {
		t.Errorf("expected line 1 (first occurrence), got %d", refs[0].lineNumber)
	}
}

func TestExtractRefsFromContent_DifferentTypesNotDeduplicated(t *testing.T) {
	content := `**e2e-testing** -- Testing skill
prompt-manager skill read e2e-testing`
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 2 {
		t.Fatalf("expected 2 refs (different types), got %d", len(refs))
	}

	kinds := make(map[EdgeKind]bool)
	for _, ref := range refs {
		kinds[ref.edgeKind] = true
	}
	if !kinds[EdgeBoldListed] {
		t.Error("missing bold-listed type")
	}
	if !kinds[EdgeCLIRead] {
		t.Error("missing cli-read type")
	}
}

func TestExtractRefsFromContent_UnknownSkillIgnored(t *testing.T) {
	content := "prompt-manager skill read nonexistent-skill"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 0 {
		t.Fatalf("expected 0 refs for unknown skill, got %d", len(refs))
	}
}

func TestExtractRefsFromContent_EmptyContent(t *testing.T) {
	refs := extractRefsFromContent("", testValidIDs)

	if len(refs) != 0 {
		t.Fatalf("expected 0 refs for empty content, got %d", len(refs))
	}
}

func TestIsValidSkillToken(t *testing.T) {
	tests := []struct {
		token string
		valid bool
	}{
		{"e2e-testing", true},
		{"cli-steer", true},
		{"simple", true},
		{"a1", true},
		{"Not-Valid", false},
		{"123-invalid", false},
		{"-invalid", false},
		{"{{SKILL}}", false},
		{"<skill-id>", false},
		{"has spaces", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isValidSkillToken(tt.token)
		if got != tt.valid {
			t.Errorf("isValidSkillToken(%q) = %v, want %v", tt.token, got, tt.valid)
		}
	}
}

func TestCLINodeID(t *testing.T) {
	tests := []struct {
		cmd    string
		expect string
	}{
		{"vrooli scenario start foo", "cli:vrooli"},
		{"prompt-manager skill read test", "cli:prompt-manager"},
		{"", "cli:unknown"},
	}
	for _, tt := range tests {
		got := cliNodeID(tt.cmd)
		if got != tt.expect {
			t.Errorf("cliNodeID(%q) = %q, want %q", tt.cmd, got, tt.expect)
		}
	}
}
