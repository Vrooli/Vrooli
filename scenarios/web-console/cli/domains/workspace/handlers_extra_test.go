package workspace

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestWorkspaceValidation(t *testing.T) {
	h := &handlers{}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{
		Flags:       []cliapp.Flag{{Name: "body-file"}},
		Positionals: []cliapp.Positional{{Name: "session-id"}, {Name: "group-id"}},
	}})
	if err := h.layoutSave(ctx); err == nil {
		t.Fatal("missing layout body unexpectedly succeeded")
	}
	if err := h.paneUpdate(ctx); err == nil {
		t.Fatal("missing pane id unexpectedly succeeded")
	}
	if err := h.paneDelete(ctx); err == nil {
		t.Fatal("missing pane id unexpectedly succeeded")
	}
	if err := h.groupCreate(ctx); err == nil {
		t.Fatal("missing group body unexpectedly succeeded")
	}
	if err := h.groupUpdate(ctx); err == nil {
		t.Fatal("missing group id unexpectedly succeeded")
	}
	if err := h.groupDelete(ctx); err == nil {
		t.Fatal("missing group id unexpectedly succeeded")
	}
}
