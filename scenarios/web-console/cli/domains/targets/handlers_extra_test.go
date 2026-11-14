package targets

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shared"
)

func TestTargetValidationHelpers(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "target-id"}}}})
	if _, err := targetID(ctx, "get"); err == nil {
		t.Fatal("missing target id succeeded")
	}
	if defaultText("", "none") != "none" || defaultText("x", "none") != "x" {
		t.Fatal("default text mismatch")
	}
	if len(readinessRows(&sharedv1.Target{})) != 1 {
		t.Fatal("empty readiness should have one explanatory row")
	}
}
