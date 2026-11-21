package percepthash

import (
	"image"
	"image/color"
	"testing"
)

// gradient returns a deterministic n×n image with a horizontal gradient and a
// bright square, so pHash has real structure to hash.
func gradient(n int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			v := uint8((x * 255) / max(1, n-1))
			c := color.NRGBA{R: v, G: v / 2, B: 255 - v, A: 255}
			if x > n/2 && y > n/2 {
				c = color.NRGBA{R: 250, G: 250, B: 250, A: 255}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func TestPHashIsStableAcrossScale(t *testing.T) {
	base := PHash(gradient(64))
	scaled := PHash(gradient(128))
	if d := Hamming(base, scaled); d > 8 {
		t.Errorf("pHash Hamming distance across a 2x scale = %d, want <= 8 (perceptual hash should be scale-robust)", d)
	}
}

func TestPHashDistinguishesDifferentImages(t *testing.T) {
	a := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	b := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			a.Set(x, y, color.NRGBA{R: 255, A: 255})
			if x < 16 {
				b.Set(x, y, color.NRGBA{B: 255, A: 255})
			} else {
				b.Set(x, y, color.NRGBA{G: 255, A: 255})
			}
		}
	}
	if d := Hamming(PHash(a), PHash(b)); d < 8 {
		t.Errorf("pHash distance between clearly different images = %d, want >= 8", d)
	}
}

func TestAHashAndPHashAreDeterministic(t *testing.T) {
	img := gradient(48)
	if PHash(img) != PHash(img) {
		t.Error("PHash is not deterministic for the same image")
	}
	if AHash(img) != AHash(img) {
		t.Error("AHash is not deterministic for the same image")
	}
}

func TestMedian(t *testing.T) {
	if got := Median(nil); got != 0 {
		t.Errorf("Median(nil) = %v, want 0", got)
	}
	if got := Median([]float64{3, 1, 2}); got != 2 {
		t.Errorf("Median([3 1 2]) = %v, want 2", got)
	}
	if got := Median([]float64{4, 1, 3, 2}); got != 2.5 {
		t.Errorf("Median([4 1 3 2]) = %v, want 2.5", got)
	}
	// Must not mutate the caller's slice.
	in := []float64{3, 1, 2}
	_ = Median(in)
	if in[0] != 3 || in[1] != 1 || in[2] != 2 {
		t.Errorf("Median mutated its input: %v", in)
	}
}

func TestHammingAndPopcount(t *testing.T) {
	if got := Popcount(0); got != 0 {
		t.Errorf("Popcount(0) = %d, want 0", got)
	}
	if got := Popcount(^uint64(0)); got != 64 {
		t.Errorf("Popcount(all ones) = %d, want 64", got)
	}
	if got := Hamming(0b1010, 0b0011); got != 2 {
		t.Errorf("Hamming(1010, 0011) = %d, want 2", got)
	}
	if Hamming(7, 3) != Hamming(3, 7) {
		t.Error("Hamming must be symmetric")
	}
}
