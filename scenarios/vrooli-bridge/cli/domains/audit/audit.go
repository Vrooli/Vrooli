// Package audit is the CLI's audit-domain command surface. Mirrors the API's
// Connect-RPC AuditService read verb: list the append-only accountability trail.
// The manifest (cli/manifest.json) is the single source of truth for the
// command shape; handlers.go binds the RPC.
package audit

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "audit"

// Register builds the audit subcommand group from the embedded manifest and
// wires the Connect-RPC binding to the handler in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AuditService.ListAuditRecords": h.list,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("audit: load from manifest: %w", err)
	}
	return group, nil
}
