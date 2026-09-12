package ops

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
	opsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops/ops_v1connect"
)

type handlers struct {
	core *cliapp.ScenarioApp
}

func newHandlers(core *cliapp.ScenarioApp) *handlers { return &handlers{core: core} }

// list mirrors OpsService.ListOperations (the Connect discovery RPC).
func (h *handlers) list(ctx cliapp.RunContext) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := opsconnect.NewOpsServiceClient(httpClient, baseURL)
	resp, err := client.ListOperations(context.Background(), connect.NewRequest(&opsv1.ListOperationsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list operations", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Operations))
	for _, op := range resp.Msg.Operations {
		results = append(results, fmt.Sprintf("%-10s [%s] %s", op.Name, op.Category, op.Summary))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d deterministic operations.", len(resp.Msg.Operations))},
		ResultsHeading: "Operations",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("decodable: %v", resp.Msg.DecodableFormats),
			fmt.Sprintf("encodable: %v", resp.Msg.EncodableFormats),
		},
	})
}

// run wraps the common flow: build params, call the REST edge, write the output.
func (h *handlers) run(operation string, build func(ctx cliapp.RunContext) *opsv1.OpParams) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		input := ctx.Positional("input")
		if input == "" {
			return fmt.Errorf("an input image path is required")
		}
		// Only the overlay command declares an --overlay (image watermark) flag.
		overlay := ""
		if ctx.FlagDeclared("overlay") {
			overlay = ctx.Flag("overlay")
		}
		// Derive the output encoding from the --out extension so any op honors
		// `--out result.webp`. convert/compress still take an explicit --format.
		res, err := runOp(h.core, operation, input, overlay, extToFormat(ctx.Flag("out")), build(ctx))
		if err != nil {
			return err
		}
		return writeOutput(ctx, res, ctx.Flag("out"))
	}
}

// --- per-op param builders ---

func (h *handlers) resizeParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Resize{Resize: &opsv1.ResizeParams{
		Width: i32(ctx.Flag("width")), Height: i32(ctx.Flag("height")), Fit: ctx.Flag("fit"), Gravity: ctx.Flag("gravity"),
	}}}
}

func (h *handlers) cropParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Crop{Crop: &opsv1.CropParams{
		X: i32(ctx.Flag("x")), Y: i32(ctx.Flag("y")), Width: i32(ctx.Flag("width")), Height: i32(ctx.Flag("height")), Gravity: ctx.Flag("gravity"),
	}}}
}

func (h *handlers) rotateParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Rotate{Rotate: &opsv1.RotateParams{
		Angle: f64(ctx.Flag("angle")), Background: ctx.Flag("background"),
	}}}
}

func (h *handlers) flipParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Flip{Flip: &opsv1.FlipParams{Axis: ctx.Flag("axis")}}}
}

func (h *handlers) deskewParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Deskew{Deskew: &opsv1.DeskewParams{Background: ctx.Flag("background")}}}
}

func (h *handlers) thumbnailParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Thumbnail{Thumbnail: &opsv1.ThumbnailParams{
		Width: i32(ctx.Flag("width")), Height: i32(ctx.Flag("height")),
	}}}
}

func (h *handlers) canvasParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Canvas{Canvas: &opsv1.CanvasParams{
		Width: i32(ctx.Flag("width")), Height: i32(ctx.Flag("height")), Background: ctx.Flag("background"), Gravity: ctx.Flag("gravity"),
	}}}
}

func (h *handlers) adjustParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Adjust{Adjust: &opsv1.AdjustParams{
		Brightness: f64(ctx.Flag("brightness")), Contrast: f64(ctx.Flag("contrast")),
		Gamma: f64(ctx.Flag("gamma")), Saturation: f64(ctx.Flag("saturation")), Hue: f64(ctx.Flag("hue")),
	}}}
}

func (h *handlers) filterParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Filter{Filter: &opsv1.FilterParams{
		Filter: ctx.Flag("filter"), Amount: f64(ctx.Flag("amount")),
	}}}
}

func (h *handlers) duotoneParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Duotone{Duotone: &opsv1.DuotoneParams{Dark: ctx.Flag("dark"), Light: ctx.Flag("light"), Mid: ctx.Flag("mid"), MidLow: f64(ctx.Flag("mid-low")), MidHigh: f64(ctx.Flag("mid-high"))}}}
}

func (h *handlers) posterizeParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Posterize{Posterize: &opsv1.PosterizeParams{Levels: i32(ctx.Flag("levels")), Dark: ctx.Flag("dark"), Light: ctx.Flag("light")}}}
}

func (h *handlers) halftoneParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Halftone{Halftone: &opsv1.HalftoneParams{Lpi: i32(ctx.Flag("lpi")), Angle: f64(ctx.Flag("angle")), Dot: ctx.Flag("dot"), Dark: ctx.Flag("dark"), Light: ctx.Flag("light")}}}
}

func (h *handlers) ditherOrderedParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_DitherOrdered{DitherOrdered: &opsv1.DitherParams{Dark: ctx.Flag("dark"), Light: ctx.Flag("light")}}}
}

