package aisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// This file implements the two shippable Reranker impls (user-approved
// 2026-06-03) behind the Reranker interface declared in contracts.go, plus the
// degradation chain that composes them. They are deliberately in the shared
// package — the search-hub plan reuses the exact same impls and the chain so
// every federated leaf reranks consistently.
//
//   - LLMReranker:          qwen3:4b via `resource-ollama gateway generate`
//     (dependency-free, always-available fallback).
//   - CrossEncoderReranker: BAAI/bge-reranker-v2-m3 served by the dedicated
//     `reranker` TEI resource (Phase 4) via RERANKER_URL /rerank.
//   - RerankerChain:        cross-encoder -> LLM -> fused (RRF) order.

// DefaultRerankModel is the substrate-standard LLM-as-reranker model. qwen3:4b
// is small, already pulled, and reasons well over a short candidate list.
const DefaultRerankModel = "qwen3:4b"

// rerankCandidateCharCap bounds each candidate's text in the LLM prompt so a
// long doc chunk can't blow the context window (the cross-encoder resource has
// its own server-side truncation).
const rerankCandidateCharCap = 700

// =============================================================================
// LLM-as-reranker (qwen3:4b via resource-ollama gateway generate)
// =============================================================================

// GenerateRunner runs the text-generation subprocess. Injectable so tests
// substitute a fake without shelling out (mirrors embedder.EmbedRunner).
type GenerateRunner func(ctx context.Context, args []string, stdin []byte) ([]byte, error)

// LLMReranker scores a shortlist with a local instruction model via
// `resource-ollama gateway generate`. It is the dependency-free fallback: it
// needs only Ollama (already required for embedding), no extra resource.
type LLMReranker struct {
	bin   string
	model string
	run   GenerateRunner
}

// NewLLMReranker returns the production LLM reranker (shells out to
// resource-ollama). An empty model defaults to qwen3:4b.
func NewLLMReranker(model string) *LLMReranker {
	if strings.TrimSpace(model) == "" {
		model = DefaultRerankModel
	}
	return &LLMReranker{bin: defaultEmbedderBin, model: model, run: defaultGenerateRunner}
}

// NewLLMRerankerWithRunner injects a runner (tests).
func NewLLMRerankerWithRunner(model string, run GenerateRunner) *LLMReranker {
	if strings.TrimSpace(model) == "" {
		model = DefaultRerankModel
	}
	return &LLMReranker{bin: defaultEmbedderBin, model: model, run: run}
}

func defaultGenerateRunner(ctx context.Context, args []string, stdin []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return nil, err
		}
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return stdout.Bytes(), nil
}

// Name identifies the active reranker for StatusReport / SearchResponse.
func (r *LLMReranker) Name() string { return "llm:" + r.model }

// Available probes the model with a one-token generation. Cheap relative to a
// full rerank and only consulted when the cross-encoder is down (the chain
// tries the cross-encoder first).
func (r *LLMReranker) Available(ctx context.Context) bool {
	if r.run == nil {
		return false
	}
	probeCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	args := []string{r.bin, "gateway", "generate", "--model", r.model, "--max-tokens", "1", "--json", "--prompt-stdin"}
	_, err := r.run(probeCtx, args, []byte("ok"))
	return err == nil
}

type generateResponse struct {
	Response string `json:"response"`
}

type llmScore struct {
	Index int     `json:"index"`
	Score float64 `json:"score"`
}

