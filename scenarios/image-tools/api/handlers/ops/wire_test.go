package ops

import (
	"testing"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
	"google.golang.org/protobuf/encoding/protojson"
)

// TestEveryEmittedParameterSurvivesTheWire is the gate that would have caught a
// defect shipped on 2026-08-12: backdrop-studio styles requested "normalize"
// and brand inks on the Tier-2 screens, but neither existed on the proto
// messages. protojson rejects unknown fields, so those styles did not degrade
// quietly — they failed the render with a 400. The unit suites never saw it
// because they run against a fake executor that skips the REST edge.
//
// Every case below is a parameter object a real caller sends. If a treatment
// grows a knob and the proto is not extended, this fails at the boundary rather
// than in production.
func TestEveryEmittedParameterSurvivesTheWire(t *testing.T) {
	cases := map[string]string{
		"duotone":          `{"duotone":{"dark":"#1B3FD8","light":"#EDE6D2","mid":"#0A1F6E","midLow":0.38,"midHigh":0.62,"normalize":true}}`,
		"posterize":        `{"posterize":{"levels":7,"dark":"#1B3FD8","light":"#EDE6D2","normalize":true}}`,
		"halftone":         `{"halftone":{"lpi":72,"angle":15,"dot":"circle","dark":"#1B3FD8","light":"#EDE6D2","normalize":true}}`,
		"dither_ordered":   `{"ditherOrdered":{"dark":"#1B3FD8","light":"#EDE6D2","normalize":true}}`,
		"dither_diffusion": `{"ditherDiffusion":{"dark":"#1B3FD8","light":"#EDE6D2","normalize":true}}`,
		"grain":            `{"grain":{"seed":11,"amount":0.05,"contrastMultiplier":1.0}}`,
		"scrim":            `{"scrim":{"color":"#1B3FD8","opacity":0.55,"direction":"left","regionX":0.06,"regionY":0.1,"regionWidth":0.42,"regionHeight":0.3,"regionFeather":0.08}}`,
		"line_screen":      `{"lineScreen":{"spacing":6,"angle":45,"dark":"#1B3FD8","light":"#EDE6D2","normalize":true}}`,
		"stipple":          `{"stipple":{"spacing":7,"seed":19,"dark":"#1B3FD8","light":"#EDE6D2","normalize":true}}`,
		"engraving":        `{"engraving":{"spacing":7,"dark":"#1B3FD8","light":"#EDE6D2","normalize":true}}`,
		"ascii_mosaic":     `{"asciiMosaic":{"blockSize":7,"dark":"#1B3FD8","light":"#EDE6D2","normalize":true}}`,
		"aberration":       `{"aberration":{"distance":9}}`,
		"bloom":            `{"bloom":{"threshold":0.62,"radius":18}}`,
		"displacement":     `{"displacement":{"amplitude":18,"spacing":26,"seed":5}}`,
		"pixel_sort":       `{"pixelSort":{"threshold":0.55,"axis":"horizontal"}}`,
		"curve":            `{"curve":{"exponent":1.4}}`,
		"defocus":          `{"defocus":{"radius":8,"bladeCount":6}}`,
		"motion_blur":      `{"motionBlur":{"distance":24,"angle":20}}`,
		"composite":        `{"composite":{"width":1440,"height":720,"background":"#EDE6D2","plates":[{"name":"sky","depth":0,"blend":"normal","opacity":1},{"name":"ink","depth":1,"blend":"multiply","opacity":0.85},{"name":"glow","depth":2,"blend":"screen","opacity":0.4}]}}`,
	}
	for op, raw := range cases {
		op, raw := op, raw
		t.Run(op, func(t *testing.T) {
			pb := &opsv1.OpParams{}
			if err := protojson.Unmarshal([]byte(raw), pb); err != nil {
				t.Fatalf("%s params rejected at the wire: %v\npayload: %s", op, err, raw)
			}
			p, err := translateParams(op, pb)
			if err != nil {
				t.Fatalf("%s translate: %v", op, err)
			}
			if p == nil {
				t.Fatalf("%s produced nil params", op)
			}
		})
	}
}

// TestNormalizeReachesTheEngine pins the specific field end to end, since a
// silently-dropped normalize would degrade output rather than fail loudly.
func TestNormalizeReachesTheEngine(t *testing.T) {
	for _, tc := range []struct{ op, raw string }{
		{"duotone", `{"duotone":{"normalize":true}}`},
		{"posterize", `{"posterize":{"levels":5,"normalize":true}}`},
		{"halftone", `{"halftone":{"lpi":48,"normalize":true}}`},
		{"dither_ordered", `{"ditherOrdered":{"normalize":true}}`},
		{"dither_diffusion", `{"ditherDiffusion":{"normalize":true}}`},
		{"line_screen", `{"lineScreen":{"spacing":6,"normalize":true}}`},
		{"stipple", `{"stipple":{"spacing":7,"normalize":true}}`},
		{"engraving", `{"engraving":{"spacing":7,"normalize":true}}`},
		{"ascii_mosaic", `{"asciiMosaic":{"blockSize":7,"normalize":true}}`},
	} {
		pb := &opsv1.OpParams{}
		if err := protojson.Unmarshal([]byte(tc.raw), pb); err != nil {
			t.Fatalf("%s: %v", tc.op, err)
		}
		p, err := translateParams(tc.op, pb)
		if err != nil {
			t.Fatalf("%s: %v", tc.op, err)
		}
		if !p.Normalize {
			t.Errorf("%s: normalize=true did not reach the engine", tc.op)
		}
	}
}

