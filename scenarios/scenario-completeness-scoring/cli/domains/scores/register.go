// Package scores is the CLI's scoring-domain command surface. Mirrors the
// API's Connect-RPC ScoreService: `score get <scenario> [--json]` renders
// the fast cached status payload (maturity rung as of tree digest, 0-100
// composite with breakdown, recommendations with point impact, action plan,
// per-phase freshness verdicts with a copy-pastable refresh command).
package scores

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "score"

// Register builds the score subcommand group from the embedded manifest and
// wires the Connect-RPC binding to the handler.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ScoreService.GetScore": h.get,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("score: load from manifest: %w", err)
	}
	return group, nil
}