// Rerank scores every candidate 0..1 with one listwise generation call and
// returns scores keyed by candidate ID. Candidates the model omits keep an
// implicit 0 (handled by the caller's stable-sort fallback to fused order).
func (r *LLMReranker) Rerank(ctx context.Context, query string, candidates []RerankCandidate) ([]RerankScore, error) {
	if r.run == nil {
		return nil, fmt.Errorf("llm reranker: runner not configured")
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	prompt := buildRerankPrompt(query, candidates)
	args := []string{r.bin, "gateway", "generate", "--model", r.model, "--max-tokens", "512", "--temperature", "0", "--json", "--prompt-stdin"}
	out, err := r.run(ctx, args, []byte(prompt))
	if err != nil {
		return nil, fmt.Errorf("resource-ollama gateway generate: %w", err)
	}
	var decoded generateResponse
	if err := json.Unmarshal(bytes.TrimSpace(out), &decoded); err != nil {
		return nil, fmt.Errorf("decode generate response: %w", err)
	}
	scores, err := parseLLMScores(decoded.Response)
	if err != nil {
		return nil, err
	}
	out2 := make([]RerankScore, 0, len(scores))
	for _, s := range scores {
		if s.Index < 0 || s.Index >= len(candidates) {
			continue
		}
		out2 = append(out2, RerankScore{ID: candidates[s.Index].ID, Score: s.Score})
	}
	if len(out2) == 0 {
		return nil, fmt.Errorf("llm reranker: no usable scores in response")
	}
	return out2, nil
}

func buildRerankPrompt(query string, candidates []RerankCandidate) string {
	var b strings.Builder
	// /no_think tells qwen3-class models to skip the <think> reasoning block so
	// the response is the bare JSON array (parseLLMScores also strips any block
	// defensively for models that ignore the directive).
	b.WriteString("/no_think\n")
	b.WriteString("You are a documentation-search relevance judge. Score how well each candidate passage answers the user query, from 0.0 (irrelevant) to 1.0 (directly answers it).\n\n")
	b.WriteString("Query: ")
	b.WriteString(strings.TrimSpace(query))
	b.WriteString("\n\nCandidates:\n")
	for i, c := range candidates {
		text := strings.TrimSpace(c.Text)
		if len(text) > rerankCandidateCharCap {
			text = text[:rerankCandidateCharCap]
		}
		text = strings.ReplaceAll(text, "\n", " ")
		fmt.Fprintf(&b, "[%d] %s\n", i, text)
	}
	b.WriteString("\nRespond with ONLY a compact JSON array, one object per candidate, like ")
	b.WriteString(`[{"index":0,"score":0.9},{"index":1,"score":0.2}]`)
	b.WriteString(". No prose, no markdown fences.")
	return b.String()
}

// parseLLMScores extracts the JSON score array from a model response,
// tolerating reasoning-model preamble (e.g. qwen3 <think> blocks, which may
// themselves contain stray brackets), markdown fences, and surrounding prose by
// stripping think blocks and scanning every balanced `[...]` span for the first
// that decodes into score objects.
func parseLLMScores(resp string) ([]llmScore, error) {
	resp = stripThinkBlocks(resp)
	for _, span := range balancedArraySpans(resp) {
		var scores []llmScore
		if err := json.Unmarshal([]byte(span), &scores); err != nil {
			continue
		}
		if len(scores) > 0 {
			return scores, nil
		}
	}
	return nil, fmt.Errorf("llm reranker: no JSON score array in response")
}

// stripThinkBlocks removes <think>...</think> sections (qwen3 reasoning),
// including an unterminated trailing block.
func stripThinkBlocks(s string) string {
	for {
		open := strings.Index(s, "<think>")
		if open < 0 {
			break
		}
		rel := strings.Index(s[open:], "</think>")
		if rel < 0 {
			s = s[:open] // unterminated; drop the rest
			break
		}
		s = s[:open] + s[open+rel+len("</think>"):]
	}
	return s
}

// balancedArraySpans returns every top-level `[...]` substring (bracket-matched,
// string-literal aware) in source order, so the caller can try each as JSON.
func balancedArraySpans(s string) []string {
	var spans []string
	depth, start := 0, -1
	inStr, esc := false, false
	for i, r := range s {
		switch {
		case esc:
			esc = false
		case r == '\\' && inStr:
			esc = true
		case r == '"':
			inStr = !inStr
		case inStr:
			// ignore brackets inside string literals
		case r == '[':
			if depth == 0 {
				start = i
			}
			depth++
		case r == ']':
			if depth > 0 {
				depth--
				if depth == 0 && start >= 0 {
					spans = append(spans, s[start:i+1])
					start = -1
				}
			}
		}
	}
	return spans
}

// =============================================================================
// Cross-encoder reranker (bge-reranker-v2-m3 via the TEI `reranker` resource)
// =============================================================================

// CrossEncoderReranker calls the dedicated `reranker` TEI resource (Phase 4)
// over HTTP. The base URL is resolved from the resource's exported environment
// (RERANKER_URL), never computed by the caller.
type CrossEncoderReranker struct {
	baseURL string
	model   string
	http    httpDoer
}

// NewCrossEncoderReranker resolves the base URL from the environment exported
// by the reranker resource. When unset it falls back to the resource's default
// host:port so a locally-started resource works without explicit wiring.
func NewCrossEncoderReranker() *CrossEncoderReranker {
	return &CrossEncoderReranker{
		baseURL: ResolveRerankerURL(os.Getenv),
		model:   envOr("RERANKER_MODEL", "bge-reranker-v2-m3"),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// NewCrossEncoderRerankerWithClient injects an httpDoer + base URL (tests).
func NewCrossEncoderRerankerWithClient(baseURL string, client httpDoer) *CrossEncoderReranker {
	return &CrossEncoderReranker{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   "bge-reranker-v2-m3",
		http:    client,
	}
}

// ResolveRerankerURL mirrors the reranker resource CLI's resolution order
// (RERANKER_BASE_URL, RERANKER_URL, RERANKER_HOST+RERANKER_PORT, default
// 127.0.0.1:11453). Exported + getenv-injected so it is unit-testable.
func ResolveRerankerURL(getenv func(string) string) string {
	for _, key := range []string{"RERANKER_BASE_URL", "RERANKER_URL"} {
		if raw := strings.TrimSpace(getenv(key)); raw != "" {
			return strings.TrimRight(raw, "/")
		}
	}
	host := strings.TrimSpace(getenv("RERANKER_HOST"))
	port := strings.TrimSpace(getenv("RERANKER_PORT"))
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "11453"
	}
	if strings.Contains(host, ":") {
		return "http://" + host
	}
	return "http://" + host + ":" + port
}

func envOr(key, def string) string {
	if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
		return raw
	}
	return def
}

// Name identifies the active reranker for StatusReport / SearchResponse.
func (r *CrossEncoderReranker) Name() string { return "cross-encoder:" + r.model }

// Available probes the resource's /health endpoint (fast, local).
func (r *CrossEncoderReranker) Available(ctx context.Context) bool {
	if r.http == nil || r.baseURL == "" {
		return false
	}
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, r.baseURL+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

type teiRerankRequest struct {
	Query      string   `json:"query"`
	Texts      []string `json:"texts"`
	RawScores  bool     `json:"raw_scores"`
	ReturnText bool     `json:"return_text"`
}

type teiRankResult struct {
	Index int     `json:"index"`
	Score float64 `json:"score"`
}

// Rerank posts the shortlist to TEI /rerank and maps the returned, score-sorted
// indices back to candidate IDs.
func (r *CrossEncoderReranker) Rerank(ctx context.Context, query string, candidates []RerankCandidate) ([]RerankScore, error) {
	if r.http == nil || r.baseURL == "" {
		return nil, fmt.Errorf("cross-encoder reranker: base URL not resolved")
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	texts := make([]string, len(candidates))
	for i, c := range candidates {
		texts[i] = c.Text
	}
	payload, err := json.Marshal(teiRerankRequest{Query: query, Texts: texts})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/rerank", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rerank: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rerank: HTTP %d", resp.StatusCode)
	}
	var results []teiRankResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode rerank response: %w", err)
	}
	out := make([]RerankScore, 0, len(results))
	for _, res := range results {
		if res.Index < 0 || res.Index >= len(candidates) {
			continue
		}
		out = append(out, RerankScore{ID: candidates[res.Index].ID, Score: res.Score})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("cross-encoder reranker: empty response")
	}
	return out, nil
}

