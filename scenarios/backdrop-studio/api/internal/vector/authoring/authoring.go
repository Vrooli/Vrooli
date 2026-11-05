// Package authoring lets a model write a new vector generator.
//
// The hand-written generators in the parent package are Go functions, and a
// model cannot add one of those to a running binary. So an authored generator
// is a *template*: an SVG document whose numbers are expressions of declared
// parameters, the frame size, and a seeded noise field. That is a real
// difference in expressive power and it is stated rather than papered over — a
// template cannot trace marching-squares iso-lines, and the four hand-written
// generators remain the shipping lane. What a template can do is compose:
// arrange, repeat, scale and vary marks across a frame, which is what most art
// directions actually are.
//
// Three properties every authored generator must hold, because they are what
// make a generator catalog data rather than a stored surprise:
//
//   - Deterministic. Same (template, size, seed, params) means the same bytes,
//     always. Validation renders twice and compares.
//   - Declared. Every parameter the template reads is in its parameter list
//     with a range, so a caller can tune it without reading the template and a
//     seed file can record what it set.
//   - Inert. No script, no external reference, no remote URL, no entity
//     declaration. An SVG is a document this system rasterizes in a browser, so
//     an authored one is untrusted input that reaches a renderer.
//
// The last is the one with teeth. A generator arrives from a model, is stored
// as catalog data, and is later rasterized by headless Chrome — so an authored
// document that could fetch a URL would be an exfiltration path from a machine
// that ran a render, and one that could run script would be worse. Validation
// refuses those before storage, not at render time, because a stored generator
// is one a later code path may reasonably assume was checked.
package authoring

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"backdrop-studio/internal/vector"
)

// Generator is one model-authored vector generator, stored as catalog data.
type Generator struct {
	// ID is the name a style's scaffold binding uses to reach it. It sits in
	// the same namespace as the built-in presets and may not collide with one:
	// a style naming `colonnade` must always get the hand-written colonnade.
	ID   string `json:"id"`
	Name string `json:"name"`
	// Template renders the SVG body — the marks, not the envelope. The
	// document element, the plane grouping and the ink resolution belong to the
	// parent package so an authored generator gets the same guarantees a
	// hand-written one does, and cannot opt out of them.
	Template string `json:"template"`
	// Params are the knobs the template reads, each with a range and a default.
	Params []ParamSpec `json:"params"`
	// Inks are the "$brand.*" slots the template references, recorded so a
	// style binding to this generator can be checked for ink coverage before it
	// is ever rendered.
	Inks []string `json:"inks"`
	// Prompt is the authoring request, verbatim. An authored generator whose
	// prompt nobody kept is a generator nobody can re-author, re-review, or
	// explain — the same reason the seed files carry retune_reasons.
	Prompt string `json:"prompt"`
	// ModelID is the model that wrote it, as ai-gateway reported it. It is the
	// disclosure record: an asset released from an authored generator names the
	// model that authored the generator, not only the one that drew pixels.
	ModelID string `json:"model_id"`
	// Validation is the verdict that admitted it, kept so an operator can read
	// what was actually checked rather than trusting that something was.
	Validation Report `json:"validation"`
}

// ParamSpec is one declared knob.
type ParamSpec struct {
	Name string `json:"name"`
	// Min and Max bound what a caller may set. A template always reads a value
	// inside them, whatever arrives, so an author reasons about one range
	// rather than about every number a JSON blob could carry.
	Min float64 `json:"min"`
	Max float64 `json:"max"`
	// Default is what the template reads when a caller sets nothing. It must
	// sit inside the range, which validation checks.
	Default     float64 `json:"default"`
	Description string  `json:"description,omitempty"`
}

// Render draws an authored generator at a size and seed.
//
// The signature mirrors vector.Render deliberately: the render path should not
// have to know whether a generator was written by hand or by a model, only that
// it produces a document. What differs is where the generator came from and how
// it was admitted, which is exactly what the catalog records.
func (g Generator) Render(width, height int, seed int64, params map[string]float64, inks map[string]string) (vector.Result, error) {
	body, err := g.body(width, height, seed, params)
	if err != nil {
		return vector.Result{}, err
	}
	return vector.RenderAuthored(vector.AuthoredRequest{
		ID:     g.ID,
		Body:   body,
		Width:  width,
		Height: height,
		Inks:   inks,
	})
}

