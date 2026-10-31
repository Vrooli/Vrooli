package ai

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai/ai_v1connect"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/selection"
)

type handlers struct {
	core *cliapp.ScenarioApp
}

func newHandlers(core *cliapp.ScenarioApp) *handlers { return &handlers{core: core} }

// list mirrors AIService.ListAIOperations (the Connect discovery RPC).
func (h *handlers) list(ctx cliapp.RunContext) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := aiconnect.NewAIServiceClient(httpClient, baseURL)
	resp, err := client.ListAIOperations(context.Background(), connect.NewRequest(&aiv1.ListAIOperationsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list ai operations", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Operations))
	for _, op := range resp.Msg.Operations {
		results = append(results, fmt.Sprintf("%-18s [%s] %s (default: %s)", op.Name, op.Category, op.Summary, op.DefaultModelId))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d AI operations.", len(resp.Msg.Operations))},
		ResultsHeading: "Operations",
		Results:        results,
	})
}

// submit returns a RunCtx handler for one AI op. inputField/maskField name the
// positional/flag the op reads (empty when the op needs none).
func (h *handlers) submit(operation string, needsInput, needsMask bool) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		// --explain is a read-only dry-run: resolve which model/technique would run
		// and exit 0 without submitting (or requiring an input image/mask).
		if boolOr(ctx, "explain") {
			return h.explainResolution(ctx, operation)
		}
		input := ""
		if needsInput {
			input = ctx.Positional("input")
			if input == "" {
				return fmt.Errorf("an input image path is required")
			}
		}
		mask := ""
		if needsMask || (operation == "image_to_object" && ctx.FlagDeclared("mask")) {
			mask = ctx.Flag("mask")
			if needsMask && mask == "" {
				return fmt.Errorf("--mask is required for %s", operation)
			}
		}
		params := buildParams(ctx)
		if operation == "inpaint" {
			contextRaw, edgeRaw := flagOr(ctx, "crop-context-pixels"), flagOr(ctx, "crop-max-edge-px")
			neutralize := boolOr(ctx, "neutralize-masked-pixels")
			if boolOr(ctx, "crop-to-mask") || contextRaw != "" || edgeRaw != "" || neutralize {
				if !boolOr(ctx, "crop-to-mask") {
					return fmt.Errorf("inpaint crop controls and --neutralize-masked-pixels require --crop-to-mask")
				}
				crop := &aiv1.InpaintParams{CropToMask: true, NeutralizeMaskedPixels: neutralize}
				if contextRaw != "" {
					value, err := strconv.ParseUint(contextRaw, 10, 32)
					if err != nil || value > 512 {
						return fmt.Errorf("invalid --crop-context-pixels %q: expected 0..512", contextRaw)
					}
					crop.ContextPixels = uint32(value)
				}
				if edgeRaw != "" {
					value, err := strconv.ParseUint(edgeRaw, 10, 32)
					if err != nil || (value != 512 && value != 1024) {
						return fmt.Errorf("invalid --crop-max-edge-px %q: expected 512 or 1024", edgeRaw)
					}
					crop.MaxEdgePx = uint32(value)
				}
				params.Inpaint = crop
			}
		}
		var removalRegions []string
		if ctx.FlagDeclared("remove-region") {
			removalRegions = ctx.FlagValues("remove-region")
		}
		if raw := flagOr(ctx, "mask-expansion"); raw != "" || len(removalRegions) > 0 {
			expansion := uint64(0)
			if raw != "" {
				parsed, err := strconv.ParseUint(raw, 10, 32)
				if err != nil || parsed > 64 {
					return fmt.Errorf("invalid --mask-expansion %q: expected 0..64", raw)
				}
				expansion = parsed
			}
			params.BackgroundRepair = &aiv1.BackgroundRepairParams{MaskExpansionPixels: uint32(expansion)}
			for _, raw := range removalRegions {
				box := &selectionv1.Box{}
				if err := protojson.Unmarshal([]byte(raw), box); err != nil {
					return fmt.Errorf("invalid --remove-region: %w", err)
				}
				params.BackgroundRepair.RemovalRegions = append(params.BackgroundRepair.RemovalRegions, box)
			}
		}
		if raw := flagOr(ctx, "selection"); raw != "" {
			params.Segmentation = &aiv1.SegmentationPrompt{}
			if err := protojson.Unmarshal([]byte(raw), params.Segmentation); err != nil {
				return fmt.Errorf("invalid --selection: %w", err)
			}
		}
		if raw := flagOr(ctx, "support-roi"); raw != "" {
			params.SceneGeometry = &aiv1.SceneGeometryParams{}
			if err := protojson.Unmarshal([]byte(raw), params.SceneGeometry); err != nil {
				return fmt.Errorf("invalid --support-roi: %w", err)
			}
		}
		if raw := flagOr(ctx, "mesh-resolution"); raw != "" {
			resolution, err := strconv.ParseUint(raw, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid --mesh-resolution %q: expected 128 or 256", raw)
			}
			params.ImageToObject = &aiv1.ImageToObjectParams{MeshResolution: uint32(resolution)}
		}
		resp, err := submitAI(h.core, operation, input, mask, params, flagOr(ctx, "operation-id"))
		if err != nil {
			return err
		}

		out := flagOr(ctx, "out")
		wait := boolOr(ctx, "wait")
		if !wait {
			// --json emits the typed SubmitAIResponse so a caller can read the
			// job id, ETA, model, tier and warnings without parsing prose.
			if ctx.JSON() {
				return cliapp.PrintProtoJSON(ctx.Stdout(), resp)
			}
			return ctx.RenderMutation(cliapp.MutationReport{
				Result: []string{fmt.Sprintf("Submitted %s as job %s (~%ds) on %s/%s", operation, resp.JobId, resp.EstimatedSeconds, resp.ModelId, resp.Tier)},
				Changes: append(warnLines(resp.Warnings),
					fmt.Sprintf("wait: image-tools jobs wait %s", resp.JobId)),
			})
		}
		job, written, werr := waitAndDownload(h.core, resp.JobId, out)
		if werr != nil {
			return werr
		}
		warnings := resp.Warnings
		if written != "" && written != out {
			warnings = append(append([]string(nil), warnings...),
				fmt.Sprintf("the result's format does not match %s; wrote %s instead", out, written))
		}
		if ctx.JSON() {
			return cliapp.PrintJSON(ctx.Stdout(), struct {
				JobId      string   `json:"job_id"`
				State      string   `json:"state"`
				OutputPath string   `json:"output_path,omitempty"`
				ResultRefs []string `json:"result_refs,omitempty"`
				Warnings   []string `json:"warnings,omitempty"`
			}{
				JobId:      job.GetId(),
				State:      stateName(job.GetState()),
				OutputPath: written,
				ResultRefs: job.GetResultRefs(),
				Warnings:   warnings,
			})
		}
		changes := warnLines(warnings)
		if written != "" {
			changes = append(changes, fmt.Sprintf("wrote %s (job=%s)", written, job.GetId()))
		}
		return ctx.RenderMutation(cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("%s %s on %s/%s", operation, stateName(job.GetState()), resp.ModelId, resp.Tier)},
			Changes: changes,
		})
	}
}

