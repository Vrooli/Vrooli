package aisearch

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// service.go is the shared read-path Service: the query -> floor -> rerank ->
// project -> text-fallback pipeline plus reindex job control that every adopter
// used to copy-paste (cli-health ~649 LOC, KO ~456 LOC). It is configurable
// through a handful of function seams so the ORDER of operations — the part that
// is easy to get subtly wrong — is owned here exactly once, while the
// corpus-specific shaping (which payload filter, which projection, which grep)
// stays adopter-owned. The non-negotiable order it bakes in (plan §8):
//
//	over-fetch shortlist -> rerank (active leg) -> ApplyRelevanceFloor(FloorForMethodLeg)
//	AFTER rerank -> LabelWeakForMethod -> project/decorate/trim
//
// An adopter supplies a Source (for indexing) plus the seams below; it never
// re-derives the pipeline.

// Projector fills a raw SearchResult's projection fields from its Payload,
// producing the adopter's hit shape IN PLACE (the input already carries ID,
// Score, Payload from the vector store). A doc adopter fills
// RelativePath/Snippet/Path/SourceID; a command adopter that keeps its
// corpus-specific fields in Payload can return the result unchanged. nil => the
// raw result is used as-is.
type Projector func(SearchResult) SearchResult

// QueryFilterFunc builds the per-query payload filter (scope / facets) applied to
// every vector leg. nil => no filter.
type QueryFilterFunc func(SearchQuery) *QueryFilter

// PostFilterFunc trims hits client-side after projection — e.g. an exact
// path-prefix scope Qdrant's keyword match cannot express. nil => identity.
type PostFilterFunc func([]SearchResult, SearchQuery) []SearchResult

// ScoreDecorator adjusts hit scores in place after the floor — e.g. an authority
// boost nudging canonical/maintained docs up at the margins. It runs late so it
// edges the final order regardless of which leg produced it. It receives the
// originating SearchQuery so a decorator can be query-aware (e.g. boost a
// canonical command only when the query does not name a different scenario);
// a decorator that only uses static payload facets (KO's maturity/canonicalFor
// boost) simply ignores it. nil => no-op.
type ScoreDecorator func([]SearchResult, SearchQuery)

// RerankTextFunc composes the passage handed to a reranker for one hit (the
// contextual text the reranker scores). nil => the payload "body" string.
type RerankTextFunc func(SearchResult) string

// TextFallbackFunc is the offline-safe keyword leg (grep / tokenized scan) the
// auto chain degrades to when the vector backend is down. nil => no text leg
// (auto then returns an empty vector response rather than grep).
type TextFallbackFunc func(context.Context, SearchQuery) ([]SearchResult, error)

