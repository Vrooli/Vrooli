// Package audit is the CLI's audit-domain command surface. It mirrors the API's
// AuditService: orchestrate a profile-mode perf capture of a scenario.
package audit

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "audit"

// Register builds the audit subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AuditService.RunAudit": h.run,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("audit: load from manifest: %w", err)
	}
	return group, nil
}
