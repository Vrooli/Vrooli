package analyze

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	analysisv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/analysis"
	analysisconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/analysis/analysis_v1connect"
)

type handlers struct {
	core *cliapp.ScenarioApp
}

func newHandlers(core *cliapp.ScenarioApp) *handlers { return &handlers{core: core} }

// list mirrors AnalysisService.ListAnalysisOperations (Connect discovery).
func (h *handlers) list(ctx cliapp.RunContext) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := analysisconnect.NewAnalysisServiceClient(httpClient, baseURL)
	resp, err := client.ListAnalysisOperations(context.Background(), connect.NewRequest(&analysisv1.ListAnalysisOperationsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list analysis operations", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Operations))
	for _, op := range resp.Msg.Operations {
		backing := "pure-go"
		if op.ModelBacked {
			backing = "model: " + op.DefaultModelId
		}
		results = append(results, fmt.Sprintf("%-14s %s (%s)", op.Name, op.Summary, backing))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d analysis operations.", len(resp.Msg.Operations))},
		ResultsHeading: "Operations",
		Results:        results,
	})
}

// run returns a handler for one analysis op, rendering its structured result.
func (h *handlers) run(operation string) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		input := ctx.Positional("input")
		if input == "" {
			return fmt.Errorf("an input image path is required")
		}
		resp, err := analyze(h.core, operation, input)
		if err != nil {
			return err
		}
		return render(ctx, resp)
	}
}

func render(ctx cliapp.RunContext, resp *analysisv1.AnalyzeResponse) error {
	switch r := resp.Result.(type) {
	case *analysisv1.AnalyzeResponse_Probe:
		p := r.Probe
		results := []string{
			fmt.Sprintf("dimensions : %dx%d (%.2f MP)", p.Width, p.Height, p.Megapixels),
			fmt.Sprintf("format     : %s (%s, alpha=%v)", p.Format, p.ColorModel, p.HasAlpha),
			fmt.Sprintf("bytes      : %d", p.SizeBytes),
			fmt.Sprintf("frames     : %d", p.FrameCount),
			fmt.Sprintf("exif       : present=%v gps=%v orientation=%d", p.HasExif, p.HasGps, p.Orientation),
		}
		for _, c := range p.DominantColors {
			results = append(results, fmt.Sprintf("color      : %s (%.0f%%)", c.Hex, c.Fraction*100))
		}
		return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{
			Summary: []string{fmt.Sprintf("Probe: %dx%d %s", p.Width, p.Height, p.Format)}, ResultsHeading: "Info", Results: results,
		})
	case *analysisv1.AnalyzeResponse_Ocr:
		o := r.Ocr
		return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("OCR (%s): %d chars, %d blocks", o.Language, len(o.FullText), len(o.Blocks))},
			ResultsHeading: "Text", Results: []string{o.FullText},
		})
	case *analysisv1.AnalyzeResponse_Nsfw:
		n := r.Nsfw
		results := []string{fmt.Sprintf("verdict: %s (score %.3f, threshold %.2f)", n.Label, n.Score, n.Threshold)}
		for _, c := range n.Categories {
			results = append(results, fmt.Sprintf("%-10s %.3f", c.Label, c.Score))
		}
		return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{
			Summary: []string{fmt.Sprintf("NSFW: %v (%.3f)", n.Nsfw, n.Score)}, ResultsHeading: "Scores", Results: results,
		})
	case *analysisv1.AnalyzeResponse_Duplicate:
		d := r.Duplicate
		return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("Perceptual fingerprint (%d-bit)", d.HashBits)},
			ResultsHeading: "Hashes",
			Results: []string{
				fmt.Sprintf("phash: %s", d.PhashHex),
				fmt.Sprintf("ahash: %s", d.AhashHex),
			},
		})
	case *analysisv1.AnalyzeResponse_Quality:
		q := r.Quality
		results := []string{
			fmt.Sprintf("overall    : %.2f", q.OverallScore),
			fmt.Sprintf("sharpness  : %.2f (blurry=%v)", q.Sharpness, q.Blurry),
			fmt.Sprintf("brightness : %.1f (%s)", q.Brightness, q.Exposure),
			fmt.Sprintf("contrast   : %.1f", q.Contrast),
		}
		for _, n := range q.Notes {
			results = append(results, "note       : "+n)
		}
		return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{
			Summary: []string{fmt.Sprintf("Quality: %.2f", q.OverallScore)}, ResultsHeading: "Metrics", Results: results,
		})
	default:
		return fmt.Errorf("empty analysis result")
	}
}
