package ops

import (
	"image"
	"image/color"
	"testing"
)

// testImage builds a deterministic 16x12 gradient with an alpha corner so
// format round-trips exercise color + (where supported) alpha.
func testImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 16, 12))
	for y := 0; y < 12; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 16), G: uint8(y * 21), B: 128, A: 255})
		}
	}
	return img
}

func TestEncodeDecodeRoundTripAllFormats(t *testing.T) {
	src := testImage()
	for _, format := range EncodableFormats {
		format := format
		t.Run(format, func(t *testing.T) {
			data, err := Encode(src, format, EncodeOptions{Quality: 90})
			if err != nil {
				t.Fatalf("Encode(%s): %v", format, err)
			}
			if len(data) == 0 {
				t.Fatalf("Encode(%s): empty output", format)
			}
			img, meta, err := Decode(data)
			if err != nil {
				t.Fatalf("Decode(%s): %v", format, err)
			}
			if got := img.Bounds(); got.Dx() != 16 || got.Dy() != 12 {
				t.Fatalf("Decode(%s): size = %dx%d, want 16x12", format, got.Dx(), got.Dy())
			}
			if meta.Format != normalizeFormat(format) {
				t.Fatalf("Decode(%s): meta.Format = %q, want %q", format, meta.Format, normalizeFormat(format))
			}
		})
	}
}

func TestDecodeSVGRasterizes(t *testing.T) {
	svg := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 30" width="40" height="30"><rect width="40" height="30" fill="#3366cc"/></svg>`)
	img, meta, err := Decode(svg)
	if err != nil {
		t.Fatalf("Decode(svg): %v", err)
	}
	if meta.Format != FormatSVG {
		t.Fatalf("meta.Format = %q, want svg", meta.Format)
	}
	if b := img.Bounds(); b.Dx() != 40 || b.Dy() != 30 {
		t.Fatalf("svg raster size = %dx%d, want 40x30", b.Dx(), b.Dy())
	}
}

func TestEncodeUnsupportedFormat(t *testing.T) {
	if _, err := Encode(testImage(), FormatHEIC, EncodeOptions{}); err == nil {
		t.Fatal("expected HEIC encode to be unsupported (decode-only format)")
	}
	if _, err := Encode(testImage(), FormatSVG, EncodeOptions{}); err == nil {
		t.Fatal("expected SVG encode to be unsupported (raster import only)")
	}
}

func TestDecodeEmpty(t *testing.T) {
	if _, _, err := Decode(nil); err == nil {
		t.Fatal("expected error decoding empty input")
	}
}

func TestFormatHelpers(t *testing.T) {
	cases := map[string]string{"jpg": FormatJPEG, ".JPEG": FormatJPEG, "tif": FormatTIFF, "heif": FormatHEIC, "png": FormatPNG, "x": ""}
	for ext, want := range cases {
		if got := FormatFromExt(ext); got != want {
			t.Errorf("FormatFromExt(%q) = %q, want %q", ext, got, want)
		}
	}
	if !CanEncode("jpg") {
		t.Error("CanEncode(jpg) = false")
	}
	if CanEncode(FormatHEIC) {
		t.Error("CanEncode(heic) = true, want false")
	}
	if MIMEFor(FormatWebP) != "image/webp" {
		t.Errorf("MIMEFor(webp) = %q", MIMEFor(FormatWebP))
	}
}
