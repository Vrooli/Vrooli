package diff

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	diffv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff"
	diffconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff/diff_v1connect"
)

type handlers struct {
	core *cliapp.ScenarioApp
}

func newHandlers(core *cliapp.ScenarioApp) *handlers { return &handlers{core: core} }

// modes mirrors DiffService.ListDiffModes (Connect discovery).
func (h *handlers) modes(ctx cliapp.RunContext) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := diffconnect.NewDiffServiceClient(httpClient, baseURL)
	resp, err := client.ListDiffModes(context.Background(), connect.NewRequest(&diffv1.ListDiffModesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list diff modes", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Modes))
	for _, m := range resp.Msg.Modes {
		results = append(results, fmt.Sprintf("%-11s %s", m.Name, m.Summary))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d comparison modes.", len(resp.Msg.Modes))},
		ResultsHeading: "Modes",
		Results:        results,
	})
}

// compare drives the REST compare edge and reports the metrics + heat-map.
func (h *handlers) compare(ctx cliapp.RunContext) error {
	base := ctx.Positional("base")
	cmp := ctx.Positional("compare")
	if base == "" || cmp == "" {
		return fmt.Errorf("two image paths are required: <base> <compare>")
	}
	params, err := buildDiffParams(ctx)
	if err != nil {
		return err
	}
	res, err := compare(h.core, base, cmp, params)
	if err != nil {
		return err
	}

	changes := warnLines(res.Warnings)
	if out := flagOr(ctx, "out"); out != "" && res.HeatmapRef != "" {
		if err := downloadBlob(h.core, res.HeatmapRef, out); err != nil {
			return err
		}
		changes = append(changes, fmt.Sprintf("wrote heat-map %s", out))
	} else if res.HeatmapRef != "" {
		changes = append(changes, fmt.Sprintf("heat-map blob: %s (fetch via image-tools blobs)", res.HeatmapRef))
	}
	changes = append(changes,
		fmt.Sprintf("changed pixels: %d / %d (%.2f%%)", res.ChangedPixels, res.TotalPixels, res.ChangedFraction*100),
		fmt.Sprintf("MAE %.2f · RMSE %.2f · PSNR %.2f dB", res.Mae, res.Rmse, res.Psnr),
		fmt.Sprintf("pHash distance %d (similarity %.2f) · SSIM %.4f", res.PhashDistance, res.PhashSimilarity, res.Ssim),
	)
	if !res.DimensionsMatch {
		changes = append(changes, fmt.Sprintf("dimensions differ: base %dx%d vs compare %dx%d",
			res.BaseWidth, res.BaseHeight, res.CompareWidth, res.CompareHeight))
	}

	return ctx.RenderMutation(cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Verdict: %s", strings.ToUpper(res.Verdict))},
		Changes: changes,
	})
}

// buildDiffParams assembles DiffParams from the compare command flags.
func buildDiffParams(ctx cliapp.RunContext) (*diffv1.DiffParams, error) {
	// Heat-map on by default; --no-heatmap turns it off.
	p := &diffv1.DiffParams{IncludeHeatmap: true}
	switch strings.ToLower(flagOr(ctx, "mode")) {
	case "perceptual":
		p.Mode = diffv1.DiffMode_DIFF_MODE_PERCEPTUAL
	case "pixel", "":
		p.Mode = diffv1.DiffMode_DIFF_MODE_PIXEL
	default:
		return nil, fmt.Errorf("unknown --mode %q (pixel|perceptual)", flagOr(ctx, "mode"))
	}
	if tol := flagOr(ctx, "tolerance"); tol != "" {
		v, err := strconv.ParseFloat(tol, 64)
		if err != nil {
			return nil, fmt.Errorf("--tolerance %q: %w", tol, err)
		}
		p.Tolerance = v
	}
	if hex := flagOr(ctx, "highlight"); hex != "" {
		p.HighlightHex = hex
	}
	if ctx.FlagDeclared("no-heatmap") {
		p.IncludeHeatmap = false
	}
	return p, nil
}

func warnLines(warnings []string) []string {
	out := make([]string, 0, len(warnings))
	for _, w := range warnings {
		out = append(out, "warning: "+w)
	}
	return out
}

func flagOr(ctx cliapp.RunContext, name string) string {
	if ctx.FlagDeclared(name) {
		return ctx.Flag(name)
	}
	return ""
}
