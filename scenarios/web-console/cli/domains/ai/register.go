// Package ai is the CLI's ai-domain command surface. It mirrors the API's
// Connect-RPC AIService — command generation, suggestions, provider config,
// and provider health — and is built from the embedded manifest.
package ai

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

// GroupName is the manifest group name this package owns.
const GroupName = "ai"

// Register builds the `ai` subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AIService.Generate":     h.generate,
		"AIService.Suggest":      h.suggest,
		"AIService.GetConfig":    h.configGet,
		"AIService.UpdateConfig": h.configSet,
		"AIService.GetHealth":    h.health,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("ai: load from manifest: %w", err)
	}
	// Preserve the pre-manifest subcommand alias (cli-manifest/v1 has no
	// per-command alias field).
	support.ApplyAliases(group.Subcommands, map[string][]string{
		"config-get": {"config"},
	})
	return group, nil
}
