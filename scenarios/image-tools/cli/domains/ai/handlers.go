package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"image-tools/cli/internal/jobstate"
	"image-tools/cli/internal/verdictfmt"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai/ai_v1connect"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
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
		results = append(results, formatOperation(op))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d AI operations.", len(resp.Msg.Operations))},
		ResultsHeading: "Operations",
		Results:        results,
		RetrievalHints: []string{
			"the default is the model a submit would run now (`models select <op>` agrees); the label after it is its verdict",
			"`models get <id>` — see the evidence behind a verdict",
		},
	})
}

// formatOperation renders one discovery row: the model a submit would run now
// with its verdict, or why no model can run the operation.
func formatOperation(op *aiv1.AIOperationInfo) string {
	row := fmt.Sprintf("%-18s [%s] %s", op.Name, op.Category, op.Summary)
	if op.DefaultModelId == "" {
		reason := op.DefaultUnavailable
		if reason == "" {
			reason = "no model selected"
		}
		return row + " (no default: " + reason + ")"
	}
	return fmt.Sprintf("%s (default: %s, %s)", row, op.DefaultModelId, verdictfmt.Label(op.DefaultVerdict.GetGrade()))
}

// submit returns a RunCtx handler for one AI op. inputField/maskField name the
// positional/flag the op reads (empty when the op needs none).
func (h *handlers) submit(operation string, needsInput, needsMask, needsReference bool) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		if boolOr(ctx, "pose-control") && !needsInput {
			return fmt.Errorf("--pose-control requires an input-image command such as ai img2img or ai edit")
		}
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
		reference := ""
		if needsReference {
			reference = ctx.Flag("reference")
			if reference == "" {
				return fmt.Errorf("--reference is required for %s: the image that supplies the face", operation)
			}
		}
		params, err := buildParams(ctx)
		if err != nil {
			return err
		}
		if operation == "face_swap" {
			faceParams, err := faceSwapParams(ctx)
			if err != nil {
				return err
			}
			if err := mergeOpParams(params, faceParams); err != nil {
				return err
			}
		}
		if operation == "color_transfer" || operation == "mask_composite" {
			transferParams, err := regionOpParams(ctx)
			if err != nil {
				return err
			}
			if err := mergeOpParams(params, transferParams); err != nil {
				return err
			}
		}
		if operation == "inpaint" {
			contextRaw, edgeRaw := flagOr(ctx, "crop-context-pixels"), flagOr(ctx, "crop-max-edge-px")
			neutralize := boolOr(ctx, "neutralize-masked-pixels")
			if boolOr(ctx, "crop-to-mask") || contextRaw != "" || edgeRaw != "" || neutralize {
				if !boolOr(ctx, "crop-to-mask") {
					return fmt.Errorf("inpaint crop controls and --neutralize-masked-pixels require --crop-to-mask")
				}
				// Inpaint knobs travel through op_params; the capability manifest's
				// schema (and its requiresTrue rule) is the server-side contract.
				extra := map[string]any{
					"inpaint_crop_to_mask":             true,
					"inpaint_neutralize_masked_pixels": neutralize,
				}
				if contextRaw != "" {
					value, err := strconv.ParseUint(contextRaw, 10, 32)
					if err != nil || value > 512 {
						return fmt.Errorf("invalid --crop-context-pixels %q: expected 0..512", contextRaw)
					}
					extra["inpaint_context_pixels"] = float64(value)
				}
				if edgeRaw != "" {
					value, err := strconv.ParseUint(edgeRaw, 10, 32)
					if err != nil || (value != 512 && value != 1024) {
						return fmt.Errorf("invalid --crop-max-edge-px %q: expected 512 or 1024", edgeRaw)
					}
					extra["inpaint_max_edge_px"] = float64(value)
				}
				if err := mergeOpParams(params, extra); err != nil {
					return err
				}
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
			repair := map[string]any{"maskExpansionPixels": float64(expansion)}
			regions := make([]any, 0, len(removalRegions))
			for _, raw := range removalRegions {
				var box map[string]any
				if err := json.Unmarshal([]byte(raw), &box); err != nil {
					return fmt.Errorf("invalid --remove-region: %w", err)
				}
				regions = append(regions, box)
			}
			if len(regions) > 0 {
				repair["removalRegions"] = regions
			}
			if err := mergeOpParams(params, map[string]any{"background_repair": repair}); err != nil {
				return err
			}
		}
		if raw := flagOr(ctx, "selection"); raw != "" {
			var prompt map[string]any
			if err := json.Unmarshal([]byte(raw), &prompt); err != nil {
				return fmt.Errorf("invalid --selection: %w", err)
			}
			if err := mergeOpParams(params, map[string]any{"segmentation": prompt}); err != nil {
				return err
			}
		}
		if phrase := flagOr(ctx, "text"); phrase != "" {
			if flagOr(ctx, "selection") != "" {
				return fmt.Errorf("--text and --selection cannot be combined")
			}
			if err := mergeOpParams(params, map[string]any{"segmentation": map[string]any{"text": phrase}}); err != nil {
				return err
			}
		}
		if raw := flagOr(ctx, "support-roi"); raw != "" {
			var box map[string]any
			if err := json.Unmarshal([]byte(raw), &box); err != nil {
				return fmt.Errorf("invalid --support-roi: %w", err)
			}
			// Accept either a bare rectangle or the historical {"supportRoi": {...}} wrapper.
			if inner, ok := box["supportRoi"].(map[string]any); ok {
				box = inner
			}
			if err := mergeOpParams(params, map[string]any{"support_roi": box}); err != nil {
				return err
			}
		}
		if raw := flagOr(ctx, "mesh-resolution"); raw != "" {
			resolution, err := strconv.ParseUint(raw, 10, 32)
			if err != nil || (resolution != 128 && resolution != 256) {
				return fmt.Errorf("invalid --mesh-resolution %q: expected 128 or 256", raw)
			}
			if err := mergeOpParams(params, map[string]any{"mesh_resolution": float64(resolution)}); err != nil {
				return err
			}
		}
		resp, err := submitAI(h.core, operation, input, mask, reference, params, flagOr(ctx, "operation-id"))
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
		downloadPath := out
		if operation == "caption" {
			// Caption results are structured text, not image bytes. Do not write
			// their JSON blob to a path that may have an image extension.
			downloadPath = ""
		}
		job, written, werr := waitAndDownload(h.core, resp.JobId, downloadPath)
		if werr != nil {
			return werr
		}
		var caption *captionOutput
		if operation == "caption" {
			data, err := fetchBlob(h.core, job.GetResultRef())
			if err != nil {
				return err
			}
			caption, err = parseCaptionOutput(data)
			if err != nil {
				return err
			}
			if out != "" {
				written = pathForCaption(out)
				if err := os.WriteFile(written, []byte(caption.Caption), 0o644); err != nil {
					return fmt.Errorf("write caption %q: %w", written, err)
				}
			}
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
				Caption    string   `json:"caption,omitempty"`
				Backend    string   `json:"backend,omitempty"`
				Model      string   `json:"model,omitempty"`
			}{
				JobId:      job.GetId(),
				State:      jobstate.DisplayName(job.GetState()),
				OutputPath: written,
				ResultRefs: job.GetResultRefs(),
				Warnings:   warnings,
				Caption:    captionValue(caption, func(c captionOutput) string { return c.Caption }),
				Backend:    captionValue(caption, func(c captionOutput) string { return c.Backend }),
				Model:      captionValue(caption, func(c captionOutput) string { return c.Model }),
			})
		}
		changes := warnLines(warnings)
		if written != "" {
			changes = append(changes, fmt.Sprintf("wrote %s (job=%s)", written, job.GetId()))
		}
		resultLines := []string{fmt.Sprintf("%s %s on %s/%s", operation, jobstate.DisplayName(job.GetState()), resp.ModelId, resp.Tier)}
		if caption != nil {
			resultLines = append(resultLines, caption.Caption)
		}
		return ctx.RenderMutation(cliapp.MutationReport{
			Result:  resultLines,
			Changes: changes,
		})
	}
}

