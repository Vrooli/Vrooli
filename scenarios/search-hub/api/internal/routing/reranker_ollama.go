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

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

type (
	generateFn func(ctx context.Context, role, prompt string, maxTokens int) ([]byte, error)
	availFn    func(ctx context.Context) bool
)

// Salvage regexes recover a rerank decision from output that failed strict JSON
// decoding (small models occasionally wrap or malform the array). rerankObjRe
// pulls each {…} score object; idxRe / scoreRe read the two fields inside it,
// order-independently. Over-/under-extraction is safe: an out-of-range index is
// dropped and a missing score defaults to 0 (least relevant), so a partial
// recovery still degrades gracefully toward the bottom of the list.
var (
	rerankObjRe    = regexp.MustCompile(`\{[^{}]*\}`)
	rerankIdxRe    = regexp.MustCompile(`"index"\s*:\s*(\d+)`)
	rerankScoreRe  = regexp.MustCompile(`"score"\s*:\s*(-?[0-9]*\.?[0-9]+)`)
	rerankScoresRe = regexp.MustCompile(`"scores"`)
)

// defaultRerankerRole is the local Ollama role that reranks candidates.
//
// Phase 0 nominated qwen3:4b as the LLM-as-reranker, but it is NOT viable
// through the resource-ollama gateway: qwen3:4b does not honor the `/no_think`
// directive (it reasons in plain prose with no think-block to strip) and so
// generates unbounded chain-of-thought that exceeds the gateway's fixed ~60s
// request deadline — every rerank would time out and degrade. qwen3:1.7b DOES
// honor `/no_think`, emits the scores JSON directly, and reranks correctly in
// ~6s (verified: a clearly-relevant candidate scores 10, the rest 0). It is the
// same model the classifier uses, so no extra weight is loaded. This is a
// pragmatic LLM-as-reranker stand-in: when the approved bge-reranker-v2-m3
// dedicated resource lands, point a new Reranker impl at it — the router
// contract does not change (plan Appendix A.3 update). Override the model with
// SEARCH_HUB_RERANKER_ROLE.
const defaultRerankerRole = "rerank.llm_fallback"

// rerankerMaxTokens caps the generation. The model emits one short score object
// per candidate; 1024 leaves ample room for a few dozen candidates.
const rerankerMaxTokens = 1024

// OllamaReranker is the production Reranker: it asks a local Ollama model to
// score each candidate's relevance to the query (pointwise 0–10) and orders by
// that. It holds no provider-specific knowledge — it reasons only over the
// candidates' own title/snippet/type, so the thin-router invariant holds.
type OllamaReranker struct {
	role      string
	maxTokens int
	generate  generateFn
	checkUp   availFn
}

// NewOllamaReranker returns the CLI-backed reranker. Role resolves from
// SEARCH_HUB_RERANKER_ROLE or falls back to defaultRerankerRole.
func NewOllamaReranker() *OllamaReranker {
	role := strings.TrimSpace(os.Getenv("SEARCH_HUB_RERANKER_ROLE"))
	if role == "" {
		role = defaultRerankerRole
	}
	return &OllamaReranker{
		role:      role,
		maxTokens: rerankerMaxTokens,
		generate:  ollama.Generate,
		checkUp:   ollama.Available,
	}
}

// Rerank scores every candidate against the query and returns them ordered
// most-relevant first with RerankScore set. It returns an error on any
// model/parse failure — the router converts that into "keep grouping + degraded"
// (never a failed query).
func (c *OllamaReranker) Rerank(ctx context.Context, query string, candidates []*routingv1.SearchHit) ([]*routingv1.SearchHit, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("empty query")
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates to rerank")
	}
	prompt := buildRerankPrompt(query, candidates)

	out, err := c.generate(ctx, c.role, prompt, c.maxTokens)
	if err != nil {
		return nil, fmt.Errorf("ollama generate: %w", err)
	}
	scores, err := parseRerankResponse(out, len(candidates))
	if err != nil {
		return nil, err
	}
	return applyRerank(candidates, scores), nil
}

// Available reports whether the Ollama daemon is reachable. Not called on the
// query hot path (that relies on Rerank's error for graceful degradation);
// reserved for the Phase 7 Status surface.
func (c *OllamaReranker) Available(ctx context.Context) bool {
	if c.checkUp == nil {
		return false
	}
	return c.checkUp(ctx)
}

