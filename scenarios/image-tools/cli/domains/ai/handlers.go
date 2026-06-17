package ai

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai/ai_v1connect"
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

// buildParams assembles AIParams from whichever flags the command declared.
func buildParams(ctx cliapp.RunContext) *aiv1.AIParams {
	p := &aiv1.AIParams{
		Prompt:         flagOr(ctx, "prompt"),
		NegativePrompt: flagOr(ctx, "negative"),
		Seed:           i64(flagOr(ctx, "seed")),
		Width:          i32(flagOr(ctx, "width")),
		Height:         i32(flagOr(ctx, "height")),
		Steps:          i32(flagOr(ctx, "steps")),
		CfgScale:       f64(flagOr(ctx, "cfg-scale")),
		Variations:     i32(flagOr(ctx, "variations")),
		Strength:       f64(flagOr(ctx, "strength")),
		Scale:          i32(flagOr(ctx, "scale")),
		ModelOverride:  flagOr(ctx, "model"),
		AllowByok:      boolOr(ctx, "byok"),
		AutoScanNsfw:   boolOr(ctx, "auto-scan"),
	}
	return p
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