type captionOutput struct {
	Caption string `json:"caption"`
	Backend string `json:"backend"`
	Model   string `json:"model"`
}

func parseCaptionOutput(data []byte) (*captionOutput, error) {
	var output captionOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("decode caption result: %w", err)
	}
	if strings.TrimSpace(output.Caption) == "" {
		return nil, fmt.Errorf("caption result has no text")
	}
	output.Caption = strings.TrimSpace(output.Caption)
	return &output, nil
}

func captionValue[T any](caption *captionOutput, pick func(captionOutput) T) T {
	if caption == nil {
		var zero T
		return zero
	}
	return pick(*caption)
}

// consentLine states whether a submit of this request would need --consent and
// why, in the words the server's rule gave.
func consentLine(r *modelsv1.Resolution) string {
	if !r.GetConsentRequired() {
		return "consent: not required"
	}
	line := "consent: required — pass --consent to affirm it"
	if reason := r.GetConsentReason(); reason != "" {
		line += " (" + reason + ")"
	}
	return line
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
		ForceCpu:       boolOr(ctx, "force-cpu"),
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
		consentLine(r),
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
// mergeOpParams merges operation-specific values into AIParams.op_params,
// preserving anything already provided by --op-params.
// faceSwapParams turns the face_swap flags into the operation's op_params.
// Only flags the person set are sent, so the server's defaults stay the
// defaults; the capability schema is the contract for ranges.
func faceSwapParams(ctx cliapp.RunContext) (map[string]any, error) {
	out := map[string]any{}
	for flag, key := range map[string]string{"target-face": "target_face", "source-face": "source_face"} {
		raw := flagOr(ctx, flag)
		if raw == "" {
			continue
		}
		index, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || index > 63 {
			return nil, fmt.Errorf("invalid --%s %q: expected a face index 0..63 (largest face first)", flag, raw)
		}
		out[key] = float64(index)
	}
	if boolOr(ctx, "all-faces") {
		out["all_faces"] = true
	}
	if boolOr(ctx, "no-occlusion") {
		out["occlusion"] = false
	}
	if raw := flagOr(ctx, "color-match"); raw != "" {
		strength, err := strconv.ParseFloat(raw, 64)
		if err != nil || strength < 0 || strength > 1 {
			return nil, fmt.Errorf("invalid --color-match %q: expected 0..1", raw)
		}
		out["color_match"] = strength
	}
	if raw := flagOr(ctx, "face-strength"); raw != "" {
		strength, err := strconv.ParseFloat(raw, 64)
		if err != nil || strength < 0 || strength > 1 {
			return nil, fmt.Errorf("invalid --face-strength %q: expected 0..1", raw)
		}
		out["face_strength"] = strength
	}
	// The extent and match vocabularies are the manifest's: the server validates
	// the ids against its declaration and says what it accepts.
	if extent := flagOr(ctx, "extent"); extent != "" {
		out["extent"] = strings.ReplaceAll(strings.ToLower(extent), "-", "_")
	}
	if target := flagOr(ctx, "skin-target"); target != "" {
		out["skin_target"] = strings.ReplaceAll(strings.ToLower(target), "-", "_")
	}
	if raw := flagOr(ctx, "match"); raw != "" {
		var stages []any
		for _, stage := range strings.Split(raw, ",") {
			if stage = strings.TrimSpace(stage); stage != "" {
				stages = append(stages, strings.ReplaceAll(strings.ToLower(stage), "-", "_"))
			}
		}
		out["match"] = stages
	}
	return out, nil
}

// regionOpParams turns the flags of the region operations (colour transfer, mask
// composite) into op_params. --strength is the shared AIParams field and travels
// with the other common flags.
func regionOpParams(ctx cliapp.RunContext) (map[string]any, error) {
	out := map[string]any{}
	for flag, spec := range map[string]struct {
		key      string
		max      float64
		describe string
	}{
		"luminance":  {"luminance", 1, "0..1"},
		"flatten":    {"flatten", 1, "0..1"},
		"feather-px": {"feather_px", 64, "0..64"},
		"grow-px":    {"grow_px", 64, "0..64"},
	} {
		raw := flagOr(ctx, flag)
		if raw == "" {
			continue
		}
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil || value < 0 || value > spec.max {
			return nil, fmt.Errorf("invalid --%s %q: expected %s", flag, raw, spec.describe)
		}
		out[spec.key] = value
	}
	return out, nil
}

func mergeOpParams(p *aiv1.AIParams, extra map[string]any) error {
	if len(extra) == 0 {
		return nil
	}
	values := map[string]any{}
	if p.GetOpParams() != nil {
		values = p.GetOpParams().AsMap()
	}
	for k, v := range extra {
		values[k] = v
	}
	structured, err := structpb.NewStruct(values)
	if err != nil {
		return fmt.Errorf("invalid op params: %w", err)
	}
	p.OpParams = structured
	return nil
}

func buildParams(ctx cliapp.RunContext) (*aiv1.AIParams, error) {
	p := &aiv1.AIParams{
		Prompt:           flagOr(ctx, "prompt"),
		NegativePrompt:   flagOr(ctx, "negative"),
		Seed:             i64(flagOr(ctx, "seed")),
		Width:            i32(flagOr(ctx, "width")),
		Height:           i32(flagOr(ctx, "height")),
		Steps:            i32(flagOr(ctx, "steps")),
		CfgScale:         f64(flagOr(ctx, "cfg-scale")),
		Variations:       i32(flagOr(ctx, "variations")),
		Strength:         f64(flagOr(ctx, "strength")),
		Scale:            i32(flagOr(ctx, "scale")),
		Realism:          f64(flagOr(ctx, "realism")),
		FaceAware:        boolOr(ctx, "face-aware"),
		ModelOverride:    flagOr(ctx, "model"),
		AllowByok:        boolOr(ctx, "byok"),
		QualityPolicy:    flagOr(ctx, "quality-policy"),
		FallbackPolicy:   flagOr(ctx, "fallback-policy"),
		OpenrouterRole:   flagOr(ctx, "role"),
		Priority:         flagOr(ctx, "priority"),
		AllowReclaim:     optionalReclaim(ctx),
		AutoAcquire:      optionalAutoAcquire(ctx),
		RequireGpu:       boolOr(ctx, "require-gpu"),
		ForceCpu:         boolOr(ctx, "force-cpu"),
		AutoScanNsfw:     boolOr(ctx, "auto-scan"),
		ConsentAffirmed:  boolOr(ctx, "consent"),
		Adapters:         adapterRefs(ctx),
		PreserveIdentity: optionalPreserveIdentity(ctx),
	}
	var err error
	if p.MaxWaitSeconds, err = optionalMaxWait(ctx); err != nil {
		return nil, err
	}
	if p.MaxCostUsd, err = optionalMaxCost(ctx); err != nil {
		return nil, err
	}
	if p.EditStyle, err = editStyle(ctx); err != nil {
		return nil, err
	}
	if raw := flagOr(ctx, "op-params"); raw != "" {
		var values map[string]any
		if err := json.Unmarshal([]byte(raw), &values); err != nil {
			return nil, fmt.Errorf("invalid --op-params JSON: %w", err)
		}
		structured, err := structpb.NewStruct(values)
		if err != nil {
			return nil, fmt.Errorf("invalid --op-params: %w", err)
		}
		p.OpParams = structured
	}
	return p, nil
}

// optionalPreserveIdentity sends false only when the caller opted out; unset
// keeps the service default (identity preservation on for edit_instruct).
func optionalPreserveIdentity(ctx cliapp.RunContext) *bool {
	if boolOr(ctx, "no-preserve-identity") {
		v := false
		return &v
	}
	return nil
}

// optionalMaxWait parses --max-wait (a Go duration such as 90s or 10m, or a bare
// number of seconds; 0 means no limit). Unset keeps the deployment default.
func optionalMaxWait(ctx cliapp.RunContext) (*int32, error) {
	raw := strings.TrimSpace(flagOr(ctx, "max-wait"))
	if raw == "" {
		return nil, nil
	}
	seconds, err := parseWaitSeconds(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid --max-wait %q: %w", raw, err)
	}
	return &seconds, nil
}

func parseWaitSeconds(raw string) (int32, error) {
	var seconds float64
	if d, err := time.ParseDuration(raw); err == nil {
		seconds = d.Seconds()
	} else if n, nerr := strconv.ParseFloat(raw, 64); nerr == nil {
		seconds = n
	} else {
		return 0, fmt.Errorf("use a duration such as 90s or 10m, or a number of seconds")
	}
	if seconds < 0 || seconds > math.MaxInt32 {
		return 0, fmt.Errorf("must be between 0 and %d seconds", math.MaxInt32)
	}
	return int32(math.Ceil(seconds)), nil
}

// optionalMaxCost parses --max-cost in USD. Unset keeps the deployment default.
func optionalMaxCost(ctx cliapp.RunContext) (*float64, error) {
	raw := strings.TrimSpace(strings.TrimPrefix(flagOr(ctx, "max-cost"), "$"))
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return nil, fmt.Errorf("invalid --max-cost %q: use a non-negative dollar amount such as 0.25", flagOr(ctx, "max-cost"))
	}
	return &v, nil
}