// explainResolution renders the read-only resolution for an op (the `--explain`
// dry-run): which model/technique would run, native-vs-derived, tier, safety
// weight — without submitting a job.
func (h *handlers) explainResolution(ctx cliapp.RunContext, operation string) error {
	r, err := explainResolution(h.core, &modelsv1.ExplainResolutionRequest{
		Operation:      operation,
		ModelId:        flagOr(ctx, "model"),
		AllowByok:      boolOr(ctx, "byok"),
		Adapters:       explainAdapterRefs(ctx),
		QualityPolicy:  flagOr(ctx, "quality-policy"),
		OpenrouterRole: flagOr(ctx, "role"),
		FallbackPolicy: flagOr(ctx, "fallback-policy"),
	})
	if err != nil {
		return err
	}
	results := []string{
		fmt.Sprintf("model: %s (%s)", r.GetModelId(), r.GetModelName()),
		"support: " + r.GetSupport(),
	}
	if r.GetTechnique() != "" {
		results = append(results, "technique: "+r.GetTechnique())
	}
	if r.GetPipelineClass() != "" {
		results = append(results, "pipeline_class: "+r.GetPipelineClass())
	}
	if r.GetTier() != "" {
		results = append(results, "tier: "+r.GetTier())
	}
	results = append(results,
		fmt.Sprintf("gpu_viable: %t", r.GetGpuViable()),
		"safety_weight: "+r.GetWeight(),
	)
	if r.GetCaveat() != "" {
		results = append(results, "caveat: "+r.GetCaveat())
	}
	for _, a := range r.GetAdapters() {
		results = append(results, fmt.Sprintf("adapter: %s (%s) scale=%.3g", a.GetId(), a.GetKind(), a.GetScale()))
	}
	for _, w := range r.GetWarnings() {
		results = append(results, "warning: "+w)
	}
	return cliapp.RenderProtoList(ctx, r, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s would run %q as a %s op (no job submitted).", r.GetModelId(), operation, r.GetSupport())},
		ResultsHeading: "Resolution",
		Results:        results,
	})
}

