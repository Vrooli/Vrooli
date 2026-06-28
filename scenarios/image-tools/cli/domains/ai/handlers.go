package ai

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

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
		if needsMask {
			mask = ctx.Flag("mask")
			if mask == "" {
				return fmt.Errorf("--mask is required for %s", operation)
			}
		}
		params := buildParams(ctx)
		resp, err := submitAI(h.core, operation, input, mask, params)
		if err != nil {
			return err
		}

		out := flagOr(ctx, "out")
		wait := boolOr(ctx, "wait")
		if !wait {
			return ctx.RenderMutation(cliapp.MutationReport{
				Result: []string{fmt.Sprintf("Submitted %s as job %s (~%ds) on %s/%s", operation, resp.JobId, resp.EstimatedSeconds, resp.ModelId, resp.Tier)},
				Changes: append(warnLines(resp.Warnings),
					fmt.Sprintf("wait: image-tools jobs wait %s", resp.JobId)),
			})
		}
		job, werr := waitAndDownload(h.core, resp.JobId, out)
		if werr != nil {
			return werr
		}
		changes := warnLines(resp.Warnings)
		if out != "" {
			changes = append(changes, fmt.Sprintf("wrote %s (job=%s)", out, job.GetId()))
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
	r, err := explainResolution(h.core, operation, flagOr(ctx, "model"), boolOr(ctx, "byok"), explainAdapterRefs(ctx))
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
		AutoScanNsfw:    boolOr(ctx, "auto-scan"),
		ConsentAffirmed: boolOr(ctx, "consent"),
		Adapters:        adapterRefs(ctx),
	}
	return p
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
