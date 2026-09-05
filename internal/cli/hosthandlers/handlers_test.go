package hosthandlers

import (
	"bytes"
	"io"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

type testContext struct {
	out bytes.Buffer
}

func TestRootHandlerDelegatesHelp(t *testing.T) {
	ctx := &testContext{}
	handler := RootHandler(HandlerDeps[*testContext]{
		Root:    func(*testContext) string { return "." },
		Globals: func(*testContext) rootcli.GlobalOptions { return rootcli.GlobalOptions{} },
		Stdout:  func(ctx *testContext) io.Writer { return &ctx.out },
		Stderr:  func(ctx *testContext) io.Writer { return &ctx.out },
	})
	if err := handler(ctx, []string{"--help"}); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if ctx.out.Len() == 0 {
		t.Fatal("host handler should write help")
	}
}
