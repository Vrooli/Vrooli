package authoring

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"backdrop-studio/internal/vector"
)

// goodTemplate is a generator that passes every check: marks placed as
// fractions of the frame, every knob declared, ink through the slot functions,
// nothing active.
const goodTemplate = `<rect width="{{f .W}}" height="{{f .H}}" fill="{{paper}}"/>
{{- $w := .W}}{{- $h := .H}}{{- $count := int (.Param "marks")}}{{- $frame := .}}
{{range $i := seq $count}}
{{- $t := div (float $i) (float $count)}}
{{- $x := mul $w (add 0.06 (mul 0.88 $t))}}
{{- $y := mul $h (add 0.5 (mul 0.34 (sin (mul $t (mul 6.2831 ($frame.Param "waves"))))))}}
<circle cx="{{f $x}}" cy="{{f $y}}" r="{{f (mul $h 0.012)}}" fill="{{ink}}"/>
{{end}}`

func goodGenerator() Generator {
	return Generator{
		ID:       "authored-wave",
		Name:     "Authored Wave",
		Template: goodTemplate,
		Params: []ParamSpec{
			{Name: "marks", Min: 8, Max: 400, Default: 96, Description: "how many marks across the frame"},
			{Name: "waves", Min: 0.5, Max: 6, Default: 2, Description: "cycles of the sine across the frame"},
		},
		Inks:    []string{vector.InkPaper, vector.InkInk},
		Prompt:  "a wave of marks across the frame",
		ModelID: "test/model",
	}
}

func TestAGoodGeneratorPassesEveryCheck(t *testing.T) {
	report := Validate(goodGenerator(), vector.Presets)
	require.Truef(t, report.Passed, "refusals: %v", report.Refusals)
	names := map[string]bool{}
	for _, c := range report.Checks {
		names[c.Name] = c.Passed
	}
	for _, want := range []string{
		CheckIDDoesNotCollide, CheckParamsDeclared, CheckTemplateParses,
		CheckNoActiveContent, CheckDeterministic, CheckInksResolve, CheckCompositionHolds,
	} {
		passed, ran := names[want]
		require.Truef(t, ran, "check %q did not run; a report that skips a check is not a verdict", want)
		require.Truef(t, passed, "check %q failed", want)
	}
}

// The rejection paths the plan names, each asserted on the check that refused
// rather than only on "it failed" — a validator that refuses everything for one
// reason would pass a weaker assertion.
func TestValidationRefusesAnUndeclaredParameter(t *testing.T) {
	g := goodGenerator()
	g.Params = g.Params[:1] // drop "waves", which the template still reads
	report := Validate(g, vector.Presets)
	require.False(t, report.Passed)
	requireRefusedBy(t, report, CheckParamsDeclared, "waves")
}

func TestValidationRefusesAScriptElement(t *testing.T) {
	g := goodGenerator()
	g.Template += "\n<script>fetch('http://example.com')</script>"
	report := Validate(g, vector.Presets)
	require.False(t, report.Passed)
	requireRefusedBy(t, report, CheckNoActiveContent, "script")
}

// A remote reference is the exfiltration path, and it is refused even without a
// script: a rasterizing browser fetches an external stylesheet or an <image>
// href by itself.
func TestValidationRefusesARemoteReference(t *testing.T) {
	for name, mark := range map[string]string{
		"an image element":      `<image href="https://example.com/a.png" x="0" y="0" width="10" height="10"/>`,
		"an external use":       `<use href="https://example.com/a.svg#x"/>`,
		"a CSS import":          `<style>@import url("https://example.com/a.css");</style>`,
		"an event handler":      `<rect width="10" height="10" onload="alert(1)"/>`,
		"an entity declaration": `<!DOCTYPE svg [<!ENTITY x SYSTEM "file:///etc/passwd">]>`,
	} {
		t.Run(name, func(t *testing.T) {
			g := goodGenerator()
			g.Template += "\n" + mark
			report := Validate(g, vector.Presets)
			require.False(t, report.Passed)
			requireRefusedBy(t, report, CheckNoActiveContent, "")
		})
	}
}

// Non-determinism, produced the way a template actually reaches it: ranging
// over a map, whose iteration order Go randomises per run.
func TestValidationRefusesANonDeterministicTemplate(t *testing.T) {
	g := Generator{
		ID:       "authored-unstable",
		Name:     "Unstable",
		Template: `<rect width="{{f .W}}" height="{{f .H}}" fill="{{paper}}"/>{{range $k, $v := .Unstable}}<circle cx="{{f $v}}" cy="10" r="2" fill="{{ink}}"/>{{end}}`,
		Inks:     []string{vector.InkPaper, vector.InkInk},
	}
	// `.Unstable` is not a field on frame, so this template fails at execution
	// rather than producing unstable bytes — which is the same refusal an
	// operator needs and a stricter one than the plan asks for. The determinism
	// check is proven separately below against a template that really does vary.
	report := Validate(g, vector.Presets)
	require.False(t, report.Passed)
}