// buildParams assembles AIParams from whichever flags the command declared.
func buildParams(ctx cliapp.RunContext) *aiv1.AIParams {
	p := &aiv1.AIParams{
		Prompt:          flagOr(ctx, "prompt"),
		NegativePrompt:  flagOr(ctx, "negative"),
		Seed:            i64(flagOr(ctx, "seed")),
		Width:           i32(flagOr(ctx, "width")),
		Height:          i32(flagOr(ctx, "height")),
		Steps:           i32(flagOr(ctx, "steps")),
		CfgScale:        f64(flagOr(ctx, "cfg-scale")),
		Variations:      i32(flagOr(ctx, "variations")),
		Strength:        f64(flagOr(ctx, "strength")),
		Scale:           i32(flagOr(ctx, "scale")),
		Realism:         f64(flagOr(ctx, "realism")),
		FaceAware:       boolOr(ctx, "face-aware"),
		ModelOverride:   flagOr(ctx, "model"),
		AllowByok:       boolOr(ctx, "byok"),
		QualityPolicy:   flagOr(ctx, "quality-policy"),
		FallbackPolicy:  flagOr(ctx, "fallback-policy"),
		OpenrouterRole:  flagOr(ctx, "role"),
		Priority:        flagOr(ctx, "priority"),
		AllowReclaim:    optionalReclaim(ctx),
		AutoScanNsfw:    boolOr(ctx, "auto-scan"),
		ConsentAffirmed: boolOr(ctx, "consent"),
		Adapters:        adapterRefs(ctx),
	}
	return p
}

func optionalReclaim(ctx cliapp.RunContext) *bool {
	if boolOr(ctx, "no-reclaim") {
		v := false
		return &v
	}
	return nil
}

// adapterRefs parses the repeatable --lora / --controlnet / --ip-adapter flags
// into the typed conditioning stack (decision C2). Spec forms:
//
//	--lora id[:scale]
//	--controlnet id[:scale[:conditioning_image_key]]
//	--ip-adapter id[:scale]:reference_image_key
//
// The colon-delimited tail is positional: the 2nd field (when numeric) is the
// scale, and a trailing non-numeric field is the conditioning/reference image key.
// Order across flags is LoRA → ControlNet → IP-Adapter (the resolver re-sorts into
// canonical application order regardless).
type adapterSpec struct {
	id    string
	scale float64
	key   string
}

