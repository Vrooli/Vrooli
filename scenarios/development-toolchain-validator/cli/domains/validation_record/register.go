// Package validation_record is the CLI's record command surface. Mirrors
// the API's Connect-RPC ValidationRecordService. Command surface loads
// from cli/manifest.json via cliapp.LoadFromManifest.
package validation_record

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "record"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ValidationRecordService.ListRecords": h.list,
		"ValidationRecordService.GetRecord":   h.get,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("validation_record: load from manifest: %w", err)
	}
	return group, nil
}
