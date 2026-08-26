package credentialshandlers

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

type testContext struct {
	stdin, stdout, stderr *bytes.Buffer
}

func TestRootHandlerAdaptsAContextWithoutOwningCredentialLogic(t *testing.T) {
	ctx := &testContext{stdin: &bytes.Buffer{}, stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	handler := RootHandler(HandlerDeps[*testContext]{
		Root:    func(*testContext) string { return "" },
		Globals: func(*testContext) rootcli.GlobalOptions { return rootcli.GlobalOptions{} },
		Stdin:   func(c *testContext) io.Reader { return c.stdin },
		Stdout:  func(c *testContext) io.Writer { return c.stdout },
		Stderr:  func(c *testContext) io.Writer { return c.stderr },
	})
	if err := handler(ctx, []string{"--help"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ctx.stdout.String(), "vrooli credentials doctor") {
		t.Fatalf("help = %q", ctx.stdout.String())
	}
}