func editStyle(ctx cliapp.RunContext) (string, error) {
	switch v := strings.ToLower(strings.TrimSpace(flagOr(ctx, "edit-style"))); v {
	case "", "identity", "stylized":
		return v, nil
	case "stylised":
		return "stylized", nil
	default:
		return "", fmt.Errorf("invalid --edit-style %q: use identity or stylized", v)
	}
}

// optionalAutoAcquire sends false only when the caller opted out; unset keeps
// the service default (acquire missing governed weights).
func optionalAutoAcquire(ctx cliapp.RunContext) *bool {
	if boolOr(ctx, "no-auto-acquire") {
		v := false
		return &v
	}
	return nil
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
	poseControl := boolOr(ctx, "pose-control")
	if len(specs) == 0 && !poseControl {
		return nil
	}
	out := make([]*aiv1.AdapterRef, 0, len(specs)+1)
	for _, s := range specs {
		out = append(out, &aiv1.AdapterRef{AdapterId: s.id, Scale: s.scale, ConditioningImageKey: s.key, LicenseTermsAccepted: boolOr(ctx, "accept-openrail-m")})
	}
	if poseControl {
		out = append(out, &aiv1.AdapterRef{
			AdapterId:            "controlnet-openpose-sd15",
			Scale:                1,
			PreprocessorOverride: "pose",
			LicenseTermsAccepted: boolOr(ctx, "accept-openrail-m"),
		})
	}
	return out
}

