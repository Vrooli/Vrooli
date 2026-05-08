package health

import (
	"testing"

	"vrooli-autoheal/cli/internal/support"
)

func TestRegisterProvidesOperationsCommands(t *testing.T) {
	group := Register(nil, support.Dependencies{})
	want := []string{"status", "tick", "loop", "platform", "diagnose-port"}
	if group.Title != "Operations" {
		t.Fatalf("Register().Title = %q, want Operations", group.Title)
	}
	if len(group.Commands) != len(want) {
		t.Fatalf("Register() command count = %d, want %d", len(group.Commands), len(want))
	}
	for i, name := range want {
		if group.Commands[i].Name != name {
			t.Fatalf("Register() command[%d] = %q, want %q", i, group.Commands[i].Name, name)
		}
	}
}
