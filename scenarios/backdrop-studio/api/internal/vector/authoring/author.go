package authoring

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"backdrop-studio/internal/vector"
)

// Role is the ai-gateway inference role an authoring request resolves through.
//
// A role, never a model slug. ai-gateway's conformance policy refuses a
// concrete model name in a runtime surface, and the reason is not bureaucratic:
// a slug in this file would have to be edited every time the provider catalog
// moves, by someone who does not know what this scenario needs. The role says
// what is needed — long-form code authoring — and ai-gateway's catalog says
// which model currently meets it.
const Role = "author.generator"

// Client is the text-generation seam. It is an interface so the authoring flow
// can be tested without a running gateway, and so this package holds no
// provider URL, credential or model name of its own.
type Client interface {
	// Author sends a prompt and returns the model's text plus the model
	// ai-gateway actually resolved. The second is the disclosure record and is
	// never a model this scenario asked for.
	Author(ctx context.Context, prompt string) (text string, modelID string, err error)
}

// Request is one authoring ask.
type Request struct {
	// ID is the generator id to author. It must not shadow a built-in preset.
	ID string
	// Brief is the art direction, in the operator's words. It is the only part
	// of the prompt a caller writes; the contract around it is this package's.
	Brief string
}

// Author asks a model for a generator, validates it, and returns both.
//
// A generator that fails validation is returned with its report rather than
// discarded, because the report is what an operator needs to decide whether to
// re-ask with a sharper brief or give up on the idea — and because the plan
// asks for the measured failure rate, which nobody can compute from silence.
// Nothing is stored here: storage is the caller's decision and validation's
// verdict is its precondition.
func Author(ctx context.Context, client Client, req Request) (Generator, Report, error) {
	if client == nil {
		return Generator{}, Report{}, fmt.Errorf("authoring: no ai-gateway client is configured; generator authoring is unavailable on this host")
	}
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return Generator{}, Report{}, fmt.Errorf("authoring: a generator id is required")
	}
	if strings.TrimSpace(req.Brief) == "" {
		return Generator{}, Report{}, fmt.Errorf("authoring: a brief is required; a generator authored from no art direction is a random picture")
	}
	prompt := Prompt(req)
	text, modelID, err := client.Author(ctx, prompt)
	if err != nil {
		return Generator{}, Report{}, fmt.Errorf("authoring: ai-gateway role %q: %w", Role, err)
	}
	authored, err := parseResponse(text)
	if err != nil {
		return Generator{}, Report{}, err
	}
	authored.ID = id
	authored.Prompt = prompt
	authored.ModelID = modelID
	if strings.TrimSpace(authored.Name) == "" {
		authored.Name = id
	}
	report := Validate(authored, vector.Presets)
	authored.Validation = report
	return authored, report, nil
}

// authoredResponse is the shape a model must return.
type authoredResponse struct {
	Name     string      `json:"name"`
	Template string      `json:"template"`
	Params   []ParamSpec `json:"params"`
	Inks     []string    `json:"inks"`
}

// parseResponse reads the model's answer, tolerating the fenced code block
// models habitually wrap JSON in and nothing else. A response this cannot read
// is an error naming what was expected, not a silent empty generator.
func parseResponse(text string) (Generator, error) {
	body := strings.TrimSpace(text)
	if fenced := extractFenced(body); fenced != "" {
		body = fenced
	}
	var decoded authoredResponse
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		preview := body
		if len(preview) > 240 {
			preview = preview[:240] + "…"
		}
		return Generator{}, fmt.Errorf(
			"authoring: the model's answer is not the declared JSON object {name, template, params, inks}: %w (got %q)", err, preview)
	}
	if strings.TrimSpace(decoded.Template) == "" {
		return Generator{}, fmt.Errorf("authoring: the model returned no template")
	}
	return Generator{
		Name:     decoded.Name,
		Template: decoded.Template,
		Params:   decoded.Params,
		Inks:     decoded.Inks,
	}, nil
}

func extractFenced(text string) string {
	start := strings.Index(text, "```")
	if start < 0 {
		return ""
	}
	rest := text[start+3:]
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		rest = rest[nl+1:]
	}
	end := strings.Index(rest, "```")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}

