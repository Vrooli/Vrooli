package upstreamverb

import (
	"testing"

	"github.com/vrooli/cli-core/upstreamcheck"
)

func TestCommandsShape(t *testing.T) {
	g := Commands(upstreamcheck.Default(upstreamcheck.Config{DisplayName: "opencode"}))
	if g.Name != "upstream-check" || g.DefaultSubcommand != "check" {
		t.Fatalf("unexpected group: %+v", g)
	}
	if len(g.Subcommands) != 1 || g.Subcommands[0].Name != "check" {
		t.Fatalf("unexpected subcommands: %+v", g.Subcommands)
	}
}
