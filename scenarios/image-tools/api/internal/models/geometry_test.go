package models

import "testing"

// The registry declares a native resolution per model; the architecture decides
// the stride. Both matter, and a caller that guessed either got a canvas the
// model draws badly — which is the defect this type replaces.
func TestGeometryComesFromTheModelNotAConstant(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	cases := []struct {
		id                  string
		wantW, wantH, wantQ int
	}{
		{"sd-1.5", 512, 512, 64},
		{"sdxl-1.0", 1024, 1024, 64},
		{"flux-1-schnell", 1024, 1024, 16},
	}
	for _, tc := range cases {
		m, ok := reg.ByID(tc.id)
		if !ok {
			t.Fatalf("model %q missing from the seed registry", tc.id)
		}
		g := m.Geometry()
		if g.NativeWidth != tc.wantW || g.NativeHeight != tc.wantH {
			t.Errorf("%s native = %dx%d, want %dx%d", tc.id, g.NativeWidth, g.NativeHeight, tc.wantW, tc.wantH)
		}
		if g.SizeQuantum != tc.wantQ {
			t.Errorf("%s quantum = %d, want %d", tc.id, g.SizeQuantum, tc.wantQ)
		}
		if !g.Declared() {
			t.Errorf("%s declares a native resolution but Declared() is false", tc.id)
		}
	}
}

// "model-dependent" is not a resolution. A cloud entry must report no declared
// geometry rather than a parsed zero, because a caller that treats zero as a
// measurement quantises against it and asks for a 0x0 canvas.
func TestACloudModelDeclaresNoGeometry(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	m, ok := reg.ByID("openrouter-image")
	if !ok {
		t.Skip("no BYOK cloud entry in the seed registry")
	}
	g := m.Geometry()
	if g.Declared() {
		t.Fatalf("cloud model reports a declared geometry %+v; the provider owns it", g)
	}
	// The provider owns geometry, so the delivery size passes through intact.
	w, h := g.Fit(1440, 720)
	if w != 1440 || h != 720 {
		t.Fatalf("Fit(1440,720) = %dx%d, want the delivery size unchanged", w, h)
	}
}

// Fit is the replacement for three hardcoded SD-1.5 constants. The properties
// that matter are the ones those constants existed to protect: the short edge
// stays at native, both axes land on the stride, and the long edge is capped so
// the model extends the composition instead of repeating it.
func TestFitHoldsTheModelsDrawableCanvas(t *testing.T) {
	sd := Geometry{NativeWidth: 512, NativeHeight: 512, SizeQuantum: 64, MaxEdge: 768}
	xl := Geometry{NativeWidth: 1024, NativeHeight: 1024, SizeQuantum: 64, MaxEdge: 1536}

	for _, tc := range []struct {
		name         string
		g            Geometry
		dw, dh       int
		wantW, wantH int
	}{
		{"square delivery stays native", sd, 800, 800, 512, 512},
		{"2:1 hero caps the long edge", sd, 1440, 720, 768, 512},
		{"portrait caps the tall edge", sd, 390, 844, 512, 768},
		{"sdxl draws its own larger canvas", xl, 1440, 720, 1536, 1024},
	} {
		w, h := tc.g.Fit(tc.dw, tc.dh)
		if w != tc.wantW || h != tc.wantH {
			t.Errorf("%s: Fit(%d,%d) = %dx%d, want %dx%d", tc.name, tc.dw, tc.dh, w, h, tc.wantW, tc.wantH)
		}
		if w%tc.g.SizeQuantum != 0 || h%tc.g.SizeQuantum != 0 {
			t.Errorf("%s: %dx%d is off the %dpx latent stride", tc.name, w, h, tc.g.SizeQuantum)
		}
	}
}
