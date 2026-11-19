package ops

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// Register builds the ops subcommand group. `list` is bound from the manifest
// to the OpsService.ListOperations Connect RPC; the per-operation run commands
// drive the REST multipart edge and are appended from the generated table
// (cli-manifest binds only Connect calls, so these cannot be declared there).
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"OpsService.ListOperations": h.list,
		"import-link":               h.importLink,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("ops: load from manifest: %w", err)
	}
	group.Subcommands = append(group.Subcommands, h.runCommands()...)
	return group, nil
}

// inputArg is the shared required input positional.
func inputArg() cliapp.Positional {
	return cliapp.Positional{Name: "input", Required: true, Description: "Path to the input image"}
}

func outFlag() cliapp.Flag {
	return cliapp.Flag{Name: "out", Description: "Path to write the result (required for image outputs)"}
}

// runCommands returns the per-operation REST run commands. Shape comes from the
// generated table (rendered from OpParams); this function only turns specs into
// commands, so a new deterministic operation or parameter needs no edit here.
func (h *handlers) runCommands() []cliapp.Command {
	commands := make([]cliapp.Command, 0, len(generatedRunSpecs)+2)
	for _, spec := range generatedRunSpecs {
		flags := make([]cliapp.Flag, 0, len(spec.Flags)+2)
		flags = append(flags, outFlag())
		for _, f := range spec.Flags {
			if _, ok := flagDescriptions[f.Name]; !ok {
				panic(fmt.Sprintf("ops: generated flag %q for operation %s has no description", f.Name, spec.Op))
			}
			flags = append(flags, flagConnector(f))
		}
		// The overlay operation takes its watermark as a second multipart part,
		// so the flag is not a wire field and cannot come from the descriptor.
		if spec.Op == "overlay" {
			flags = append(flags, cliapp.Flag{Name: "overlay", Description: "Path to an image watermark"})
		}
		commands = append(commands, cliapp.Command{
			Name:        spec.Op,
			Description: spec.Desc,
			NeedsAPI:    true,
			Args:        cliapp.ArgSchema{Positionals: []cliapp.Positional{inputArg()}, Flags: flags},
			RunCtx:      h.run(spec),
		})
	}
	// The layered views carry multipart plate stacks and both cameras, so they
	// are built by hand rather than from scalar flags.
	return append(commands,
		h.layeredCommand("composite", "Compose prepared depth layers into one PNG view"),
		h.layeredCommand("parallax", "Render prepared layers into a lossless parallax PNG sequence ZIP"),
	)
}
