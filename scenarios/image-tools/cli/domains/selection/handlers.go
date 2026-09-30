package selection

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/selection"
	selectionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/selection/selection_v1connect"
)

type handlers struct {
	core *cliapp.ScenarioApp
}

func newHandlers(core *cliapp.ScenarioApp) *handlers { return &handlers{core: core} }

// classes mirrors SelectionService.ListRegionClasses (Connect discovery).
func (h *handlers) classes(ctx cliapp.RunContext) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := selectionconnect.NewSelectionServiceClient(httpClient, baseURL)
	resp, err := client.ListRegionClasses(context.Background(), connect.NewRequest(&selectionv1.ListRegionClassesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list region classes", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Classes))
	for _, c := range resp.Msg.Classes {
		results = append(results, fmt.Sprintf("%-11s %s (%d edits)", c.Name, c.Summary, len(c.Edits)))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d region classes.", len(resp.Msg.Classes))},
		ResultsHeading: "Classes",
		Results:        results,
	})
}

// suggest mirrors SelectionService.SuggestEdits (the contextual-edit compiler).
func (h *handlers) suggest(ctx cliapp.RunContext) error {
	class := ctx.Positional("class")
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := selectionconnect.NewSelectionServiceClient(httpClient, baseURL)
	resp, err := client.SuggestEdits(context.Background(), connect.NewRequest(&selectionv1.SuggestEditsRequest{RegionClass: class}))
	if err != nil {
		return cliapp.WrapAPIError("suggest edits", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Contextual edits for %q:", resp.Msg.RegionClass)},
		ResultsHeading: "Edits",
		Results:        editLines(resp.Msg.Edits),
	})
}

// segment drives the REST segment edge and writes the produced mask to --out.
func (h *handlers) segment(ctx cliapp.RunContext) error {
	input := ctx.Positional("input")
	if input == "" {
		return fmt.Errorf("an input image path is required")
	}
	params, err := buildSegmentParams(ctx)
	if err != nil {
		return err
	}
	res, err := segment(h.core, input, params)
	if err != nil {
		return err
	}

	out := ""
	if ctx.FlagDeclared("out") {
		out = ctx.Flag("out")
	}
	changes := warnLines(res.Warnings)
	if out != "" && res.MaskRef != "" {
		if err := downloadBlob(h.core, res.MaskRef, out); err != nil {
			return err
		}
		changes = append(changes, fmt.Sprintf("wrote mask %s", out))
	} else {
		changes = append(changes, fmt.Sprintf("mask blob: %s (fetch via image-tools blobs)", res.MaskRef))
	}
	changes = append(changes, editLines(res.SuggestedEdits)...)

	return ctx.RenderMutation(cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Selected %s (confidence %.2f, %.1f%% of image) on %s",
			res.RegionClass, res.Confidence, res.AreaFraction*100, res.Tier)},
		Changes: changes,
	})
}

// buildSegmentParams assembles SegmentParams from the segment command flags.
func buildSegmentParams(ctx cliapp.RunContext) (*selectionv1.SegmentParams, error) {
	p := &selectionv1.SegmentParams{}
	switch strings.ToLower(flagOr(ctx, "mode")) {
	case "point":
		p.Mode = selectionv1.SegmentMode_SEGMENT_MODE_POINT
	case "box":
		p.Mode = selectionv1.SegmentMode_SEGMENT_MODE_BOX
	case "auto", "":
		if flagOr(ctx, "point") != "" {
			p.Mode = selectionv1.SegmentMode_SEGMENT_MODE_POINT
		} else if flagOr(ctx, "box") != "" {
			p.Mode = selectionv1.SegmentMode_SEGMENT_MODE_BOX
		} else {
			p.Mode = selectionv1.SegmentMode_SEGMENT_MODE_AUTO
		}
	default:
		return nil, fmt.Errorf("unknown --mode %q (point|box|auto)", flagOr(ctx, "mode"))
	}
	if pt := flagOr(ctx, "point"); pt != "" {
		x, y, err := parsePair(pt)
		if err != nil {
			return nil, fmt.Errorf("--point %q: %w (want x,y in 0..1)", pt, err)
		}
		p.Points = []*selectionv1.Point{{X: x, Y: y}}
	}
	if bx := flagOr(ctx, "box"); bx != "" {
		vals, err := parseQuad(bx)
		if err != nil {
			return nil, fmt.Errorf("--box %q: %w (want x,y,w,h in 0..1)", bx, err)
		}
		p.Box = &selectionv1.Box{X: vals[0], Y: vals[1], Width: vals[2], Height: vals[3]}
	}
	if tol := flagOr(ctx, "tolerance"); tol != "" {
		v, err := strconv.ParseFloat(tol, 64)
		if err != nil {
			return nil, fmt.Errorf("--tolerance %q: %w", tol, err)
		}
		p.Tolerance = v
	}
	if m := flagOr(ctx, "model"); m != "" {
		p.ModelOverride = m
	}
	return p, nil
}

func editLines(edits []*selectionv1.SuggestedEdit) []string {
	out := make([]string, 0, len(edits))
	for _, e := range edits {
		mask := ""
		if e.RequiresMask {
			mask = " [masked]"
		}
		prompt := ""
		if e.RequiresPrompt {
			prompt = " (needs prompt)"
		}
		out = append(out, fmt.Sprintf("%-16s → %s%s%s — %s", e.Id, e.Operation, mask, prompt, e.Label))
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

func flagOr(ctx cliapp.RunContext, name string) string {
	if ctx.FlagDeclared(name) {
		return ctx.Flag(name)
	}
	return ""
}

func parsePair(s string) (float64, float64, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("want two comma-separated values")
	}
	x, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, err
	}
	y, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

func parseQuad(s string) ([4]float64, error) {
	var out [4]float64
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return out, fmt.Errorf("want four comma-separated values")
	}
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return out, err
		}
		out[i] = v
	}
	return out, nil
}
