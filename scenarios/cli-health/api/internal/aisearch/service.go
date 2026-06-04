package aisearch

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	pkg "github.com/vrooli/aisearch-go"
)

// SearchMode tags how Search should retrieve results. It is the proto-facing
// enum the Connect handler maps to and is intentionally kept local rather than
// adopting pkg.SearchMode (which has no "ai" member) — see the read-path
// adoption note in models.go and docs/internal/DECISIONS.md (WS5).
type SearchMode string

const (
	ModeAuto SearchMode = "auto" // prefer ai, fall back to text
	ModeAI   SearchMode = "ai"   // ai only; error if unavailable
	ModeText SearchMode = "text" // text only; never embed
)

// Service is the cli-health AI-search orchestrator. It owns the search/reindex
// surface the Connect handlers call and delegates the index/store/reconcile core
// to the shared engine (packages/aisearch-go): the embedder, vector store, and
// reconciler are all package types.
type Service struct {
	embedder    pkg.Embedder
	vectorStore pkg.VectorStore
	discovery   DiscoverySource
	reconciler  *pkg.Reconciler
	spec        pkg.CollectionSpec

	threshold float64
	floor     pkg.FloorConfig

	rerankEnabled bool
	reranker      *pkg.RerankerChain

	mu      sync.Mutex
	jobs    map[string]*reindexJob
	lastJob string
}

// reindexJob is the per-job state surfaced by ReindexStatus.
type reindexJob struct {
	ID         string
	State      string // queued|running|succeeded|failed|cancelled
	Scenario   string
	DryRun     bool
	Plan       *pkg.DriftReport
	Apply      *pkg.ApplyResult
	Err        string
	StartedAt  time.Time
	FinishedAt time.Time
	cancel     context.CancelFunc
}

// Options configure NewService.
type Options struct {
	Embedder         pkg.Embedder
	VectorStore      pkg.VectorStore
	Discovery        DiscoverySource
	Parallelism      int
	MaxEmbedsPerTick int
	Threshold        float64
	// Floor tunes the WS2 relative relevance cutoff applied to AI results.
	Floor pkg.FloorConfig
	// RerankEnabled gates the WS4 reranker pass; Reranker is the degradation
	// chain (cross-encoder -> llm -> fused order). When disabled or nil, results
	// keep their dense order — a pure addition.
	RerankEnabled bool
	Reranker      *pkg.RerankerChain
}

// NewService builds a Service over the shared engine. cli-health indexes
// commands as single-chunk sources (identity chunker), dense-only (no sparse
// vector); hybrid/rerank arrive for the documentation consumer in later phases.
func NewService(opts Options) *Service {
	// Commands are dense-only, single-chunk sources whose Body is the
	// pre-composed embedding text — exactly the NewDenseBinding common case
	// (identity chunker + identity composer, no sparse vector).
	binding := pkg.NewDenseBinding(commandKind, idPrefix, opts.VectorStore, newCommandSource(opts.Discovery))
	rec := pkg.NewReconciler(opts.Embedder, []pkg.SourceBinding{binding}, opts.Parallelism)
	rec.MaxEmbedsPerTick = opts.MaxEmbedsPerTick

	threshold := opts.Threshold
	if threshold < 0 {
		threshold = 0
	}
	return &Service{
		embedder:    opts.Embedder,
		vectorStore: opts.VectorStore,
		discovery:   opts.Discovery,
		reconciler:  rec,
		spec: pkg.CollectionSpec{
			Name:          DefaultCollection,
			DenseSize:     pkg.DefaultVectorSize,
			DenseDistance: pkg.DefaultDenseDistance,
			Model:         pkg.DefaultEmbedModel,
		},
		threshold:     threshold,
		floor:         opts.Floor,
		rerankEnabled: opts.RerankEnabled,
		reranker:      opts.Reranker,
		jobs:          make(map[string]*reindexJob),
	}
}

// Reconciler exposes the underlying reconciler for the sync loop.
func (s *Service) Reconciler() *pkg.Reconciler { return s.reconciler }

// EnsureCollection is called once at startup; idempotent.
func (s *Service) EnsureCollection(ctx context.Context) error {
	return s.vectorStore.EnsureCollection(ctx, s.spec)
}

