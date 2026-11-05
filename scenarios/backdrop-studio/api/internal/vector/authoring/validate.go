package authoring

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"backdrop-studio/internal/vector"
)

// Report is the verdict that admits or refuses an authored generator. It is
// stored beside the generator so an operator reads what was actually checked
// rather than trusting that something was.
type Report struct {
	Passed   bool     `json:"passed"`
	Checks   []Check  `json:"checks"`
	Refusals []string `json:"refusals,omitempty"`
}

// Check is one named test and what it measured.
type Check struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

// Check names, exported because the CLI, the store and the tests all key on
// them and a typo in a literal three packages away is not a failure anyone
// wants to debug.
const (
	CheckTemplateParses   = "template_parses"
	CheckParamsDeclared   = "params_declared"
	CheckNoActiveContent  = "no_active_content"
	CheckDeterministic    = "deterministic"
	CheckInksResolve      = "inks_resolve"
	CheckCompositionHolds = "composition_holds"
	CheckIDDoesNotCollide = "id_does_not_collide"
)

// compositionAgreementFloor is how much of the composition must survive a
// four-fold reduction in frame size.
//
// The same bar the hand-written generators are held to, and for the same
// reason: a generator whose marks are placed in pixels rather than in fractions
// of the frame looks correct at the size its author saw and collapses at every
// other. That is not a stylistic difference — a backdrop is asked for at
// eighteen surface geometries in this catalog alone.
const compositionAgreementFloor = 0.80

// validationInks are stand-in colours used only to prove a document's slots
// resolve. They are never stored: the real palette comes from the bound brand
// at render time.
var validationInks = map[string]string{
	vector.InkPaper:  "#f1e7d2",
	vector.InkInk:    "#123a6b",
	vector.InkAccent: "#b1442c",
}