// Prompt is the authoring request sent to the model.
//
// It states the contract rather than describing it, because every rejection
// path in Validate corresponds to a sentence here: a model that is not told the
// template may read only declared parameters will read undeclared ones, and a
// model that is not told the frame is variable will place marks in pixels. The
// prompt and the validator are two halves of one specification and drift
// between them shows up as a rejection rate.
func Prompt(req Request) string {
	var b strings.Builder
	b.WriteString("You are writing a backdrop generator for a design system. Return ONLY a JSON object, no prose.\n\n")

	b.WriteString("ART DIRECTION\n")
	b.WriteString(strings.TrimSpace(req.Brief))
	b.WriteString("\n\n")

	b.WriteString("SHAPE\n")
	b.WriteString(`{"name": "...", "template": "...", "params": [{"name":"...","min":0,"max":1,"default":0.5,"description":"..."}], "inks": ["$brand.background", "$brand.primary"]}` + "\n\n")

	b.WriteString("THE TEMPLATE\n")
	b.WriteString("Go text/template producing SVG MARKS ONLY — no <svg> element, no <defs>, no <g>.\n")
	b.WriteString("The document element, the depth plane and the ink substitution are added around your output.\n\n")

	b.WriteString("AVAILABLE IN THE TEMPLATE\n")
	b.WriteString("  .W .H              frame width and height in user units (floats). NEVER assume a size.\n")
	b.WriteString("  .Param \"name\"      a declared parameter, already clamped to its range\n")
	b.WriteString("  .Noise x y         deterministic value noise in [0,1]\n")
	b.WriteString("  .Fbm x y octaves   deterministic fractal noise in [0,1]\n")
	b.WriteString("  .Rand i            deterministic pseudo-random value in [0,1] keyed by an index\n")
	b.WriteString("  paper ink accent   the three brand ink slots, as colour values\n")
	b.WriteString("  seq n              a slice of 0..n-1, for repeating a mark (n is capped at 20000)\n")
	b.WriteString("  f v                format a coordinate; wrap EVERY number you emit in it\n")
	b.WriteString("  add sub mul div min max abs sin cos sqrt pow hypot pi float int\n\n")

	b.WriteString("RULES — each is checked before the generator is stored, and a failure means it is rejected\n")
	b.WriteString("  1. Every number you emit must be a fraction of .W or .H. A generator with pixel\n")
	b.WriteString("     constants is rejected: it is asked for at frames from 390x844 to 2048x2732.\n")
	b.WriteString("  2. Every parameter the template reads must appear in params, with min, max and a\n")
	b.WriteString("     default inside that range.\n")
	b.WriteString("  3. Deterministic. Use .Noise/.Fbm/.Rand for variation, never a range over a map.\n")
	b.WriteString("     The same seed must produce the same bytes.\n")
	b.WriteString("  4. Inert. No <script>, <image>, <use>, <foreignObject>, no href to anything, no\n")
	b.WriteString("     http:// or data: URL, no on* attribute, no @import, no <!DOCTYPE or <!ENTITY.\n")
	b.WriteString("  5. Colour comes only from paper/ink/accent, and every slot you use goes in inks.\n")
	b.WriteString("  6. Tone is carried by the density of marks, not by flat fills. That is the whole\n")
	b.WriteString("     point of this lane: a flat shape has no tone for a screen to modulate, and the\n")
	b.WriteString("     styles that failed this catalog's review failed exactly that way.\n")
	b.WriteString("  7. Leave the upper-left third open enough for a headline to sit on it.\n\n")

	b.WriteString("EXAMPLE OF THE FORM (not the art direction — write your own)\n")
	b.WriteString(`  <rect width="{{f .W}}" height="{{f .H}}" fill="{{paper}}"/>` + "\n")
	b.WriteString(`  {{$f := .}}{{range $i := seq 120}}` + "\n")
	b.WriteString(`  <circle cx="{{f (mul $f.W ($f.Rand $i))}}" cy="{{f (mul $f.H 0.5)}}" r="{{f (mul $f.H 0.01)}}" fill="{{ink}}"/>` + "\n")
	b.WriteString(`  {{end}}` + "\n")
	return b.String()
}