// Search performs retrieval with AI-first, text-fallback semantics.
func (s *Service) Search(ctx context.Context, query string, limit int, mode SearchMode) (*SearchResponse, error) {
	if strings.TrimSpace(query) == "" {
		return &SearchResponse{Query: query, Method: "text"}, nil
	}
	if limit <= 0 {
		limit = 10
	}
	if mode == "" {
		mode = ModeAuto
	}

	if mode == ModeText {
		return s.textSearch(ctx, query, limit)
	}

	if s.embedder == nil {
		if mode == ModeAI {
			return nil, errors.New("ai mode requested but embedder is not configured")
		}
		return s.textSearch(ctx, query, limit)
	}

	vec, err := s.embedder.Embed(ctx, query)
	if err != nil {
		if mode == ModeAI {
			return nil, fmt.Errorf("embed query: %w", err)
		}
		log.Printf("[cli-health/aisearch] embed failed, falling back to text: %v", err)
		return s.textSearch(ctx, query, limit)
	}

	// WS4: when reranking is active, over-fetch a shortlist so the reranker has
	// candidates to reorder beyond the requested page.
	doRerank := s.rerankEnabled && s.reranker != nil
	queryLimit := limit
	if doRerank && rerankShortlist(limit) > queryLimit {
		queryLimit = rerankShortlist(limit)
	}

	results, err := s.vectorStore.Query(ctx, pkg.HybridQuery{
		Dense:          vec,
		Limit:          queryLimit,
		ScoreThreshold: s.threshold,
	})
	if err != nil {
		if mode == ModeAI {
			return nil, fmt.Errorf("qdrant search: %w", err)
		}
		log.Printf("[cli-health/aisearch] vector search failed, falling back to text: %v", err)
		return s.textSearch(ctx, query, limit)
	}

	// WS4: rerank the shortlist with the degradation chain FIRST, so the floor
	// below operates on the reranker's calibrated scores. When no leg is
	// reachable, behavior is exactly the dense order — a pure addition.
	reranker := "none"
	if doRerank {
		if active := s.reranker.Active(ctx); active != nil {
			results = rerankResults(ctx, active, query, results)
			reranker = active.Name()
		}
	}

	// WS2: drop weak/garbage hits with a query-adaptive relative cutoff AFTER
	// rerank (cap-fabecce56b518120) — junk the reranker drives to ~0 then
	// collapses out of the page count instead of riding along on its dense
	// score. With rerank off this is identical to flooring the dense scores.
	// Keeps the top hit; never hides correct answers to sparse queries (a fixed
	// floor would, given the weak-real/gibberish overlap).
	results = pkg.ApplyRelevanceFloor(results, s.floor)

	if len(results) > limit {
		results = results[:limit]
	}

	hits := make([]SearchHit, 0, len(results))
	for _, r := range results {
		hits = append(hits, payloadToHit(r.ID, r.Score, r.Payload))
	}
	return &SearchResponse{
		Results:  hits,
		Total:    len(hits),
		Query:    query,
		Method:   "ai",
		Reranker: reranker,
	}, nil
}

// rerankShortlist is the candidate count fetched for reranking — at least 20,
// or the page size when larger — so the reranker reorders a meaningful pool.
func rerankShortlist(limit int) int {
	const minShortlist = 20
	if limit > minShortlist {
		return limit
	}
	return minShortlist
}

// rerankResults reorders a result set by a reranker's scores, preserving the
// upstream (dense) order for any candidate the reranker omitted. On reranker
// error the original order is kept (degrade cleanly). Mirrors pkg.ApplyRerank
// at the SearchResult level (cli-health does not adopt the generic SearchHit).
func rerankResults(ctx context.Context, r pkg.Reranker, query string, results []pkg.SearchResult) []pkg.SearchResult {
	if len(results) < 2 {
		return results
	}
	candidates := make([]pkg.RerankCandidate, len(results))
	for i, res := range results {
		candidates[i] = pkg.RerankCandidate{ID: res.ID, Text: candidateText(res.Payload)}
	}
	scores, err := r.Rerank(ctx, query, candidates)
	if err != nil {
		log.Printf("[cli-health/aisearch] rerank failed, keeping dense order: %v", err)
		return results
	}
	if len(scores) == 0 {
		return results
	}
	scoreByID := make(map[string]float64, len(scores))
	for _, sc := range scores {
		scoreByID[sc.ID] = sc.Score
	}
	type ranked struct {
		res    pkg.SearchResult
		score  float64
		scored bool
		order  int
	}
	items := make([]ranked, len(results))
	for i, res := range results {
		sc, ok := scoreByID[res.ID]
		items[i] = ranked{res: res, score: sc, scored: ok, order: i}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].scored != items[j].scored {
			return items[i].scored
		}
		if items[i].scored && items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].order < items[j].order
	})
	out := make([]pkg.SearchResult, len(items))
	for i, it := range items {
		res := it.res
		if it.scored {
			res.Score = it.score
		}
		out[i] = res
	}
	return out
}

