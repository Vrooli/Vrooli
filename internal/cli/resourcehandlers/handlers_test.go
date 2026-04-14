package resourcehandlers

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
)

type testContext struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func TestRootHandlerRendersHelpWithNoArgs(t *testing.T) {
	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	handler := RootHandler(HandlerDeps[testContext]{
		Stdout:  func(ctx testContext) io.Writer { return ctx.stdout },
		Stderr:  func(ctx testContext) io.Writer { return ctx.stderr },
		Globals: func(testContext) rootcli.GlobalOptions { return rootcli.GlobalOptions{} },
		OutputFormat: func(testContext) (cliout.Format, error) {
			return cliout.FormatHuman, nil
		},
	})

	if err := handler(ctx, nil); err != nil {
		t.Fatalf("RootHandler() error = %v", err)
	}
	if got := ctx.stdout.String(); !strings.Contains(got, "vrooli resource") {
		t.Fatalf("RootHandler() help missing resource usage: %q", got)
	}
}
