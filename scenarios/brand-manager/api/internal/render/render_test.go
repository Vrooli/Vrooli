package render

import (
	"bytes"
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

func parseGroupScale(t *testing.T, svg string) float64 {
	t.Helper()
	idx := strings.Index(svg, "scale(")
	if idx < 0 {
		t.Fatal("no scale in output")
	}
	rest := svg[idx+len("scale("):]
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