// ServiceOptions configures the shared Service. Embedder + VectorStore are
// required for the vector legs; SparseEncoder (non-nil) enables the hybrid
// (dense+sparse RRF) leg; Reconciler (optional) feeds Status + reindex jobs.
//
// Control-surface map: the fields here are three kinds — (1) WIRING/SEAMS
// (Embedder, SparseEncoder, VectorStore, Reranker, Reconciler, and the *Func
// seams: Project/Filter/PostFilter/Decorate/RerankText/TextFallback); (2) the
// resolved GENUINE FACTORS the read path honors per construction (RerankEnabled,
// RerankBlend, Shortlist, ApplyFloor, Floor) — an adopter on the SSOT fills these
// from TuningConfig (see NewServiceForTuning / tuning.go), not by hand; (3)
// numeric BOUNDS (DefaultLimit/MaxLimit/PrefetchLimit/RRFK/Threshold) that shape
// the page. The factor taxonomy itself lives in tuning.go; this struct is the
// wiring it resolves into.
type ServiceOptions struct {
	Embedder      Embedder
	SparseEncoder SparseEncoder
	VectorStore   VectorStore
	Reranker      *RerankerChain
	Reconciler    *Reconciler

	// RerankEnabled gates the rerank pass (the one genuine per-corpus lever).
	// Shortlist is the over-fetch depth handed to the reranker (<=0 =>
	// DefaultRerankShortlist).
	RerankEnabled bool
	Shortlist     int
	// RerankBlend fuses the reranker order with the retrieval order via RRF
	// (ApplyRerankRRF) instead of letting the reranker order win outright. It
	// preserves the reranker's junk rejection while preventing it from burying a
	// strongly-retrieved canonical result beneath literal-token lookalikes — the
	// measured cli-health failure mode. When set, the fused (rank-signal) score is
	// classified with the fusion regime for the floor/weak label. RerankRRFK tunes
	// the fusion constant (<=0 => DefaultRRFK).
	RerankBlend bool
	RerankRRFK  int
	// HybridFusion selects the Qdrant fusion strategy for dense+sparse queries.
	// Empty means HybridFusionRRF. It is a query-time factor and is ignored by
	// dense-only services.
	HybridFusion string

	// ApplyFloor gates ApplyRelevanceFloor. It exists because the regime floor
	// bands (cross-encoder / llm / cosine) assume a 0..1 score; an RRF-fused
	// hybrid leg with rerank OFF produces ~0.01 scores the cosine HardFloor would
	// wrongly annihilate. cli-health (dense + rerank) sets it true; a doc adopter
	// running rerank-off hybrid sets it false. Floor is the operator override
	// merged onto the regime default by FloorForMethodLeg.
	ApplyFloor bool
	Floor      FloorConfig

	// Threshold is the dense-leg ScoreThreshold (ignored for fused queries).
	Threshold float64

	// DefaultLimit / MaxLimit bound the returned result count (<=0 => 10 / 100).
	// PrefetchLimit is the per-leg shortlist before RRF fusion (<=0 => Shortlist).
	DefaultLimit  int
	MaxLimit      int
	PrefetchLimit int

	// Seams (see the *Func types above).
	Project    Projector
	Filter     QueryFilterFunc
	PostFilter PostFilterFunc
	// PreFloorDecorate adjusts scores after reranking but before the relevance
	// floor. It is for a bounded second-leg signal (for example, an exact lexical
	// match that should rescue a relevant candidate from a cross-encoder floor).
	// Decorate remains the late, post-floor authority seam.
	PreFloorDecorate ScoreDecorator
	Decorate         ScoreDecorator
	RerankText       RerankTextFunc
	TextFallback     TextFallbackFunc

	// OverridePolicy gates the per-request query-time override channel (see
	// override.go). nil => DenyOverrides: the secure default in which the Service
	// honors no per-request override regardless of what a Search call passes.
	OverridePolicy OverridePolicy
}

// Service is the concrete shared read path. Construct it with NewService.
type Service struct {
	embedder   Embedder
	sparse     SparseEncoder
	store      VectorStore
	reranker   *RerankerChain
	reconciler *Reconciler

	rerankEnabled bool
	rerankBlend   bool
	rrfK          int
	hybridFusion  string
	shortlist     int
	applyFloor    bool
	floor         FloorConfig
	threshold     float64
	defaultLimit  int
	maxLimit      int
	prefetchLimit int

	project          Projector
	filter           QueryFilterFunc
	postFilter       PostFilterFunc
	preFloorDecorate ScoreDecorator
	decorate         ScoreDecorator
	rerankText       RerankTextFunc
	text             TextFallbackFunc

	overridePolicy OverridePolicy

	mu      sync.Mutex
	jobs    map[string]*ReindexJob
	lastJob string
}

// NewService wires the shared read path from the given options, filling the
// numeric defaults so the pipeline is always meaningful.
func NewService(opts ServiceOptions) *Service {
	shortlist := opts.Shortlist
	if shortlist <= 0 {
		shortlist = DefaultRerankShortlist
	}
	defaultLimit := opts.DefaultLimit
	if defaultLimit <= 0 {
		defaultLimit = 10
	}
	maxLimit := opts.MaxLimit
	if maxLimit <= 0 {
		maxLimit = 100
	}
	prefetch := opts.PrefetchLimit
	if prefetch <= 0 {
		prefetch = shortlist
	}
	rrfK := opts.RerankRRFK
	if rrfK <= 0 {
		rrfK = DefaultRRFK
	}
	hybridFusion := normalizeHybridFusion(opts.HybridFusion)
	if hybridFusion == "" {
		hybridFusion = HybridFusionRRF
	}
	threshold := opts.Threshold
	if threshold < 0 {
		threshold = 0
	}
	return &Service{
		embedder:         opts.Embedder,
		sparse:           opts.SparseEncoder,
		store:            opts.VectorStore,
		reranker:         opts.Reranker,
		reconciler:       opts.Reconciler,
		rerankEnabled:    opts.RerankEnabled,
		rerankBlend:      opts.RerankBlend,
		rrfK:             rrfK,
		hybridFusion:     hybridFusion,
		shortlist:        shortlist,
		applyFloor:       opts.ApplyFloor,
		floor:            opts.Floor,
		threshold:        threshold,
		defaultLimit:     defaultLimit,
		maxLimit:         maxLimit,
		prefetchLimit:    prefetch,
		project:          opts.Project,
		filter:           opts.Filter,
		postFilter:       opts.PostFilter,
		preFloorDecorate: opts.PreFloorDecorate,
		decorate:         opts.Decorate,
		rerankText:       opts.RerankText,
		text:             opts.TextFallback,
		overridePolicy:   opts.OverridePolicy,
		jobs:             make(map[string]*ReindexJob),
	}
}

