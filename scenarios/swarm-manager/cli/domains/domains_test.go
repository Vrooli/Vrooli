package domains

import (
	"testing"

	"swarm-manager/cli/internal/testutil"
)

func TestCommandGroups(t *testing.T) {
	groups := CommandGroups(testutil.StubDeps())

	wantTitles := []string{"Overview", "Migration"}
	if len(groups) != len(wantTitles) {
		t.Fatalf("CommandGroups count = %d, want %d", len(groups), len(wantTitles))
	}
	for i, g := range groups {
		if g.Title != wantTitles[i] {
			t.Errorf("group[%d] Title = %q, want %q", i, g.Title, wantTitles[i])
		}
		if len(g.Commands) == 0 {
			t.Errorf("group %q has no commands", g.Title)
		}
		for _, c := range g.Commands {
			if c.Name == "" {
				t.Errorf("group %q has command with empty Name", g.Title)
			}
			if c.Run == nil {
				t.Errorf("group %q command %q has nil Run handler", g.Title, c.Name)
			}
		}
	}
}

func TestSubcommandGroups(t *testing.T) {
	groups := SubcommandGroups(testutil.StubDeps())

	wantNames := []string{
		"backlog", "scenarios", "settings", "queue", "execution", "evidence", "prompts",
		"goals", "proposals", "captures", "records", "related", "agent-manager",
		"operations", "portfolio", "sessions", "stats", "ai-search", "autofiler",
	}
	if len(groups) != len(wantNames) {
		t.Fatalf("SubcommandGroups count = %d, want %d", len(groups), len(wantNames))
	}

	seen := map[string]bool{}
	for i, g := range groups {
		if g.Name != wantNames[i] {
			t.Errorf("group[%d] Name = %q, want %q", i, g.Name, wantNames[i])
		}
		if seen[g.Name] {
			t.Errorf("duplicate group name %q", g.Name)
		}
		seen[g.Name] = true
		if g.Description == "" {
			t.Errorf("group %q has empty Description", g.Name)
		}
		if len(g.Subcommands) == 0 {
			t.Errorf("group %q has no subcommands", g.Name)
		}
		for _, c := range g.Subcommands {
			if c.Name == "" {
				t.Errorf("group %q has subcommand with empty Name", g.Name)
			}
			if c.Run == nil {
				t.Errorf("group %q subcommand %q has nil Run handler", g.Name, c.Name)
			}
		}
	}
}
