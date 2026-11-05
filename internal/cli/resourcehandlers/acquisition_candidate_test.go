package resourcehandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/resources"
)

func TestComposeCandidateActualHandlerHelpAndArguments(t *testing.T) {
	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := rootcli.HandlerDeps[testContext]{Stdout: func(c testContext) io.Writer { return c.stdout }}
	handler := buildResourceAcquisitionCommandHandlers(deps)["compose-candidate"]
	if handler == nil {
		t.Fatal("not registered")
	}
	if err := handler(ctx, nil, []string{"--help"}); err != nil || !bytes.Contains(ctx.stdout.Bytes(), []byte("--out")) {
		t.Fatal("leaf help failed", err)
	}
	for _, args := range [][]string{{}, {"fixture"}, {"fixture", "--out"}, {"fixture", "--bogus"}} {
		if err := handler(ctx, nil, args); err == nil {
			t.Fatal("invalid arguments accepted", args)
		}
	}
}

func TestComposeCandidateActualHandlerContextAndTruthfulReceipt(t *testing.T) {
	old := composeCandidateRun
	t.Cleanup(func() { composeCandidateRun = old })
	called := 0
	composeCandidateRun = func(ctx context.Context, c *resources.Controller, name, out string) (resources.ComposeCandidateReceipt, error) {
		called++
		if name != "fixture" || out != "new-candidate" {
			t.Fatal(name, out)
		}
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 15*time.Minute || time.Until(deadline) < 14*time.Minute {
			t.Fatal("missing bounded owner context")
		}
		if err := ctx.Err(); err != nil {
			return resources.ComposeCandidateReceipt{}, err
		}
		return resources.ComposeCandidateReceipt{Status: "candidate-only", ArtifactSHA256: "observed"}, nil
	}
	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := rootcli.HandlerDeps[testContext]{Stdout: func(c testContext) io.Writer { return c.stdout }}
	if err := resourceComposeCandidateHandler(deps)(ctx, nil, []string{"fixture", "--out", "new-candidate"}); err != nil {
		t.Fatal(err)
	}
	var receipt resources.ComposeCandidateReceipt
	if err := json.Unmarshal(ctx.stdout.Bytes(), &receipt); err != nil || receipt.Status != "candidate-only" || called != 1 {
		t.Fatal("incorrect receipt", receipt, err)
	}
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	deps.OperationContext = func(testContext) context.Context { return parent }
	if err := resourceComposeCandidateHandler(deps)(ctx, nil, []string{"fixture", "--out", "new-candidate"}); err == nil || called != 1 {
		t.Fatalf("cancelled operation reached preparation: %v calls%d", err, called)
	}
}
