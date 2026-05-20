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
)

// SearchMode tags how Search should retrieve results.
type SearchMode string

const (
	ModeAuto SearchMode = "auto" // prefer ai, fall back to text
	ModeAI   SearchMode = "ai"   // ai only; error if unavailable
	ModeText SearchMode = "text" // text only; never embed
)

// Service is the cli-health AI-search orchestrator. It owns the embedder,
// vector store, discovery source, and reconciler.
type Service struct {
	embedder    Embedder
	vectorStore VectorStore
	discovery   DiscoverySource
	reconciler  *Reconciler

	threshold float64

	mu       sync.Mutex
	jobs     map[string]*reindexJob
	lastJob  string
	lastSync time.Time
}

// reindexJob is the per-job state surfaced by ReindexStatus.
type reindexJob struct {
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

// Options configure NewService.
type Options struct {
	Embedder    Embedder
	VectorStore VectorStore
	Discovery   DiscoverySource
	Parallelism int
	Threshold   float64
}

// NewService builds a Service with a Reconciler around the given seams.
func NewService(opts Options) *Service {
	desc := NewCommandDescriptor(opts.VectorStore, opts.Discovery)
	rec := NewReconciler(opts.Embedder, []CollectionDescriptor{desc}, opts.Parallelism)
	threshold := opts.Threshold
	if threshold <= 0 {
		threshold = 0.0
	}
	return &Service{
		embedder:    opts.Embedder,
		vectorStore: opts.VectorStore,
		discovery:   opts.Discovery,
		reconciler:  rec,
		threshold:   threshold,
		jobs:        make(map[string]*reindexJob),
	}
}

// Reconciler exposes the underlying reconciler for the sync loop.
func (s *Service) Reconciler() *Reconciler { return s.reconciler }

// EnsureCollection is called once at startup; idempotent.
func (s *Service) EnsureCollection(ctx context.Context) error {
	return s.vectorStore.EnsureCollection(ctx)
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

	results, err := s.vectorStore.Search(ctx, vec, limit, s.threshold)
	if err != nil {
		if mode == ModeAI {
			return nil, fmt.Errorf("qdrant search: %w", err)
		}
		log.Printf("[cli-health/aisearch] vector search failed, falling back to text: %v", err)
		return s.textSearch(ctx, query, limit)
	}

	hits := make([]SearchHit, 0, len(results))
	for _, r := range results {
		hits = append(hits, payloadToHit(r.ID, r.Score, r.Payload))
	}
	return &SearchResponse{
		Results: hits,
		Total:   len(hits),
		Query:   query,
		Method:  "ai",
	}, nil
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
					ID:           PointIDForCommand(r.FullPath),
					Scenario:     r.Scenario,
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

// Reindex queues a single reconcile job. Scenario filter is not yet honored;
// v1 always reconciles the full corpus. The flag is accepted for forward
// compatibility (plan §7 Phase 3) and surfaces as scenario= in the job.
func (s *Service) Reindex(ctx context.Context, scenario string, dryRun bool) (*reindexJob, error) {
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
		s.lastSync = job.FinishedAt
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
