package ops

import (
	"context"
	"fmt"
	"image"
	"sort"

	"image-tools/internal/treatments"
)

// RunInput is what an operation receives: the original encoded bytes, the
// decoded image, its source metadata, and the typed parameters.
type RunInput struct {
	Context context.Context
	Bytes   []byte
	Img     image.Image
	Meta    Meta
	Params  *Params
}

// RunResult is an operation's output: the encoded result bytes plus the result
// format/MIME and (for image results) dimensions. A non-image result (metadata
// read) sets Format "json" and Width/Height 0.
type RunResult struct {
	Bytes  []byte
	Format string
	Mime   string
	Width  int
	Height int
	// ResolvedParams reports the pixel value each relative spatial parameter
	// resolved to for this input, keyed by the absolute parameter's name. It is
	// nil when the request sent none. Without it a caller sending `spacing_rel`
	// cannot learn what actually ran.
	ResolvedParams map[string]float64
}

// RunFunc executes one operation end-to-end (decode already done by Execute).
type RunFunc func(in RunInput) (RunResult, error)

// Op is a registered deterministic operation.
// Op is one in-process operation. Discovery metadata (family, summary, ports)
// comes from the operation's capability manifest; this table owns only execution.
type Op struct {
	Name string
	run  RunFunc
}

// registry is the canonical deterministic-op table. It is the single source of
// truth consumed by the job runner (execution), the OpsService (discovery), and
// the CLI (command surface).
var registry = func() map[string]Op {
	ops := []Op{
		{Name: "resize", run: pixel(Resize)},
		{Name: "crop", run: pixel(Crop)},
		{Name: "rotate", run: pixel(Rotate)},
		{Name: "flip", run: pixel(Flip)},
		{Name: "deskew", run: pixel(Deskew)},
		{Name: "thumbnail", run: pixel(Thumbnail)},
		{Name: "canvas", run: pixel(Canvas)},
		{Name: "adjust", run: pixel(Adjust)},
		{Name: "filter", run: pixel(Filter)},
		{Name: "duotone", run: treatment(treatments.Duotone)},
		{Name: "posterize", run: treatment(treatments.Posterize)},
		{Name: "halftone", run: treatment(treatments.Halftone)},
		{Name: "dither_ordered", run: treatment(treatments.DitherOrdered)},
		{Name: "dither_diffusion", run: treatment(treatments.DitherDiffusion)},
		{Name: "grain", run: treatment(treatments.Grain)},
		{Name: "scrim", run: treatment(treatments.Scrim)},
		{Name: "line_screen", run: tier2("line_screen")},
		{Name: "stipple", run: tier2("stipple")},
		{Name: "engraving", run: tier2("engraving")},
		{Name: "aberration", run: tier2("aberration")},
		{Name: "bloom", run: tier2("bloom")},
		{Name: "curve", run: tier2("curve")},
		{Name: "defocus", run: tier2("defocus")},
		{Name: "motion_blur", run: tier2("motion_blur")},
		{Name: "ascii_mosaic", run: tier2("ascii_mosaic")},
		{Name: "pixel_sort", run: tier2("pixel_sort")},
		{Name: "displacement", run: tier2("displacement")},
		{Name: "canny", run: pixel(Canny)},
		{Name: "screenshot_cleanup", run: pixel(ScreenshotCleanup)},
		{Name: "convert", run: pixel(convertImage)},
		{Name: "rasterize", run: Rasterize},
		{Name: "vectorize", run: Vectorize},
		{Name: "icon_container", run: IconContainer},
		{Name: "compress", run: compressRun},
		{Name: "overlay", run: pixel(Overlay)},
		{Name: "parallax", run: parallaxSequence},
		{Name: "composite", run: pixel(Composite)},
		{Name: "metadata", run: metadataRun},
	}
	m := make(map[string]Op, len(ops))
	for _, o := range ops {
		m[o.Name] = o
	}
	return m
}()

// Names returns the registered operation names in stable (sorted) order.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func tier2(name string) RunFunc {
	return pixel(func(img image.Image, p *Params) (image.Image, error) {
		tp := treatments.Params{Normalize: p.Normalize, Seed: p.Seed, Angle: p.Angle, Spacing: p.Spacing, Radius: p.Radius, BladeCount: p.BladeCount, Distance: p.Distance, Amplitude: p.Amplitude, Threshold: p.Threshold, Curve: p.Curve, BlockSize: p.BlockSize, Axis: p.Axis}
		knockoutFrom(p, &tp)
		return treatments.Tier2(treatments.ReserveBefore(img, tp), name, tp)
	})
}

