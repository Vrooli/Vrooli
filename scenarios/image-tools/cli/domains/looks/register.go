// Package looks is the CLI's looks-domain command surface. Mirrors the API's
// Connect-RPC LooksService: browse + manage the Look/Style library (reusable
// transformation recipes) and resolve a Look into concrete request shapes.
//
//	looks list [--kind film]      — browse the library (built-in + custom)
//	looks get <id>                — show one Look
//	looks create --file look.json — register a custom Look (protojson)
//	looks update --file look.json — replace a custom Look
//	looks delete <id>             — remove a custom Look
//	looks compile <id> --subject  — resolve a Look into op/AI request shapes
//	looks render-preview <id>     — render the Look's preview thumbnail
package looks

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "looks"

// Register builds the looks subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"LooksService.ListLooks":     h.list,
		"LooksService.GetLook":       h.get,
		"LooksService.CreateLook":    h.create,
		"LooksService.UpdateLook":    h.update,
		"LooksService.DeleteLook":    h.del,
		"LooksService.CompileLook":   h.compile,
		"LooksService.RenderPreview": h.renderPreview,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("looks: load from manifest: %w", err)
	}
	return group, nil
}
