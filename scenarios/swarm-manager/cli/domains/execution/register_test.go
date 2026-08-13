package execution

import (
	"testing"

	testutil "github.com/vrooli/cli-core/cliapptest"
)

func TestRegister(t *testing.T) {
	g := Register(testutil.StubDeps())

	if g.Name != "execution" {
		t.Errorf("group Name = %q, want %q", g.Name, "execution")
	}
	if g.Description == "" {
		t.Error("group Description is empty")
	}

	want := []string{"list", "get", "create", "policy-get", "policy-update", "prompt-trace", "start", "cancel", "retry", "circuit-breaker-reset"}
	wantSet := make(map[string]bool, len(want))
	for _, n := range want {
		wantSet[n] = true
	}

	gotSet := make(map[string]bool, len(g.Subcommands))
	for _, c := range g.Subcommands {
		if c.Name == "" {
			t.Error("subcommand has empty Name")
			continue
		}
		if gotSet[c.Name] {
			t.Errorf("duplicate subcommand name %q", c.Name)
		}
		gotSet[c.Name] = true
		if c.Description == "" {
			t.Errorf("subcommand %q has empty Description", c.Name)
		}
		if c.Run == nil {
			t.Errorf("subcommand %q has nil Run handler (unwired dependency)", c.Name)
		}
		if !wantSet[c.Name] {
			t.Errorf("unexpected subcommand %q (not in expected set)", c.Name)
		}
	}

	for _, n := range want {
		if !gotSet[n] {
			t.Errorf("missing expected subcommand %q", n)
		}
	}

	if len(g.Subcommands) != len(want) {
		t.Errorf("subcommand count = %d, want %d", len(g.Subcommands), len(want))
	}
}
