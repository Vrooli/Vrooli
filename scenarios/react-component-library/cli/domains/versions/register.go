// Package versions is the CLI's version-history surface. Mirrors the
// API's Connect-RPC VersionsService. Command surface loads from
// cli/manifest.json via cliapp.LoadFromManifest.
package versions

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "versions"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"VersionsService.ListVersions":   cliapp.ProtoList(h.listCall, h.listReport),
		"VersionsService.GetVersion":     cliapp.ProtoList(h.showCall, h.showReport),
		"VersionsService.DiffVersions":   cliapp.ProtoList(h.diffCall, h.diffReport),
		"VersionLifecycleService.Doctor": cliapp.ProtoList(h.doctorCall, h.doctorReport),
		"reap":                           {Run: h.reap},
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("versions: load from manifest: %w", err)
	}
	return group, nil
}
