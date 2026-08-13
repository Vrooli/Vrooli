package aisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// This file implements the two shippable Reranker impls (user-approved
// 2026-06-03) behind the Reranker interface declared in contracts.go, plus the
// degradation chain that composes them. They are deliberately in the shared
// package — the search-hub plan reuses the exact same impls and the chain so
// every federated leaf reranks consistently.
//
//   - LLMReranker:          an Ollama policy role via
//     `resource-ollama gateway generate` (dependency-free fallback for hosts
//     without the cross-encoder GPU resource).
//   - CrossEncoderReranker: BAAI/bge-reranker-v2-m3 served by the dedicated
//     `reranker` TEI resource (Phase 4) via RERANKER_URL /rerank.
//   - RerankerChain:        cross-encoder -> LLM -> fused (RRF) order.

// DefaultRerankRole is the substrate-standard LLM-as-reranker role.
const DefaultRerankRole = "rerank.llm_fallback"

// rerankCandidateCharCap bounds each candidate's text in the LLM prompt so a
// long doc chunk can't blow the context window (the cross-encoder resource has
// its own server-side truncation).
const rerankCandidateCharCap = 700

// maxCrossEncoderBatch bounds how many candidates go in one TEI /rerank request.
// The TEI server enforces --max-client-batch-size (default 32) and answers a
// larger batch with HTTP 413, so the engine chunks the shortlist into requests
// of at most this many candidates and merges the scores. This decouples the
// RERANK_SHORTLIST lever from the server limit: any shortlist reranks correctly
// (the /rerank scores are per query-candidate pair, so they are comparable and
// mergeable across chunks).
const maxCrossEncoderBatch = 32

// =============================================================================
// LLM-as-reranker (DefaultRerankRole via resource-ollama gateway generate)
// =============================================================================

// GenerateRunner runs the text-generation subprocess. Injectable so tests
// substitute a fake without shelling out. Shares the default implementation
// (runLocalCLI) with the embedder's EmbedRunner — identical signature, identical
// gateway-subprocess plumbing.
type GenerateRunner func(ctx context.Context, args []string, stdin []byte) ([]byte, error)

// LLMReranker scores a shortlist with a local instruction model via
// `resource-ollama gateway generate`. It is the dependency-free fallback: it
// needs only Ollama (already required for embedding), no extra resource.
type LLMReranker struct {
	bin  string
	role string
	run  GenerateRunner
}

// NewLLMReranker returns the production LLM reranker (shells out to
// resource-ollama). An empty role defaults to DefaultRerankRole.
func NewLLMReranker(role string) *LLMReranker {
	if strings.TrimSpace(role) == "" {
		role = DefaultRerankRole
	}
	return &LLMReranker{bin: defaultEmbedderBin, role: role, run: runLocalCLI}
}

// NewLLMRerankerWithRunner injects a runner (tests).
func NewLLMRerankerWithRunner(role string, run GenerateRunner) *LLMReranker {
	if strings.TrimSpace(role) == "" {
		role = DefaultRerankRole
	}
	return &LLMReranker{bin: defaultEmbedderBin, role: role, run: run}
}

// Name identifies the active reranker for StatusReport / SearchResponse.
func (r *LLMReranker) Name() string { return "llm:" + r.role }

