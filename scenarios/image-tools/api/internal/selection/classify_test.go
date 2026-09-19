package selection

import (
	"image"
	"image/color"
	"testing"
)

// classifyWhole runs the classifier over an entire solid/painted image by
// segmenting in AUTO-equivalent fashion: it marks every pixel selected.
func classifyAll(img *image.NRGBA, area float64) (string, float64) {
	b := img.Bounds()
	m := newMask(b.Dx(), b.Dy())
	for i := range m.sel {
		m.sel[i] = true
	}
	return Classify(img, m, area)
}

func TestClassifyBackground(t *testing.T) {
	// A large region touching all borders.
	img := solidImage(60, 60, color.NRGBA{R: 120, G: 120, B: 120, A: 255})
	class, conf := classifyAll(img, 0.9)
	if class != ClassBackground {
		t.Errorf("class = %q, want %q", class, ClassBackground)
	}
	if conf <= 0 {
		t.Error("confidence should be > 0")
	}
}

func TestClassifySky(t *testing.T) {
	// Bright bluish region in the TOP of the frame (centroid high), not touching
	// all borders so it isn't classified as background.
	img := solidImage(60, 60, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	m := newMask(60, 60)
	for y := 0; y < 20; y++ {
		for x := 10; x < 50; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 150, G: 190, B: 240, A: 255})
			m.set(x, y, true)
		}
	}
	class, _ := Classify(img, m, 0.2)
	if class != ClassSky {
		t.Errorf("class = %q, want %q", class, ClassSky)
	}
}

func TestClassifyPerson(t *testing.T) {
	// A skin-tone region (Kovac daylight rule: R>95,G>40,B>20, R>G>B).
	img := solidImage(60, 60, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	m := newMask(60, 60)
	skin := color.NRGBA{R: 230, G: 170, B: 140, A: 255}
	for y := 25; y < 45; y++ {
		for x := 25; x < 45; x++ {
			img.SetNRGBA(x, y, skin)
			m.set(x, y, true)
		}
	}
	class, _ := Classify(img, m, 0.11)
	if class != ClassPerson {
		t.Errorf("class = %q, want %q", class, ClassPerson)
	}
}

func TestClassifyFoliage(t *testing.T) {
	img := solidImage(60, 60, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	m := newMask(60, 60)
	green := color.NRGBA{R: 40, G: 150, B: 50, A: 255}
	for y := 30; y < 50; y++ {
		for x := 20; x < 40; x++ {
			img.SetNRGBA(x, y, green)
			m.set(x, y, true)
		}
	}
	class, _ := Classify(img, m, 0.11)
	if class != ClassFoliage {
		t.Errorf("class = %q, want %q", class, ClassFoliage)
	}
}

func TestClassifyObjectFallback(t *testing.T) {
	// A small saturated red region low in the frame — not sky, skin, or green.
	img := solidImage(60, 60, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	m := newMask(60, 60)
	for y := 40; y < 55; y++ {
		for x := 40; x < 55; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 210, G: 30, B: 30, A: 255})
			m.set(x, y, true)
		}
	}
	class, _ := Classify(img, m, 0.06)
	if class != ClassObject {
		t.Errorf("class = %q, want %q", class, ClassObject)
	}
}

func TestClassifyEmptyMask(t *testing.T) {
	img := solidImage(20, 20, blue)
	m := newMask(20, 20)
	class, conf := Classify(img, m, 0)
	if class != ClassObject || conf <= 0 {
		t.Errorf("empty mask: class=%q conf=%.2f, want object/>0", class, conf)
	}
}

func TestIsSkinRejectsNonSkin(t *testing.T) {
	if isSkin(30, 60, 220) { // blue
		t.Error("blue classified as skin")
	}
	if isSkin(40, 150, 50) { // green
		t.Error("green classified as skin")
	}
	if !isSkin(230, 170, 140) { // a skin tone
		t.Error("skin tone not recognized")
	}
}
