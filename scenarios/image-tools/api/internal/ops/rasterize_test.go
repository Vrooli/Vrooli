package ops

import (
	"bytes"
	"image"
	"testing"
)

const rasterizeFixtureSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" width="512" height="512">` +
	`<defs><linearGradient id="g" x1="0" y1="0" x2="0" y2="1">` +
	`<stop offset="0" stop-color="#15243c"/><stop offset="1" stop-color="#0b1728"/></linearGradient></defs>` +
	`<rect x="0" y="0" width="512" height="512" rx="112" fill="url(#g)"/>` +
	`<path fill="#ffffff" d="M100 100H400V400H100Z M200 200V300H300V200Z"/></svg>`

func TestRasterizeExactSizeAndDeterministic(t *testing.T) {
	for _, size := range []int{16, 32, 512} {
		p := &Params{Width: size, Height: size}
		first, err := Rasterize(RunInput{Bytes: []byte(rasterizeFixtureSVG), Params: p})
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		if first.Format != FormatPNG || first.Width != size || first.Height != size {
			t.Fatalf("size %d: got %s %dx%d", size, first.Format, first.Width, first.Height)
		}
		second, err := Rasterize(RunInput{Bytes: []byte(rasterizeFixtureSVG), Params: p})
		if err != nil {
			t.Fatalf("size %d second run: %v", size, err)
		}
		if !bytes.Equal(first.Bytes, second.Bytes) {
			t.Fatalf("size %d: rasterization is not deterministic", size)
		}
	}
}

func TestRasterizeRejectsOversize(t *testing.T) {
	_, err := Rasterize(RunInput{Bytes: []byte(rasterizeFixtureSVG), Params: &Params{Width: MaxSVGRasterDimension + 1, Height: 10}})
	if err == nil {
		t.Fatal("expected an oversize error")
	}
}

func TestRasterizeFilterFreeNeverNeedsChrome(t *testing.T) {
	if feats := HighFidelitySVGFeatures([]byte(rasterizeFixtureSVG)); len(feats) != 0 {
		t.Fatalf("fixture should be filter-free, got %v", feats)
	}
	filtered := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><filter id="f"/><rect width="4" height="4" filter="url(#f)"/></svg>`)
	if feats := HighFidelitySVGFeatures(filtered); len(feats) == 0 {
		t.Fatal("expected the filter SVG to require the high-fidelity rasterizer")
	}
}

func TestRasterizeRejectsNonSVG(t *testing.T) {
	if _, err := Rasterize(RunInput{Bytes: []byte("not an svg"), Params: &Params{Width: 8, Height: 8}}); err == nil {
		t.Fatal("expected non-SVG input to be rejected")
	}
}

func TestRasterizeOpaqueBackground(t *testing.T) {
	res, err := Rasterize(RunInput{Bytes: []byte(rasterizeFixtureSVG), Params: &Params{Width: 32, Height: 32, Background: "#ffffff"}})
	if err != nil {
		t.Fatalf("rasterize with background: %v", err)
	}
	img, _, err := Decode(res.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if !opaqueImage(img) {
		t.Fatal("expected an opaque result when a background is given")
	}
}

func opaqueImage(img image.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0xffff {
				return false
			}
		}
	}
	return true
}
