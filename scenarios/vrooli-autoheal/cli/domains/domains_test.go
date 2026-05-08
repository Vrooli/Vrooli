package domains

import (
	"testing"
	"vrooli-autoheal/cli/internal/support"
)

func TestCommandGroupsExposeLegacyEntrypoints(t *testing.T) {
	groups := CommandGroups(nil, support.Dependencies{})
	if len(groups) != 2 {
		t.Fatalf("CommandGroups() count = %d, want 2", len(groups))
	}
	if groups[0].Title != "Operations" {
		t.Fatalf("first command group title = %q, want Operations", groups[0].Title)
	}
	if groups[1].Title != "Recovery" {
		t.Fatalf("second command group title = %q, want Recovery", groups[1].Title)
	}
}

func TestSubcommandGroupsExposeScenarioDomains(t *testing.T) {
	groups := SubcommandGroups(nil, support.Dependencies{})
	want := []string{"check", "config", "host", "incidents", "monitoring", "actions"}
	if len(groups) != len(want) {
		t.Fatalf("SubcommandGroups() count = %d, want %d", len(groups), len(want))
	}
	for i, name := range want {
		if groups[i].Name != name {
			t.Fatalf("SubcommandGroups()[%d].Name = %q, want %q", i, groups[i].Name, name)
		}
	}
}