func (h *handlers) ditherDiffusionParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_DitherDiffusion{DitherDiffusion: &opsv1.DitherParams{Dark: ctx.Flag("dark"), Light: ctx.Flag("light")}}}
}

func (h *handlers) grainParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Grain{Grain: &opsv1.GrainParams{Seed: i64(ctx.Flag("seed")), Amount: f64(ctx.Flag("amount")), ContrastMultiplier: f64(ctx.Flag("contrast-multiplier"))}}}
}

func (h *handlers) scrimParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Scrim{Scrim: &opsv1.ScrimParams{Color: ctx.Flag("color"), Opacity: f64(ctx.Flag("opacity")), Direction: ctx.Flag("direction")}}}
}

func (h *handlers) lineScreenParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_LineScreen{LineScreen: &opsv1.LineScreenParams{Spacing: f64(ctx.Flag("spacing")), SpacingRel: f64(ctx.Flag("spacing-rel")), Angle: f64(ctx.Flag("angle"))}}}
}

func (h *handlers) stippleParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Stipple{Stipple: &opsv1.StippleParams{Spacing: f64(ctx.Flag("spacing")), SpacingRel: f64(ctx.Flag("spacing-rel")), Seed: i64(ctx.Flag("seed"))}}}
}

func (h *handlers) engravingParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Engraving{Engraving: &opsv1.EngravingParams{Spacing: f64(ctx.Flag("spacing")), SpacingRel: f64(ctx.Flag("spacing-rel"))}}}
}

func (h *handlers) aberrationParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Aberration{Aberration: &opsv1.AberrationParams{Amplitude: f64(ctx.Flag("amplitude")), DistanceRel: f64(ctx.Flag("distance-rel"))}}}
}

func (h *handlers) bloomParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Bloom{Bloom: &opsv1.BloomParams{Radius: i32(ctx.Flag("radius")), RadiusRel: f64(ctx.Flag("radius-rel")), Threshold: f64(ctx.Flag("threshold"))}}}
}

func (h *handlers) curveParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Curve{Curve: &opsv1.CurveParams{Exponent: f64(ctx.Flag("exponent"))}}}
}

func (h *handlers) defocusParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Defocus{Defocus: &opsv1.DefocusParams{Radius: i32(ctx.Flag("radius")), RadiusRel: f64(ctx.Flag("radius-rel")), BladeCount: i32(ctx.Flag("blade-count"))}}}
}

func (h *handlers) motionBlurParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_MotionBlur{MotionBlur: &opsv1.MotionBlurParams{Distance: i32(ctx.Flag("distance")), DistanceRel: f64(ctx.Flag("distance-rel")), Angle: f64(ctx.Flag("angle"))}}}
}

func (h *handlers) asciiMosaicParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_AsciiMosaic{AsciiMosaic: &opsv1.AsciiMosaicParams{BlockSize: i32(ctx.Flag("block-size")), BlockSizeRel: f64(ctx.Flag("block-size-rel"))}}}
}

func (h *handlers) pixelSortParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_PixelSort{PixelSort: &opsv1.PixelSortParams{Threshold: f64(ctx.Flag("threshold")), Axis: ctx.Flag("axis")}}}
}

func (h *handlers) displacementParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Displacement{Displacement: &opsv1.DisplacementParams{Amplitude: f64(ctx.Flag("amplitude")), AmplitudeRel: f64(ctx.Flag("amplitude-rel")), Spacing: f64(ctx.Flag("spacing")), SpacingRel: f64(ctx.Flag("spacing-rel")), Seed: i64(ctx.Flag("seed"))}}}
}

func (h *handlers) convertParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Convert{Convert: &opsv1.ConvertParams{
		Format: ctx.Flag("format"), Quality: i32(ctx.Flag("quality")), Lossless: ctx.BoolFlag("lossless"),
	}}}
}

func (h *handlers) compressParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Compress{Compress: &opsv1.CompressParams{
		Format: ctx.Flag("format"), Quality: i32(ctx.Flag("quality")), Lossless: ctx.BoolFlag("lossless"), TargetBytes: i64(ctx.Flag("target-bytes")),
	}}}
}

func (h *handlers) overlayParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Overlay{Overlay: &opsv1.OverlayParams{
		Text: ctx.Flag("text"), Position: ctx.Flag("position"), Opacity: f64(ctx.Flag("opacity")), Color: ctx.Flag("color"), FontSize: f64(ctx.Flag("font-size")),
	}}}
}

func (h *handlers) metadataParams(ctx cliapp.RunContext) *opsv1.OpParams {
	return &opsv1.OpParams{Op: &opsv1.OpParams_Metadata{Metadata: &opsv1.MetadataParams{
		StripAll: ctx.BoolFlag("strip-all"), StripGps: ctx.BoolFlag("strip-gps"), AutoOrient: ctx.BoolFlag("auto-orient"),
	}}}
}

// --- flag parse helpers (empty/invalid → zero, the proto default) ---

func i32(s string) int32 {
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0
	}
	return int32(n)
}

func i64(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func f64(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
