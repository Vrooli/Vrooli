// Package queue is the CLI's queue-domain command surface. Mirrors the API's
// Connect-RPC QueueService: read the live per-node job scheduler view (which
// jobs are running vs queued, per node). The manifest (cli/manifest.json) is the
// single source of truth for the command shape; handlers.go binds the RPCs.
package queue

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "queue"

// Register builds the queue subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"QueueService.ListQueue": h.list,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("queue: load from manifest: %w", err)
	}
	return group, nil
}