// Validate decides whether an authored generator may be stored.
//
// It runs every check rather than stopping at the first failure, because an
// author — model or human — fixing one rejection at a time through a paid
// round trip is a slow and expensive way to learn there were four.
func Validate(g Generator, builtinPresets []string) Report {
	report := Report{Passed: true}
	fail := func(name, detail string) {
		report.Checks = append(report.Checks, Check{Name: name, Passed: false, Detail: detail})
		report.Refusals = append(report.Refusals, detail)
		report.Passed = false
	}
	pass := func(name, detail string) {
		report.Checks = append(report.Checks, Check{Name: name, Passed: true, Detail: detail})
	}

	// An authored generator may not take a built-in preset's name. A style
	// binding to `colonnade` must always reach the hand-written colonnade; a
	// stored generator that shadowed one would silently change what an existing
	// seeded style draws.
	collides := false
	for _, preset := range builtinPresets {
		if strings.EqualFold(strings.TrimSpace(g.ID), preset) {
			fail(CheckIDDoesNotCollide, fmt.Sprintf("id %q is a built-in preset; an authored generator may not shadow one", g.ID))
			collides = true
			break
		}
	}
	if !collides {
		if strings.TrimSpace(g.ID) == "" {
			fail(CheckIDDoesNotCollide, "id is empty")
		} else {
			pass(CheckIDDoesNotCollide, fmt.Sprintf("id %q is free", g.ID))
		}
	}

	// Active content, checked on the template source *before* it is executed.
	// A template that assembles `<scr` + `ipt>` from two literals would pass a
	// check on its output only if the check ran on output too, so both are
	// checked: this one catches the source, the render check below catches the
	// result.
	if found := activeContent(g.Template); len(found) > 0 {
		fail(CheckNoActiveContent, fmt.Sprintf("template contains %s", strings.Join(found, ", ")))
	}

	declared := map[string]struct{}{}
	for _, spec := range g.Params {
		if strings.TrimSpace(spec.Name) == "" {
			fail(CheckParamsDeclared, "a declared parameter has no name")
			continue
		}
		if spec.Max < spec.Min {
			fail(CheckParamsDeclared, fmt.Sprintf("parameter %q declares max %g below min %g", spec.Name, spec.Max, spec.Min))
		}
		if spec.Default < spec.Min || spec.Default > spec.Max {
			fail(CheckParamsDeclared, fmt.Sprintf("parameter %q default %g is outside its declared range [%g, %g]", spec.Name, spec.Default, spec.Min, spec.Max))
		}
		declared[spec.Name] = struct{}{}
	}
	var undeclared []string
	for _, name := range ReadParams(g.Template) {
		if _, ok := declared[name]; !ok {
			undeclared = append(undeclared, name)
		}
	}
	if len(undeclared) > 0 {
		fail(CheckParamsDeclared, fmt.Sprintf("template reads undeclared parameter(s): %s", strings.Join(undeclared, ", ")))
	} else if len(report.Refusals) == 0 || !hasFailure(report, CheckParamsDeclared) {
		pass(CheckParamsDeclared, fmt.Sprintf("%d declared, %d read, all covered", len(g.Params), len(ReadParams(g.Template))))
	}

	// Render at the reference frame. Everything below needs a document.
	const refW, refH = 1440, 720
	first, err := g.Render(refW, refH, authoringProbeSeed, nil, validationInks)
	if err != nil {
		fail(CheckTemplateParses, err.Error())
		return report
	}
	pass(CheckTemplateParses, fmt.Sprintf("renders %d bytes at %dx%d", len(first.SVG), refW, refH))

	// Checked on the author's marks, not the wrapped document: the envelope
	// carries `xmlns="http://www.w3.org/2000/svg"`, and a remote-URL check over
	// the whole document would refuse every generator ever written, including a
	// correct one. What is untrusted is the body.
	body, bodyErr := g.Body(refW, refH, authoringProbeSeed, nil)
	switch {
	case bodyErr != nil:
		fail(CheckNoActiveContent, fmt.Sprintf("body could not be rendered for inspection: %v", bodyErr))
	default:
		if found := activeContent(body); len(found) > 0 {
			fail(CheckNoActiveContent, fmt.Sprintf("rendered marks contain %s", strings.Join(found, ", ")))
		} else if !hasFailure(report, CheckNoActiveContent) {
			pass(CheckNoActiveContent, "no script, external reference, remote URL or entity declaration")
		}
	}

	// Determinism, measured rather than assumed. The template functions are
	// pure and the noise is seeded, but an author can still reach
	// non-determinism through map iteration order in a `range`, and the whole
	// reproduce-a-released-asset story rests on this.
	second, err := g.Render(refW, refH, authoringProbeSeed, nil, validationInks)
	switch {
	case err != nil:
		fail(CheckDeterministic, fmt.Sprintf("second render failed where the first succeeded: %v", err))
	case first.SHA256 != second.SHA256:
		fail(CheckDeterministic, fmt.Sprintf("two renders of the same seed differ (%s vs %s)", first.SHA256[:12], second.SHA256[:12]))
	default:
		pass(CheckDeterministic, "two renders of one seed are byte-identical")
	}

	// Ink resolution, proven by rendering against a palette that covers nothing
	// and requiring the refusal. A generator that draws no ink slot at all is
	// legal — a paper-and-graphite plate is a real art direction — so an
	// absence of slots is reported, not refused.
	switch _, inkErr := g.Render(refW, refH, authoringProbeSeed, nil, map[string]string{}); {
	case inkErr == nil && len(g.Inks) == 0:
		pass(CheckInksResolve, "draws no brand ink slot")
	case inkErr == nil:
		fail(CheckInksResolve, fmt.Sprintf("declares ink slots %v but renders without them; the declaration is wrong or the template hardcodes a colour", g.Inks))
	default:
		pass(CheckInksResolve, "an unbound palette is refused rather than written into the document")
	}

	// Composition across sizes, on the same occupancy measure the hand-written
	// generators are held to.
	small, smallErr := g.Render(refW/4, refH/4, authoringProbeSeed, nil, validationInks)
	if smallErr != nil {
		fail(CheckCompositionHolds, fmt.Sprintf("renders at %dx%d and fails at %dx%d: %v", refW, refH, refW/4, refH/4, smallErr))
	} else {
		agreement := gridAgreement(occupancy(string(small.SVG), small.Width, small.Height), occupancy(string(first.SVG), first.Width, first.Height))
		if agreement < compositionAgreementFloor {
			fail(CheckCompositionHolds, fmt.Sprintf(
				"composition agreement %.3f between %dx%d and %dx%d is below the %.2f floor; the marks are placed in pixels rather than in fractions of the frame",
				agreement, refW/4, refH/4, refW, refH, compositionAgreementFloor))
		} else {
			pass(CheckCompositionHolds, fmt.Sprintf("composition agreement %.3f across a four-fold size change", agreement))
		}
	}
	return report
}

// authoringProbeSeed is the seed validation renders at. Fixed so two runs of
// the validator on the same generator produce the same report.
const authoringProbeSeed = 7

func hasFailure(r Report, name string) bool {
	for _, c := range r.Checks {
		if c.Name == name && !c.Passed {
			return true
		}
	}
	return false
}

