package resourcehandlers

import (
	"bytes"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"io"
	"testing"
)

func TestDataGroupHandlerRequiresExplicitGroupAndValidArguments(t *testing.T) {
	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := rootcli.HandlerDeps[testContext]{Stdout: func(c testContext) io.Writer { return c.stdout }}
	handler := buildResourceAcquisitionCommandHandlers(deps)["data"]
	if handler == nil {
		t.Fatal("missing handler")
	}
	if err := handler(ctx, nil, []string{"--help"}); err != nil || !bytes.Contains(ctx.stdout.Bytes(), []byte("--group")) {
		t.Fatal("missing help", err)
	}
	for _, args := range [][]string{nil, {"fixture"}, {"fixture", "--group"}, {"fixture", "--unknown"}} {
		if err := handler(ctx, nil, args); err == nil {
			t.Fatal("invalid acquisition accepted", args)
		}
	}
}
