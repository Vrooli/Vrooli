package transfer_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"device-sync-hub/internal/transfer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestImageThumbnailer_DownscalesLargeImage(t *testing.T) {
	data := makePNG(t, 600, 400)
	thumb, mime, ok := transfer.ImageThumbnailer{}.Generate(data, "image/png")
	require.True(t, ok)
	assert.Equal(t, "image/jpeg", mime)

	img, _, err := image.Decode(bytes.NewReader(thumb))
	require.NoError(t, err)
	b := img.Bounds()
	assert.LessOrEqual(t, b.Dx(), 256)
	assert.LessOrEqual(t, b.Dy(), 256)
	// Aspect ratio preserved: the wider side hits the cap.
	assert.Equal(t, 256, b.Dx())
}

func TestImageThumbnailer_SkipsNonImages(t *testing.T) {
	_, _, ok := transfer.ImageThumbnailer{}.Generate([]byte("not an image"), "text/plain")
	assert.False(t, ok)

	// An image MIME with undecodable bytes is also a clean skip, not a panic.
	_, _, ok = transfer.ImageThumbnailer{}.Generate([]byte("garbage"), "image/png")
	assert.False(t, ok)
}
