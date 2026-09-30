package catalog

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestReadinessFloorAllowsLegacyHealthAlias(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{})
	if got := readinessFloor(ctx); got != "" {
		t.Fatalf("floor = %q, want empty for a command without --floor", got)
	}
}

func TestReadinessFloorReadsDeclaredFlag(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "floor"}}},
		Flags:  map[string]string{"floor": "verified"},
	})
	if got := readinessFloor(ctx); got != "verified" {
		t.Fatalf("floor = %q, want verified", got)
	}
}