// rerankJSON mirrors the JSON the model is asked to emit.
type rerankJSON struct {
	Scores []struct {
		Index int     `json:"index"`
		Score float64 `json:"score"`
	} `json:"scores"`
}

// buildRerankPrompt renders the pointwise relevance-scoring prompt. Each
// candidate is listed by index with its type, title, and a short snippet (the
// only relevance signal); the model returns an integer 0–10 per index.
// `/no_think` disables qwen3's chain-of-thought so the reply is just the JSON.
func buildRerankPrompt(query string, candidates []*routingv1.SearchHit) string {
	var b strings.Builder
	b.WriteString("You are a search reranker. Score how well each candidate answers the user's query.\n")
	b.WriteString("Give each candidate an integer relevance score from 0 (irrelevant) to 10 (perfectly answers the query).\n\n")
	fmt.Fprintf(&b, "Query: %s\n\n", query)
	b.WriteString("Candidates:\n")
	for i, h := range candidates {
		title := strings.TrimSpace(h.GetTitle())
		if title == "" {
			title = h.GetId()
		}
		line := fmt.Sprintf("[%d] (%s) %s", i, h.GetType(), title)
		if snip := truncateForErr(h.GetSnippet()); snip != "" {
			line += " — " + snip
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\nRules:\n")
	b.WriteString("- Score EVERY candidate, referenced by its index.\n")
	b.WriteString("- Judge relevance to the query only; ignore the order candidates are listed in.\n")
	b.WriteString("- Output ONLY one JSON object, no prose: {\"scores\":[{\"index\":<i>,\"score\":<0..10>},...]}\n\n")
	b.WriteString("/no_think")
	return b.String()
}

// parseRerankResponse unwraps the gateway envelope, strips qwen3 <think> blocks,
// and decodes the per-candidate scores into a positional []float64 of length n
// (a candidate the model omitted keeps score 0 ⇒ ranked last). It tries strict
// JSON first, then a salvage pass for the malformed cases small models hit.
func parseRerankResponse(raw []byte, n int) ([]float64, error) {
	text := ollama.StripThink(ollama.UnwrapResponse(raw))

	// Strict path: a well-formed {"scores":[{"index":i,"score":s},...]} object.
	if obj := ollama.ExtractJSONObject(text); obj != "" {
		var decoded rerankJSON
		if err := json.Unmarshal([]byte(obj), &decoded); err == nil && len(decoded.Scores) > 0 {
			scores := make([]float64, n)
			for _, s := range decoded.Scores {
				if s.Index >= 0 && s.Index < n {
					scores[s.Index] = normalizeRerankScore(s.Score)
				}
			}
			return scores, nil
		}
	}

	// Salvage path: recover index/score pairs heuristically from malformed output.
	if scores, ok := salvageRerank(text, n); ok {
		return scores, nil
	}

	return nil, fmt.Errorf("no usable reranker JSON in model response: %q", truncateForErr(text))
}

// salvageRerank recovers (index, score) pairs from output that failed strict
// decoding. It scopes extraction to the "scores":[ … ] span when present (so it
// does not vacuum braces out of surrounding prose), then reads each {…} object's
// index and score independently of their order. ok is false when no pair is
// recovered.
func salvageRerank(text string, n int) ([]float64, bool) {
	span := text
	if si := rerankScoresRe.FindStringIndex(text); si != nil {
		if lb := strings.Index(text[si[0]:], "["); lb != -1 {
			start := si[0] + lb
			if rb := strings.Index(text[start:], "]"); rb != -1 {
				span = text[start : start+rb+1]
			} else {
				span = text[start:]
			}
		}
	}
	scores := make([]float64, n)
	any := false
	for _, obj := range rerankObjRe.FindAllString(span, -1) {
		im := rerankIdxRe.FindStringSubmatch(obj)
		sm := rerankScoreRe.FindStringSubmatch(obj)
		if im == nil || sm == nil {
			continue
		}
		idx, errI := strconv.Atoi(im[1])
		sc, errS := strconv.ParseFloat(sm[1], 64)
		if errI != nil || errS != nil {
			continue
		}
		if idx >= 0 && idx < n {
			scores[idx] = normalizeRerankScore(sc)
			any = true
		}
	}
	return scores, any
}