// Body renders the template alone, without the document envelope.
//
// Validation checks this rather than the wrapped document, because the envelope
// legitimately carries `xmlns="http://www.w3.org/2000/svg"` and a remote-URL
// check on the whole document therefore refuses every generator ever written —
// including a correct one. What is untrusted is the author's marks, and this is
// exactly those.
func (g Generator) Body(width, height int, seed int64, params map[string]float64) (string, error) {
	return g.body(width, height, seed, params)
}

func (g Generator) body(width, height int, seed int64, params map[string]float64) (string, error) {
	tmpl, err := template.New(g.ID).Funcs(templateFuncs()).Option("missingkey=error").Parse(g.Template)
	if err != nil {
		return "", fmt.Errorf("authoring: generator %q template does not parse: %w", g.ID, err)
	}
	values := g.resolveParams(params)
	var out bytes.Buffer
	data := &frame{
		W: float64(width), H: float64(height),
		Seed:   seed,
		params: values,
		id:     g.ID,
	}
	if err := tmpl.Execute(&out, data); err != nil {
		return "", fmt.Errorf("authoring: generator %q failed to render: %w", g.ID, err)
	}
	if data.err != nil {
		return "", data.err
	}
	return out.String(), nil
}

// resolveParams clamps supplied values into their declared ranges and fills the
// declared default for anything absent. A template therefore always reads a
// value inside the range its author declared, whatever a caller sends.
func (g Generator) resolveParams(supplied map[string]float64) map[string]float64 {
	values := make(map[string]float64, len(g.Params))
	for _, spec := range g.Params {
		v, ok := supplied[spec.Name]
		if !ok {
			v = spec.Default
		}
		values[spec.Name] = math.Min(spec.Max, math.Max(spec.Min, v))
	}
	return values
}

// ParamNames returns the declared names, sorted.
func (g Generator) ParamNames() []string {
	names := make([]string, 0, len(g.Params))
	for _, p := range g.Params {
		names = append(names, p.Name)
	}
	sort.Strings(names)
	return names
}

// frame is the data a template sees. It is deliberately small: a template gets
// the frame, its declared parameters, and deterministic noise, and nothing that
// could reach a clock, the filesystem or the network.
type frame struct {
	W, H float64
	Seed int64

	params map[string]float64
	id     string
	// err records the first undeclared-parameter read. text/template swallows a
	// function's second return only when the signature declares one, and a
	// template that reads a knob its author forgot to declare should fail the
	// render rather than silently draw with zero.
	err error
}

// Param reads a declared parameter. Reading an undeclared one is an error
// rather than a zero, which is the rejection path the plan names first: a
// template that draws with an undeclared knob is one whose behaviour its own
// parameter list does not describe.
func (f *frame) Param(name string) float64 {
	v, ok := f.params[name]
	if !ok {
		if f.err == nil {
			f.err = &UndeclaredParamError{GeneratorID: f.id, Param: name}
		}
		return 0
	}
	return v
}

// Noise is the same value-noise field the hand-written generators use, so a
// seed means the same thing across the whole vector family.
func (f *frame) Noise(x, y any) float64 { return vector.Noise(num(x), num(y), f.Seed) }

// Fbm is fractal noise: several octaves of Noise, which is what makes a field
// read as terrain or cloud rather than as a grid.
func (f *frame) Fbm(x, y any, octaves int) float64 {
	return vector.Fbm(num(x), num(y), octaves, f.Seed)
}

// Rand is a deterministic pseudo-random value in [0,1) keyed by an index, for a
// template that wants a scatter rather than a field. It is a function of the
// seed and the index alone — there is no hidden stream position — so a template
// that draws mark 40 gets the same value whatever it drew before it.
func (f *frame) Rand(index any) float64 { return vector.Noise(num(index)*12.9898, 78.233, f.Seed) }

