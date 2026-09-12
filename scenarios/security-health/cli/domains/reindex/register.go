// Package reindex is the CLI surface for reconciling the fleet dependency
// corpus. It mirrors the API's ReindexService.
package reindex

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "reindex"

// Register builds the reindex subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ReindexService.Reindex":       h.run,
		"ReindexService.ReindexStatus": h.status,
		"ReindexService.ReindexCancel": h.cancel,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("reindex: load from manifest: %w", err)
	}
	return group, nil
}
