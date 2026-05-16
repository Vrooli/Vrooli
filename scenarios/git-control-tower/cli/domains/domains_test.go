package domains

import "testing"

func TestSubcommandGroupsRegistersExpectedDomains(t *testing.T) {
	groups := SubcommandGroups(nil)
	if len(groups) != 5 {
		t.Fatalf("SubcommandGroups() returned %d groups, want 5", len(groups))
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