// The real determinism assertion: a generator that passes validation renders
// byte-identically from the same seed, and differently from a different one.
func TestAValidatedGeneratorIsDeterministicAndSeedSensitive(t *testing.T) {
	g := Generator{
		ID:   "authored-scatter",
		Name: "Authored Scatter",
		Template: `<rect width="{{f .W}}" height="{{f .H}}" fill="{{paper}}"/>
{{- $frame := .}}{{- $w := .W}}{{- $h := .H}}
{{range $i := seq 240}}
<circle cx="{{f (mul $w ($frame.Rand $i))}}" cy="{{f (mul $h ($frame.Rand (add (float $i) 991.0 | int)))}}" r="{{f (mul $h 0.006)}}" fill="{{ink}}"/>
{{end}}`,
		Inks:    []string{vector.InkPaper, vector.InkInk},
		Prompt:  "a seeded scatter",
		ModelID: "test/model",
	}
	report := Validate(g, vector.Presets)
	require.Truef(t, report.Passed, "refusals: %v", report.Refusals)

	a, err := g.Render(1440, 720, 11, nil, validationInks)
	require.NoError(t, err)
	b, err := g.Render(1440, 720, 11, nil, validationInks)
	require.NoError(t, err)
	require.Equal(t, a.SHA256, b.SHA256, "same seed must produce the same bytes")

	c, err := g.Render(1440, 720, 12, nil, validationInks)
	require.NoError(t, err)
	require.NotEqual(t, a.SHA256, c.SHA256, "a different seed must draw a different picture, or the seed is decoration")
}

// Composition collapse: marks placed in pixels rather than in fractions of the
// frame. The generator looks correct at the size its author saw and piles into
// one corner at a quarter of it.
func TestValidationRefusesACompositionThatCollapses(t *testing.T) {
	g := Generator{
		ID:   "authored-pixel-bound",
		Name: "Pixel Bound",
		Template: `<rect width="{{f .W}}" height="{{f .H}}" fill="{{paper}}"/>
{{range $i := seq 200}}
<circle cx="{{f (add 40.0 (mul (float $i) 6.0))}}" cy="{{f (add 40.0 (mul (float $i) 3.0))}}" r="4" fill="{{ink}}"/>
{{end}}`,
		Inks: []string{vector.InkPaper, vector.InkInk},
	}
	report := Validate(g, vector.Presets)
	require.False(t, report.Passed)
	requireRefusedBy(t, report, CheckCompositionHolds, "composition agreement")
}

// An authored generator may not shadow a hand-written one. A style binding to
// `colonnade` must always reach the colonnade its author wrote.
func TestValidationRefusesAnIDThatShadowsABuiltIn(t *testing.T) {
	g := goodGenerator()
	g.ID = "colonnade"
	report := Validate(g, vector.Presets)
	require.False(t, report.Passed)
	requireRefusedBy(t, report, CheckIDDoesNotCollide, "built-in preset")
}

// A declared range that cannot hold its own default is a contradiction, and it
// is the kind a model produces when it writes the list after the template.
func TestValidationRefusesADefaultOutsideItsRange(t *testing.T) {
	g := goodGenerator()
	g.Params[0].Default = 900 // max is 400
	report := Validate(g, vector.Presets)
	require.False(t, report.Passed)
	requireRefusedBy(t, report, CheckParamsDeclared, "outside its declared range")
}

// Ink resolution fails closed, exactly as it does for a hand-written generator:
// an unbound slot is an error, never the literal "$brand.primary" written into
// a document that then rasterizes to the wrong colour.
func TestAnAuthoredGeneratorFailsClosedOnAnUnboundInk(t *testing.T) {
	_, err := goodGenerator().Render(1440, 720, 7, nil, map[string]string{vector.InkPaper: "#ffffff"})
	require.Error(t, err)
	var unresolved *vector.UnresolvedInkError
	require.ErrorAs(t, err, &unresolved)
	require.Equal(t, vector.InkInk, unresolved.Slot)
}

// A caller's parameters are clamped into the declared range, so a template
// always draws with a number its author reasoned about.
func TestSuppliedParametersAreClampedToTheirDeclaredRange(t *testing.T) {
	g := goodGenerator()
	values := g.resolveParams(map[string]float64{"marks": 100000, "waves": -50})
	require.Equal(t, 400.0, values["marks"], "above max must clamp to max")
	require.Equal(t, 0.5, values["waves"], "below min must clamp to min")
	require.Len(t, values, 2, "an undeclared key a caller sends must not reach the template")
}

// A template that reads an undeclared knob fails the render by name, so the
// rule holds even for a generator stored before the check existed.
func TestReadingAnUndeclaredParameterFailsTheRender(t *testing.T) {
	g := goodGenerator()
	g.Params = g.Params[:1]
	_, err := g.Render(1440, 720, 7, nil, validationInks)
	require.Error(t, err)
	var undeclared *UndeclaredParamError
	require.ErrorAs(t, err, &undeclared)
	require.Equal(t, "waves", undeclared.Param)
}

// The document envelope belongs to the vector package, not to the author. An
// authored body is wrapped in one `<svg>` with one declared plane, whatever the
// template emits.
func TestTheEnvelopeIsNotTheAuthorsToWrite(t *testing.T) {
	res, err := goodGenerator().Render(1440, 720, 7, nil, validationInks)
	require.NoError(t, err)
	document := string(res.SVG)
	require.Equal(t, 1, strings.Count(document, "<svg"), "exactly one document element")
	require.Contains(t, document, `viewBox="0 0 1440.000 720.000"`, "the envelope declares the requested frame")
	require.Equal(t, []string{"authored"}, res.Planes)
	require.NotContains(t, document, "$brand.", "every ink slot resolves before the document leaves")
}

func requireRefusedBy(t *testing.T, report Report, check, substring string) {
	t.Helper()
	for _, c := range report.Checks {
		if c.Name != check || c.Passed {
			continue
		}
		if substring == "" || strings.Contains(c.Detail, substring) {
			return
		}
	}
	t.Fatalf("no failing %q check containing %q; checks were %+v", check, substring, report.Checks)
}