// candidateText is the passage handed to the reranker for one hit: the stored
// retrievable body, falling back to the command's path + description.
func candidateText(payload map[string]any) string {
	if body, _ := payload["body"].(string); strings.TrimSpace(body) != "" {
		return body
	}
	full, _ := payload["full_path"].(string)
	desc, _ := payload["description"].(string)
	return strings.TrimSpace(full + " " + desc)
}

// textSearch implements BM25-ish substring scoring over freshly discovered
// records. Slow (re-scans on every call) but correct — used when ollama or
// qdrant is unavailable.
func (s *Service) textSearch(ctx context.Context, query string, limit int) (*SearchResponse, error) {
	scenarios, err := s.discovery.ListScenarios(ctx)
	if err != nil {
		return nil, fmt.Errorf("list scenarios: %w", err)
	}
	terms := tokenize(query)
	if len(terms) == 0 {
		return &SearchResponse{Query: query, Method: "text"}, nil
	}

	type scored struct {
		hit   SearchHit
		score float64
	}
	scoredHits := make([]scored, 0, 64)
	for _, scenario := range scenarios {
		records, err := s.discovery.Discover(ctx, scenario)
		if err != nil {
			continue
		}
		for _, r := range records {
			score := scoreRecord(r, terms)
			if score <= 0 {
				continue
			}
			scoredHits = append(scoredHits, scored{
				hit: SearchHit{
					ID:           pointIDForCommand(r.FullPath),
					Origin:       r.Origin,
					Group:        r.Group,
					Name:         r.Name,
					FullPath:     r.FullPath,
					Description:  r.Description,
					Tags:         r.Tags,
					Binding:      r.Binding,
					Source:       r.Source,
					Score:        score,
					ScorePercent: int(score*100 + 0.5),
				},
				score: score,
			})
		}
	}

	sort.Slice(scoredHits, func(i, j int) bool {
		if scoredHits[i].score != scoredHits[j].score {
			return scoredHits[i].score > scoredHits[j].score
		}
		return scoredHits[i].hit.FullPath < scoredHits[j].hit.FullPath
	})
	if len(scoredHits) > limit {
		scoredHits = scoredHits[:limit]
	}
	out := make([]SearchHit, 0, len(scoredHits))
	for _, sh := range scoredHits {
		out = append(out, sh.hit)
	}
	return &SearchResponse{
		Results: out,
		Total:   len(out),
		Query:   query,
		Method:  "text",
	}, nil
}

func tokenize(q string) []string {
	q = strings.ToLower(q)
	fields := strings.FieldsFunc(q, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', ',', ';', ':', '/', '\\', '-', '_', '.', '"', '\'', '(', ')', '[', ']':
			return true
		}
		return false
	})
	out := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if len(f) < 2 {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}

func scoreRecord(r CommandRecord, terms []string) float64 {
	if len(terms) == 0 {
		return 0
	}
	// Weighted fields: full path > name > group > description > tags.
	name := strings.ToLower(r.FullPath + " " + r.Name + " " + r.Group)
	desc := strings.ToLower(r.Description)
	tags := strings.ToLower(strings.Join(r.Tags, " "))
	bind := strings.ToLower(r.Binding)
	var score float64
	for _, t := range terms {
		hit := false
		if strings.Contains(name, t) {
			score += 1.0
			hit = true
		}
		if strings.Contains(desc, t) {
			score += 0.4
			hit = true
		}
		if strings.Contains(tags, t) {
			score += 0.3
			hit = true
		}
		if strings.Contains(bind, t) {
			score += 0.2
			hit = true
		}
		if !hit {
			// Missing term shouldn't tank an otherwise good match.
			score -= 0.05
		}
	}
	if score <= 0 {
		return 0
	}
	// Normalize to (0, 1] for ScorePercent — divide by max possible.
	max := float64(len(terms)) * 1.9
	if max <= 0 {
		return 0
	}
	n := score / max
	if n > 1 {
		n = 1
	}
	if n < 0 {
		n = 0
	}
	return n
}

