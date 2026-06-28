// Package log is the CLI's log-domain command surface. It owns the `log`
// manifest group — record decisions/findings/bugs/records/notes, list/inspect
// entries, update/promote/sync — all backed by the API's LogService. The
// manifest (cli/manifest.json) carries the declarative command shape; handlers.go
// builds each typed request and renders the response.
package log

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "log"

// Register builds the log subcommand group from the embedded manifest and wires
// Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"LogService.AddDecision":  h.decisionAdd,
		"LogService.AddFinding":   h.findingAdd,
		"LogService.AddBug":       h.bugAdd,
		"LogService.AddRecord":    h.recordAdd,
		"LogService.AddNote":      h.noteAdd,
		"LogService.ListEntries":  h.list,
		"LogService.GetEntry":     h.get,
		"LogService.UpdateEntry":  h.update,
		"LogService.PromoteEntry": h.promote,
		"LogService.SyncEntry":    h.sync,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("log: load log group: %w", err)
	}
	return group, nil
}
