package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"search-hub/internal/ollama"
)

// Salvage regexes recover a routing decision from the *malformed* JSON small
// models occasionally emit (e.g. qwen3:1.7b merges keys into the types array:
// {"types":["command","confidence":0.45,...]} — a colon inside an array, which
// strict JSON rejects). quotedTokenRe pulls quoted strings; confidenceRe and
// reasonRe pull the two scalar fields. Over-extraction is safe: widenPolicy
// intersects the recovered types against the live registry and drops the rest.
var (
	quotedTokenRe = regexp.MustCompile(`"([^"\\]*(?:\\.[^"\\]*)*)"`)
	confidenceRe  = regexp.MustCompile(`"confidence"\s*:\s*([0-9]*\.?[0-9]+)`)
	reasonRe      = regexp.MustCompile(`"reason"\s*:\s*"([^"\\]*(?:\\.[^"\\]*)*)"`)
)

// defaultClassifierModel is the local Ollama model that routes queries. qwen3:1.7b
// is small, fast, already pulled (per the Phase 0 dependency decision), and emits
// the JSON we ask for. Override with SEARCH_HUB_CLASSIFIER_MODEL.
const defaultClassifierModel = "qwen3:1.7b"

// classifierMaxTokens caps the generation. The model only emits a short JSON
// object ({"types":[…],"confidence":…}); 256 is ample headroom.
const classifierMaxTokens = 256

// generateFn shells out one completion (via internal/ollama). Seamed so tests
// inject a deterministic runner instead of the real model.
type generateFn func(ctx context.Context, model, prompt string, maxTokens int) ([]byte, error)

// availFn reports daemon reachability for the Phase 7 Status surface. Seamed for
// tests.
type availFn func(ctx context.Context) bool

// OllamaClassifier is the production Classifier: it asks a local Ollama model to
// pick provider types for a query, reading only the registry's provider
// descriptions (passed in as profiles). It contains no provider-specific
// knowledge — swapping the registry's descriptions changes routing with no code
// change here, holding the thin-router invariant.
type OllamaClassifier struct {
	model     string
	maxTokens int
	generate  generateFn
	checkUp   availFn
}

// NewOllamaClassifier returns the CLI-backed classifier. Model resolves from
// SEARCH_HUB_CLASSIFIER_MODEL or falls back to defaultClassifierModel.
func NewOllamaClassifier() *OllamaClassifier {
	model := strings.TrimSpace(os.Getenv("SEARCH_HUB_CLASSIFIER_MODEL"))
	if model == "" {
		model = defaultClassifierModel
	}
	return &OllamaClassifier{
		model:     model,
		maxTokens: classifierMaxTokens,
		generate:  ollama.Generate,
		checkUp:   ollama.Available,
	}
}

// Classify asks the model which provider types best match the query, given the
// available provider descriptions. It returns ClassifyResult{} + an error on any
// model/parse failure — the router converts that into widen-to-all + degraded.
func (c *OllamaClassifier) Classify(ctx context.Context, query string, profiles []ProviderProfile) (ClassifyResult, error) {
	if strings.TrimSpace(query) == "" {
		return ClassifyResult{}, fmt.Errorf("empty query")
	}
	if len(profiles) == 0 {
		return ClassifyResult{}, fmt.Errorf("no provider profiles to route over")
	}
	prompt := buildClassifierPrompt(query, profiles)

	out, err := c.generate(ctx, c.model, prompt, c.maxTokens)
	if err != nil {
		return ClassifyResult{}, fmt.Errorf("ollama generate: %w", err)
	}
	return parseClassifierResponse(out)
}

// Available reports whether the Ollama daemon is reachable. Not called on the
// query hot path (that relies on Classify's error for graceful degradation);
// reserved for the Phase 7 Status surface.
func (c *OllamaClassifier) Available(ctx context.Context) bool {
	if c.checkUp == nil {
		return false
	}
	return c.checkUp(ctx)
}

// classifierType mirrors the JSON the model is asked to emit.
type classifierJSON struct {
	Types      []string `json:"types"`
	Confidence float64  `json:"confidence"`
	Reason     string   `json:"reason"`
	WebShaped  bool     `json:"web_shaped"`
}

