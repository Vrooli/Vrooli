package render

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"testing"
)

const markSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 80"><path fill="#ffffff" d="M10 10H90V70H10Z"/><path fill="#22d3ee" d="M30 30H40V40H30Z M60 50H70V60H60Z"/></svg>`

func testStyle() Style {
	return Style{
		Shape:            "rounded_square",
		CornerRatio:      0.21875,
		BackgroundTop:    "#15243c",
		BackgroundBottom: "#0b1728",
		MarkScale:        0.86,
		MaskableScale:    0.40,
		AccentColor:      "#22d3ee",
		Glow:             []GlowLayer{{Width: 2, Opacity: 0.45}, {Width: 5, Opacity: 0.22}, {Width: 9, Opacity: 0.10}},
	}
}

func TestComposeVariantsAreFilterFreeAndDeterministic(t *testing.T) {
	for _, v := range []Variant{Rounded, FullBleed, Maskable, SocialCard, Vector} {
		out1, err := Compose([]byte(markSVG), nil, testStyle(), v, 512, false)
		if err != nil {
			t.Fatalf("%s: %v", v, err)
		}
		out2, err := Compose([]byte(markSVG), nil, testStyle(), v, 512, false)
		if err != nil {
			t.Fatalf("%s: %v", v, err)
		}
		if !bytes.Equal(out1, out2) {
			t.Fatalf("%s: composition is not deterministic", v)
		}
		if ContainsFilterFeatures(out1) {
			t.Fatalf("%s: output contains a filter/mask/pattern/style element:\n%s", v, out1)
		}
	}
}

// image-tools' pure-Go rasterizer drops a group whose transform is a list such
// as "translate(x y) scale(s)", which shipped every PNG target as an empty tile.
// Every composed variant must place the mark with a single matrix().
func TestComposePlacesTheMarkWithASingleMatrix(t *testing.T) {
	for _, v := range []Variant{Rounded, FullBleed, Maskable, SocialCard} {
		out, err := Compose([]byte(markSVG), nil, testStyle(), v, 180, false)
		if err != nil {
			t.Fatalf("%s: %v", v, err)
		}
		s := string(out)
		if strings.Contains(s, "translate(") || strings.Contains(s, "scale(") {
			t.Fatalf("%s: output uses a transform list oksvg drops:\n%s", v, s)
		}
		if !strings.Contains(s, `transform="matrix(`) {
			t.Fatalf("%s: mark is not placed with matrix():\n%s", v, s)
		}
	}
}

// A traced mark has a 2048-unit viewBox but only paints part of it. The mark's
// INK — not its viewBox — must come out at MarkScale of the tile edge, so the
// padding a trace happens to leave never changes how large the mark is drawn.
func TestComposeFitsATracedMarkToTheTileByItsInk(t *testing.T) {
	// Ink spans 100..1948 = 1848 units inside the 2048-unit viewBox.
	traced := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 2048 2048"><path fill="#ffffff" d="M100 100H1948V1948H100Z"/></svg>`
	for _, edge := range []int{16, 32, 180, 512} {
		out, err := Compose([]byte(traced), nil, testStyle(), Rounded, edge, false)
		if err != nil {
			t.Fatal(err)
		}
		scale := parseGroupScale(t, string(out))
		got := scale * 1848
		want := 0.86 * float64(edge)
		if got < want-0.5 || got > want+0.5 {
			t.Fatalf("edge %d: mark ink = %.2f px, want %.2f", edge, got, want)
		}
	}
}

// Two marks with the same ink but different surrounding padding must compose to
// the same drawn size in the same place. This is what keeps one product line's
// icons consistent when each mark is traced from a different raster.
func TestComposeIsInvariantToMarkPadding(t *testing.T) {
	edge := 512
	cases := []struct {
		name           string
		svg            string
		inkMin, inkLen float64
	}{
		{"tight", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><path fill="#ffffff" d="M10 10H90V90H10Z"/></svg>`, 10, 80},
		{"padded", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 2048 2048"><path fill="#ffffff" d="M984 984H1064V1064H984Z"/></svg>`, 984, 80},
	}
	for _, tc := range cases {
		out, err := Compose([]byte(tc.svg), nil, testStyle(), Rounded, edge, false)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		scale, tx, ty := parseGroupMatrix(t, string(out))
		if got, want := tc.inkLen*scale, 0.86*float64(edge); math.Abs(got-want) > 0.5 {
			t.Fatalf("%s: drawn ink = %.2f px, want %.2f", tc.name, got, want)
		}
		centreX := tx + (tc.inkMin+tc.inkLen/2)*scale
		centreY := ty + (tc.inkMin+tc.inkLen/2)*scale
		if math.Abs(centreX-float64(edge)/2) > 0.5 || math.Abs(centreY-float64(edge)/2) > 0.5 {
			t.Fatalf("%s: ink centre = (%.2f, %.2f), want (%d, %d)", tc.name, centreX, centreY, edge/2, edge/2)
		}
	}
}

