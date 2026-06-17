package main

import (
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// TestAPICLIParity locks in the API↔CLI contract: every Connect-RPC endpoint in
// .vrooli/endpoints.json must declare a cli_mapping whose command is registered
// in this CLI's domain tree, and every cli_commands[] entry must resolve to a
// registered command. The shared cli-core helper does the walking; this test is
// the thin per-scenario wiring that constructs the app and points the helper at
// the scenario's endpoints.json.
//
// To exempt an endpoint that genuinely has no CLI form (server stream,
// long-lived subscription), add "cli:skip" to its rest_exception note or pass
// its ID in the skipIDs map below — never weaken the assertion.
func TestAPICLIParity(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	endpointsPath := filepath.Join("..", ".vrooli", "endpoints.json")
	cliapp.AssertAPICLIParity(t, app.core, endpointsPath, map[string]struct{}{
		"validation_describe_scenarios_protos": {},
	})
}
