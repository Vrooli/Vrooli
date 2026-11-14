package evals

import (
	"os"
	"path/filepath"
	"testing"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"

	"github.com/vrooli/cli-core/cliapp"
)

// TestEvalsManifestCoversEvalService asserts every RPC on EvalService has a
// bound command in cli/manifest.json — the CLI-side parity guard mirroring the
// API's TestProtoConnectParity. Adding an RPC to eval.proto without a CLI
// command (or vice versa) fails here.
func TestEvalsManifestCoversEvalService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, evalv1.File_search_hub_v1_eval_eval_proto, "EvalService")
}
