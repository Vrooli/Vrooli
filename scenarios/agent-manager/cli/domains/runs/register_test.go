package runs

import (
	"testing"

	"agent-manager/cli/internal/support"
)

func TestRegister(t *testing.T) {
	g := Register(support.Dependencies{})
	if g.Title == "" || len(g.Commands) != 2 {
		t.Fatalf("invalid group: %+v", g)
	}
}