// UndeclaredParamError reports a template reading a knob its author did not
// declare.
type UndeclaredParamError struct {
	GeneratorID, Param string
}

func (e *UndeclaredParamError) Error() string {
	return fmt.Sprintf(
		"authoring: generator %q reads parameter %q, which its declared parameter list does not contain; "+
			"declare it with a range and a default, or stop reading it",
		e.GeneratorID, e.Param)
}

// templateFuncs is the whole function surface a template gets. It is a closed
// list rather than a starting point: every entry is arithmetic or formatting,
// none can reach outside the process, and adding one is a decision about what
// an untrusted document may do.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b any) float64 { return num(a) + num(b) },
		"sub": func(a, b any) float64 { return num(a) - num(b) },
		"mul": func(a, b any) float64 { return num(a) * num(b) },
		"div": func(a, b any) float64 {
			d := num(b)
			if d == 0 {
				return 0
			}
			return num(a) / d
		},
		"min":   func(a, b any) float64 { return math.Min(num(a), num(b)) },
		"max":   func(a, b any) float64 { return math.Max(num(a), num(b)) },
		"abs":   func(a any) float64 { return math.Abs(num(a)) },
		"sin":   func(a any) float64 { return math.Sin(num(a)) },
		"cos":   func(a any) float64 { return math.Cos(num(a)) },
		"sqrt":  func(a any) float64 { return math.Sqrt(math.Abs(num(a))) },
		"pow":   func(a, b any) float64 { return math.Pow(num(a), num(b)) },
		"hypot": func(a, b any) float64 { return math.Hypot(num(a), num(b)) },
		"pi":    func() float64 { return math.Pi },
		"float": func(a any) float64 { return num(a) },
		"int":   func(a any) int { return int(num(a)) },
		// seq is how a template repeats a mark. Bounded at 20000 so an authored
		// generator cannot produce a document that takes a rasterizer minutes.
		"seq": func(n any) []int {
			count := int(num(n))
			if count < 0 {
				count = 0
			}
			if count > 20000 {
				count = 20000
			}
			out := make([]int, count)
			for i := range out {
				out[i] = i
			}
			return out
		},
		// f formats a coordinate. Fixed at three decimals for the same reason
		// the hand-written generators do it: a golden file that changes in the
		// seventeenth decimal because a CPU rounded differently is a test that
		// fails for no reason anybody can act on.
		"f": func(v any) string {
			value := num(v)
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return "0"
			}
			return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.3f", value), "0"), ".")
		},
		// Ink slots, so a template names them through a function rather than
		// typing the literal and getting it subtly wrong.
		"paper":  func() string { return vector.InkPaper },
		"ink":    func() string { return vector.InkInk },
		"accent": func() string { return vector.InkAccent },
	}
}

// num coerces whatever a template hands an arithmetic helper.
//
// Every helper takes `any` rather than float64 because text/template does not
// convert, and `range $i := seq 12` yields an int. A model that writes
// `mul .W $i` — the obvious thing — would otherwise get "wrong type for value;
// expected float64; got int" and burn a paid round trip on a type error rather
// than on the art direction. The plan's stated risk for this phase is that
// authored generators fail validation often enough to be useless, and this is
// the single largest avoidable source of that.
func num(v any) float64 {
	switch value := v.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	default:
		return 0
	}
}

// paramReadPattern finds every `Param "name"` read in a template, so validation
// can compare what the template reads against what its author declared without
// executing it.
var paramReadPattern = regexp.MustCompile(`\.Param\s+"([^"]+)"`)

// ReadParams returns the parameter names a template reads.
func ReadParams(tmpl string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, m := range paramReadPattern.FindAllStringSubmatch(tmpl, -1) {
		if _, dup := seen[m[1]]; dup {
			continue
		}
		seen[m[1]] = struct{}{}
		out = append(out, m[1])
	}
	sort.Strings(out)
	return out
}