// Reconciler exposes the underlying reconciler (for the sync loop / EnsureCollection).
func (s *Service) Reconciler() *Reconciler { return s.reconciler }

// Options returns a copy of the read-path configuration currently held by the
// service. Adopters that own a provider-controlled shadow index can use this
// as the starting point for an isolated engine while preserving the provider's
// filters, projection and reranker policy. The returned reconciler is the
// current one and callers that build a new source binding must replace it.
func (s *Service) Options() ServiceOptions {
	return ServiceOptions{
		Embedder:         s.embedder,
		SparseEncoder:    s.sparse,
		VectorStore:      s.store,
		Reranker:         s.reranker,
		Reconciler:       s.reconciler,
		RerankEnabled:    s.rerankEnabled,
		RerankBlend:      s.rerankBlend,
		RerankRRFK:       s.rrfK,
		HybridFusion:     s.hybridFusion,
		Shortlist:        s.shortlist,
		ApplyFloor:       s.applyFloor,
		Floor:            s.floor,
		Threshold:        s.threshold,
		DefaultLimit:     s.defaultLimit,
		MaxLimit:         s.maxLimit,
		PrefetchLimit:    s.prefetchLimit,
		Project:          s.project,
		Filter:           s.filter,
		PostFilter:       s.postFilter,
		PreFloorDecorate: s.preFloorDecorate,
		Decorate:         s.decorate,
		RerankText:       s.rerankText,
		TextFallback:     s.text,
		OverridePolicy:   s.overridePolicy,
	}
}

func (s *Service) normalize(q SearchQuery) SearchQuery {
	q.Query = strings.TrimSpace(q.Query)
	if q.Mode == "" {
		q.Mode = ModeAuto
	}
	if q.Scope.Kind == "" {
		q.Scope.Kind = ScopeGlobal
	}
	if q.Limit <= 0 {
		q.Limit = s.defaultLimit
	}
	if q.Limit > s.maxLimit {
		q.Limit = s.maxLimit
	}
	return q
}

// Search runs one query, dispatching on mode and (for ModeAuto) walking the
// hybrid -> dense -> text fallback chain so it never hard-fails. Optional
// SearchOptions (WithOverrides) vary the query-time factors for this one call,
// subject to the Service's OverridePolicy and clamping; without options the
// Service's constructed defaults apply unchanged.
func (s *Service) Search(ctx context.Context, q SearchQuery, opts ...SearchOption) (SearchResponse, error) {
	q = s.normalize(q)
	if q.Query == "" {
		return SearchResponse{}, fmt.Errorf("query is required")
	}
	eff := s.resolveEffective(opts...)
	switch q.Mode {
	case ModeText:
		return s.textSearch(ctx, q)
	case ModeDense:
		return s.vectorSearch(ctx, q, false, eff)
	case ModeHybrid:
		if s.sparse == nil {
			return SearchResponse{}, fmt.Errorf("hybrid mode requires a sparse encoder")
		}
		return s.vectorSearch(ctx, q, true, eff)
	case ModeAuto:
		return s.autoSearch(ctx, q, eff)
	default:
		return SearchResponse{}, fmt.Errorf("unknown search mode %q", q.Mode)
	}
}

