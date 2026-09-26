package transfer

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"strings"

	// Register decoders so image.Decode handles the common upload formats. We
	// re-encode every thumbnail as JPEG, so only the decoders are needed.
	_ "image/gif"
	_ "image/png"
)

// Thumbnailer generates a small preview image for an uploaded file. It is a
// seam so the upload handler can substitute a deterministic generator in tests
// and so a future swap to a higher-quality scaler (golang.org/x/image) is a
// one-line wiring change, not a handler rewrite.
type Thumbnailer interface {
	// Generate returns a JPEG thumbnail for data when mime is a supported image
	// type and decoding succeeds. ok=false (with nil bytes) means "no thumbnail"
	// — a non-image, an unsupported format, or a decode failure — which is NOT
	// an error: the item is simply stored without a thumbnail.
	Generate(data []byte, mime string) (thumb []byte, thumbMIME string, ok bool)
}

// thumbMaxEdge is the longest side of a generated thumbnail in pixels.
const thumbMaxEdge = 256

// ImageThumbnailer is the production Thumbnailer: pure-stdlib decode +
// nearest-neighbor downscale + JPEG encode. Nearest-neighbor is intentionally
// dependency-free; a thumbnail is a glanceable preview, not an archival render.
type ImageThumbnailer struct{}

// Compile-time guarantee.
var _ Thumbnailer = ImageThumbnailer{}

func (ImageThumbnailer) Generate(data []byte, mime string) ([]byte, string, bool) {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(mime)), "image/") {
		return nil, "", false
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", false
	}
	dst := downscale(src, thumbMaxEdge)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 80}); err != nil {
		return nil, "", false
	}
	return buf.Bytes(), "image/jpeg", true
}

// downscale returns an image whose longest edge is at most maxEdge, preserving
// aspect ratio via nearest-neighbor sampling. An image already within bounds is
// returned unscaled.
func downscale(src image.Image, maxEdge int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return src
	}
	if w <= maxEdge && h <= maxEdge {
		return src
	}
	scale := float64(maxEdge) / float64(maxBound(w, h))
	dw := int(float64(w) * scale)
	dh := int(float64(h) * scale)
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		sy := b.Min.Y + int(float64(y)/float64(dh)*float64(h))
		for x := 0; x < dw; x++ {
			sx := b.Min.X + int(float64(x)/float64(dw)*float64(w))
			dst.Set(x, y, color.RGBAModel.Convert(src.At(sx, sy)))
		}
	}
	return dst
}

func maxBound(a, b int) int {
	if a > b {
		return a
	}
	return b
}
