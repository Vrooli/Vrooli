package ai

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// Register builds the ai subcommand group. `list` is bound from the manifest to
// the AIService.ListAIOperations Connect RPC; the per-operation submit commands
// drive the REST multipart edge and are appended directly (documented in the
// manifest's `omitted` array).
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AIService.ListAIOperations": h.list,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("ai: load from manifest: %w", err)
	}
	group.Subcommands = append(group.Subcommands, h.submitCommands()...)
	return group, nil
}

func inputArg() cliapp.Positional {
	return cliapp.Positional{Name: "input", Required: true, Description: "Path to the input image"}
}

// commonFlags are shared by every submit command.
func commonFlags() []cliapp.Flag {
	return []cliapp.Flag{
		{Name: "out", Description: "Path to write the result (with --wait)"},
		{Name: "wait", Bool: true, Description: "Block once until the job finishes and download the result"},
		{Name: "model", Description: "Force a specific model id (override hardware-fit)"},
		{Name: "byok", Bool: true, Description: "Allow a paid BYOK cloud provider when no local backend is available"},
	}
}

func genFlags() []cliapp.Flag {
	return []cliapp.Flag{
		{Name: "prompt", Description: "Positive text prompt"},
		{Name: "negative", Description: "Negative prompt"},
		{Name: "seed", Description: "RNG seed (0 = random)"},
		{Name: "width", Description: "Output width in px"},
		{Name: "height", Description: "Output height in px"},
		{Name: "steps", Description: "Sampler steps"},
		{Name: "cfg-scale", Description: "Prompt-adherence strength"},
		{Name: "variations", Description: "Number of outputs to produce"},
		{Name: "auto-scan", Bool: true, Description: "Run an NSFW scan on the output"},
	}
}

func (h *handlers) submitCommands() []cliapp.Command {
	cmd := func(name, op, desc string, needsInput, needsMask bool, flags ...cliapp.Flag) cliapp.Command {
		positionals := []cliapp.Positional{}
		if needsInput {
			positionals = append(positionals, inputArg())
		}
		return cliapp.Command{
			Name:        name,
			Description: desc,
			NeedsAPI:    true,
			Args:        cliapp.ArgSchema{Positionals: positionals, Flags: append(append([]cliapp.Flag{}, flags...), commonFlags()...)},
			RunCtx:      h.submit(op, needsInput, needsMask),
		}
	}
	maskFlag := cliapp.Flag{Name: "mask", Description: "Path to a mask image (white = edit region)"}
	return []cliapp.Command{
		cmd("generate", "text_to_image", "Generate an image from a text prompt", false, false, genFlags()...),
		cmd("img2img", "image_to_image", "Transform an input image guided by a prompt", true, false,
			append(genFlags(), cliapp.Flag{Name: "strength", Description: "img2img denoising strength 0..1"})...),
		cmd("edit", "edit_instruct", "Edit an image from a natural-language instruction (identity-preserving)", true, false,
			append(genFlags(), cliapp.Flag{Name: "strength", Description: "image-guidance: how faithful to the source (higher = preserve more)"})...),
		cmd("inpaint", "inpaint", "Regenerate a masked region from a prompt", true, true,
			append(genFlags(), maskFlag)...),
		cmd("object-removal", "object_removal", "Remove a masked object and fill the gap", true, true, maskFlag),
		cmd("upscale", "upscale", "Super-resolve / enlarge an image", true, false,
			cliapp.Flag{Name: "scale", Description: "Upscale factor (2 or 4)"}),
		cmd("bg-removal", "background_removal", "Remove the background to transparency", true, false),
		cmd("denoise", "denoise", "Reduce noise / deblur an image", true, false),
		cmd("naturalize", "naturalize", "Reintroduce realistic texture/grain to an over-smoothed (restored/upscaled) image", true, false,
			cliapp.Flag{Name: "realism", Description: "Fidelity↔realism knob 0..1 (default 0.5)"},
			cliapp.Flag{Name: "face-aware", Bool: true, Description: "Bias texture/grain toward midtone (skin) regions"}),
	}
}