// autoSearch degrades hybrid -> dense -> text on unavailability or empty result,
// so search always returns something useful.
func (s *Service) autoSearch(ctx context.Context, q SearchQuery, eff effectiveParams) (SearchResponse, error) {
	storeUp := s.store != nil && s.store.Available(ctx)
	embedUp := s.embedder != nil && s.embedder.Available(ctx)
	if storeUp && embedUp {
		if s.sparse != nil {
			if resp, err := s.vectorSearch(ctx, q, true, eff); err == nil && len(resp.Results) > 0 {
				return resp, nil
			}
		}
		if resp, err := s.vectorSearch(ctx, q, false, eff); err == nil && len(resp.Results) > 0 {
			return resp, nil
		}
	}
	if s.text != nil {
		return s.textSearch(ctx, q)
	}
	// No text leg: return an empty (not errored) vector response so callers see a
	// clean "no results" rather than a hard failure.
	return SearchResponse{Query: q.Query, Method: "dense", Reranker: "none", Regime: RegimeForMethod("dense", "none")}, nil
}

// vectorSearch runs the dense or hybrid leg, then projects, post-filters,
// reranks, floors, decorates, weak-labels, and trims — the one true ordering.
// eff carries the resolved query-time parameters (Service defaults overlaid by
// any permitted per-call overrides) the rerank/floor stages read.
func (s *Service) vectorSearch(ctx context.Context, q SearchQuery, hybrid bool, eff effectiveParams) (SearchResponse, error) {
	if s.store == nil || s.embedder == nil {
		return SearchResponse{}, fmt.Errorf("vector search requires an embedder and vector store")
	}
	dense, err := embedQueryText(ctx, s.embedder, q.Query)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("embed query: %w", err)
	}

	// Over-fetch a shortlist whenever a stage downstream of the query can drop or
	// reorder hits before the page is cut — reranking, a client-side PostFilter
	// (e.g. exact path scope), or a ScoreDecorator (authority boost). Without the
	// headroom those stages would operate on too few candidates and the page
	// could lose in-scope or better-ranked results.
	shortlist := q.Limit
	overfetch := (eff.rerankEnabled && s.reranker != nil) || s.postFilter != nil || s.preFloorDecorate != nil || s.decorate != nil
	if overfetch && eff.shortlist > shortlist {
		shortlist = eff.shortlist
	}

	hq := HybridQuery{
		Dense:         dense,
		Limit:         shortlist,
		PrefetchLimit: s.prefetchLimit,
	}
	if s.filter != nil {
		hq.Filter = s.filter(q)
	}
	method := "dense"
	if hybrid {
		sparse := s.sparse.Encode(q.Query)
		hq.Sparse = &sparse
		hq.Fusion = eff.hybridFusion
		method = "hybrid"
	} else {
		// Threshold only applies to a pure dense query; fused queries ignore it.
		hq.ScoreThreshold = s.threshold
	}

	raw, err := s.store.Query(ctx, hq)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("vector query: %w", err)
	}

	hits := s.projectAll(raw)
	if s.postFilter != nil {
		hits = s.postFilter(hits, q)
	}
	sortByScoreDesc(hits)

	leg := "none"
	reranked := false
	if eff.rerankEnabled && s.reranker != nil {
		if active := s.reranker.Active(ctx); active != nil {
			reorder := s.applyRerank
			if eff.rerankBlend {
				reorder = s.applyRerankRRF
			}
			if reordered, rerr := reorder(ctx, q.Query, hits, active); rerr == nil {
				hits = reordered
				leg = active.Name()
				reranked = true
			} else {
				log.Printf("[aisearch] rerank failed, keeping fused order: %v", rerr)
			}
		}
	}

	if s.preFloorDecorate != nil {
		s.preFloorDecorate(hits, q)
		sortByScoreDesc(hits)
	}

	// Choose the regime that classifies the post-rerank scores for the floor and
	// weak label. A pure rerank leaves cross-encoder/llm 0..1 scores (judged in
	// that leg's regime); an RRF BLEND replaces them with a rank-fusion signal, so
	// it is classified by the fusion regime (relative MaxGap only, no absolute
	// HardFloor) exactly like rerank-off hybrid. respLeg keeps the reranker name
	// for observability.
	floorMethod, floorLeg, respLeg := method, leg, leg
	if reranked && eff.rerankBlend {
		floorMethod, floorLeg = "hybrid", "" // force the fusion regime
		respLeg = "blend:" + leg
	}

	// Floor AFTER rerank (plan §8): junk the reranker drives to ~0 then drops out
	// of the page instead of riding its raw dense score. Regime-aware via
	// FloorForMethodLeg — which classifies a rerank-off hybrid leg (and a blended
	// leg) as the fusion band (relative MaxGap only, no absolute HardFloor) rather
	// than cosine, so ApplyFloor is safe to leave on for a fused adopter. Still
	// gated by ApplyFloor for an adopter that wants no floor at all.
	if eff.applyFloor {
		hits = ApplyRelevanceFloor(hits, FloorForMethodLeg(floorMethod, floorLeg, eff.floor))
	}

	if s.decorate != nil {
		s.decorate(hits, q)
		sortByScoreDesc(hits)
	}

	// The blend path labeled weakness from the reranker's raw scores inside
	// applyRerankRRF (a blended rank-fusion score cannot express relevance);
	// every other path is judged on the post-rerank score's own regime here.
	if !(reranked && eff.rerankBlend) {
		labelWeak(hits, floorMethod, floorLeg)
	}

	total := len(hits)
	if len(hits) > q.Limit {
		hits = hits[:q.Limit]
	}
	return SearchResponse{
		Results:  hits,
		Total:    total,
		Query:    q.Query,
		Method:   method,
		Reranker: respLeg,
		// Resolve the regime from the SAME (method, leg) the floor + weak label
		// used — not respLeg, whose "blend:" prefix is observability-only. This is
		// the single authoritative regime; adopters read it off the response.
		Regime: RegimeForMethod(floorMethod, floorLeg),
	}, nil
}