// buildClassifierPrompt renders the routing prompt. It lists each available
// type with its NL description (the only routing knowledge) and asks for a
// strict JSON answer. `/no_think` disables qwen3's chain-of-thought so the reply
// is just the JSON (fast, cheap).
func buildClassifierPrompt(query string, profiles []ProviderProfile) string {
	var b strings.Builder
	b.WriteString("You are a search router. Choose which corpus types can answer the user's query.\n")
	b.WriteString("Available corpus types (id — what it holds):\n")
	for _, p := range profiles {
		fmt.Fprintf(&b, "- %s — %s\n", p.Type, p.Description)
	}
	b.WriteString("\nRules:\n")
	b.WriteString("- Return EVERY type that could plausibly contain a relevant result; prefer recall over precision.\n")
	b.WriteString("- Use only type ids from the list above.\n")
	b.WriteString("- confidence is your certainty in [0,1] that the chosen types are sufficient; use a LOW value (below 0.45) when unsure so the router widens its search.\n")
	b.WriteString("- web_shaped is true ONLY when the query wants fresh, live, public-web information (current events, \"latest\"/\"today\", real-time facts, things outside an internal corpus); false for questions answerable from internal/project knowledge.\n")
	b.WriteString("- Output ONLY one JSON object, no prose: {\"types\":[\"<id>\",...],\"confidence\":<0..1>,\"reason\":\"<short>\",\"web_shaped\":<true|false>}\n\n")
	fmt.Fprintf(&b, "Query: %s /no_think", query)
	return b.String()
}

// parseClassifierResponse unwraps the gateway envelope ({"response":"…"}),
// strips qwen3 <think> blocks, and decodes the routing JSON — first strictly,
// then via a salvage pass for the malformed-JSON cases small models hit.
func parseClassifierResponse(raw []byte) (ClassifyResult, error) {
	text := ollama.StripThink(ollama.UnwrapResponse(raw))

	// Strict path: a well-formed {"types":[…],"confidence":…,"reason":…} object.
	if obj := ollama.ExtractJSONObject(text); obj != "" {
		var decoded classifierJSON
		if err := json.Unmarshal([]byte(obj), &decoded); err == nil {
			res := normalizeClassifier(decoded.Types, decoded.Confidence, decoded.Reason)
			res.WebShaped = decoded.WebShaped
			return res, nil
		}
	}

	// Salvage path: recover the type tokens + scalars heuristically from
	// malformed output (see the salvage regexes). widenPolicy filters non-types.
	if types, conf, reason, ok := salvageClassifier(text); ok {
		return normalizeClassifier(types, conf, reason), nil
	}

	return ClassifyResult{}, fmt.Errorf("no usable classifier JSON in model response: %q", truncateForErr(text))
}

// normalizeClassifier trims the type tokens and clamps confidence to [0,1].
func normalizeClassifier(rawTypes []string, conf float64, reason string) ClassifyResult {
	types := make([]string, 0, len(rawTypes))
	for _, t := range rawTypes {
		if t = strings.TrimSpace(t); t != "" {
			types = append(types, t)
		}
	}
	if conf < 0 {
		conf = 0
	}
	if conf > 1 {
		conf = 1
	}
	return ClassifyResult{Types: types, Confidence: conf, Rationale: strings.TrimSpace(reason)}
}

// salvageClassifier recovers (types, confidence, reason) from output that failed
// strict JSON decoding. It scopes type extraction to the "types":[ … ] span so
// it does not vacuum quoted words out of prose, then reads confidence/reason
// from anywhere in the text. ok is false when no types span is found.
func salvageClassifier(text string) (types []string, confidence float64, reason string, ok bool) {
	ti := strings.Index(text, `"types"`)
	if ti == -1 {
		return nil, 0, "", false
	}
	lb := strings.Index(text[ti:], "[")
	if lb == -1 {
		return nil, 0, "", false
	}
	start := ti + lb
	rb := strings.Index(text[start:], "]")
	if rb == -1 {
		return nil, 0, "", false
	}
	span := text[start : start+rb+1]
	for _, m := range quotedTokenRe.FindAllStringSubmatch(span, -1) {
		if tok := strings.TrimSpace(m[1]); tok != "" {
			types = append(types, tok)
		}
	}
	if len(types) == 0 {
		return nil, 0, "", false
	}
	if m := confidenceRe.FindStringSubmatch(text); m != nil {
		confidence, _ = strconv.ParseFloat(m[1], 64)
	}
	if m := reasonRe.FindStringSubmatch(text); m != nil {
		reason = m[1]
	}
	return types, confidence, reason, true
}

func truncateForErr(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 120 {
		return s[:120] + "…"
	}
	return s
}
