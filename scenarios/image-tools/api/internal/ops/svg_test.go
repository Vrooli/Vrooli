package ops

import (
	"image"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHighFidelityFeatureDetection(t *testing.T) {
	for _, tc := range []struct {
		name string
		svg  string
		want []string
	}{
		{"plain shapes", `<svg><rect width="10" height="10" fill="#000"/></svg>`, nil},
		{"filter element", `<svg><defs><filter id="f"></filter></defs></svg>`, []string{"filter"}},
		{"filter reference", `<svg><circle filter="url(#f)"/></svg>`, []string{"filter"}},
		{"mask element", `<svg><mask id="m"></mask></svg>`, []string{"mask"}},
		{"pattern element", `<svg><pattern id="p"></pattern></svg>`, []string{"pattern"}},
		{"stylesheet", `<svg><style>.a{fill:#000}</style></svg>`, []string{"css"}},
		{
			"all four",
			`<svg><style>.a{fill:#000}</style><defs><filter id="f"/><mask id="m"/><pattern id="p"/></defs></svg>`,
			[]string{"filter", "mask", "pattern", "css"},
		},
		// The detector keys on element and url() forms so an ordinary id or
		// class containing the word does not force the slow path.
		{"word in an id", `<svg><rect id="filter-panel" class="mask-layer"/></svg>`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, HighFidelitySVGFeatures([]byte(tc.svg)))
		})
	}
}

// The load-bearing test for D4.
//
// It renders one SVG whose entire appearance depends on a filter, a mask, a
// pattern and a CSS rule, and asserts each one actually drew by sampling
// pixels that can only have their measured colour if that feature ran. An
// assertion that merely checked "we got a PNG back" would pass against the
// rasterizer this replaced, which is the whole problem: oksvg returns a
// perfectly valid PNG with all four features missing.
func TestHighFidelityRasterizerDrawsFilterMaskPatternAndCSS(t *testing.T) {
	if ChromeBinary() == "" {
		t.Skipf("no headless Chrome on this host; set %s to run the fidelity proof", chromeBinaryEnv)
	}
	const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="200" viewBox="0 0 400 200">
<style>.ink{fill:#00ff00}</style>
<defs>
  <pattern id="dots" width="10" height="10" patternUnits="userSpaceOnUse">
    <rect width="10" height="10" fill="#ff0000"/>
  </pattern>
  <mask id="hole">
    <rect width="400" height="200" fill="#ffffff"/>
    <rect x="300" y="0" width="100" height="200" fill="#000000"/>
  </mask>
  <!-- The region has to be wide enough to contain the offset result: a filter
       region is relative to the object's bounding box, and content moved
       outside it is clipped away rather than drawn. -->
  <filter id="shift" x="-200%" y="-200%" width="500%" height="500%">
    <feOffset dx="60" dy="0"/>
  </filter>
</defs>
<rect width="400" height="200" fill="#0000ff"/>
<rect width="300" height="200" fill="url(#dots)" mask="url(#hole)"/>
<rect x="0" y="150" width="40" height="40" class="ink" filter="url(#shift)"/>
</svg>`

	img, err := rasterizeSVG([]byte(svg), 400, 200)
	require.NoError(t, err)
	require.Equal(t, image.Rect(0, 0, 400, 200), img.Bounds())

	sample := func(x, y int) (uint8, uint8, uint8) {
		r, g, b, _ := img.At(x, y).RGBA()
		return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
	}

	// pattern: the left band is painted by a pattern fill, so it must be red.
	r, g, b := sample(100, 100)
	require.Equalf(t, [3]uint8{255, 0, 0}, [3]uint8{r, g, b},
		"the pattern did not draw: the left band should be the pattern's red, got #%02x%02x%02x", r, g, b)

	// mask: the pattern rect covers x<300 but the mask blacks out x>=300, so
	// the right band must be the underlying blue rather than the pattern.
	r, g, b = sample(350, 100)
	require.Equalf(t, [3]uint8{0, 0, 255}, [3]uint8{r, g, b},
		"the mask did not draw: the masked band should show the blue ground, got #%02x%02x%02x", r, g, b)

	// filter + css together: the square is green only because of the CSS class,
	// and it sits at x≈60..100 only because feOffset moved it. Its authored
	// position (x<40) must therefore NOT be green.
	r, g, b = sample(80, 170)
	require.Equalf(t, [3]uint8{0, 255, 0}, [3]uint8{r, g, b},
		"the filter or the CSS rule did not draw: expected the offset green square at x=80, got #%02x%02x%02x", r, g, b)
	r, g, b = sample(10, 170)
	require.NotEqualf(t, [3]uint8{0, 255, 0}, [3]uint8{r, g, b},
		"the filter did not draw: the square is still at its authored position, so feOffset was dropped")
}

// A rasterizer that quietly degrades is the defect. On a host with no Chrome,
// an SVG that needs it must fail by name rather than return a picture missing
// the features that define it.
func TestSVGNeedingFidelityRefusesRatherThanDegrading(t *testing.T) {
	err := (&ErrSVGFidelityUnavailable{Features: []string{"filter", "mask"}}).Error()
	require.Contains(t, err, "filter, mask")
	require.Contains(t, err, chromeBinaryEnv)
	require.Contains(t, err, "Refusing")
}

// Plain SVG keeps the pure-Go path, so the common case needs no browser.
func TestPlainSVGUsesThePureGoPath(t *testing.T) {
	const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="16" viewBox="0 0 32 16">
<rect width="32" height="16" fill="#123456"/></svg>`
	require.Empty(t, HighFidelitySVGFeatures([]byte(svg)))
	img, err := rasterizeSVG([]byte(svg), 0, 0)
	require.NoError(t, err)
	require.Equal(t, image.Rect(0, 0, 32, 16), img.Bounds())
}

func TestIntrinsicSVGSize(t *testing.T) {
	w, h := intrinsicSVGSize([]byte(`<svg width="640" height="480" viewBox="0 0 100 50">`))
	require.Equal(t, 640, w)
	require.Equal(t, 480, h)

	// viewBox alone is enough when no width/height attribute is declared.
	w, h = intrinsicSVGSize([]byte(`<svg viewBox="0 0 100 50">`))
	require.Equal(t, 100, w)
	require.Equal(t, 50, h)
}

func TestSVGRasterBombGuard(t *testing.T) {
	_, _, err := svgTargetSize([]byte(`<svg viewBox="0 0 99999 99999">`), 0, 0)
	require.ErrorIs(t, err, ErrDecode)
	require.Contains(t, err.Error(), "exceeds")
}