func collectAdapterSpecs(ctx cliapp.RunContext) []adapterSpec {
	var out []adapterSpec
	collect := func(flag string) {
		if !ctx.FlagDeclared(flag) {
			return
		}
		for _, spec := range ctx.FlagValues(flag) {
			spec = strings.TrimSpace(spec)
			if spec == "" {
				continue
			}
			out = append(out, parseAdapterSpec(spec))
		}
	}
	collect("lora")
	collect("controlnet")
	collect("ip-adapter")
	return out
}

func parseAdapterSpec(spec string) adapterSpec {
	parts := strings.Split(spec, ":")
	out := adapterSpec{id: strings.TrimSpace(parts[0])}
	for _, raw := range parts[1:] {
		field := strings.TrimSpace(raw)
		if field == "" {
			continue
		}
		if v, err := strconv.ParseFloat(field, 64); err == nil {
			out.scale = v
			continue
		}
		// A non-numeric tail field is the conditioning / reference image key.
		out.key = field
	}
	return out
}

func adapterRefs(ctx cliapp.RunContext) []*aiv1.AdapterRef {
	specs := collectAdapterSpecs(ctx)
	if len(specs) == 0 {
		return nil
	}
	out := make([]*aiv1.AdapterRef, 0, len(specs))
	for _, s := range specs {
		out = append(out, &aiv1.AdapterRef{AdapterId: s.id, Scale: s.scale, ConditioningImageKey: s.key})
	}
	return out
}

func explainAdapterRefs(ctx cliapp.RunContext) []*modelsv1.AdapterRef {
	specs := collectAdapterSpecs(ctx)
	if len(specs) == 0 {
		return nil
	}
	out := make([]*modelsv1.AdapterRef, 0, len(specs))
	for _, s := range specs {
		out = append(out, &modelsv1.AdapterRef{AdapterId: s.id, Scale: s.scale, ConditioningImageKey: s.key})
	}
	return out
}

func warnLines(warnings []string) []string {
	out := make([]string, 0, len(warnings))
	for _, w := range warnings {
		out = append(out, "warning: "+w)
	}
	return out
}

// flagOr returns the flag value when the command declared it, else "".
func flagOr(ctx cliapp.RunContext, name string) string {
	if ctx.FlagDeclared(name) {
		return ctx.Flag(name)
	}
	return ""
}

func boolOr(ctx cliapp.RunContext, name string) bool {
	if ctx.FlagDeclared(name) {
		return ctx.BoolFlag(name)
	}
	return false
}

func i32(s string) int32 {
	// ParseInt with bitSize=32 bounds the result to int32 range, so the
	// conversion below cannot overflow (avoids gosec G109).
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0
	}
	return int32(n)
}

func i64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func f64(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// submitStored composes native artifact refs without reuploading source bytes.
func (h *handlers) submitStored(ctx cliapp.RunContext) error {
	params := &aiv1.AIParams{}
	if raw := ctx.Flag("params"); raw != "" {
		if err := protojson.Unmarshal([]byte(raw), params); err != nil {
			return fmt.Errorf("invalid --params: %w", err)
		}
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	response, err := aiconnect.NewAIServiceClient(httpClient, baseURL).SubmitStoredAI(context.Background(), connect.NewRequest(&aiv1.SubmitStoredAIRequest{Operation: ctx.Positional("operation"), Params: params, InputRef: ctx.Flag("input-ref"), MaskRef: ctx.Flag("mask-ref"), OperationId: ctx.Flag("operation-id")}))
	if err != nil {
		return cliapp.WrapAPIError("submit stored AI operation", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, response.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Submitted %s as %s on %s/%s", ctx.Positional("operation"), response.Msg.JobId, response.Msg.ModelId, response.Msg.Tier)}, Changes: []string{"wait: image-tools jobs wait " + response.Msg.JobId}})
}
