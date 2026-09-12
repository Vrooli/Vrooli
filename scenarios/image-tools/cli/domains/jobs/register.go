// Package jobs is the CLI's jobs-domain command surface. Mirrors the API's
// Connect-RPC JobsService — the read + control verbs over server-owned durable
// async jobs (get / wait / list / cancel) — plus the `watch` server-stream
// exception appended outside the manifest.
//
// `jobs wait` is the canonical block-once verb: it blocks server-side until the
// job is terminal and returns it. Never poll `jobs get` in a loop.
package jobs

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "jobs"

// Register builds the jobs subcommand group from the embedded manifest and wires
// Connect-RPC bindings to handlers. WatchJob is a server-streaming RPC, so it is
// appended directly (cli-manifest/v1 binds unary calls only) and is documented
// in the manifest's `omitted` array.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"JobsService.GetJob":    h.get,
		"JobsService.WaitJob":   h.wait,
		"JobsService.ListJobs":  h.list,
		"JobsService.CancelJob": h.cancel,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("jobs: load from manifest: %w", err)
	}
	group.Subcommands = append(group.Subcommands, cliapp.Command{
		Name:        "watch",
		Description: "Stream a job's progress until it reaches a terminal state",
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{
				{Name: "id", Required: true, Description: "Job id"},
			},
		},
		RunCtx: h.watch,
	})
	return group, nil
}