// Status reports backend availability and the indexed point count.
func (s *Service) Status(ctx context.Context) StatusReport {
	rep := StatusReport{}
	if s.embedder != nil {
		rep.Ollama = s.embedder.Available(ctx)
	}
	if s.vectorStore != nil {
		rep.Qdrant = s.vectorStore.Available(ctx)
		if rep.Qdrant {
			if n, err := s.vectorStore.CountPoints(ctx); err == nil {
				rep.IndexedCount = n
			}
		}
	}
	rep.Available = rep.Ollama && rep.Qdrant
	rep.Reranker = s.rerankerStatus(ctx)
	st := s.reconciler.Status()
	if st.FinishedAt != "" {
		rep.LastReconcileAt = st.FinishedAt
	}
	switch {
	case st.LastError != "":
		rep.LastReconcileOutcome = "error: " + st.LastError
	case st.LastResult != nil:
		var upserts, deletes, errs int
		for _, c := range st.LastResult.Collections {
			upserts += c.Upserted
			deletes += c.Deleted
		}
		errs = len(st.LastResult.Errors)
		rep.LastReconcileOutcome = fmt.Sprintf("upserts=%d deletes=%d errors=%d", upserts, deletes, errs)
	}
	return rep
}

// rerankerStatus reports the reranker leg search would use right now: "none"
// when disabled/unconfigured, the active leg's Name() when reachable, or
// "degraded" when enabled but no leg is currently reachable.
func (s *Service) rerankerStatus(ctx context.Context) string {
	if !s.rerankEnabled || s.reranker == nil {
		return "none"
	}
	if active := s.reranker.Active(ctx); active != nil {
		return active.Name()
	}
	return "degraded"
}

// Reindex queues a single reconcile job. Scenario filter is not yet honored;
// v1 always reconciles the full corpus. The flag is accepted for forward
// compatibility and surfaces as scenario= in the job.
func (s *Service) Reindex(_ context.Context, scenario string, dryRun bool) (*reindexJob, error) {
	jobID := uuid.NewString()
	jobCtx, cancel := context.WithCancel(context.Background())
	job := &reindexJob{
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

func (s *Service) runReindex(ctx context.Context, job *reindexJob) {
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

// ReindexStatus returns the current state of a single job.
func (s *Service) ReindexStatus(jobID string) (*reindexJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if jobID == "" {
		jobID = s.lastJob
	}
	job, ok := s.jobs[jobID]
	return job, ok
}

// ReindexCancel asks an in-flight job to stop. Returns true when a running
// job was cancelled; false otherwise (already terminal or unknown).
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
	s.reconciler.Cancel()
	return true
}

// JobExport projects a reindexJob to the wire shape (string fields only).
func (s *Service) JobExport(job *reindexJob) map[string]interface{} {
	if job == nil {
		return nil
	}
	plannedUpserts := 0
	plannedDeletes := 0
	if job.Plan != nil {
		for _, c := range job.Plan.Collections {
			plannedUpserts += len(c.ToUpsert)
			plannedDeletes += len(c.ToDelete)
		}
	}
	processed := 0
	total := plannedUpserts + plannedDeletes
	if job.Apply != nil {
		for _, c := range job.Apply.Collections {
			processed += c.Upserted + c.Deleted
		}
	}
	return map[string]interface{}{
		"job_id":          job.ID,
		"state":           job.State,
		"scenario":        job.Scenario,
		"dry_run":         job.DryRun,
		"planned_upserts": plannedUpserts,
		"planned_deletes": plannedDeletes,
		"processed":       processed,
		"total":           total,
		"error":           job.Err,
		"started_at":      job.StartedAt.Format(time.RFC3339),
		"finished_at":     formatIfNotZero(job.FinishedAt),
	}
}

func formatIfNotZero(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
