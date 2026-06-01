package dependencies

import (
	"context"
	"fmt"
	"sync"
	"time"

	"security-health/internal/clock"
	"security-health/internal/validation"
)

// Service is the dependencies domain's orchestration core: it discovers fleet
// dependencies, annotates them with vuln status, persists them to the SQLite
// corpus, and serves search/status. Reindex runs as a cancellable async job;
// a 5-minute reconcile loop drives periodic refreshes.
//
// Search is TEXT + structured-filter over the SQLite corpus today (always
// available). A semantic (AI) layer over a pre-embedded Qdrant index is the
// declared next increment; until then a MODE_AI request degrades to TEXT and
// Status reports the AI backends' availability honestly.
type Service struct {
	repoRoot string
	store    *Store
	annot    *Annotator
	clock    clock.Clock

	// aiProbe reports whether the AI backends (ollama embeddings, qdrant
	// vector store) are reachable. Nil ⇒ AI unavailable. Wired when the
	// semantic layer lands; today Status uses it only for reporting.
	aiProbe func(ctx context.Context) (ollama, qdrant bool)

	mu      sync.Mutex
	jobs    map[string]*reindexJob
	jobSeq  int
	running bool
}

// Deps wires the service. A nil Annotator/Clock defaults to the real ones.
type Deps struct {
	RepoRoot  string
	Store     *Store
	Annotator *Annotator
	Clock     clock.Clock
	AIProbe   func(ctx context.Context) (ollama, qdrant bool)
}

// NewService constructs the dependencies service.
func NewService(d Deps) *Service {
	clk := d.Clock
	if clk == nil {
		clk = clock.System{}
	}
	annot := d.Annotator
	if annot == nil {
		annot = NewAnnotator(d.RepoRoot, validation.NewExecCommander())
	}
	return &Service{
		repoRoot: d.RepoRoot,
		store:    d.Store,
		annot:    annot,
		clock:    clk,
		aiProbe:  d.AIProbe,
		jobs:     map[string]*reindexJob{},
	}
}

// Search runs the query against the corpus. MODE_AI is accepted but degrades
// to TEXT until the semantic layer is wired; the response reports the mode
// actually used so callers can render a "(text mode)" hint.
func (s *Service) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	hits, err := s.store.Search(ctx, req)
	if err != nil {
		return SearchResponse{}, err
	}
	return SearchResponse{Results: hits, ModeUsed: ModeText}, nil
}

// Status reports backend availability and corpus/reconcile state.
func (s *Service) Status(ctx context.Context) (Status, error) {
	count, err := s.store.Count(ctx)
	if err != nil {
		return Status{}, err
	}
	vulnCount, err := s.store.VulnerableCount(ctx)
	if err != nil {
		return Status{}, err
	}
	at, outcome := s.store.ReconcileState(ctx)
	var ollama, qdrant bool
	if s.aiProbe != nil {
		ollama, qdrant = s.aiProbe(ctx)
	}
	return Status{
		Available:            true, // TEXT/structured search always works
		Ollama:               ollama,
		Qdrant:               qdrant,
		IndexedCount:         count,
		VulnerableCount:      vulnCount,
		LastReconcileAt:      at,
		LastReconcileOutcome: outcome,
	}, nil
}

// reindexJob tracks one async reconcile.
type reindexJob struct {
	mu        sync.Mutex
	id        string
	state     string // pending | running | succeeded | failed | cancelled
	processed int
	total     int
	errMsg    string
	cancel    context.CancelFunc
}

func (j *reindexJob) snapshot() (state string, processed, total int, errMsg string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.state, j.processed, j.total, j.errMsg
}

func (j *reindexJob) set(state string, processed, total int, errMsg string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.state = state
	if total >= 0 {
		j.total = total
	}
	if processed >= 0 {
		j.processed = processed
	}
	if errMsg != "" {
		j.errMsg = errMsg
	}
}

// ReindexResult is returned by Reindex.
type ReindexResult struct {
	JobID          string
	PlannedUpserts int
	PlannedDeletes int
	DryRun         bool
}

