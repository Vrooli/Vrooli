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
// models occasionally emit (e.g. qwen3:1.7b merges keys into the provider_ids array:
// {"provider_ids":["provider.leaf","confidence":0.45,...]} — a colon inside an array, which
// strict JSON rejects). quotedTokenRe pulls quoted strings; confidenceRe and
// reasonRe pull the two scalar fields. Over-extraction is safe: widenPolicy
// intersects the recovered ids against the live registry and drops the rest.
var (
	quotedTokenRe = regexp.MustCompile(`"([^"\\]*(?:\\.[^"\\]*)*)"`)
	confidenceRe  = regexp.MustCompile(`"confidence"\s*:\s*([0-9]*\.?[0-9]+)`)
	reasonRe      = regexp.MustCompile(`"reason"\s*:\s*"([^"\\]*(?:\\.[^"\\]*)*)"`)
)

const defaultClassifierRole = "classify.routing"

// classifierMaxTokens caps the generation. The model only emits a short JSON
// object ({"provider_ids":[…],"confidence":…}); 256 is ample headroom.
const classifierMaxTokens = 256

const (
	classifierMaxProfiles         = 64
	classifierMaxDescriptionBytes = 12000
)

// generateFn shells out one completion (via internal/ollama). Seamed so tests
// inject a deterministic runner instead of the real model.
type generateFn func(ctx context.Context, role, prompt string, maxTokens int) ([]byte, error)

// availFn reports daemon reachability for the Phase 7 Status surface. Seamed for
// tests.
type availFn func(ctx context.Context) bool

// OllamaClassifier is the production Classifier: it asks a local Ollama model to
// pick provider leaves for a query, reading only the registry's provider
// descriptions (passed in as profiles). It contains no provider-specific
// knowledge — swapping the registry's descriptions changes routing with no code
// change here, holding the thin-router invariant.
type OllamaClassifier struct {
	role      string
	maxTokens int
	generate  generateFn
	checkUp   availFn
}

// NewOllamaClassifier returns the CLI-backed classifier. Role resolves from
// SEARCH_HUB_CLASSIFIER_ROLE or falls back to defaultClassifierRole.
func NewOllamaClassifier() *OllamaClassifier {
	role := strings.TrimSpace(os.Getenv("SEARCH_HUB_CLASSIFIER_ROLE"))
	if role == "" {
		role = defaultClassifierRole
	}
	return &OllamaClassifier{
		role:      role,
		maxTokens: classifierMaxTokens,
		generate:  ollama.Generate,
		checkUp:   ollama.Available,
	}
}

// Classify asks the model which provider leaves best match the query, given the
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

	out, err := c.generate(ctx, c.role, prompt, c.maxTokens)
	if err != nil {
		return ClassifyResult{}, fmt.Errorf("ollama generate: %w", err)
	}
	result, err := parseClassifierResponse(out)
	if err != nil {
		return ClassifyResult{}, err
	}
	// A small model can confidently omit a corpus whose description is an
	// exact match for an external-fact query. Preserve the description-driven
	// boundary with a conservative recall guard: it can only add profiles that
	// explicitly advertise external-world facts, and never applies to queries
	// that identify project/code/configuration work.
	return addDescriptionBackedRecall(query, profiles, result), nil
}

func addDescriptionBackedRecall(query string, profiles []ProviderProfile, result ClassifyResult) ClassifyResult {
	if !looksLikeExternalFact(query) {
		return result
	}

	selected := make(map[string]struct{}, len(result.ProviderIDs))
	for _, id := range result.ProviderIDs {
		selected[id] = struct{}{}
	}
	for _, profile := range profiles {
		if !describesExternalFacts(profile.Description) {
			continue
		}
		if _, already := selected[profile.ProviderID]; already {
			continue
		}
		result.ProviderIDs = append(result.ProviderIDs, profile.ProviderID)
		selected[profile.ProviderID] = struct{}{}
	}
	return result
}

