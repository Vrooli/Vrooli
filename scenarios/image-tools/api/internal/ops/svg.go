package ops

import (
	"bytes"
	"fmt"
	"image"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// MaxSVGRasterDimension caps either side of an SVG rasterization so a tiny SVG
// declaring an enormous viewBox can't allocate an unbounded raster (the
// vector-format analogue of the decompression-bomb guard).
const MaxSVGRasterDimension = 8192

// rasterizeSVG renders SVG bytes to an *image.RGBA. width/height of 0 use the
// SVG's intrinsic viewBox size; a non-zero pair scales to that target. The
// result is bounded by MaxSVGRasterDimension on each side.
func rasterizeSVG(data []byte, width, height int) (image.Image, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: svg parse: %v", ErrDecode, err)
	}

	w := int(icon.ViewBox.W)
	h := int(icon.ViewBox.H)
	if width > 0 {
		w = width
	}
	if height > 0 {
		h = height
	}
	if w <= 0 || h <= 0 {
		// No intrinsic size and no target: fall back to a sane default canvas.
		w, h = 512, 512
	}
	if w > MaxSVGRasterDimension || h > MaxSVGRasterDimension {
		return nil, fmt.Errorf("%w: svg raster %dx%d exceeds %d px per side", ErrDecode, w, h, MaxSVGRasterDimension)
	}

	icon.SetTarget(0, 0, float64(w), float64(h))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	raster := rasterx.NewDasher(w, h, scanner)
	icon.Draw(raster, 1.0)
	return img, nil
}