// textSearch is the keyword degradation leg. The fallback's scores are dense-
// cosine-band judged (text method => cosine regime in LabelWeakForMethod).
func (s *Service) textSearch(ctx context.Context, q SearchQuery) (SearchResponse, error) {
	if s.text == nil {
		return SearchResponse{}, fmt.Errorf("text fallback is not configured")
	}
	hits, err := s.text(ctx, q)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("text search: %w", err)
	}
	total := len(hits)
	if len(hits) > q.Limit {
		hits = hits[:q.Limit]
	}
	labelWeak(hits, "text", "")
	return SearchResponse{Results: hits, Total: total, Query: q.Query, Method: "text", Reranker: "none", Regime: RegimeForMethod("text", "none")}, nil
}

func (s *Service) projectAll(raw []SearchResult) []SearchResult {
	if s.project == nil {
		return raw
	}
	out := make([]SearchResult, len(raw))
	for i, r := range raw {
		out[i] = s.project(r)
	}
	return out
}

// applyRerank scores the shortlist with the active leg and reorders it via the
// shared ApplyRerank (preserving fused order for omitted candidates).
func (s *Service) applyRerank(ctx context.Context, query string, hits []SearchResult, active Reranker) ([]SearchResult, error) {
	cands := make([]RerankCandidate, len(hits))
	for i, h := range hits {
		cands[i] = RerankCandidate{ID: h.ID, Text: s.rerankCandidateText(h)}
	}
	scores, err := active.Rerank(ctx, query, cands)
	if err != nil {
		return nil, err
	}
	return ApplyRerank(hits, scores), nil
}

// applyRerankRRF scores the shortlist with the active leg and fuses its order
// with the retrieval order via ApplyRerankRRF (the blend path).
//
// It also owns the WEAK LABEL for blended hits: the blended score is a pure
// rank-fusion signal whose magnitude says nothing about relevance — on a tiny
// corpus every hit (junk included) blends to within epsilon of 2/(K+1), so any
// absolute threshold either flags everything weak (the bug this fixes: blend
// scores ≈0.033 judged against the 0.20 fusion band tuned for qdrant
// server-side RRF) or nothing. The reranker's RAW scores are calibrated for
// exactly this question, and they are in hand here — so weakness is judged
// from the raw score in the active leg's own regime, while the blended score
// keeps owning the ORDER. Hits the reranker did not score at all are weak: a
// shortlist member the leg declined to vouch for.
func (s *Service) applyRerankRRF(ctx context.Context, query string, hits []SearchResult, active Reranker) ([]SearchResult, error) {
	cands := make([]RerankCandidate, len(hits))
	for i, h := range hits {
		cands[i] = RerankCandidate{ID: h.ID, Text: s.rerankCandidateText(h)}
	}
	scores, err := active.Rerank(ctx, query, cands)
	if err != nil {
		return nil, err
	}
	blended := ApplyRerankRRF(hits, scores, s.rrfK)

	rawByID := make(map[string]float64, len(scores))
	for _, sc := range scores {
		rawByID[sc.ID] = sc.Score
	}
	leg := active.Name()
	for i := range blended {
		raw, scored := rawByID[blended[i].ID]
		blended[i].Weak = !scored || LabelWeakForMethod("dense", leg, raw)
	}
	return blended, nil
}

