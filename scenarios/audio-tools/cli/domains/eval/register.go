// Package eval hosts the `audio-tools eval ...` subtree — the STT strategy
// comparison harness. The command surface is declared in cli/manifest.json;
// Register binds EvalService.RunEval to the handler in handlers.go.
package eval

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "eval"

// Register builds the eval subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"EvalService.RunEval": h.run,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("eval: load from manifest: %w", err)
	}
	return group, nil
}