// Reindex discovers (and annotates) the fleet (or one scenario) and computes
// planned upsert/delete counts. For dry_run it returns the plan synchronously
// without writing. For a real run it starts an async job that applies the
// changes and returns the planned counts plus the job id to poll.
func (s *Service) Reindex(ctx context.Context, scenario string, dryRun bool) (ReindexResult, error) {
	fresh, err := s.discover(ctx, scenario)
	if err != nil {
		return ReindexResult{}, err
	}
	upserts, deletes, err := s.store.Diff(ctx, scenario, fresh)
	if err != nil {
		return ReindexResult{}, err
	}
	if dryRun {
		return ReindexResult{JobID: "", PlannedUpserts: upserts, PlannedDeletes: deletes, DryRun: true}, nil
	}

	jobID := s.nextJobID()
	jobCtx, cancel := context.WithCancel(context.Background())
	job := &reindexJob{id: jobID, state: "pending", total: len(fresh), cancel: cancel}
	s.mu.Lock()
	s.jobs[jobID] = job
	s.mu.Unlock()

	go s.runReindexJob(jobCtx, job, scenario, fresh)

	return ReindexResult{JobID: jobID, PlannedUpserts: upserts, PlannedDeletes: deletes, DryRun: false}, nil
}

// RunReconcileOnce performs a synchronous fleet reconcile (used by the sync
// loop). It discovers, annotates, applies, and records reconcile state.
func (s *Service) RunReconcileOnce(ctx context.Context) error {
	fresh, err := s.discover(ctx, "")
	if err != nil {
		_ = s.store.SetReconcileState(ctx, s.now(), "discovery failed: "+err.Error())
		return err
	}
	if err := s.store.Apply(ctx, "", fresh, s.now()); err != nil {
		_ = s.store.SetReconcileState(ctx, s.now(), "apply failed: "+err.Error())
		return err
	}
	_ = s.store.SetReconcileState(ctx, s.now(), fmt.Sprintf("reconciled %d records", len(fresh)))
	return nil
}

func (s *Service) runReindexJob(ctx context.Context, job *reindexJob, scenario string, fresh []DependencyRecord) {
	job.set("running", 0, len(fresh), "")
	if ctx.Err() != nil {
		job.set("cancelled", -1, -1, "")
		return
	}
	if err := s.store.Apply(ctx, scenario, fresh, s.now()); err != nil {
		job.set("failed", -1, -1, err.Error())
		_ = s.store.SetReconcileState(ctx, s.now(), "reindex failed: "+err.Error())
		return
	}
	if ctx.Err() != nil {
		job.set("cancelled", -1, -1, "")
		return
	}
	job.set("succeeded", len(fresh), len(fresh), "")
	_ = s.store.SetReconcileState(ctx, s.now(), fmt.Sprintf("reindexed %d records", len(fresh)))
}

// ReindexStatus returns the state of a job, or ok=false if unknown.
func (s *Service) ReindexStatus(jobID string) (state string, processed, total int, errMsg string, ok bool) {
	s.mu.Lock()
	job, found := s.jobs[jobID]
	s.mu.Unlock()
	if !found {
		return "", 0, 0, "", false
	}
	state, processed, total, errMsg = job.snapshot()
	return state, processed, total, errMsg, true
}

// ReindexCancel requests cooperative cancellation. Returns cancelled=false when
// the job is unknown or already terminal.
func (s *Service) ReindexCancel(jobID string) (cancelled, ok bool) {
	s.mu.Lock()
	job, found := s.jobs[jobID]
	s.mu.Unlock()
	if !found {
		return false, false
	}
	state, _, _, _ := job.snapshot()
	if isTerminal(state) {
		return false, true
	}
	job.cancel()
	job.set("cancelled", -1, -1, "")
	return true, true
}

func isTerminal(state string) bool {
	switch state {
	case "succeeded", "failed", "cancelled":
		return true
	}
	return false
}

// discover walks the fleet (or one scenario) and annotates with vuln status.
func (s *Service) discover(ctx context.Context, scenario string) ([]DependencyRecord, error) {
	var fresh []DependencyRecord
	var err error
	if scenario == "" {
		fresh, err = DiscoverFleet(s.repoRoot)
	} else {
		fresh, err = DiscoverScenario(scenarioDir(s.repoRoot, scenario), scenario)
	}
	if err != nil {
		return nil, err
	}
	s.annot.Annotate(ctx, fresh)
	return fresh, nil
}

func (s *Service) now() string { return s.clock.Now().UTC().Format(time.RFC3339) }

func (s *Service) nextJobID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobSeq++
	return fmt.Sprintf("reindex-%s-%d", s.clock.Now().UTC().Format("20060102T150405"), s.jobSeq)
}

func scenarioDir(repoRoot, scenario string) string {
	return repoRoot + "/scenarios/" + scenario
}
