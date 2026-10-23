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
		"AIService.SubmitStoredAI":   h.submitStored,
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
		{Name: "operation-id", Description: "Stable UUID for retrying the same operation without duplicate inference"},
		{Name: "out", Description: "Path to write the result (with --wait)"},
		{Name: "wait", Bool: true, Description: "Block once until the job finishes and download the result"},
		{Name: "model", Description: "Force a specific model id (override hardware-fit)"},
		{Name: "byok", Bool: true, Description: "Allow a paid BYOK cloud provider unless --fallback-policy=local_only"},
		{Name: "quality-policy", Description: "Candidate ordering policy: quality, balanced, or fast"},
		{Name: "fallback-policy", Description: "Provider fallback policy: any, cloud_allowed, or local_only"},
		{Name: "role", Description: "OpenRouter model role (e.g. image.vector.default, image.generate.logo) for the gateway"},
		{Name: "priority", Description: "Capacity priority: service, interactive, or batch"},
		{Name: "no-reclaim", Bool: true, Description: "Do not reclaim cold-idle local resources for this job"},
		{Name: "explain", Bool: true, Description: "Print which model/technique would run (native-vs-derived) and exit without submitting"},
		{Name: "consent", Bool: true, Description: "Affirm you have the right to edit the people in this image (required for identity-altering ops on a public deployment)"},
		{Name: "lora", Description: "Conditioning LoRA to apply: id[:scale] (repeatable)"},
		{Name: "controlnet", Description: "Conditioning ControlNet: id[:scale[:conditioning_image_key]] (repeatable)"},
		{Name: "ip-adapter", Description: "Conditioning IP-Adapter: id[:scale]:reference_image_key (repeatable)"},
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
			append(append(genFlags(), maskFlag),
				cliapp.Flag{Name: "crop-to-mask", Bool: true, Description: "Crop inference to the white mask bounds with surrounding context"},
				cliapp.Flag{Name: "crop-context-pixels", Description: "Source-pixel context around the mask crop (0..512)"},
				cliapp.Flag{Name: "crop-max-edge-px", Description: "Inference crop maximum edge in pixels (512 or 1024)"},
				cliapp.Flag{Name: "neutralize-masked-pixels", Bool: true, Description: "Replace masked init-image pixels with nearest unmasked context before inpaint"},
			)...),
		cmd("outpaint", "outpaint", "Expand an image beyond its borders, generating the new region from a prompt", true, true,
			append(genFlags(), maskFlag)...),
		cmd("object-removal", "object_removal", "Remove a masked object and fill the gap", true, true,
			maskFlag,
			cliapp.Flag{Name: "mask-expansion", Description: "Explicitly expand the removal mask by 0..64 source pixels"},
			cliapp.Flag{Name: "remove-region", Description: "Additional normalized removal box JSON {\"x\":0.2,\"y\":0.7,\"width\":0.1,\"height\":0.05}; repeatable"}),
		cmd("background-replace", "background_replace", "Replace the background behind a masked subject from a prompt", true, true,
			append(genFlags(), maskFlag)...),
		cmd("upscale", "upscale", "Super-resolve / enlarge an image", true, false,
			cliapp.Flag{Name: "scale", Description: "Upscale factor (2 or 4)"}),
		cmd("bg-removal", "background_removal", "Remove the background to transparency", true, false),
		cmd("denoise", "denoise", "Reduce noise / deblur an image", true, false),
		cmd("colorize", "colorize", "Add realistic colour to a grayscale / black-and-white image", true, false),
		cmd("depth", "depth_map", "Estimate a per-pixel depth map from a single image", true, false),
		cmd("geometry", "scene_geometry", "Estimate a camera and full floating-point scene geometry bundle", true, false,
			cliapp.Flag{Name: "support-roi", Description: "Optional normalized support-plane region JSON {\"supportRoi\":{\"x\":0,\"y\":0.85,\"width\":1,\"height\":0.15}}; fitted scale remains estimated"}),
		cmd("image-to-object", "image_to_object", "Reconstruct a 3D prop mesh from one image and optional foreground mask", true, false,
			maskFlag,
			cliapp.Flag{Name: "mesh-resolution", Description: "Marching-cubes resolution: 128 or 256 (default 256)"}),
		cmd("segment", "segment", "Segment an object with a SAM model in a durable job", true, false,
			cliapp.Flag{Name: "selection", Description: "JSON normalized points and/or box, e.g. {\"points\":[{\"x\":0.7,\"y\":0.5}]}"}),
		cmd("normal-map", "normal_map", "Convert image luminance/depth into a tangent-space normal map", true, false),
		cmd("naturalize", "naturalize", "Reintroduce realistic texture/grain to an over-smoothed (restored/upscaled) image", true, false,
			cliapp.Flag{Name: "realism", Description: "Fidelity↔realism knob 0..1 (default 0.5)"},
			cliapp.Flag{Name: "face-aware", Bool: true, Description: "Bias texture/grain toward midtone (skin) regions"}),
	}
}
