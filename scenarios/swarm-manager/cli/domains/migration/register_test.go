package migration

import (
	"testing"

	"swarm-manager/cli/internal/testutil"
)

func TestRegister(t *testing.T) {
	g := Register(testutil.StubDeps())

	if g.Title != "Migration" {
		t.Errorf("group Title = %q, want %q", g.Title, "Migration")
	}

	want := []string{"migrate-workshop"}
	if len(g.Commands) != len(want) {
		t.Fatalf("command count = %d, want %d", len(g.Commands), len(want))
	}

	for i, c := range g.Commands {
		if c.Name != want[i] {
			t.Errorf("command[%d] Name = %q, want %q", i, c.Name, want[i])
		}
		if c.Description == "" {
			t.Errorf("command %q has empty Description", c.Name)
		}
		if c.Run == nil {
			t.Errorf("command %q has nil Run handler", c.Name)
		}
	}
}