func (s *Service) rerankCandidateText(h SearchResult) string {
	if s.rerankText != nil {
		return s.rerankText(h)
	}
	if body, _ := h.Payload["body"].(string); strings.TrimSpace(body) != "" {
		return body
	}
	return h.Snippet
}

// labelWeak sets the regime-aware weak flag on every hit using the retrieval
// method + active leg (so a rerank-off hybrid leg is judged on the fusion band).
func labelWeak(hits []SearchResult, method, leg string) {
	for i := range hits {
		hits[i].Weak = LabelWeakForMethod(method, leg, hits[i].Score)
	}
}

func sortByScoreDesc(hits []SearchResult) {
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
}

// Status reports backend availability + the indexed point count + last reconcile
// outcome for `search status`.
func (s *Service) Status(ctx context.Context) StatusReport {
	rep := StatusReport{Reranker: "none"}
	if s.embedder != nil {
		rep.Ollama = s.embedder.Available(ctx)
	}
	if s.store != nil {
		rep.Qdrant = s.store.Available(ctx)
		if rep.Qdrant {
			if n, err := s.store.CountPoints(ctx); err == nil {
				rep.IndexedCount = n
			}
		}
	}
	// Search never hard-fails: by this default it is available whenever a vector
	// backend OR the grep fallback can answer (the doc-corpus "degrade to grep is
	// still available" semantics). An adopter for whom AI search IS the product
	// (text is a degradation, not "available") overrides this in its own Status
	// wrapper — see the StatusReport.Available doc comment and cli-health.
	rep.Available = rep.Qdrant || s.text != nil
	if s.rerankEnabled && s.reranker != nil {
		// Live readout: bypass the per-query TTL cache so status reflects the
		// reranker's true current reachability.
		if active := s.reranker.ActiveUncached(ctx); active != nil {
			rep.Reranker = active.Name()
		} else {
			rep.Reranker = "degraded"
		}
	}
	if s.reconciler != nil {
		st := s.reconciler.Status()
		rep.LastReconcileAt = st.FinishedAt
		// A periodic reconciler can start the next run immediately after a
		// completed apply. In that interval its top-level FinishedAt is
		// intentionally withheld while LastResult still carries the last
		// completed apply. Expose that completed timestamp so status remains
		// truthful instead of becoming spuriously unreported during a run.
		if rep.LastReconcileAt == "" && st.LastResult != nil && !st.LastResult.FinishedAt.IsZero() {
			rep.LastReconcileAt = st.LastResult.FinishedAt.Format(time.RFC3339)
		}
		switch {
		case st.Running:
			rep.LastReconcileOutcome = "running"
		case st.Canceled:
			rep.LastReconcileOutcome = "canceled"
		case st.LastError != "":
			rep.LastReconcileOutcome = "error: " + st.LastError
		case st.LastResult != nil:
			var upserts, deletes int
			for _, c := range st.LastResult.Collections {
				upserts += c.Upserted
				deletes += c.Deleted
			}
			rep.LastReconcileOutcome = fmt.Sprintf("upserts=%d deletes=%d errors=%d", upserts, deletes, len(st.LastResult.Errors))
		case st.FinishedAt != "":
			rep.LastReconcileOutcome = "ok"
		}
	}
	return rep
}

// =============================================================================
// Reindex job control (lifted from cli-health; generic over the reconciler)
// =============================================================================

// ReindexJob is the per-job state surfaced by ReindexStatus.
type ReindexJob struct {
	ID         string
	State      string // queued|running|succeeded|failed|cancelled
	Scenario   string
	DryRun     bool
	Plan       *DriftReport
	Apply      *ApplyResult
	Err        string
	StartedAt  time.Time
	FinishedAt time.Time
	cancel     context.CancelFunc
}