func looksLikeExternalFact(query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return false
	}
	for _, projectTerm := range []string{
		"project", "repository", "repo", "code", "configuration", "config",
		"scenario", "cli", "api", "retry logic", "where is", "which command",
	} {
		if strings.Contains(q, projectTerm) {
			return false
		}
	}
	for _, externalTerm := range []string{
		"latest", "newest", "current", "version", "release", "releases",
		"feature", "features", "current events", "vendor", "product", "news",
		"what changed",
	} {
		if strings.Contains(q, externalTerm) {
			return true
		}
	}
	return false
}

func describesExternalFacts(description string) bool {
	d := strings.ToLower(description)
	for _, marker := range []string{
		"external world", "external-world", "current events", "third-party",
		"software release", "outside-world factual",
	} {
		if strings.Contains(d, marker) {
			return true
		}
	}
	return false
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
	ProviderIDs []string `json:"provider_ids"`
	// Types is a decode-only compatibility path for a stale model response.
	// New prompts never request it, and the router still intersects every token
	// against exact registered provider ids.
	Types      []string `json:"types"`
	Confidence float64  `json:"confidence"`
	Reason     string   `json:"reason"`
	WebShaped  bool     `json:"web_shaped"`
}

// buildClassifierPrompt renders the routing prompt. It lists each available
// provider leaf with its NL description (the only routing knowledge) and asks for a
// strict JSON answer. `/no_think` disables qwen3's chain-of-thought so the reply
// is just the JSON (fast, cheap).
func buildClassifierPrompt(query string, profiles []ProviderProfile) string {
	var b strings.Builder
	b.WriteString("You are a search router. Choose which registered corpus leaves can answer the user's query.\n")
	b.WriteString("Available corpus leaves (provider_id — type/group — what it holds):\n")
	descriptions := 0
	truncated := false
	for i, p := range profiles {
		if i >= classifierMaxProfiles || descriptions >= classifierMaxDescriptionBytes {
			truncated = true
			break
		}
		description := p.Description
		remaining := classifierMaxDescriptionBytes - descriptions
		if len(description) > remaining {
			description = description[:remaining]
			truncated = true
		}
		fmt.Fprintf(&b, "- %s — %s/%s — %s\n", p.ProviderID, p.Type, p.Group, description)
		descriptions += len(description)
	}
	if truncated {
		fmt.Fprintf(&b, "- [candidate descriptions truncated at %d bytes / %d leaves; omitted candidates are handled by bounded recall fallback]\n", classifierMaxDescriptionBytes, classifierMaxProfiles)
	}
	for _, p := range profiles {
		if len(p.OmittedProviderIDs) == 0 {
			continue
		}
		fmt.Fprintf(&b, "- [omitted provider_ids: %s]\n", strings.Join(p.OmittedProviderIDs, ", "))
		break
	}
	b.WriteString("\nRules:\n")
	b.WriteString("- Return EVERY provider_id that could plausibly contain a relevant result; prefer recall over precision.\n")
	b.WriteString("- Match the query against what each corpus HOLDS per its description, not against surface keywords. A question about outside-world facts (software releases/versions/features, current events, products, vendors) matches every corpus whose description says it stores such external-world knowledge — include those alongside any other plausible types. Questions about THIS project's own code, configuration, or how past work here was done are NOT outside-world facts.\n")
	b.WriteString("- For implementation questions asking where code, symbols, call paths, retry logic, configuration, or commands live, prefer leaves whose descriptions explicitly hold code/reference, domain, documentation, or command evidence. Do not choose a narrative record or memory corpus solely because it mentions a past implementation; use it only when the question asks about history, decisions, or prior work.\n")
	b.WriteString("- Hard rule for code-location questions (for example 'where is the retry logic' or 'where is this implemented'): never route only to skill or record leaves when any code, command, domain, or documentation leaf is listed; select at least one of those evidence leaves.\n")
	b.WriteString("- Use only provider_id values from the list above.\n")
	b.WriteString("- confidence is your certainty in [0,1] that the chosen types are sufficient; use a LOW value (below 0.45) when unsure so the router widens its search.\n")
	b.WriteString("- web_shaped is true ONLY when the query wants fresh, live, public-web information (current events, \"latest\"/\"today\", real-time facts, things outside an internal corpus); false for questions answerable from internal/project knowledge.\n")
	b.WriteString("- Output ONLY one JSON object, no prose: {\"provider_ids\":[\"<provider_id>\",...],\"confidence\":<0..1>,\"reason\":\"<short>\",\"web_shaped\":<true|false>}\n")
	b.WriteString("- Keep reason under 8 words; never restate the query or the descriptions.\n\n")
	fmt.Fprintf(&b, "Query: %s /no_think", query)
	return b.String()
}

