// Package adapters is the CLI's adapters-domain command surface. Mirrors the
// API's Connect-RPC AdaptersService: the read, enable/disable, install, remove,
// guided-import, and compatibility verbs over the declarative conditioning-
// adapter catalog (LoRA / ControlNet / IP-Adapter).
//
// `adapters compatible` lists adapters compatible with a base model (by
// architecture); `adapters enable` toggles the persisted overlay over the
// read-only seed catalog.
package adapters

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "adapters"

// Register builds the adapters subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AdaptersService.ListAdapters":           h.list,
		"AdaptersService.GetAdapter":             h.get,
		"AdaptersService.SetAdapterEnabled":      h.setEnabled,
		"AdaptersService.InstallAdapter":         h.install,
		"AdaptersService.RemoveAdapter":          h.remove,
		"AdaptersService.InspectAdapterSource":   h.inspect,
		"AdaptersService.ImportAdapter":          h.importAdapter,
		"AdaptersService.ListCompatibleAdapters": h.compatible,
		"AdaptersService.DoctorCatalog":          h.doctor,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("adapters: load from manifest: %w", err)
	}
	return group, nil
}
