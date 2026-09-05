// Package drift is the CLI's drift-domain command surface. It binds to the
// ConfigService RPCs that expose and reconcile ingress ownership drift —
// GetDrift / AdoptIngress / IgnoreIngress / PruneIngress — under a dedicated
// `drift` command group so operators reason about live-vs-desired ingress
// separately from base config.
//
// Follows the canonical domain shape: a Register(core, manifest) returning a
// cliapp.SubcommandGroup built from cli/manifest.json via
// cliapp.LoadFromManifest, plus one handler per Connect-RPC subcommand in
// handlers.go.
package drift

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "drift"

// Register builds the drift subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"ConfigService.GetDrift":      cliapp.ProtoList(h.listCall, h.listReport),
		"ConfigService.AdoptIngress":  cliapp.ProtoMutation(h.adoptCall, h.adoptReport),
		"ConfigService.IgnoreIngress": cliapp.ProtoMutation(h.ignoreCall, h.ignoreReport),
		"ConfigService.PruneIngress":  cliapp.ProtoMutation(h.pruneCall, h.pruneReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("drift: load from manifest: %w", err)
	}
	return group, nil
}
