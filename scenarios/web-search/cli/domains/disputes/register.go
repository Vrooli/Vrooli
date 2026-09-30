// Package disputes is the CLI's dispute-review command surface. The DISPUTED
// review queue is a view over the findings store, so this group binds to the
// API's FindingsService (ListFindings filtered to DISPUTED for `disputes list`,
// ResolveDispute for `disputes resolve`). Built from cli/manifest.json via
// cliapp.LoadFromManifest with one handler per subcommand in handlers.go.
package disputes

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "disputes"

// Register builds the disputes subcommand group from the embedded manifest and
// wires each command to a handler in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"FindingsService.ListDisputes":   h.list,
		"FindingsService.ResolveDispute": h.resolve,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("disputes: load from manifest: %w", err)
	}
	return group, nil
}
