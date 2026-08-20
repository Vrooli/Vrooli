package domains

import (
	"testing"

	"agent-manager/cli/internal/support"
)

func TestCommandGroupsRegistersEveryTopLevelCommand(t *testing.T) {
	groups := CommandGroups(support.Dependencies{})
	if len(groups) != 15 {
		t.Fatalf("group count = %d, want 15", len(groups))
	}
	for _, group := range groups {
		if group.Title == "" || len(group.Commands) == 0 {
			t.Fatalf("invalid group: %+v", group)
		}
	}
}
