package selection

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// Register builds the select subcommand group. `classes` and `suggest` are bound
// from the manifest to the SelectionService Connect RPCs; the `segment` command
// drives the REST multipart edge and is appended directly (documented in the
// manifest's `omitted` array).
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"SelectionService.ListRegionClasses": h.classes,
		"SelectionService.SuggestEdits":      h.suggest,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("select: load from manifest: %w", err)
	}
	group.Subcommands = append(group.Subcommands, h.segmentCommand())
	return group, nil
}

func (h *handlers) segmentCommand() cliapp.Command {
	return cliapp.Command{
		Name:        "segment",
		Description: "Smart-select a region (point/box/auto), classify it, and write the mask",
		NeedsAPI:    true,
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{{Name: "input", Required: true, Description: "Path to the input image"}},
			Flags: []cliapp.Flag{
				{Name: "mode", Description: "Segmentation mode: point | box | auto (default: inferred from --point/--box)"},
				{Name: "point", Description: "Seed point x,y in 0..1 (point mode), e.g. 0.5,0.5"},
				{Name: "box", Description: "Box x,y,w,h in 0..1 (box mode), e.g. 0.3,0.3,0.4,0.4"},
				{Name: "tolerance", Description: "Region-grow colour threshold 0..1 (default ~0.14)"},
				{Name: "model", Description: "Force a SAM model id (falls back to the built-in if unwired)"},
				{Name: "out", Description: "Path to write the produced mask PNG"},
			},
		},
		RunCtx: h.segment,
	}
}
