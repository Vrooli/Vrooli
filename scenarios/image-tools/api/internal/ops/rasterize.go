package ops

import (
	"fmt"
	"image"
	"image/draw"
)

// Rasterize renders an SVG at an exact pixel size. Unlike every other op it
// reads the raw source bytes rather than the decoded image: Decode() rasterizes
// an SVG at its intrinsic size, which discards the target geometry this op
// exists to apply. The renderer is pure-Go oksvg unless the SVG uses a feature
// oksvg silently drops, in which case rasterizeSVG escalates to headless Chrome
// or fails by name.
func Rasterize(in RunInput) (RunResult, error) {
	if !looksLikeSVG(in.Bytes) {
		return RunResult{}, fmt.Errorf("%w: rasterize expects SVG input", ErrDecode)
	}
	w, h := in.Params.Width, in.Params.Height
	if w <= 0 || h <= 0 {
		return RunResult{}, fmt.Errorf("ops: rasterize requires a positive width and height")
	}
	if w > MaxSVGRasterDimension || h > MaxSVGRasterDimension {
		return RunResult{}, fmt.Errorf("%w: svg raster %dx%d exceeds %d px per side", ErrDecode, w, h, MaxSVGRasterDimension)
	}
	img, err := rasterizeSVG(in.Bytes, w, h)
	if err != nil {
		return RunResult{}, err
	}
	if in.Params.Background != "" {
		img, err = flattenOnto(img, in.Params.Background)
		if err != nil {
			return RunResult{}, err
		}
	}
	data, err := Encode(img, FormatPNG, EncodeOptions{})
	if err != nil {
		return RunResult{}, err
	}
	b := img.Bounds()
	return RunResult{Bytes: data, Format: FormatPNG, Mime: MIMEFor(FormatPNG), Width: b.Dx(), Height: b.Dy()}, nil
}

// flattenOnto composites img over an opaque background colour, returning a PNG
// whose alpha channel is fully opaque. It is how a full-bleed or maskable
// target gets an opaque ground.
func flattenOnto(img image.Image, hex string) (image.Image, error) {
	bg, err := parseHexColor(hex)
	if err != nil {
		return nil, err
	}
	bg.A = 255
	out := image.NewNRGBA(img.Bounds())
	draw.Draw(out, out.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	draw.Draw(out, out.Bounds(), img, img.Bounds().Min, draw.Over)
	return out, nil
}