// Available probes the model with a one-token generation. Cheap relative to a
// full rerank and only consulted when the cross-encoder is down (the chain
// tries the cross-encoder first). The timeout matches the cross-encoder probe
// (3 s) so an unreachable LLM leg can never reintroduce the per-query latency
// cliff — and RerankerChain TTL-caches the probe, so it runs at most once per
// window regardless.
func (r *LLMReranker) Available(ctx context.Context) bool {
	if r.run == nil {
		return false
	}
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	args := []string{r.bin, "gateway", "generate", "--role", r.role, "--max-tokens", "1", "--json", "--prompt-stdin"}
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
	args := []string{r.bin, "gateway", "generate", "--role", r.role, "--max-tokens", "512", "--temperature", "0", "--json", "--prompt-stdin"}
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

// CrossEncoderReranker calls the dedicated `reranker` TEI resource over HTTP.
// The base URL and model flow in from Config (prefix-aware), so two scenarios on
// one host can target different rerankers; when a value is empty the constructor
// falls back to the resource's own unprefixed env, preserving zero-config use.
type CrossEncoderReranker struct {
	baseURL string
	model   string
	http    httpDoer
}

// NewCrossEncoderReranker builds the cross-encoder leg from the Config-resolved
// base URL and model. An empty baseURL falls back to ResolveRerankerURL over the
// process env (RERANKER_BASE_URL/RERANKER_URL/RERANKER_HOST+PORT, then the
// resource default); an empty model falls back to RERANKER_MODEL, then the
// bge-reranker-v2-m3 default. Pass cfg.RerankerURL / cfg.RerankerModel.
func NewCrossEncoderReranker(baseURL, model string) *CrossEncoderReranker {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = ResolveRerankerURL(os.Getenv)
	}
	if strings.TrimSpace(model) == "" {
		model = envOr("RERANKER_MODEL", "bge-reranker-v2-m3")
	}
	return &CrossEncoderReranker{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		model:   strings.TrimSpace(model),
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

// Rerank scores the shortlist with the cross-encoder, chunking into TEI
// /rerank requests of at most maxCrossEncoderBatch candidates (the server's
// --max-client-batch-size; a larger batch is answered with HTTP 413) and
// merging the per-candidate scores. Any chunk failing aborts the whole rerank
// so the caller cleanly degrades to fused/dense order rather than reordering on
// a partial set.
func (r *CrossEncoderReranker) Rerank(ctx context.Context, query string, candidates []RerankCandidate) ([]RerankScore, error) {
	if r.http == nil || r.baseURL == "" {
		return nil, fmt.Errorf("cross-encoder reranker: base URL not resolved")
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	out := make([]RerankScore, 0, len(candidates))
	for start := 0; start < len(candidates); start += maxCrossEncoderBatch {
		end := start + maxCrossEncoderBatch
		if end > len(candidates) {
			end = len(candidates)
		}
		scores, err := r.rerankBatch(ctx, query, candidates[start:end])
		if err != nil {
			return nil, err
		}
		out = append(out, scores...)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("cross-encoder reranker: empty response")
	}
	return out, nil
}

// rerankBatch posts one ≤maxCrossEncoderBatch chunk to TEI /rerank and maps the
// returned indices (relative to the chunk) back to candidate IDs.
func (r *CrossEncoderReranker) rerankBatch(ctx context.Context, query string, chunk []RerankCandidate) ([]RerankScore, error) {
	texts := make([]string, len(chunk))
	for i, c := range chunk {
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
		if res.Index < 0 || res.Index >= len(chunk) {
			continue
		}
		out = append(out, RerankScore{ID: chunk[res.Index].ID, Score: res.Score})
	}
	return out, nil
}

// =============================================================================
// Degradation chain
// =============================================================================

// DefaultRerankerProbeTTL bounds how long RerankerChain caches its availability
// probe. Without it, every Search re-ran the per-leg Available() probe (a
// cross-encoder /health GET, or worse an LLM generate) — so a single down leg
// imposed a probe on every query. Caching the probe for a short window caps that
// to one probe per TTL and reflects an outage or recovery within one window.
const DefaultRerankerProbeTTL = 20 * time.Second

// RerankerChain composes rerankers in preference order and routes a Rerank call
// to the first Available one (cross-encoder -> LLM by convention). When none is
// available it returns (nil, nil) so the caller keeps the upstream fused order.
// It satisfies the Reranker interface itself, so a consumer can hold a single
// Reranker and the search-hub plan can reuse the same composition.
//
// Active() is TTL-cached behind an injected clock (the same func()time.Time
// seam Reconciler/SyncLoop use) so the availability probe is off the per-query
// hot path.
type RerankerChain struct {
	rerankers []Reranker
	clock     func() time.Time
	ttl       time.Duration

	mu       sync.Mutex
	cached   Reranker
	cachedAt time.Time
	primed   bool
}

// NewRerankerChain builds a chain in preference order with the default probe
// TTL and wall clock. Nil entries are dropped.
func NewRerankerChain(rerankers ...Reranker) *RerankerChain {
	return newRerankerChain(time.Now, DefaultRerankerProbeTTL, rerankers...)
}

// NewRerankerChainWithClock injects the clock + probe TTL (tests prove the
// cache hits/expires without real time). Extends the existing func()time.Time
// clock seam rather than introducing a parallel one.
func NewRerankerChainWithClock(clock func() time.Time, ttl time.Duration, rerankers ...Reranker) *RerankerChain {
	return newRerankerChain(clock, ttl, rerankers...)
}

func newRerankerChain(clock func() time.Time, ttl time.Duration, rerankers ...Reranker) *RerankerChain {
	filtered := make([]Reranker, 0, len(rerankers))
	for _, r := range rerankers {
		if r != nil {
			filtered = append(filtered, r)
		}
	}
	if clock == nil {
		clock = time.Now
	}
	if ttl <= 0 {
		ttl = DefaultRerankerProbeTTL
	}
	return &RerankerChain{rerankers: filtered, clock: clock, ttl: ttl}
}

// Active returns the first available reranker (cross-encoder -> LLM), or nil
// when none is reachable. The result is cached for ttl so the availability probe
// runs at most once per window, never once per query. A live readout (e.g. a
// status surface) should use ActiveUncached.
func (c *RerankerChain) Active(ctx context.Context) Reranker {
	return c.ActiveWithPreference(ctx, RerankPreferenceCrossEncoderPreferred)
}

// ActiveWithPreference selects the first available leg subject to the
// provider's reranker policy. A required cross-encoder never silently spends
// the LLM fallback budget; a preferred policy preserves the existing
// cross-encoder-first, LLM-second behavior.
func (c *RerankerChain) ActiveWithPreference(ctx context.Context, preference string) Reranker {
	preference = strings.TrimSpace(preference)
	if preference == RerankPreferenceCrossEncoderRequired {
		// A required policy must probe the cross-encoder on every decision. A
		// preferred-cache entry may legitimately be an LLM fallback and must
		// never turn a recovered required leg into a stale refusal.
		return c.refreshWithPreference(ctx, preference)
	}
	c.mu.Lock()
	if c.primed && c.clock().Sub(c.cachedAt) < c.ttl {
		active := c.cached
		c.mu.Unlock()
		if preference == RerankPreferenceCrossEncoderRequired && !isCrossEncoder(active) {
			return nil
		}
		return active
	}
	c.mu.Unlock()
	return c.refreshWithPreference(ctx, preference)
}

// ActiveUncached probes the legs live (bypassing the TTL cache) and refreshes
// it. The cache exists for the search hot path; a rarely-called status probe
// wants the current truth, not a cached one (plan §13).
func (c *RerankerChain) ActiveUncached(ctx context.Context) Reranker {
	return c.refreshWithPreference(ctx, RerankPreferenceCrossEncoderPreferred)
}

func (c *RerankerChain) refreshWithPreference(ctx context.Context, preference string) Reranker {
	var active Reranker
	for _, r := range c.rerankers {
		if preference == RerankPreferenceCrossEncoderRequired && !isCrossEncoder(r) {
			continue
		}
		if r.Available(ctx) {
			active = r
			break
		}
	}
	// The required policy is stricter than the shared preferred cache. Do not
	// let a required probe that found no cross-encoder poison a subsequent
	// preferred request for the duration of the TTL.
	if preference != RerankPreferenceCrossEncoderRequired {
		c.mu.Lock()
		c.cached = active
		c.cachedAt = c.clock()
		c.primed = true
		c.mu.Unlock()
	}
	return active
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
	return c.ActiveNameWithPreference(ctx, RerankPreferenceCrossEncoderPreferred)
}

// ActiveNameWithPreference is the policy-aware leg name for status and query
// observability. It returns "none" when the required leg is unavailable.
func (c *RerankerChain) ActiveNameWithPreference(ctx context.Context, preference string) string {
	if r := c.ActiveWithPreference(ctx, preference); r != nil {
		return r.Name()
	}
	return "none"
}

func isCrossEncoder(r Reranker) bool {
	return r != nil && strings.HasPrefix(strings.TrimSpace(r.Name()), "cross")
}

// Available reports whether any leg is reachable.
func (c *RerankerChain) Available(ctx context.Context) bool {
	return c.Active(ctx) != nil
}

// Rerank delegates to the first available leg. (nil, nil) means "no reranker
// available — keep fused order."
func (c *RerankerChain) Rerank(ctx context.Context, query string, candidates []RerankCandidate) ([]RerankScore, error) {
	return c.RerankWithPreference(ctx, query, candidates, RerankPreferenceCrossEncoderPreferred)
}

// RerankWithPreference executes the selected leg under the declared policy.
func (c *RerankerChain) RerankWithPreference(ctx context.Context, query string, candidates []RerankCandidate, preference string) ([]RerankScore, error) {
	active := c.ActiveWithPreference(ctx, preference)
	if active == nil {
		return nil, nil
	}
	return active.Rerank(ctx, query, candidates)
}

// ApplyRerank reorders results by a reranker's scores, preserving the upstream
// (fused) order as the stable tie-break and for any result the reranker omitted.
// Results the reranker scored sort to the front by descending score; unscored
// results keep their relative order behind them. It operates on the unified
// SearchResult — the same type VectorStore.Query returns and ApplyRelevanceFloor
// consumes — so no adopter re-implements reordering against a second type.
// ApplyRerankRRF fuses the upstream (retrieval) order with the reranker's order
// via Reciprocal Rank Fusion instead of letting the reranker's order win
// outright. Each result's fused score is 1/(k+retrievalRank) + 1/(k+rerankRank).
//
// This keeps the reranker's junk-rejection power — a candidate ranked low by
// BOTH retrieval and the reranker stays low, and a cross-encoder that collapses
// gibberish to ~0 still pushes it to the bottom — while preventing the reranker
// from BURYING a strongly-retrieved canonical result beneath literal-token
// lookalikes it happens to score higher. That burial is the measured failure
// mode on the cli-health command corpus, where the cross-encoder costs ~0.20
// recall@5 by pure-reordering; blending recovers most of it without giving up
// junk rejection. The fused score replaces Score and is a rank signal, not a
// 0..1 relevance probability, so classify it with the fusion regime (relative
// gap only, no absolute hard floor) — the Service does this automatically when
// RerankBlend is set. Results the reranker omitted keep only their retrieval-leg
// contribution; ties fall back to retrieval order.
func ApplyRerankRRF(hits []SearchResult, scores []RerankScore, k int) []SearchResult {
	if len(hits) == 0 {
		return hits
	}
	if k <= 0 {
		k = DefaultRRFK
	}
	rerankRankByID := make(map[string]int, len(scores))
	sorted := make([]RerankScore, len(scores))
	copy(sorted, scores)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Score > sorted[j].Score })
	for rank, s := range sorted {
		rerankRankByID[s.ID] = rank
	}
	type fused struct {
		hit   SearchResult
		score float64
		order int
	}
	items := make([]fused, len(hits))
	for i, h := range hits {
		rrf := 1.0 / float64(k+i) // retrieval leg always contributes
		if rr, ok := rerankRankByID[h.ID]; ok {
			rrf += 1.0 / float64(k+rr)
		}
		items[i] = fused{hit: h, score: rrf, order: i}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].order < items[j].order
	})
	out := make([]SearchResult, len(items))
	for i, it := range items {
		h := it.hit
		h.Score = it.score
		out[i] = h
	}
	return out
}

func ApplyRerank(hits []SearchResult, scores []RerankScore) []SearchResult {
	if len(scores) == 0 || len(hits) == 0 {
		return hits
	}
	scoreByID := make(map[string]float64, len(scores))
	for _, s := range scores {
		scoreByID[s.ID] = s.Score
	}
	type ranked struct {
		hit    SearchResult
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
	out := make([]SearchResult, len(items))
	for i, it := range items {
		h := it.hit
		if it.scored {
			h.Score = it.score
		}
		out[i] = h
	}
	return out
}
