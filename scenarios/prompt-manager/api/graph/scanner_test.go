package graph

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"prompt-manager/store"
)

func TestScanAll_RepositoryAndGeneratedPromptReferenceSources(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("prompt-manager skill read agent-system-audit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	workflowPath := filepath.Join(root, "scenarios", "swarm-manager", ".vrooli", "agent-manager", "plan-author.json")
	if err := os.MkdirAll(filepath.Dir(workflowPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workflowPath, []byte(`{"nodes":[{"run":{"promptRef":{"skillId":"swarm-manager-workflow-plan-author"}}}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewScanner(
		&mockAgentLister{agents: []store.Agent{{ID: "member-1"}}},
		&mockTeamLister{teams: []store.Team{{ID: "team-1"}}},
		&mockSkillLister{skills: []store.Skill{{ID: "agent-system-audit"}, {ID: "swarm-manager-workflow-plan-author"}, {ID: "writing-standards"}}},
		&mockRelationStore{members: map[string][]store.TeamMemberRelation{"team-1": {{AgentID: "member-1"}}}},
		nil,
	)
	s.SetRepositoryRoot(root)
	s.SetGeneratedPromptProvider(func(_ context.Context, teamID, agentID string) (string, error) {
		if teamID != "team-1" || agentID != "member-1" {
			t.Fatalf("unexpected prompt request %s/%s", teamID, agentID)
		}
		return "prompt-manager skill read writing-standards", nil
	})

	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("ScanAll: %v", err)
	}
	want := map[string]string{
		"system:agents": "agent-system-audit",
		"workflow:scenarios/swarm-manager/.vrooli/agent-manager/plan-author.json": "swarm-manager-workflow-plan-author",
		"member-1": "writing-standards",
	}
	for from, to := range want {
		found := false
		for _, edge := range edges {
			if edge.From == from && edge.To == to && edge.Kind == EdgeCLIRead {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing %s -> %s CLI-read edge: %+v", from, to, edges)
		}
	}
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

func TestScanAll_ActionUseRefs(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{
			agents: []store.Agent{{ID: "agent-1"}},
			files: map[string][]store.AgentFileEntry{
				"agent-1": {{Path: "SOUL.md"}},
			},
			contents: map[string]string{
				"agent-1/SOUL.md": "Use action:scenario.status.show or `prompt-manager action run scenario.status.show`.",
			},
		},
		&mockTeamLister{},
		&mockSkillLister{},
		nil,
		nil,
		&mockActionLister{actions: []store.Action{{ID: "scenario.status.show"}}},
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected deduped action-use edge, got %d: %+v", len(edges), edges)
	}
	if edges[0].From != "agent-1" || edges[0].To != "action:scenario.status.show" || edges[0].Kind != EdgeActionUse {
		t.Fatalf("unexpected action-use edge: %+v", edges[0])
	}
	if edges[0].SourceFile != "SOUL.md" || edges[0].LineNumber != 1 {
		t.Fatalf("unexpected action-use location: %+v", edges[0])
	}
}

func TestScanAll_ActionUseIgnoresUnknownAction(t *testing.T) {
	s := NewScanner(
		&mockAgentLister{
			agents: []store.Agent{{ID: "agent-1"}},
			files: map[string][]store.AgentFileEntry{
				"agent-1": {{Path: "SOUL.md"}},
			},
			contents: map[string]string{
				"agent-1/SOUL.md": "Use action:missing.action.",
			},
		},
		&mockTeamLister{},
		&mockSkillLister{},
		nil,
		nil,
		&mockActionLister{actions: []store.Action{{ID: "scenario.status.show"}}},
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 0 {
		t.Fatalf("expected no edges for unknown action, got %+v", edges)
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

func TestParseCommandParts(t *testing.T) {
	tests := []struct {
		name           string
		cmd            string
		wantCommand    string
		wantSubcommand string
	}{
		{
			name:           "command with subcommand",
			cmd:            "scenario-completeness-scoring score prompt-manager --json",
			wantCommand:    "scenario-completeness-scoring",
			wantSubcommand: "score",
		},
		{
			name:           "command with only flags after",
			cmd:            "grep --help",
			wantCommand:    "grep",
			wantSubcommand: "",
		},
		{
			name:           "empty command",
			cmd:            "",
			wantCommand:    "",
			wantSubcommand: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCommand, gotSubcommand := parseCommandParts(tt.cmd)
			if gotCommand != tt.wantCommand || gotSubcommand != tt.wantSubcommand {
				t.Fatalf("parseCommandParts(%q) = (%q,%q), want (%q,%q)",
					tt.cmd, gotCommand, gotSubcommand, tt.wantCommand, tt.wantSubcommand)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Skill and Team code-usage edge tests
// ---------------------------------------------------------------------------

func TestScanAll_SkillCodeUsage(t *testing.T) {
	det := NewCLIDetector([]string{"prompt-manager"})
	s := NewScanner(
		&mockAgentLister{},
		&mockTeamLister{},
		&mockSkillLister{
			skills: []store.Skill{{ID: "skill-a"}},
			contentMap: map[string]string{
				"skill-a": "Run `vrooli scenario start my-app` to deploy",
			},
		},
		nil, det,
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d: %+v", len(edges), edges)
	}
	e := edges[0]
	if e.From != "skill-a" || e.To != "cli:vrooli" || e.Kind != EdgeCodeUsage {
		t.Errorf("unexpected edge: %+v", e)
	}
	if e.Command != "vrooli" || e.Subcommand != "scenario" {
		t.Errorf("expected command metadata (vrooli,scenario), got (%s,%s)", e.Command, e.Subcommand)
	}
	if e.SourceFile != "SKILL.md" {
		t.Errorf("expected SourceFile SKILL.md, got %s", e.SourceFile)
	}
}

func TestScanAll_TeamCodeUsage(t *testing.T) {
	det := NewCLIDetector([]string{"prompt-manager"})
	s := NewScanner(
		&mockAgentLister{},
		&mockTeamLister{
			teams: []store.Team{{ID: "team-1"}},
			files: map[string][]store.TeamFileEntry{
				"team-1": {{Path: "shared.md"}},
			},
			contents: map[string]string{
				"team-1/shared.md": "Deploy with `vrooli scenario start demo`",
			},
		},
		&mockSkillLister{},
		nil, det,
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d: %+v", len(edges), edges)
	}
	e := edges[0]
	if e.From != "team-1" || e.To != "cli:vrooli" || e.Kind != EdgeCodeUsage {
		t.Errorf("unexpected edge: %+v", e)
	}
	if e.Command != "vrooli" || e.Subcommand != "scenario" {
		t.Errorf("expected command metadata (vrooli,scenario), got (%s,%s)", e.Command, e.Subcommand)
	}
	if e.SourceFile != "shared.md" {
		t.Errorf("expected SourceFile shared.md, got %s", e.SourceFile)
	}
}

func TestScanAll_ScriptEdgesCreated(t *testing.T) {
	// Shell script references in backticks produce CodeScript edges.
	det := NewCLIDetector(nil)
	s := NewScanner(
		&mockAgentLister{
			agents: []store.Agent{{ID: "agent-1"}},
			files: map[string][]store.AgentFileEntry{
				"agent-1": {{Path: "SOUL.md"}},
			},
			contents: map[string]string{
				"agent-1/SOUL.md": "Run `scripts/deploy.sh`",
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
		t.Fatalf("expected 1 edge, got %d: %+v", len(edges), edges)
	}
	if edges[0].Category != CodeScript {
		t.Errorf("expected category script, got %s", edges[0].Category)
	}
}

func TestScanAll_APICallExcluded(t *testing.T) {
	// Bare HTTP patterns (CodeAPICall) do NOT produce edges — they're
	// documentation of API endpoints, not tool invocations.
	det := NewCLIDetector(nil)
	s := NewScanner(
		&mockAgentLister{
			agents: []store.Agent{{ID: "agent-1"}},
			files: map[string][]store.AgentFileEntry{
				"agent-1": {{Path: "SOUL.md"}},
			},
			contents: map[string]string{
				"agent-1/SOUL.md": "GET https://api.example.com/data",
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
	if len(edges) != 0 {
		t.Fatalf("expected 0 edges (API calls excluded), got %d: %+v", len(edges), edges)
	}
}

// ---------------------------------------------------------------------------
// codeUsageEdgesFromContent unit tests
// ---------------------------------------------------------------------------

func TestCodeUsageEdgesFromContent_NilDetector(t *testing.T) {
	s := &Scanner{} // cliDetector is nil
	edges := s.codeUsageEdgesFromContent("src-1", "file.md", "Run `vrooli help`")
	if len(edges) != 0 {
		t.Fatalf("expected 0 edges with nil detector, got %d", len(edges))
	}
}

func TestCodeUsageEdgesFromContent_AllowedCategories(t *testing.T) {
	// ScenarioCLI, ExternalTool, and Script produce edges. APICall does not.
	s := &Scanner{
		cliDetector: &stubCodeDetector{
			refs: []CodeReference{
				{Category: CodeScenarioCLI, Value: "vrooli scenario start x", Line: 1},
				{Category: CodeAPICall, Value: "GET https://example.com", Line: 2},
				{Category: CodeScript, Value: "scripts/deploy.sh", Line: 3},
				{Category: CodeExternalTool, Value: "grep -r pattern", Line: 4},
			},
		},
	}
	edges := s.codeUsageEdgesFromContent("src-1", "file.md", "irrelevant")
	if len(edges) != 3 {
		t.Fatalf("expected 3 edges (CLI + Script + ExternalTool), got %d: %+v", len(edges), edges)
	}
	cats := make(map[CodeCategory]bool)
	for _, e := range edges {
		cats[e.Category] = true
	}
	if !cats[CodeScenarioCLI] {
		t.Error("missing CodeScenarioCLI edge")
	}
	if !cats[CodeScript] {
		t.Error("missing CodeScript edge")
	}
	if !cats[CodeExternalTool] {
		t.Error("missing CodeExternalTool edge")
	}
	if cats[CodeAPICall] {
		t.Error("CodeAPICall should not produce edges")
	}
}

func TestScanAll_MockDetector(t *testing.T) {
	// Demonstrates the codeDetector interface seam works with a stub.
	stub := &stubCodeDetector{
		refs: []CodeReference{
			{Category: CodeScenarioCLI, Value: "custom-tool deploy", Line: 5},
		},
	}
	s := NewScanner(
		&mockAgentLister{
			agents: []store.Agent{{ID: "agent-1"}},
			files: map[string][]store.AgentFileEntry{
				"agent-1": {{Path: "SOUL.md"}},
			},
			contents: map[string]string{
				"agent-1/SOUL.md": "some content",
			},
		},
		&mockTeamLister{},
		&mockSkillLister{},
		nil, stub,
	)
	edges, err := s.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge from stub detector, got %d", len(edges))
	}
	if edges[0].To != "cli:custom-tool" {
		t.Errorf("expected cli:custom-tool, got %s", edges[0].To)
	}
}

// ---------------------------------------------------------------------------
// CLI-read exclusion and category field tests
// ---------------------------------------------------------------------------

func TestCodeUsageEdges_SkipsCLIRead(t *testing.T) {
	// "prompt-manager skill read" is a Skill→Skill relation (EdgeCLIRead),
	// not a code-usage edge. Verify it's excluded.
	s := &Scanner{
		cliDetector: &stubCodeDetector{
			refs: []CodeReference{
				{Category: CodeScenarioCLI, Value: "prompt-manager skill read my-skill", Line: 1},
				{Category: CodeScenarioCLI, Value: "prompt-manager skills read a b", Line: 2},
				{Category: CodeScenarioCLI, Value: "vrooli scenario start x", Line: 3},
			},
		},
	}
	edges := s.codeUsageEdgesFromContent("skill-a", "SKILL.md", "irrelevant")
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge (skill read excluded), got %d: %+v", len(edges), edges)
	}
	if edges[0].To != "cli:vrooli" {
		t.Errorf("expected cli:vrooli, got %s", edges[0].To)
	}
}

func TestCodeUsageEdges_CategoryFieldSet(t *testing.T) {
	// Verify the Category field is populated on emitted edges.
	s := &Scanner{
		cliDetector: &stubCodeDetector{
			refs: []CodeReference{
				{Category: CodeScenarioCLI, Value: "vrooli help", Line: 1},
				{Category: CodeExternalTool, Value: "grep pattern", Line: 2},
				{Category: CodeScript, Value: "deploy.sh", Line: 3},
			},
		},
	}
	edges := s.codeUsageEdgesFromContent("src-1", "file.md", "irrelevant")
	if len(edges) != 3 {
		t.Fatalf("expected 3 edges, got %d: %+v", len(edges), edges)
	}
	for _, e := range edges {
		if e.Category == "" {
			t.Errorf("edge to %s has empty Category", e.To)
		}
		if e.Kind != EdgeCodeUsage {
			t.Errorf("expected EdgeCodeUsage, got %s", e.Kind)
		}
	}
}

func TestCodeUsageEdges_ExternalToolEdges(t *testing.T) {
	det := NewCLIDetector([]string{"prompt-manager"})
	s := NewScanner(
		&mockAgentLister{
			agents: []store.Agent{{ID: "agent-1"}},
			files: map[string][]store.AgentFileEntry{
				"agent-1": {{Path: "SOUL.md"}},
			},
			contents: map[string]string{
				"agent-1/SOUL.md": "Run `grep -r pattern .` to search",
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
		t.Fatalf("expected 1 edge, got %d: %+v", len(edges), edges)
	}
	if edges[0].Category != CodeExternalTool {
		t.Errorf("expected category external-tool, got %s", edges[0].Category)
	}
	if edges[0].To != "cli:grep" {
		t.Errorf("expected cli:grep, got %s", edges[0].To)
	}
}

func TestCodeUsageEdges_Deduplication(t *testing.T) {
	// Same tool referenced twice should produce only one edge.
	s := &Scanner{
		cliDetector: &stubCodeDetector{
			refs: []CodeReference{
				{Category: CodeExternalTool, Value: "grep foo", Line: 1},
				{Category: CodeExternalTool, Value: "grep bar", Line: 3},
			},
		},
	}
	edges := s.codeUsageEdgesFromContent("src-1", "file.md", "irrelevant")
	if len(edges) != 1 {
		t.Fatalf("expected 1 deduplicated edge, got %d: %+v", len(edges), edges)
	}
}
