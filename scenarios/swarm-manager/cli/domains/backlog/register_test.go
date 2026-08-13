package backlog

import (
	"testing"

	testutil "github.com/vrooli/cli-core/cliapptest"
)

func TestRegister(t *testing.T) {
	g := Register(testutil.StubDeps())

	if g.Name != "backlog" {
		t.Errorf("group Name = %q, want %q", g.Name, "backlog")
	}
	if g.Description == "" {
		t.Error("group Description is empty")
	}

	want := []string{"list", "pending-questions", "get", "criteria-set", "create", "update", "delete", "dismiss", "plan-workshop", "recreate", "reset-artifacts", "files", "file-get", "file-upload", "process-preflight", "queue", "batch-create", "batch-queue", "export", "import", "review-decide", "recover-review", "retry", "search-ai"}
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
		if c.Run == nil && c.RunCtx == nil {
			t.Errorf("subcommand %q has no handler", c.Name)
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
