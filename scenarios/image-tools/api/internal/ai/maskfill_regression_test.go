package ai

import (
	"image"
	"image/color"
	"os"
	"testing"
)

// TestMaskFillPointerRegression runs the 2026-09-15 pointer-removal case: the
// deliberately undersized mask is grown to absorb the pointer and its glow, and
// the filled region must carry no cyan residue and match the accepted result.
func TestMaskFillPointerRegression(t *testing.T) {
	orig := loadFixture(t, "testdata/aquila-original-512.png")
	clean := loadFixture(t, "testdata/aquila-clean-512.png")
	maskImg := loadFixture(t, "testdata/mask-pointer-512.png")
	if orig == nil || clean == nil || maskImg == nil {
		t.Skip("fixtures missing")
	}

	p := DefaultMaskFillParams()
	out := MaskFill(orig, maskImg, p)

	b := orig.Bounds()
	w, h := b.Dx(), b.Dy()
	// Rebuild the grown region exactly as MaskFill does, to scope the checks.
	base := make([]bool, w*h)
	mb := maskImg.Bounds()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			mx, my := x*mb.Dx()/w, y*mb.Dy()/h
			r, g, bl, a := maskImg.At(mb.Min.X+mx, mb.Min.Y+my).RGBA()
			base[y*w+x] = a > 0 && (int(r>>8)+int(g>>8)+int(bl>>8))/3 > 127
		}
	}
	region := dilateMask(base, w, h, p.GrowPx)
	region = growHalo(toNRGBA(orig), region, w, h, p.HaloThreshold, 4)

	// 1. No cyan residue in the grown region.
	residue := 0
	for i, m := range region {
		if !m {
			continue
		}
		c := out.NRGBAAt(i%w, i/w)
		if c.R < 120 && c.G > 120 && c.B > 150 {
			residue++
		}
	}
	if residue > 0 {
		t.Fatalf("%d cyan-residue pixel(s) remain in the filled region", residue)
	}

	// 2. The filled region matches the accepted result.
	diff, total := 0, 0
	for i, m := range region {
		if !m {
			continue
		}
		total++
		a := out.NRGBAAt(i%w, i/w)
		c := color.NRGBAModel.Convert(clean.At(b.Min.X+i%w, b.Min.Y+i/w)).(color.NRGBA)
		if maxDiff(a, c) > 32 {
			diff++
		}
	}
	if total == 0 {
		t.Fatal("grown region is empty")
	}
	if frac := float64(diff) / float64(total); frac > 0.02 {
		t.Fatalf("filled region mismatch %.4f exceeds 0.02", frac)
	}
}

func loadFixture(t *testing.T, path string) image.Image {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := decodeNaturalizeInput(data)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return img
}

func toNRGBA(src image.Image) []color.NRGBA {
	b := src.Bounds()
	out := make([]color.NRGBA, b.Dx()*b.Dy())
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			out[y*b.Dx()+x] = color.NRGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
		}
	}
	return out
}