func explainAdapterRefs(ctx cliapp.RunContext) []*modelsv1.AdapterRef {
	specs := collectAdapterSpecs(ctx)
	poseControl := boolOr(ctx, "pose-control")
	if len(specs) == 0 && !poseControl {
		return nil
	}
	out := make([]*modelsv1.AdapterRef, 0, len(specs)+1)
	for _, s := range specs {
		out = append(out, &modelsv1.AdapterRef{AdapterId: s.id, Scale: s.scale, ConditioningImageKey: s.key, LicenseTermsAccepted: boolOr(ctx, "accept-openrail-m")})
	}
	if poseControl {
		out = append(out, &modelsv1.AdapterRef{
			AdapterId:            "controlnet-openpose-sd15",
			Scale:                1,
			PreprocessorOverride: "pose",
			LicenseTermsAccepted: boolOr(ctx, "accept-openrail-m"),
		})
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
	response, err := aiconnect.NewAIServiceClient(httpClient, baseURL).SubmitStoredAI(context.Background(), connect.NewRequest(&aiv1.SubmitStoredAIRequest{Operation: ctx.Positional("operation"), Params: params, InputRef: ctx.Flag("input-ref"), MaskRef: ctx.Flag("mask-ref"), ReferenceRef: ctx.Flag("reference-ref"), OperationId: ctx.Flag("operation-id")}))
	if err != nil {
		return cliapp.WrapAPIError("submit stored AI operation", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, response.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Submitted %s as %s on %s/%s", ctx.Positional("operation"), response.Msg.JobId, response.Msg.ModelId, response.Msg.Tier)}, Changes: []string{"wait: image-tools jobs wait " + response.Msg.JobId}})
}
