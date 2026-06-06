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
//	over-fetch shortlist -> rerank (active leg) -> ApplyRelevanceFloor(FloorForLeg)
//	AFTER rerank -> LabelWeak -> project/decorate/trim
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
// edges the final order regardless of which leg produced it. nil => no-op.
type ScoreDecorator func([]SearchResult)

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

	// ApplyFloor gates ApplyRelevanceFloor. It exists because the regime floor
	// bands (cross-encoder / llm / cosine) assume a 0..1 score; an RRF-fused
	// hybrid leg with rerank OFF produces ~0.01 scores the cosine HardFloor would
	// wrongly annihilate. cli-health (dense + rerank) sets it true; a doc adopter
	// running rerank-off hybrid sets it false. Floor is the operator override
	// merged onto the regime default by FloorForLeg.
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
	Project      Projector
	Filter       QueryFilterFunc
	PostFilter   PostFilterFunc
	Decorate     ScoreDecorator
	RerankText   RerankTextFunc
	TextFallback TextFallbackFunc
}

// Service is the concrete shared read path. Construct it with NewService.
type Service struct {
	embedder   Embedder
	sparse     SparseEncoder
	store      VectorStore
	reranker   *RerankerChain
	reconciler *Reconciler

	rerankEnabled bool
	shortlist     int
	applyFloor    bool
	floor         FloorConfig
	threshold     float64
	defaultLimit  int
	maxLimit      int
	prefetchLimit int

	project    Projector
	filter     QueryFilterFunc
	postFilter PostFilterFunc
	decorate   ScoreDecorator
	rerankText RerankTextFunc
	text       TextFallbackFunc

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
	threshold := opts.Threshold
	if threshold < 0 {
		threshold = 0
	}
	return &Service{
		embedder:      opts.Embedder,
		sparse:        opts.SparseEncoder,
		store:         opts.VectorStore,
		reranker:      opts.Reranker,
		reconciler:    opts.Reconciler,
		rerankEnabled: opts.RerankEnabled,
		shortlist:     shortlist,
		applyFloor:    opts.ApplyFloor,
		floor:         opts.Floor,
		threshold:     threshold,
		defaultLimit:  defaultLimit,
		maxLimit:      maxLimit,
		prefetchLimit: prefetch,
		project:       opts.Project,
		filter:        opts.Filter,
		postFilter:    opts.PostFilter,
		decorate:      opts.Decorate,
		rerankText:    opts.RerankText,
		text:          opts.TextFallback,
		jobs:          make(map[string]*ReindexJob),
	}
}

// Reconciler exposes the underlying reconciler (for the sync loop / EnsureCollection).
func (s *Service) Reconciler() *Reconciler { return s.reconciler }

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
// hybrid -> dense -> text fallback chain so it never hard-fails.
func (s *Service) Search(ctx context.Context, q SearchQuery) (SearchResponse, error) {
	q = s.normalize(q)
	if q.Query == "" {
		return SearchResponse{}, fmt.Errorf("query is required")
	}
	switch q.Mode {
	case ModeText:
		return s.textSearch(ctx, q)
	case ModeDense:
		return s.vectorSearch(ctx, q, false)
	case ModeHybrid:
		if s.sparse == nil {
			return SearchResponse{}, fmt.Errorf("hybrid mode requires a sparse encoder")
		}
		return s.vectorSearch(ctx, q, true)
	case ModeAuto:
		return s.autoSearch(ctx, q)
	default:
		return SearchResponse{}, fmt.Errorf("unknown search mode %q", q.Mode)
	}
}

// autoSearch degrades hybrid -> dense -> text on unavailability or empty result,
// so search always returns something useful.
func (s *Service) autoSearch(ctx context.Context, q SearchQuery) (SearchResponse, error) {
	storeUp := s.store != nil && s.store.Available(ctx)
	embedUp := s.embedder != nil && s.embedder.Available(ctx)
	if storeUp && embedUp {
		if s.sparse != nil {
			if resp, err := s.vectorSearch(ctx, q, true); err == nil && len(resp.Results) > 0 {
				return resp, nil
			}
		}
		if resp, err := s.vectorSearch(ctx, q, false); err == nil && len(resp.Results) > 0 {
			return resp, nil
		}
	}
	if s.text != nil {
		return s.textSearch(ctx, q)
	}
	// No text leg: return an empty (not errored) vector response so callers see a
	// clean "no results" rather than a hard failure.
	return SearchResponse{Query: q.Query, Method: "dense", Reranker: "none"}, nil
}

// vectorSearch runs the dense or hybrid leg, then projects, post-filters,
// reranks, floors, decorates, weak-labels, and trims — the one true ordering.
func (s *Service) vectorSearch(ctx context.Context, q SearchQuery, hybrid bool) (SearchResponse, error) {
	if s.store == nil || s.embedder == nil {
		return SearchResponse{}, fmt.Errorf("vector search requires an embedder and vector store")
	}
	dense, err := s.embedder.Embed(ctx, q.Query)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("embed query: %w", err)
	}

	// Over-fetch a shortlist whenever a stage downstream of the query can drop or
	// reorder hits before the page is cut — reranking, a client-side PostFilter
	// (e.g. exact path scope), or a ScoreDecorator (authority boost). Without the
	// headroom those stages would operate on too few candidates and the page
	// could lose in-scope or better-ranked results.
	shortlist := q.Limit
	overfetch := (s.rerankEnabled && s.reranker != nil) || s.postFilter != nil || s.decorate != nil
	if overfetch && s.shortlist > shortlist {
		shortlist = s.shortlist
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
		hq.Fusion = "rrf"
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
	if s.rerankEnabled && s.reranker != nil {
		if active := s.reranker.Active(ctx); active != nil {
			if reordered, rerr := s.applyRerank(ctx, q.Query, hits, active); rerr == nil {
				hits = reordered
				leg = active.Name()
			} else {
				log.Printf("[aisearch] rerank failed, keeping fused order: %v", rerr)
			}
		}
	}

	// Floor AFTER rerank (plan §8): junk the reranker drives to ~0 then drops out
	// of the page instead of riding its raw dense score. Regime-aware via
	// FloorForMethodLeg — which classifies a rerank-off hybrid leg as the fusion
	// band (relative MaxGap only, no absolute HardFloor) rather than cosine, so
	// ApplyFloor is safe to leave on for a fused adopter. Still gated by ApplyFloor
	// for an adopter that wants no floor at all.
	if s.applyFloor {
		hits = ApplyRelevanceFloor(hits, FloorForMethodLeg(method, leg, s.floor))
	}

	if s.decorate != nil {
		s.decorate(hits)
		sortByScoreDesc(hits)
	}

	labelWeak(hits, method, leg)

	total := len(hits)
	if len(hits) > q.Limit {
		hits = hits[:q.Limit]
	}
	return SearchResponse{
		Results:  hits,
		Total:    total,
		Query:    q.Query,
		Method:   method,
		Reranker: leg,
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
	return SearchResponse{Results: hits, Total: total, Query: q.Query, Method: "text", Reranker: "none"}, nil
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
	s.mu.Lock()
	s.jobs[jobID] = job
	s.lastJob = jobID
	s.mu.Unlock()

	go s.runReindex(jobCtx, job)
	return job, nil
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
	return job, ok
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
