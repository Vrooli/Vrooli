// Package components is the CLI's component-registry surface. Mirrors
// the API's Connect-RPC ComponentsService. Command surface loads from
// cli/manifest.json via cliapp.LoadFromManifest.
package components

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "components"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ComponentsService.ListComponents":          h.list,
		"ComponentsService.GetComponent":            h.get,
		"ComponentsService.IngestComponent":         h.ingest,
		"ComponentsService.BeginComponentVersion":   h.versionBegin,
		"ComponentsService.PublishComponentVersion": h.versionPublish,
		"ComponentsService.UpdateComponentContent":  h.contentSet,
		"ComponentTestsService.RunComponentTest":    h.testRun,
		"ComponentTestsService.SweepComponentTests": h.sweep,
		// Local binding: the manifest declares this command with
		// binding.kind "local" and no handler name, so the loader keys it by
		// the command name. It must be registered here like any other — a
		// manifest command with no entry in this map is a startup panic, not
		// a missing subcommand.
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("components: load from manifest: %w", err)
	}
	return group, nil
}
