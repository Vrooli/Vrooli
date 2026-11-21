package assets

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writePNG(t *testing.T, path string, w, h int, c color.RGBA) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, img))
	require.NoError(t, f.Close())
}

func TestGenerateLogoDerivatives(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "logos"), 0o755))
	writePNG(t, filepath.Join(tmp, "logos", "logo.png"), 800, 400, color.RGBA{200, 20, 20, 255})

	svc := &Service{uploadDir: tmp}
	derivatives, err := svc.generateDerivatives(filepath.Join(tmp, "logos", "logo.png"), "logos/logo.png", "image/png", "logo")
	require.NoError(t, err)

	for _, key := range []string{"logo_512", "logo_256", "logo_128", "logo_icon", "favicon_32", "apple_touch_180"} {
		path, ok := derivatives[key]
		require.Truef(t, ok, "missing derivative %s", key)
		info, err := os.Stat(filepath.Join(tmp, path))
		require.NoError(t, err)
		require.NotZero(t, info.Size())
	}
	f, err := os.Open(filepath.Join(tmp, derivatives["logo_512"]))
	require.NoError(t, err)
	defer f.Close()
	img, err := png.Decode(f)
	require.NoError(t, err)
	require.Equal(t, 512, img.Bounds().Dx())
	require.Equal(t, 512, img.Bounds().Dy())
}

func TestGenerateLogoDerivativesJpeg(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "logos"), 0o755))
	img := image.NewRGBA(image.Rect(0, 0, 600, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 600; x++ {
			img.Set(x, y, color.RGBA{120, 80, 10, 255})
		}
	}
	f, err := os.Create(filepath.Join(tmp, "logos", "logo.jpg"))
	require.NoError(t, err)
	require.NoError(t, jpeg.Encode(f, img, &jpeg.Options{Quality: 80}))
	require.NoError(t, f.Close())

	svc := &Service{uploadDir: tmp}
	derivatives, err := svc.generateDerivatives(filepath.Join(tmp, "logos", "logo.jpg"), "logos/logo.jpg", "image/jpeg", "logo")
	require.NoError(t, err)
	require.Contains(t, derivatives, "logo_256")
	require.Contains(t, derivatives, "favicon_32")
	require.NotEmpty(t, derivatives["logo_icon"])
}

func TestGenerateDerivativesSvgFallback(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "logos"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "logos", "logo.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100"></svg>`), 0o644))

	svc := &Service{uploadDir: tmp}
	derivatives, err := svc.generateDerivatives(filepath.Join(tmp, "logos", "logo.svg"), "logos/logo.svg", "image/svg+xml", "logo")
	require.NoError(t, err)
	for _, key := range []string{"favicon_32", "apple_touch_180", "logo_icon"} {
		require.Equal(t, "logos/logo.svg", derivatives[key])
	}
}

func TestGenerateFaviconDerivatives(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "favicons"), 0o755))
	writePNG(t, filepath.Join(tmp, "favicons", "favicon.png"), 64, 64, color.RGBA{10, 200, 10, 255})

	svc := &Service{uploadDir: tmp}
	derivatives, err := svc.generateDerivatives(filepath.Join(tmp, "favicons", "favicon.png"), "favicons/favicon.png", "image/png", "favicon")
	require.NoError(t, err)
	for _, key := range []string{"favicon_64", "favicon_32", "favicon_16", "apple_touch_180"} {
		require.Contains(t, derivatives, key)
	}
}

func TestGenerateOgDerivatives(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "og-images"), 0o755))
	writePNG(t, filepath.Join(tmp, "og-images", "og.png"), 800, 800, color.RGBA{50, 50, 220, 255})

	svc := &Service{uploadDir: tmp}
	derivatives, err := svc.generateDerivatives(filepath.Join(tmp, "og-images", "og.png"), "og-images/og.png", "image/png", "og_image")
	require.NoError(t, err)
	path, ok := derivatives["og_image_1200x630"]
	require.True(t, ok)
	f, err := os.Open(filepath.Join(tmp, path))
	require.NoError(t, err)
	defer f.Close()
	img, err := png.Decode(f)
	require.NoError(t, err)
	require.Equal(t, 1200, img.Bounds().Dx())
	require.Equal(t, 630, img.Bounds().Dy())
}