// A trace whose ink sits off-centre in its viewBox must still be centred on the
// tile: Rigel's traced tower sat at x 499..1263 of 2048 and composed off-centre.
func TestComposeCentresAnOffCentreMarkByItsInk(t *testing.T) {
	edge := 512
	offset := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 2048 2048"><path fill="#ffffff" d="M100 100H612V612H100Z"/></svg>`
	out, err := Compose([]byte(offset), nil, testStyle(), Rounded, edge, false)
	if err != nil {
		t.Fatal(err)
	}
	scale, tx, ty := parseGroupMatrix(t, string(out))
	centreX := tx + 356*scale // ink spans 100..612, centre 356
	centreY := ty + 356*scale
	if math.Abs(centreX-256) > 0.5 || math.Abs(centreY-256) > 0.5 {
		t.Fatalf("off-centre ink composed at (%.2f, %.2f), want (256, 256)", centreX, centreY)
	}
}

func TestComposeRoundedUsesCornerRadiusAndGradient(t *testing.T) {
	out, err := Compose([]byte(markSVG), nil, testStyle(), Rounded, 512, false)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `rx="112.00"`) {
		t.Fatalf("expected corner radius 112, got:\n%s", s)
	}
	if !strings.Contains(s, `stop-color="#15243c"`) || !strings.Contains(s, `stop-color="#0b1728"`) {
		t.Fatalf("expected the gradient stops, got:\n%s", s)
	}
}

func TestComposeDrawsGlowAsLayeredStrokes(t *testing.T) {
	out, err := Compose([]byte(markSVG), nil, testStyle(), Rounded, 512, false)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	// Three glow layers, each one stroke path per accent-filled path element.
	if got := strings.Count(s, `stroke-opacity=`); got != 3 {
		t.Fatalf("expected 3 glow stroke paths, got %d:\n%s", got, s)
	}
}

func TestComposeSocialCardSize(t *testing.T) {
	out, err := Compose([]byte(markSVG), nil, testStyle(), SocialCard, 512, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `viewBox="0 0 1200 630"`) {
		t.Fatalf("social card must be 1200x630:\n%s", out)
	}
}

func TestComposeMaskableKeepsMarkInsideSafeZone(t *testing.T) {
	edge := 512
	out, err := Compose([]byte(markSVG), nil, testStyle(), Maskable, edge, false)
	if err != nil {
		t.Fatal(err)
	}
	scale := parseGroupScale(t, string(out))
	// The mark's circumscribed circle (radius = 0.5 * max(vbW,vbH) * scale) must
	// fit radius 0.40 * edge within 1 px. The fixture viewBox is 100x80.
	radius := 0.5 * scale * 100.0
	if radius > 0.40*float64(edge)+1 {
		t.Fatalf("maskable mark radius %.2f exceeds safe zone %.2f", radius, 0.40*float64(edge))
	}
	if scale == 0 {
		t.Fatal("no scale parsed")
	}
}

// parseGroupMatrix returns the scale and translation of the mark's matrix(),
// emitted as "matrix(s 0 0 s tx ty)".
func parseGroupMatrix(t *testing.T, svg string) (scale, tx, ty float64) {
	t.Helper()
	idx := strings.Index(svg, "matrix(")
	if idx < 0 {
		t.Fatal("no matrix in output")
	}
	rest := svg[idx+len("matrix("):]
	end := strings.Index(rest, ")")
	if end < 0 {
		t.Fatal("unterminated matrix")
	}
	fields := strings.Fields(rest[:end])
	if len(fields) != 6 {
		t.Fatalf("expected 6 matrix terms, got %d: %q", len(fields), rest[:end])
	}
	get := func(i int) float64 {
		v, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			t.Fatalf("parse matrix term %d (%q): %v", i, fields[i], err)
		}
		return v
	}
	return get(0), get(4), get(5)
}

func parseGroupScale(t *testing.T, svg string) float64 {
	t.Helper()
	// The mark group is placed with matrix(s 0 0 s tx ty); its first term is the scale.
	idx := strings.Index(svg, "matrix(")
	if idx < 0 {
		t.Fatal("no matrix in output")
	}
	rest := svg[idx+len("matrix("):]
	end := strings.IndexAny(rest, ") ")
	if end < 0 {
		t.Fatal("bad scale")
	}
	v, err := strconv.ParseFloat(rest[:end], 64)
	if err != nil {
		t.Fatalf("parse scale %q: %v", rest[:end], err)
	}
	return v
}
