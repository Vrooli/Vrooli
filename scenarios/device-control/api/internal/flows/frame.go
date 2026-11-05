package flows

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
)

// PrepareFrame decodes the caller capture, records its original device
// dimensions, and downsizes it before the gateway boundary. The caller owns
// this policy; the gateway receives only the submitted dimensions.
func PrepareFrame(raw []byte, mediaType string, maxDimension int) (Frame, error) {
	if len(raw) == 0 {
		return Frame{}, fmt.Errorf("%w: empty image", ErrInvalidFrame)
	}
	decoded, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Frame{}, fmt.Errorf("%w: decode image: %v", ErrInvalidFrame, err)
	}
	original := decoded.Bounds().Size()
	if original.X <= 0 || original.Y <= 0 {
		return Frame{}, fmt.Errorf("%w: image has no dimensions", ErrInvalidFrame)
	}
	if maxDimension <= 0 {
		maxDimension = defaultMaxDimension
	}
	if original.X <= maxDimension && original.Y <= maxDimension {
		return Frame{Bytes: raw, MediaType: mediaType, Width: original.X, Height: original.Y, OriginalWidth: original.X, OriginalHeight: original.Y}, nil
	}

	scale := float64(maxDimension) / float64(original.X)
	if original.Y > original.X {
		scale = float64(maxDimension) / float64(original.Y)
	}
	width := max(1, int(float64(original.X)*scale))
	height := max(1, int(float64(original.Y)*scale))
	resized := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sx := x * original.X / width
			sy := y * original.Y / height
			resized.Set(x, y, decoded.At(sx, sy))
		}
	}
	var out bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		if err := jpeg.Encode(&out, resized, &jpeg.Options{Quality: 90}); err != nil {
			return Frame{}, fmt.Errorf("%w: encode resized jpeg: %v", ErrInvalidFrame, err)
		}
		mediaType = "image/jpeg"
	default:
		if err := png.Encode(&out, resized); err != nil {
			return Frame{}, fmt.Errorf("%w: encode resized png: %v", ErrInvalidFrame, err)
		}
		mediaType = "image/png"
	}
	return Frame{Bytes: out.Bytes(), MediaType: mediaType, Width: width, Height: height, OriginalWidth: original.X, OriginalHeight: original.Y}, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
