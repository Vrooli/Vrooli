package domains

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSubcommandGroupsRegistersExpectedDomains(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	groups, err := SubcommandGroups(nil, manifest)
	if err != nil {
		t.Fatalf("SubcommandGroups: %v", err)
	}
	if len(groups) != 6 {
		t.Fatalf("SubcommandGroups() returned %d groups, want 6", len(groups))
	}

	got := make(map[string]int, len(groups))
	for _, group := range groups {
		got[group.Name] = len(group.Subcommands)
		if !group.NeedsAPI {
			t.Fatalf("group %q should require API access", group.Name)
		}
	}

	want := map[string]int{
		"repo":     6,
		"branch":   4,
		"worktree": 8,
		"review":   3,
		"audit":    1,
		"baseline": 11, // 5 record verbs + 6 engagement verbs (start/check/promote/abandon/status/gc)
	}
	for name, count := range want {
		if got[name] != count {
			t.Fatalf("group %q command count = %d, want %d; all groups: %#v", name, got[name], count, got)
		}
	}
}

func TestCommandGroupsRemainUnused(t *testing.T) {
	if got := CommandGroups(nil); got != nil {
		t.Fatalf("CommandGroups() = %#v, want nil", got)
	}
}
