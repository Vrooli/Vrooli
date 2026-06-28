package ops

import (
	"fmt"
	"image"
	"sort"
)

// RunInput is what an operation receives: the original encoded bytes, the
// decoded image, its source metadata, and the typed parameters.
type RunInput struct {
	Bytes  []byte
	Img    image.Image
	Meta   Meta
	Params *Params
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
}

// RunFunc executes one operation end-to-end (decode already done by Execute).
type RunFunc func(in RunInput) (RunResult, error)

// Category groups operations for discovery/UI.
type Category string

const (
	CategoryGeometry Category = "geometry"
	CategoryColor    Category = "color"
	CategoryFormat   Category = "format"
	CategoryCompose  Category = "compose"
	CategoryMetadata Category = "metadata"
)

// Op is a registered deterministic operation.
type Op struct {
	Name     string
	Category Category
	Summary  string
	run      RunFunc
}

// registry is the canonical deterministic-op table. It is the single source of
// truth consumed by the job runner (execution), the OpsService (discovery), and
// the CLI (command surface).
var registry = func() map[string]Op {
	ops := []Op{
		{Name: "resize", Category: CategoryGeometry, Summary: "Resize (fit/fill/stretch) preserving or forcing aspect", run: pixel(Resize)},
		{Name: "crop", Category: CategoryGeometry, Summary: "Crop a rectangle or a gravity-anchored region", run: pixel(Crop)},
		{Name: "rotate", Category: CategoryGeometry, Summary: "Rotate by 90° steps or an arbitrary angle", run: pixel(Rotate)},
		{Name: "flip", Category: CategoryGeometry, Summary: "Mirror horizontally or vertically", run: pixel(Flip)},
		{Name: "deskew", Category: CategoryGeometry, Summary: "Auto-straighten a skewed scan/document", run: pixel(Deskew)},
		{Name: "thumbnail", Category: CategoryGeometry, Summary: "Produce a fill-cropped thumbnail", run: pixel(Thumbnail)},
		{Name: "canvas", Category: CategoryGeometry, Summary: "Pad/extend onto a background canvas", run: pixel(Canvas)},
		{Name: "adjust", Category: CategoryColor, Summary: "Brightness/contrast/gamma/saturation/hue", run: pixel(Adjust)},
		{Name: "filter", Category: CategoryColor, Summary: "Grayscale/sepia/invert/blur/sharpen", run: pixel(Filter)},
		{Name: "canny", Category: CategoryColor, Summary: "Deterministic Canny edge map (ControlNet conditioning preprocessor)", run: pixel(Canny)},
		{Name: "convert", Category: CategoryFormat, Summary: "Convert to another image format", run: pixel(convertClone)},
		{Name: "compress", Category: CategoryFormat, Summary: "Re-encode at a quality or to a target file size", run: compressRun},
		{Name: "overlay", Category: CategoryCompose, Summary: "Composite a watermark image or text", run: pixel(Overlay)},
		{Name: "metadata", Category: CategoryMetadata, Summary: "Read, strip, or auto-orient image metadata", run: metadataRun},
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
	return op.run(RunInput{Bytes: src, Img: img, Meta: meta, Params: p})
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