func treatment(fn func(image.Image, treatments.Params) (image.Image, error)) RunFunc {
	return pixel(func(img image.Image, p *Params) (image.Image, error) {
		tp := treatments.Params{
			Normalize: p.Normalize, Dark: p.Dark, Light: p.Light, Mid: p.Mid, MidLow: p.MidLow, MidHigh: p.MidHigh, Levels: p.Levels, LPI: p.LPI, Angle: p.Angle, Dot: p.Dot, Seed: p.Seed, Amount: p.Amount, Contrast: p.ContrastMultiplier, ScrimColor: p.ScrimColor, Direction: p.Direction, Opacity: p.Opacity,
			RegionX: p.RegionX, RegionY: p.RegionY, RegionWidth: p.RegionWidth, RegionHeight: p.RegionHeight, RegionFeather: p.RegionFeather,
		}
		knockoutFrom(p, &tp)
		return fn(treatments.ReserveBefore(img, tp), tp)
	})
}

// knockoutFrom carries the reserved area across.
//
// Both wrappers lift before the operation AND pass the same bounds through, and
// the second half matters as much as the first: the lift decides what the
// operation reads, and the bounds let an operation that auto-levels know the
// white it is looking at was put there deliberately. Without them a
// normalizing screen would treat the knockout as the picture's new highlight
// and rescale the real subject down to make room for it.
func knockoutFrom(p *Params, tp *treatments.Params) {
	tp.KnockoutX, tp.KnockoutY = p.KnockoutX, p.KnockoutY
	tp.KnockoutWidth, tp.KnockoutHeight = p.KnockoutWidth, p.KnockoutHeight
	tp.KnockoutFeather = p.KnockoutFeather
	tp.KnockoutSolid = p.KnockoutSolid
}

// List returns the operation catalog in stable order (for discovery).
func List() []Op {
	ops := make([]Op, 0, len(registry))
	for _, n := range Names() {
		ops = append(ops, registry[n])
	}
	return ops
}

// Has reports whether op is a registered deterministic operation.
func Has(name string) bool {
	_, ok := registry[name]
	return ok
}

// Execute decodes src, runs the named operation with params, and returns the
// encoded result. It is the single entry point the job runner calls; it never
// touches storage, jobs, or HTTP. The guard/ingest check happens upstream at
// the multipart boundary — Execute assumes src is already size-bounded.
func Execute(name string, src []byte, p *Params) (RunResult, error) {
	return ExecuteContext(context.Background(), name, src, p)
}

func ExecuteContext(ctx context.Context, name string, src []byte, p *Params) (RunResult, error) {
	if ctx == nil {
		return RunResult{}, fmt.Errorf("ops: execution context is required")
	}
	if err := ctx.Err(); err != nil {
		return RunResult{}, err
	}
	op, ok := registry[name]
	if !ok {
		return RunResult{}, fmt.Errorf("ops: unknown operation %q", name)
	}
	if p == nil {
		p = &Params{}
	}
	img, meta, err := Decode(src)
	if err != nil {
		return RunResult{}, err
	}
	// Relative spatial parameters resolve here, once, against the decoded
	// geometry — the first point in the pipeline where the image's size is
	// known. Resolving inside each treatment instead would mean nine
	// implementations of one rule.
	b := img.Bounds()
	resolved := ResolveSpatialParams(p, b.Dx(), b.Dy())
	res, err := op.run(RunInput{Context: ctx, Bytes: src, Img: img, Meta: meta, Params: p})
	if err != nil {
		return RunResult{}, err
	}
	res.ResolvedParams = resolved
	return res, nil
}

// pixel adapts a pixel transform (image→image) into a RunFunc by encoding the
// result in the resolved output format with the request's quality options.
func pixel(transform func(image.Image, *Params) (image.Image, error)) RunFunc {
	return func(in RunInput) (RunResult, error) {
		out, err := transform(in.Img, in.Params)
		if err != nil {
			return RunResult{}, err
		}
		format := resolveOutputFormat(in.Params.Format, in.Meta.Format)
		data, err := Encode(out, format, EncodeOptions{Quality: in.Params.Quality, Lossless: in.Params.Lossless})
		if err != nil {
			return RunResult{}, err
		}
		b := out.Bounds()
		return RunResult{Bytes: data, Format: format, Mime: MIMEFor(format), Width: b.Dx(), Height: b.Dy()}, nil
	}
}

// compressRun handles both quality re-encode and target-size compression.
func compressRun(in RunInput) (RunResult, error) {
	if in.Params.TargetBytes > 0 {
		data, format, _, err := compressToTarget(in.Img, in.Params)
		if err != nil {
			return RunResult{}, err
		}
		b := in.Img.Bounds()
		return RunResult{Bytes: data, Format: format, Mime: MIMEFor(format), Width: b.Dx(), Height: b.Dy()}, nil
	}
	// No target size: a plain quality re-encode in the resolved format.
	return pixel(func(img image.Image, _ *Params) (image.Image, error) { return img, nil })(in)
}