// =============================================================================
// Degradation chain
// =============================================================================

// RerankerChain composes rerankers in preference order and routes a Rerank call
// to the first Available one (cross-encoder -> LLM by convention). When none is
// available it returns (nil, nil) so the caller keeps the upstream fused order.
// It satisfies the Reranker interface itself, so a consumer can hold a single
// Reranker and the search-hub plan can reuse the same composition.
type RerankerChain struct {
	rerankers []Reranker
}

// NewRerankerChain builds a chain in preference order. Nil entries are dropped.
func NewRerankerChain(rerankers ...Reranker) *RerankerChain {
	filtered := make([]Reranker, 0, len(rerankers))
	for _, r := range rerankers {
		if r != nil {
			filtered = append(filtered, r)
		}
	}
	return &RerankerChain{rerankers: filtered}
}

// Active returns the first available reranker, or nil when none is reachable.
func (c *RerankerChain) Active(ctx context.Context) Reranker {
	for _, r := range c.rerankers {
		if r.Available(ctx) {
			return r
		}
	}
	return nil
}

// Name reports the chain's composition (for diagnostics). Use ActiveName for
// the leg that actually answered a query.
func (c *RerankerChain) Name() string {
	if len(c.rerankers) == 0 {
		return "none"
	}
	names := make([]string, len(c.rerankers))
	for i, r := range c.rerankers {
		names[i] = r.Name()
	}
	return "chain[" + strings.Join(names, ">") + "]"
}

// ActiveName returns the active leg's Name(), or "none" when none is available.
func (c *RerankerChain) ActiveName(ctx context.Context) string {
	if r := c.Active(ctx); r != nil {
		return r.Name()
	}
	return "none"
}

// Available reports whether any leg is reachable.
func (c *RerankerChain) Available(ctx context.Context) bool {
	return c.Active(ctx) != nil
}

// Rerank delegates to the first available leg. (nil, nil) means "no reranker
// available — keep fused order."
func (c *RerankerChain) Rerank(ctx context.Context, query string, candidates []RerankCandidate) ([]RerankScore, error) {
	active := c.Active(ctx)
	if active == nil {
		return nil, nil
	}
	return active.Rerank(ctx, query, candidates)
}

// ApplyRerank reorders hits by a reranker's scores, preserving the upstream
// (fused) order as the stable tie-break and for any hit the reranker omitted.
// Hits the reranker scored sort to the front by descending score; unscored hits
// keep their relative order behind them. This is the shared helper both KO and
// search-hub use so reordering semantics never drift.
func ApplyRerank(hits []SearchHit, scores []RerankScore) []SearchHit {
	if len(scores) == 0 || len(hits) == 0 {
		return hits
	}
	scoreByID := make(map[string]float64, len(scores))
	for _, s := range scores {
		scoreByID[s.ID] = s.Score
	}
	type ranked struct {
		hit    SearchHit
		score  float64
		scored bool
		order  int
	}
	items := make([]ranked, len(hits))
	for i, h := range hits {
		s, ok := scoreByID[h.ID]
		items[i] = ranked{hit: h, score: s, scored: ok, order: i}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].scored != items[j].scored {
			return items[i].scored // scored hits first
		}
		if items[i].scored && items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].order < items[j].order
	})
	out := make([]SearchHit, len(items))
	for i, it := range items {
		h := it.hit
		if it.scored {
			h.Score = it.score
		}
		out[i] = h
	}
	return out
}
