package resourcehandlers

import (
	"bytes"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestModelCacheHandlerRejectsInvalidMutationsBeforeOwner(t *testing.T) {
	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := rootcli.HandlerDeps[testContext]{Stdout: func(c testContext) io.Writer { return c.stdout }}
	handler := buildResourceAcquisitionCommandHandlers(deps)["model-cache"]
	if handler == nil {
		t.Fatal("missing command")
	}
	if err := handler(ctx, nil, []string{"--help"}); err != nil || !bytes.Contains(ctx.stdout.Bytes(), []byte("--apply-preview")) {
		t.Fatal("help", err)
	}
	oversized := filepath.Join(t.TempDir(), "preview.json")
	if err := os.WriteFile(oversized, bytes.Repeat([]byte(" "), 65537), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{}, {"fixture", "--pin", "maybe"}, {"fixture", "--unknown"}, {"fixture", "--apply-preview", oversized}} {
		if err := handler(ctx, nil, args); err == nil {
			t.Fatal("invalid mutation accepted", args)
		}
	}
}