// TestBrandInksReachTheTier2Screens pins the palette lock for the ink-on-paper
// screens. Without these the catalog could name a brand-locked style that
// rendered with a hardcoded ink.
func TestBrandInksReachTheTier2Screens(t *testing.T) {
	for _, tc := range []struct{ op, raw string }{
		{"line_screen", `{"lineScreen":{"spacing":6,"dark":"#1B3FD8","light":"#EDE6D2"}}`},
		{"stipple", `{"stipple":{"spacing":7,"dark":"#1B3FD8","light":"#EDE6D2"}}`},
		{"engraving", `{"engraving":{"spacing":7,"dark":"#1B3FD8","light":"#EDE6D2"}}`},
		{"ascii_mosaic", `{"asciiMosaic":{"blockSize":7,"dark":"#1B3FD8","light":"#EDE6D2"}}`},
	} {
		pb := &opsv1.OpParams{}
		if err := protojson.Unmarshal([]byte(tc.raw), pb); err != nil {
			t.Fatalf("%s: %v", tc.op, err)
		}
		p, err := translateParams(tc.op, pb)
		if err != nil {
			t.Fatalf("%s: %v", tc.op, err)
		}
		if p.Dark != "#1B3FD8" || p.Light != "#EDE6D2" {
			t.Errorf("%s: brand inks did not reach the engine (dark=%q light=%q)", tc.op, p.Dark, p.Light)
		}
	}
}

// The compositor's plate list has to survive the wire with every field intact,
// not merely parse. A plate whose blend or depth was dropped in translation
// composites into a different picture and nothing downstream can see it — which
// is the same class of defect this file exists to catch.
func TestEveryCompositePlateFieldReachesTheEngine(t *testing.T) {
	const raw = `{"composite":{"width":1440,"height":720,"background":"#EDE6D2","plates":[` +
		`{"name":"sky","depth":0,"blend":"normal","opacity":1},` +
		`{"name":"ink","depth":1,"blend":"multiply","opacity":0.85},` +
		`{"name":"glow","depth":2,"blend":"screen","opacity":0.4}]}}`

	pb := &opsv1.OpParams{}
	if err := protojson.Unmarshal([]byte(raw), pb); err != nil {
		t.Fatalf("composite params rejected at the wire: %v", err)
	}
	p, err := translateParams("composite", pb)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if p.Width != 1440 || p.Height != 720 {
		t.Fatalf("geometry %dx%d, want 1440x720", p.Width, p.Height)
	}
	if p.Background != "#EDE6D2" {
		t.Fatalf("background %q, want #EDE6D2", p.Background)
	}
	if len(p.Plates) != 3 {
		t.Fatalf("%d plates reached the engine, want 3", len(p.Plates))
	}
	for i, want := range []struct {
		name    string
		depth   int
		blend   string
		opacity float64
	}{
		{"sky", 0, "normal", 1},
		{"ink", 1, "multiply", 0.85},
		{"glow", 2, "screen", 0.4},
	} {
		got := p.Plates[i]
		if got.Name != want.name || got.Depth != want.depth || got.Blend != want.blend || got.Opacity != want.opacity {
			t.Fatalf("plate %d reached the engine as %+v, want %+v", i, got, want)
		}
	}
}

// A region-scoped scrim's geometry has to survive the wire with every field
// intact. A scrim whose region was dropped in translation silently becomes the
// whole-frame gradient it was meant to replace — dimming the entire picture to
// fix one corner, which is the behaviour the region exists to end.
func TestTheScrimRegionReachesTheEngine(t *testing.T) {
	const raw = `{"scrim":{"color":"#1B3FD8","opacity":0.55,"direction":"left",` +
		`"regionX":0.06,"regionY":0.1,"regionWidth":0.42,"regionHeight":0.3,"regionFeather":0.08}}`
	pb := &opsv1.OpParams{}
	if err := protojson.Unmarshal([]byte(raw), pb); err != nil {
		t.Fatalf("scrim params rejected at the wire: %v", err)
	}
	p, err := translateParams("scrim", pb)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	for name, got := range map[string]float64{
		"region_x": p.RegionX, "region_y": p.RegionY,
		"region_width": p.RegionWidth, "region_height": p.RegionHeight,
		"region_feather": p.RegionFeather,
	} {
		if got == 0 {
			t.Fatalf("%s did not reach the engine; the scrim would silently cover the whole frame", name)
		}
	}
	if p.RegionWidth != 0.42 || p.RegionHeight != 0.3 {
		t.Fatalf("scrim region reached the engine as %gx%g, want 0.42x0.3", p.RegionWidth, p.RegionHeight)
	}
}
