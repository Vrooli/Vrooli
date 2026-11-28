package desktopvideo

import (
	"bytes"
	"image"
	"image/jpeg"
)

// EncodeJPEG downsamples an opaque X11 frame and produces a single bounded
// intra frame for HTTP clients through the existing target proxy.
func EncodeJPEG(src *image.RGBA, limit int) ([]byte, error) {
	for _, width := range []int{1280, 960, 640, 480} {
		destination := downsample(src, width)
		var output bytes.Buffer
		if err := jpeg.Encode(&output, destination, &jpeg.Options{Quality: 65}); err != nil {
			return nil, err
		}
		if output.Len() <= limit {
			return output.Bytes(), nil
		}
	}
	return nil, ErrEncode
}

func downsample(src *image.RGBA, maximum int) *image.RGBA {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	largest := max(w, h)
	if largest <= maximum {
		return src
	}
	width, height := max(1, w*maximum/largest), max(1, h*maximum/largest)
	dest := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			dest.SetRGBA(x, y, src.RGBAAt(bounds.Min.X+x*w/width, bounds.Min.Y+y*h/height))
		}
	}
	return dest
}
