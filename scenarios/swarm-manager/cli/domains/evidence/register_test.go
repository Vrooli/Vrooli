package evidence

import (
	"testing"

	"swarm-manager/cli/internal/testutil"
)

func TestRegister(t *testing.T) {
	group := Register(testutil.StubDeps())
	want := map[string]bool{"run": true, "entity": true, "reconcile": true, "verify": true}
	if group.Name != "evidence" || group.Description == "" || len(group.Subcommands) != len(want) {
		t.Fatalf("group = %+v", group)
	}
	for _, command := range group.Subcommands {
		if !want[command.Name] || command.Run == nil || command.Description == "" {
			t.Fatalf("command = %+v", command)
		}
	}
}
