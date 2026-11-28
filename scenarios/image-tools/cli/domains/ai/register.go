package ai

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// Register builds the ai subcommand group. `list` is bound from the manifest to
// the AIService.ListAIOperations Connect RPC; the per-operation submit commands
// drive the REST multipart edge and are appended directly from the generated
// table (cli-manifest binds only Connect calls, so these cannot be declared there).
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
		{Name: "max-wait", Description: "Longest to wait for local capacity before re-routing to the cloud, e.g. 90s or 10m (0 = wait locally without limit; default: deployment, 10m). Needs the cloud to be permitted and within the spend caps"},
		{Name: "max-cost", Description: "Per-job cloud spend cap in USD, e.g. 0.25 (0 = no cloud spend for this job; default: deployment, $0.25). The daily cap is set by the deployment"},
		{Name: "edit-style", Description: "Cloud role choice for edits: identity (keep the subject recognisable) or stylized (restyle it). Default: detected from the prompt"},
		{Name: "no-preserve-identity", Bool: true, Description: "Send the edit instruction exactly as written (by default edit_instruct adds an identity-preservation sentence; the effective prompt is in the job meta)"},
		{Name: "priority", Description: "Capacity priority: service, interactive, or batch"},
		{Name: "no-reclaim", Bool: true, Description: "Do not reclaim cold-idle local resources for this job"},
		{Name: "require-gpu", Bool: true, Description: "Require local GPU execution; fail if capacity would degrade this job to CPU"},
		{Name: "force-cpu", Bool: true, Description: "Force a declared CPU model/backend even when a GPU is available"},
		{Name: "no-auto-acquire", Bool: true, Description: "Fail instead of downloading the selected model's weights when they are not installed"},
		{Name: "explain", Bool: true, Description: "Print which model/technique would run (native-vs-derived) and exit without submitting"},
		{Name: "consent", Bool: true, Description: "Affirm you have the right to edit the people in these images. Needed only when the request would leave this machine (cloud provider or remote node) or the deployment is public; local runs never need it. --explain says whether it applies"},
		{Name: "lora", Description: "Conditioning LoRA to apply: id[:scale] (repeatable)"},
		{Name: "controlnet", Description: "Conditioning ControlNet: id[:scale[:conditioning_image_key]] (repeatable)"},
		{Name: "pose-control", Bool: true, Description: "Use the input image's detected pose with controlnet-openpose-sd15 (requires that adapter to be enabled and installed)"},
		{Name: "accept-openrail-m", Bool: true, Description: "Affirm acceptance of CreativeML OpenRAIL-M v2022-08-22 use restrictions and pass-through duties for the OpenPose adapter; read adapters terms openrail-m first"},
		{Name: "ip-adapter", Description: "Conditioning IP-Adapter: id[:scale]:reference_image_key (repeatable)"},
		{Name: "op-params", Description: "Operation-specific parameters as JSON, validated against the operation's capability schema (e.g. '{\"grow_px\":12}')"},
	}
}

// submitSpec is one generated submit command definition. The data comes from
// commands.generated.go (rendered from the capability manifests); the command
// construction and flag definitions live below.
type submitSpec struct {
	Verb       string
	Op         string
	Desc       string
	NeedsInput bool
	NeedsMask  bool
	// NeedsReference marks an op that requires a reference image (the source face
	// of face_swap), supplied with --reference. An op that takes one as an option
	// declares the flag without this.
	NeedsReference bool
	Flags          []string
}

