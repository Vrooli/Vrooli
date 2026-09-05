// Package manifest is the CLI's manifest command surface. Mirrors the
// API's Connect-RPC ManifestService. Command surface loads from
// cli/manifest.json via cliapp.LoadFromManifest.
package manifest

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "manifest"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ManifestService.ListManifests":  h.list,
		"ManifestService.GetManifest":    h.get,
		"ManifestService.UpsertManifest": h.upsert,
		"ManifestService.ClearStale":     h.clearStale,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("manifest: load from manifest: %w", err)
	}
	return group, nil
}
