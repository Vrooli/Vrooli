package packagehandlers

import (
	"bytes"
	"io"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli/rootclitest"
	"github.com/vrooli/vrooli/internal/cliout"
)

type testContext struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func TestConformance(t *testing.T) {
	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	handler := RootHandler(HandlerDeps[testContext]{
		Stdout: func(ctx testContext) io.Writer { return ctx.stdout },
		Stderr: func(ctx testContext) io.Writer { return ctx.stderr },
		Root:   func(testContext) string { return "/repo" },
		OutputFormat: func(testContext) (cliout.Format, error) {
			return cliout.FormatHuman, nil
		},
	})

	rootclitest.AssertHelpWithNoArgs(t, func() error { return handler(ctx, nil) }, ctx.stdout, "vrooli package")
}
