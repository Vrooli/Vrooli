package packagehandlers

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
)

type testContext struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func TestRootHandlerRendersHelpWithNoArgs(t *testing.T) {
	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	handler := RootHandler(HandlerDeps[testContext]{
		Stdout: func(ctx testContext) io.Writer { return ctx.stdout },
		Stderr: func(ctx testContext) io.Writer { return ctx.stderr },
		Root:   func(testContext) string { return "/repo" },
		OutputFormat: func(testContext) (cliout.Format, error) {
			return cliout.FormatHuman, nil
		},
	})

	if err := handler(ctx, nil); err != nil {
		t.Fatalf("RootHandler() error = %v", err)
	}
	if got := ctx.stdout.String(); !strings.Contains(got, "vrooli package") {
		t.Fatalf("RootHandler() help missing package usage: %q", got)
	}
}