// Reindex queues a single reconcile job and runs it in the background. The
// scenario filter is recorded for forward compatibility; v1 reconciles the full
// corpus.
func (s *Service) Reindex(_ context.Context, scenario string, dryRun bool) (*ReindexJob, error) {
	if s.reconciler == nil {
		return nil, fmt.Errorf("reindex requires a reconciler")
	}
	jobID := newJobID()
	jobCtx, cancel := context.WithCancel(context.Background())
	job := &ReindexJob{
		ID:        jobID,
		State:     "queued",
		Scenario:  scenario,
		DryRun:    dryRun,
		StartedAt: time.Now(),
		cancel:    cancel,
	}
	initial := cloneReindexJob(job)
	s.mu.Lock()
	s.jobs[jobID] = job
	s.lastJob = jobID
	s.mu.Unlock()

	go s.runReindex(jobCtx, job)
	return initial, nil
}

func (s *Service) runReindex(ctx context.Context, job *ReindexJob) {
	s.mu.Lock()
	job.State = "running"
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		job.FinishedAt = time.Now()
		s.mu.Unlock()
	}()

	plan, err := s.reconciler.Plan(ctx)
	s.mu.Lock()
	job.Plan = plan
	s.mu.Unlock()
	if err != nil {
		s.mu.Lock()
		job.State = "failed"
		job.Err = err.Error()
		s.mu.Unlock()
		return
	}
	if job.DryRun {
		s.mu.Lock()
		job.State = "succeeded"
		s.mu.Unlock()
		return
	}

	apply, err := s.reconciler.Apply(ctx, plan)
	s.mu.Lock()
	defer s.mu.Unlock()
	job.Apply = apply
	switch {
	case err != nil:
		if ctx.Err() != nil {
			job.State = "cancelled"
		} else {
			job.State = "failed"
			job.Err = err.Error()
		}
	case len(apply.Errors) > 0:
		job.State = "failed"
		job.Err = fmt.Sprintf("reconcile completed with %d item error(s): %s", len(apply.Errors), apply.Errors[0].Err)
	default:
		job.State = "succeeded"
	}
}

// ReindexStatus returns a job's state (the last job when jobID is empty).
func (s *Service) ReindexStatus(jobID string) (*ReindexJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if jobID == "" {
		jobID = s.lastJob
	}
	job, ok := s.jobs[jobID]
	if !ok {
		return nil, false
	}
	return cloneReindexJob(job), true
}

// cloneReindexJob is the read-side snapshot boundary for asynchronous job
// state. ReindexStatus holds Service.mu while taking this copy, so callers can
// inspect the returned value without racing the background worker's updates.
func cloneReindexJob(job *ReindexJob) *ReindexJob {
	if job == nil {
		return nil
	}
	copy := *job
	copy.cancel = nil
	return &copy
}

// ReindexCancel asks an in-flight job to stop. Returns true when a running or
// queued job was cancelled.
func (s *Service) ReindexCancel(jobID string) bool {
	s.mu.Lock()
	job, ok := s.jobs[jobID]
	if !ok {
		s.mu.Unlock()
		return false
	}
	if job.State != "running" && job.State != "queued" {
		s.mu.Unlock()
		return false
	}
	s.mu.Unlock()
	job.cancel()
	if s.reconciler != nil {
		s.reconciler.Cancel()
	}
	return true
}

// JobExport projects a ReindexJob to the string-keyed wire shape both adopters'
// handlers/CLIs emit.
func (s *Service) JobExport(job *ReindexJob) map[string]any {
	if job == nil {
		return nil
	}
	plannedUpserts, plannedDeletes := 0, 0
	if job.Plan != nil {
		for _, c := range job.Plan.Collections {
			plannedUpserts += len(c.ToUpsert)
			plannedDeletes += len(c.ToDelete)
		}
	}
	processed := 0
	if job.Apply != nil {
		for _, c := range job.Apply.Collections {
			processed += c.Upserted + c.Deleted
		}
	}
	return map[string]any{
		"job_id":          job.ID,
		"state":           job.State,
		"scenario":        job.Scenario,
		"dry_run":         job.DryRun,
		"planned_upserts": plannedUpserts,
		"planned_deletes": plannedDeletes,
		"processed":       processed,
		"total":           plannedUpserts + plannedDeletes,
		"error":           job.Err,
		"started_at":      job.StartedAt.Format(time.RFC3339),
		"finished_at":     formatIfNotZero(job.FinishedAt),
	}
}

// newJobID returns a random 128-bit hex id (stdlib only — no external uuid dep).
func newJobID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failure is catastrophic; fall back to a time-based id so a
		// reindex can still be tracked rather than panicking the server.
		return "job-" + strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b[:])
}

func formatIfNotZero(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
