package resourcehandlers

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/resources"
)

func TestManagedUpdateRollbackDoesNotReportCommandSuccess(t *testing.T) {
	for _, phase := range []resources.ManagedUpdatePhase{resources.UpdateFailedPreserved, resources.UpdateFailedRestored} {
		var output bytes.Buffer
		err := writeManagedUpdateCommandResult(&output, "resume", resources.ManagedServiceUpdate{ID: "request", Phase: phase}, nil)
		var exit rootcli.ExitCodeError
		if !errors.As(err, &exit) || exit.Code != 1 || !strings.Contains(output.String(), string(phase)) {
			t.Fatalf("failed release reported success or lost serving recovery receipt: %s %v", output.String(), err)
		}
		output.Reset()
		if err := writeManagedUpdateCommandResult(&output, "status", resources.ManagedServiceUpdate{ID: "request", Phase: phase}, nil); err != nil {
			t.Fatalf("read-only failed-release inspection must succeed: %v", err)
		}
	}
}

func TestManagedUpdateCommandHelpDoesNotRequireAnOwner(t *testing.T) {
	for _, readOnly := range []bool{false, true} {
		ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
		deps := rootcli.HandlerDeps[testContext]{Stdout: func(c testContext) io.Writer { return c.stdout }}
		if err := resourceUpdateByOwnerHandler(deps, readOnly)(ctx, nil, []string{"--help"}); err != nil {
			t.Fatalf("ownerless help: %v", err)
		}
		if !strings.Contains(ctx.stdout.String(), "resource update") {
			t.Fatalf("missing command help: %s", ctx.stdout)
		}
	}
}

func TestManagedUpdateCommandRejectsUnusedHarnessReleaseFlags(t *testing.T) {
	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := rootcli.HandlerDeps[testContext]{Stdout: func(c testContext) io.Writer { return c.stdout }, OutputFormat: func(testContext) (cliout.Format, error) { return cliout.FormatJSON, nil }}
	c := resources.NewController(t.TempDir(), t.TempDir())
	if err := resourceUpdateByOwnerHandler(deps, false)(ctx, c, []string{"codex", "--version", "ignored"}); err == nil || !strings.Contains(err.Error(), "only to managed-service") {
		t.Fatalf("harness silently ignored managed-service release identity: %v", err)
	}
}
