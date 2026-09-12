package diff

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// Register builds the diff subcommand group. `modes` is bound from the manifest
// to the DiffService.ListDiffModes Connect RPC; the `compare` command drives the
// REST multipart compare edge and is appended directly (it is not a Connect
// procedure, so it needs no manifest binding/omission).
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"DiffService.ListDiffModes": h.modes,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("diff: load from manifest: %w", err)
	}
	group.Subcommands = append(group.Subcommands, h.compareCommand())
	return group, nil
}

func (h *handlers) compareCommand() cliapp.Command {
	return cliapp.Command{
		Name:        "compare",
		Description: "Compare two images (pixel + perceptual diff) and write a heat-map",
		NeedsAPI:    true,
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{
				{Name: "base", Required: true, Description: "Path to the base (reference) image"},
				{Name: "compare", Required: true, Description: "Path to the image to compare against the base"},
			},
			Flags: []cliapp.Flag{
				{Name: "mode", Description: "Comparison mode: pixel | perceptual (default: pixel)"},
				{Name: "tolerance", Description: "Per-channel pixel-difference threshold 0..1 (pixel mode; default 0 = exact)"},
				{Name: "highlight", Description: "Heat-map highlight colour #rrggbb (default magenta)"},
				{Name: "no-heatmap", Description: "Skip heat-map generation (metrics only)", Bool: true},
				{Name: "out", Description: "Path to write the produced heat-map PNG"},
			},
		},
		RunCtx: h.compare,
	}
}
