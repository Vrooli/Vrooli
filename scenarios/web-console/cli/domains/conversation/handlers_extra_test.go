package conversation

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestValidationAndTruncate(t *testing.T) {
	h := &handlers{}
	empty := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "session"}, {Name: "since"}, {Name: "event"}, {Name: "body-file"},
	}}})
	for _, call := range []func(cliapp.RunContext) error{h.get, h.cursorSet, h.summarize} {
		if err := call(empty); err == nil {
			t.Fatal("missing required flags unexpectedly succeeded")
		}
	}
	if got := truncate("short", 10); got != "short" {
		t.Fatalf("truncate short = %q", got)
	}
	if got := truncate(strings.Repeat("x", 5), 3); got != "xxx…" {
		t.Fatalf("truncate long = %q", got)
	}
}
