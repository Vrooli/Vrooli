package domains

import (
	"ecosystem-manager/cli/domains/discovery"
	"ecosystem-manager/cli/internal/appctx"
	"ecosystem-manager/cli/logs"
	"ecosystem-manager/cli/queue"
	"ecosystem-manager/cli/steer"
	"ecosystem-manager/cli/tasks"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates the scenario's flat (REST / token-only) domain
// command groups. These domains are NOT proto-bound; they stay code-registered
// exactly as before. The proto-bound discovery domain is built from
// cli/manifest.json via SubcommandGroups instead.
func CommandGroups(ctx appctx.Context) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		tasks.Commands(ctx)[0],
		steer.Commands(ctx),
		queue.Commands(ctx),
		logs.Commands(),
	}
}

// SubcommandGroups aggregates the scenario's hierarchical, proto-bound command
// groups. Each such domain owns a Register(core, manifest) function returning a
// SubcommandGroup built from cli/manifest.json (the CLI-surface SSOT). The
// aggregator passes the embedded manifest bytes through unchanged; per-domain
// Register implementations call cliapp.LoadFromManifest with the relevant group
// name and wire connect-rpc bindings to handlers.
//
// Only discovery is proto-bound today; the REST domains stay in CommandGroups.
func SubcommandGroups(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	discoveryGroup, err := discovery.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	return []cliapp.SubcommandGroup{discoveryGroup}, nil
}