// submitFlagDefs is the flag vocabulary referenced by manifests. A manifest
// declares flag names; this table owns their definitions. Descriptions stay
// neutral where one flag serves several operations.
var submitFlagDefs = map[string]cliapp.Flag{
	"prompt":                   {Name: "prompt", Description: "Prompt or instruction text"},
	"negative":                 {Name: "negative", Description: "Negative prompt"},
	"seed":                     {Name: "seed", Description: "RNG seed (0 = random)"},
	"width":                    {Name: "width", Description: "Output width in px"},
	"height":                   {Name: "height", Description: "Output height in px"},
	"steps":                    {Name: "steps", Description: "Sampler steps"},
	"cfg-scale":                {Name: "cfg-scale", Description: "Prompt-adherence strength"},
	"variations":               {Name: "variations", Description: "Number of outputs to produce"},
	"auto-scan":                {Name: "auto-scan", Bool: true, Description: "Run an NSFW scan on the output"},
	"strength":                 {Name: "strength", Description: "Denoising/source-guidance strength 0..1 (higher preserves more of the source for instruction edits)"},
	"mask":                     {Name: "mask", Description: "Path to a mask image (white = edit region)"},
	"crop-to-mask":             {Name: "crop-to-mask", Bool: true, Description: "Crop inference to the white mask bounds with surrounding context"},
	"crop-context-pixels":      {Name: "crop-context-pixels", Description: "Source-pixel context around the mask crop (0..512)"},
	"crop-max-edge-px":         {Name: "crop-max-edge-px", Description: "Inference crop maximum edge in pixels (512 or 1024)"},
	"neutralize-masked-pixels": {Name: "neutralize-masked-pixels", Bool: true, Description: "Replace masked init-image pixels with nearest unmasked context before inpaint"},
	"mask-expansion":           {Name: "mask-expansion", Description: "Explicitly expand the removal mask by 0..64 source pixels"},
	"remove-region":            {Name: "remove-region", Description: "Additional normalized removal box JSON {\"x\":0.2,\"y\":0.7,\"width\":0.1,\"height\":0.05}; repeatable"},
	"scale":                    {Name: "scale", Description: "Upscale factor (2 or 4)"},
	"support-roi":              {Name: "support-roi", Description: "Optional normalized support-plane region JSON {\"supportRoi\":{\"x\":0,\"y\":0.85,\"width\":1,\"height\":0.15}}"},
	"mesh-resolution":          {Name: "mesh-resolution", Description: "Marching-cubes resolution: 128 or 256 (default 256)"},
	"selection":                {Name: "selection", Description: "JSON normalized points and/or box, e.g. {\"points\":[{\"x\":0.7,\"y\":0.5}]}"},
	"text":                     {Name: "text", Description: "Select by phrase, e.g. \"the jacket\" or \"her hair\" (a grounded model finds objects, a SegFormer parser finds hair and garments; cannot combine with --selection)"},
	"realism":                  {Name: "realism", Description: "Fidelity↔realism knob 0..1 (default 0.5)"},
	"face-aware":               {Name: "face-aware", Bool: true, Description: "Bias texture/grain toward midtone (skin) regions"},
	"reference":                {Name: "reference", Description: "Path to the reference image: the face to put on the input image"},
	"target-face":              {Name: "target-face", Description: "Which face of the input image to replace, largest first from 0 (default 0)"},
	"source-face":              {Name: "source-face", Description: "Which face of the reference image to use, largest first from 0 (default 0)"},
	"all-faces":                {Name: "all-faces", Bool: true, Description: "Replace every detected face of the input image, not only one"},
	"color-match":              {Name: "color-match", Description: "How strongly the swapped face takes the colour of the face it replaces, 0..1 (default 1)"},
	"extent":                   {Name: "extent", Description: "How much of the person to carry over: face, face-skin, full-look or whole-head (default face)"},
	"match":                    {Name: "match", Description: "Comma-separated stages to add to the swap: skin, hair-style, hair-colour (each independent); whole-head is the head extent"},
	"skin-target":              {Name: "skin-target", Description: "Which way the skin stage matches tone: source (the reference person's skin, default) or body (the target's own visible hands and arms)"},
	"face-strength":            {Name: "face-strength", Description: "How much of the source face the swap takes, 0..1 (1 = full swap, 0 = the target's own face; default 1)"},
	"luminance":                {Name: "luminance", Description: "Share of the reference's lightness offset applied, 0..1 (default 0.5)"},
	"flatten":                  {Name: "flatten", Description: "How far the region's colour variation may shrink toward the reference's, 0..1 (default 0.4)"},
	"feather-px":               {Name: "feather-px", Description: "Edge softening in pixels, 0..64 (colour transfer default 4, mask composite 5)"},
	"grow-px":                  {Name: "grow-px", Description: "How far to widen the mask before blending, in pixels, 0..64 (default 6)"},
	"grid":                     {Name: "grid", Description: "Pin the chart layout as ROWSxCOLS, e.g. 3x3 (default: find each figure from the page)"},
	"gap":                      {Name: "gap", Description: "How far apart two drawings must be to be separate cells, as a share of the page's longer side, 0.005..0.2 (default 0.02)"},
	"padding":                  {Name: "padding", Description: "Margin around each cell's drawing, as a share of its size, 0..0.3 (default 0.03)"},
	"box":                      {Name: "box", Description: "Read this one person only: x0,y0,x1,y1 in pixels (default: find the people; a drawing with no detectable person is read as one figure)"},
	"prepare":                  {Name: "prepare", Description: "How the image is made readable first: auto (fill in a line drawing, leave a photo alone), none or line-art"},
	"pose-cell":                {Name: "pose-cell", Description: "The pose chart cell to use, numbered in reading order (see `ai pose-chart`); default: the whole image is the pose"},
	"pose-grid":                {Name: "pose-grid", Description: "Pin the chart layout the cell number refers to as ROWSxCOLS, e.g. 3x3 (default: find each figure from the page)"},
	"pose-prepare":             {Name: "pose-prepare", Description: "How the pose image is made readable first: auto (fill in a line drawing, leave a photo alone), none or line-art"},
	"pose-strength":            {Name: "pose-strength", Description: "How strictly the picture follows the pose, 0.1..2 (default 1)"},
	"character-strength":       {Name: "character-strength", Description: "How strongly the --reference picture steers the character, 0.05..1; higher keeps more of it and starts copying its pose"},
	"max-people":               {Name: "max-people", Description: "The most people to read when finding them, 1..16 (default 4)"},
	"no-occlusion":             {Name: "no-occlusion", Bool: true, Description: "Do not protect hands, hair and glasses in front of the face (skip the occlusion mask)"},
}

// submitCommands builds the submit command table from the generated specs. All
// per-operation knowledge (verb, description, I/O shape, flag names) comes from
// the capability manifests; only flag definitions and the submit handler are
// hand-written.
func (h *handlers) submitCommands() []cliapp.Command {
	common := commonFlags()
	out := make([]cliapp.Command, 0, len(generatedSubmitSpecs))
	for _, s := range generatedSubmitSpecs {
		flags := make([]cliapp.Flag, 0, len(s.Flags)+len(common))
		for _, name := range s.Flags {
			def, ok := submitFlagDefs[name]
			if !ok {
				panic(fmt.Sprintf("ai: manifest declares unknown flag %q for operation %s", name, s.Op))
			}
			flags = append(flags, def)
		}
		flags = append(flags, common...)
		var positionals []cliapp.Positional
		if s.NeedsInput {
			positionals = append(positionals, inputArg())
		}
		out = append(out, cliapp.Command{
			Name:        s.Verb,
			Description: s.Desc,
			NeedsAPI:    true,
			Args:        cliapp.ArgSchema{Positionals: positionals, Flags: flags},
			RunCtx:      h.submit(s.Op, s.NeedsInput, s.NeedsMask, s.NeedsReference),
		})
	}
	return out
}
