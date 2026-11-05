package measures

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Completer is the single-shot text-completion seam the constrained extractor
// depends on. The owning scenario supplies the implementation (e.g. a
// `resource-ollama gateway generate` shell-out, exactly like search-hub's
// classifier); measures-go itself stays free of any model/transport dependency
// so the contract library remains lean and offline-testable.
type Completer interface {
	// Complete returns the model's raw text for a single prompt. It must be
	// deterministic-friendly (the impl should use temperature 0).
	Complete(ctx context.Context, prompt string) (string, error)
}

// CompleterFunc adapts a function to the Completer interface.
type CompleterFunc func(ctx context.Context, prompt string) (string, error)

// Complete calls the underlying function.
func (f CompleterFunc) Complete(ctx context.Context, prompt string) (string, error) {
	return f(ctx, prompt)
}

// extractedConfidence is the self-reported confidence assigned to a constrained
// LLM extraction when the model does not emit one. It is deliberately below
// DefaultConfidenceThreshold so a bare best-effort extraction does NOT clear the
// auto-execute gate on its own — an LLM-guessed param routes to confirmation
// rather than an unattended answer unless the model is explicitly confident.
const extractedConfidence = 0.65

// LLMExtractor is the constrained-extraction implementation behind the Phase 1
// ParamExtractor seam. It is the second + third tiers of the three-tier
// degradation (the first, canonical time_window, is handled deterministically
// by ResolveParams before the extractor is consulted):
//
//   - constrained (enum / numeric bounds): the prompt pins the answer to the
//     proto-derived `allowed` set (or Min/Max), and a value outside it is
//     rejected by ResolveParams — the model cannot invent an illegal value;
//   - bare best-effort: no value space, so the prompt leans on the field's
//     description/bounds for grounding and the result is low-confidence.
//
// It abstains (Found=false) whenever the model says the param is absent or emits
// an unusable answer — abstention (→ needs[]) is always preferred over a guess.
type LLMExtractor struct {
	// Completer runs one constrained completion. Required.
	Completer Completer
}

// NewLLMExtractor constructs a constrained extractor over the given completer.
func NewLLMExtractor(c Completer) *LLMExtractor { return &LLMExtractor{Completer: c} }

// extractionReply is the strict JSON shape the extractor asks the model for.
type extractionReply struct {
	// Found reports whether the param was present in the question.
	Found bool `json:"found"`
	// Value is the extracted value (ignored when Found is false). For a
	// constrained param it must be one of the allowed members verbatim.
	Value string `json:"value"`
	// Confidence is the model's certainty in [0,1]; optional (defaults applied).
	Confidence float64 `json:"confidence"`
}

// Extract asks the model to pull a single parameter out of the question,
// constrained to `allowed` when the param has a bounded value space. It returns
// (found=false) on any abstention or parse failure so the resolver routes a
// required-but-unresolved param to needs[] rather than guessing.
func (e *LLMExtractor) Extract(ctx context.Context, question string, p Param, allowed []string) (ExtractResult, error) {
	if e.Completer == nil {
		return ExtractResult{}, fmt.Errorf("measures: LLMExtractor has no Completer")
	}
	prompt := buildExtractionPrompt(question, p, allowed)
	raw, err := e.Completer.Complete(ctx, prompt)
	if err != nil {
		return ExtractResult{}, fmt.Errorf("measures: extract %q: %w", p.Name, err)
	}
	reply, ok := parseExtractionReply(raw)
	if !ok || !reply.Found {
		return ExtractResult{}, nil // abstain
	}
	value := strings.TrimSpace(reply.Value)
	if value == "" {
		return ExtractResult{}, nil
	}
	conf := reply.Confidence
	if conf <= 0 || conf > 1 {
		conf = extractedConfidence
	}
	// The hard constraint (membership in `allowed`) is re-checked by
	// ResolveParams; we return the value verbatim and let that single
	// enforcement point reject an out-of-set answer.
	return ExtractResult{Value: value, Found: true, Confidence: conf}, nil
}

