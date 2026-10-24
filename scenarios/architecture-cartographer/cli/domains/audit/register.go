// Package audit is the CLI's audit-domain command surface. One
// command — `arch-cart audit run <scenario>` — orchestrates a
// CI-shaped drift audit and maps the response outcome to a process
// exit code (0 clean / 1 findings ≥ threshold / 2 tool error /
// 3 usage error).
package audit

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "audit"

// Register builds the audit subcommand group from the embedded CLI
// manifest and wires the single AuditService.Run binding.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AuditService.Run":    h.run,
		"AuditService.RunAll": h.runAll,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("audit: load from manifest: %w", err)
	}
	return group, nil
}