// parseClassifierResponse unwraps the gateway envelope ({"response":"…"}),
// strips qwen3 <think> blocks, and decodes the routing JSON — first strictly,
// then via a salvage pass for the malformed-JSON cases small models hit.
func parseClassifierResponse(raw []byte) (ClassifyResult, error) {
	text := ollama.StripThink(ollama.UnwrapResponse(raw))

	// Strict path: a well-formed {"provider_ids":[…],"confidence":…,"reason":…} object.
	if obj := ollama.ExtractJSONObject(text); obj != "" {
		var decoded classifierJSON
		if err := json.Unmarshal([]byte(obj), &decoded); err == nil {
			ids := decoded.ProviderIDs
			if len(ids) == 0 {
				ids = decoded.Types
			}
			res := normalizeClassifier(ids, decoded.Confidence, decoded.Reason)
			res.WebShaped = decoded.WebShaped
			return res, nil
		}
	}

	// Salvage path: recover the provider ids + scalars heuristically from
	// malformed output (see the salvage regexes). widenPolicy filters unknown ids.
	if ids, conf, reason, ok := salvageClassifier(text); ok {
		return normalizeClassifier(ids, conf, reason), nil
	}

	return ClassifyResult{}, fmt.Errorf("no usable classifier JSON in model response: %q", truncateForErr(text))
}

// normalizeClassifier trims the type tokens and clamps confidence to [0,1].
func normalizeClassifier(rawIDs []string, conf float64, reason string) ClassifyResult {
	ids := make([]string, 0, len(rawIDs))
	for _, id := range rawIDs {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	if conf < 0 {
		conf = 0
	}
	if conf > 1 {
		conf = 1
	}
	return ClassifyResult{ProviderIDs: ids, Types: append([]string(nil), ids...), Confidence: conf, Rationale: strings.TrimSpace(reason)}
}

// salvageClassifier recovers (provider ids, confidence, reason) from output that failed
// strict JSON decoding. It scopes id extraction to the "provider_ids":[ … ] span so
// it does not vacuum quoted words out of prose, then reads confidence/reason
// from anywhere in the text. ok is false when no types span is found.
func salvageClassifier(text string) (ids []string, confidence float64, reason string, ok bool) {
	ti := strings.Index(text, `"provider_ids"`)
	if ti == -1 {
		// Compatibility with a stale classifier response while the model role is
		// being refreshed. Exact-id intersection prevents type fan-out.
		ti = strings.Index(text, `"types"`)
	}
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
			ids = append(ids, tok)
		}
	}
	if len(ids) == 0 {
		return nil, 0, "", false
	}
	if m := confidenceRe.FindStringSubmatch(text); m != nil {
		confidence, _ = strconv.ParseFloat(m[1], 64)
	}
	if m := reasonRe.FindStringSubmatch(text); m != nil {
		reason = m[1]
	}
	return ids, confidence, reason, true
}

func truncateForErr(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 120 {
		return s[:120] + "…"
	}
	return s
}
