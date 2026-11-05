// Package audits is the CLI's audits-domain command surface. Mirrors the API's
// Connect-RPC AuditsService. Operators use run/get/list to run generic snapshot
// audits and review prior proofs.
//
// The manifest (cli/manifest.json) is the single source of truth for the command
// shape (flags, positionals, governance, RPC bindings); this package only wires
// bindings to handlers in handlers.go.
package audits

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "audits"

// Register builds the audits subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AuditsService.RunSnapshotAudit": h.run,
		"AuditsService.GetAudit":         h.get,
		"AuditsService.ListAudits":       h.list,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("audits: load from manifest: %w", err)
	}
	return group, nil
}