// buildExtractionPrompt renders a strict, constrained extraction prompt. The
// `/no_think` suffix mirrors search-hub's classifier so a reasoning model
// (qwen3) emits just the JSON; a non-reasoning model ignores it harmlessly.
func buildExtractionPrompt(question string, p Param, allowed []string) string {
	var b strings.Builder
	b.WriteString("You extract one parameter from a user's analytical question.\n")
	fmt.Fprintf(&b, "Parameter name: %s\n", p.Name)
	if d := strings.TrimSpace(p.Description); d != "" {
		fmt.Fprintf(&b, "Parameter meaning: %s\n", d)
	}
	switch {
	case len(allowed) > 0:
		fmt.Fprintf(&b, "The value MUST be exactly one of: %s\n", strings.Join(allowed, ", "))
	case p.Min != nil && p.Max != nil:
		fmt.Fprintf(&b, "The value MUST be an integer between %d and %d inclusive.\n", *p.Min, *p.Max)
	case p.Min != nil:
		fmt.Fprintf(&b, "The value MUST be an integer >= %d.\n", *p.Min)
	case p.Max != nil:
		fmt.Fprintf(&b, "The value MUST be an integer <= %d.\n", *p.Max)
	}
	b.WriteString("\nRules:\n")
	b.WriteString("- If the question does not specify this parameter, set found=false.\n")
	b.WriteString("- Never invent or guess a value; abstain (found=false) when unsure.\n")
	if len(allowed) > 0 {
		b.WriteString("- value must be copied verbatim from the allowed list above.\n")
	}
	b.WriteString("- Output ONLY one JSON object, no prose: {\"found\":<bool>,\"value\":\"<string>\",\"confidence\":<0..1>}\n\n")
	fmt.Fprintf(&b, "Question: %s /no_think", question)
	return b.String()
}

// parseExtractionReply unwraps the gateway envelope, strips reasoning blocks,
// and decodes the strict JSON. It returns ok=false when no usable object is
// found, so the extractor abstains rather than risking a bad value.
func parseExtractionReply(raw string) (extractionReply, bool) {
	text := stripThinkBlocks(unwrapResponse(raw))
	obj := firstJSONObject(text)
	if obj == "" {
		return extractionReply{}, false
	}
	var reply extractionReply
	if err := json.Unmarshal([]byte(obj), &reply); err != nil {
		// Tolerate a bare confidence emitted as a string ("0.7").
		var loose struct {
			Found      bool   `json:"found"`
			Value      string `json:"value"`
			Confidence any    `json:"confidence"`
		}
		if err2 := json.Unmarshal([]byte(obj), &loose); err2 != nil {
			return extractionReply{}, false
		}
		reply.Found = loose.Found
		reply.Value = loose.Value
		reply.Confidence = coerceFloat(loose.Confidence)
	}
	return reply, true
}

// coerceFloat best-efforts a JSON value (number or numeric string) into a float.
func coerceFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
			return f
		}
	}
	return 0
}

// unwrapResponse returns the model text, unwrapping the `resource-ollama gateway
// generate --json` envelope ({"response":"…"}) when present, else the raw text.
func unwrapResponse(raw string) string {
	trimmed := strings.TrimSpace(raw)
	var env struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal([]byte(trimmed), &env); err == nil && env.Response != "" {
		return env.Response
	}
	return trimmed
}

// stripThinkBlocks removes <think>…</think> spans some reasoning models emit
// even with /no_think (they come back empty but present).
func stripThinkBlocks(s string) string {
	for {
		start := strings.Index(s, "<think>")
		if start == -1 {
			break
		}
		end := strings.Index(s, "</think>")
		if end == -1 || end < start {
			return s[:start]
		}
		s = s[:start] + s[end+len("</think>"):]
	}
	return s
}

// firstJSONObject returns the first balanced {…} object in s (ignoring braces
// inside strings), or "" when none is present.
func firstJSONObject(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}
	depth, inStr, escaped := 0, false, false
	for i := start; i < len(s); i++ {
		c := s[i]
		switch {
		case escaped:
			escaped = false
		case c == '\\' && inStr:
			escaped = true
		case c == '"':
			inStr = !inStr
		case inStr:
			// inside a string literal: ignore structural chars
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}