// activeContentPatterns are what an SVG must not contain.
//
// This is a deny list, which is normally the weaker choice — but the
// alternative, parsing SVG and admitting only a known-good element set, is a
// second SVG implementation in a scenario that owns no rasterizer. The deny
// list is paired with two structural controls that do not depend on it: an
// authored generator never writes the document element, and the template
// function surface is closed and arithmetic-only.
var activeContentPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"a script element", regexp.MustCompile(`(?i)<\s*script\b`)},
	{"a foreignObject element", regexp.MustCompile(`(?i)<\s*foreignObject\b`)},
	{"an embedded image element", regexp.MustCompile(`(?i)<\s*image\b`)},
	{"a use element", regexp.MustCompile(`(?i)<\s*use\b`)},
	{"an event handler attribute", regexp.MustCompile(`(?i)\bon[a-z]+\s*=`)},
	{"a remote URL", regexp.MustCompile(`(?i)(https?:|//[a-z0-9-]+\.)`)},
	{"a data URI", regexp.MustCompile(`(?i)data:`)},
	{"an entity declaration", regexp.MustCompile(`(?i)<!\s*(DOCTYPE|ENTITY)`)},
	{"an external reference", regexp.MustCompile(`(?i)\b(xlink:href|href)\s*=\s*"[^"#]`)},
	{"a CSS import", regexp.MustCompile(`(?i)@import`)},
}

func activeContent(document string) []string {
	var found []string
	for _, p := range activeContentPatterns {
		if p.pattern.MatchString(document) {
			found = append(found, p.name)
		}
	}
	return found
}

// occupancy reduces a document to a coarse map of where its coordinates land.
// It reads numbers out of the SVG rather than rasterizing, so validation needs
// no browser — the same trade the parent package's tests make.
func occupancy(document string, width, height int) []float64 {
	const cols, rows = 12, 8
	grid := make([]float64, cols*rows)
	w, h := float64(width), float64(height)
	if w <= 0 || h <= 0 {
		return grid
	}
	for _, pt := range coordinates(document) {
		cx, cy := int(pt[0]/w*cols), int(pt[1]/h*rows)
		if cx < 0 || cy < 0 || cx >= cols || cy >= rows {
			continue
		}
		grid[cy*cols+cx]++
	}
	total := 0.0
	for _, v := range grid {
		total += v
	}
	if total == 0 {
		return grid
	}
	for i := range grid {
		grid[i] /= total
	}
	return grid
}

// coordinates pulls every drawn point out of a document.
//
// It reads three forms, because a template legitimately uses all of them and an
// extractor that saw only one measured nothing: paired attributes (`cx`/`cy`,
// `x`/`y`, `x1`/`y1`, `x2`/`y2`), and the number streams inside `d` and
// `points`. The first version read only whitespace-separated pairs, which is
// path data — so a generator built from `<circle>` elements produced an empty
// occupancy grid, two empty grids agreed perfectly, and the composition check
// passed a generator that placed every mark in absolute pixels.
func coordinates(document string) [][2]float64 {
	var out [][2]float64
	for _, pair := range [][2]string{{"cx", "cy"}, {"x1", "y1"}, {"x2", "y2"}, {"x", "y"}} {
		for _, m := range attributePairPattern(pair[0], pair[1]).FindAllStringSubmatch(document, -1) {
			x, errX := strconv.ParseFloat(m[1], 64)
			y, errY := strconv.ParseFloat(m[2], 64)
			if errX == nil && errY == nil {
				out = append(out, [2]float64{x, y})
			}
		}
	}
	for _, m := range pointStreamPattern.FindAllStringSubmatch(document, -1) {
		numbers := numberPattern.FindAllString(m[2], -1)
		for i := 0; i+1 < len(numbers); i += 2 {
			x, errX := strconv.ParseFloat(numbers[i], 64)
			y, errY := strconv.ParseFloat(numbers[i+1], 64)
			if errX == nil && errY == nil {
				out = append(out, [2]float64{x, y})
			}
		}
	}
	return out
}

var (
	pointStreamPattern = regexp.MustCompile(`\b(d|points)\s*=\s*"([^"]*)"`)
	numberPattern      = regexp.MustCompile(`-?\d+(?:\.\d+)?`)
	attributePairCache = map[string]*regexp.Regexp{}
)

// attributePairPattern matches `a="N" ... b="M"` with only attributes between,
// so a `cx`/`cy` pair on one element is read and two unrelated elements are not
// joined into a phantom point.
func attributePairPattern(a, b string) *regexp.Regexp {
	key := a + "|" + b
	if compiled, ok := attributePairCache[key]; ok {
		return compiled
	}
	compiled := regexp.MustCompile(`\b` + a + `\s*=\s*"(-?\d+(?:\.\d+)?)"[^<>]{0,120}?\b` + b + `\s*=\s*"(-?\d+(?:\.\d+)?)"`)
	attributePairCache[key] = compiled
	return compiled
}

// gridAgreement is 1 minus half the total variation distance: identical
// distributions score 1, disjoint ones score 0.
func gridAgreement(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	total := 0.0
	for i := range a {
		total += math.Abs(a[i] - b[i])
	}
	return 1 - total/2
}
