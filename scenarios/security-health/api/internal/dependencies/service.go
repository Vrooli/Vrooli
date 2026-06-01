package dependencies

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"security-health/internal/clock"
	"security-health/internal/dependencies/aisearch"
	"security-health/internal/validation"
)

// SemanticIndex is the optional AI-ranking overlay over the SBOM corpus. The
// concrete implementation (internal/dependencies/aisearch.Indexer) embeds each
// record into Qdrant; a nil SemanticIndex means TEXT-only. Defined as an
// interface so the service is testable with a fake.
type SemanticIndex interface {
	EnsureCollection(ctx context.Context) error
	Sync(ctx context.Context, items []aisearch.Item) (upserted, deleted int, err error)
	Query(ctx context.Context, query string, limit int) ([]aisearch.KeyScore, error)
	CountPoints(ctx context.Context) (int, error)
	Available(ctx context.Context) (ollama, qdrant bool)
}

const (
	// EnvReadyThreshold overrides the coverage fraction at/above which MODE_AI
	// is served instead of degrading to TEXT.
	EnvReadyThreshold = "SECURITY_HEALTH_INDEX_READY_THRESHOLD"
	// DefaultReadyThreshold is the coverage fraction the vector index must reach
	// before AI mode is served — below it the backfill is still populating and
	// AI would silently miss records TEXT would find, so we serve TEXT.
	DefaultReadyThreshold = 0.95
	// readinessTTL bounds how stale the cached coverage gate may be: Search
	// recomputes lazily past this so an out-of-band Qdrant change converges even
	// without a reconcile.
	readinessTTL = 60 * time.Second
)

// Service is the dependencies domain's orchestration core: it discovers fleet
// dependencies, annotates them with vuln status, persists them to the SQLite
// corpus, and serves search/status. Reindex runs as a cancellable async job;
// a 5-minute reconcile loop drives periodic refreshes.
//
// The SQLite corpus is the source of truth and always powers deterministic
// TEXT + structured-filter search. When a SemanticIndex is wired and its
// backends (ollama + qdrant) are up, a MODE_AI request is ranked by vector
// similarity over the same corpus; otherwise it degrades to TEXT and Status
// reports the backends' availability honestly.
type Service struct {
	repoRoot string
	store    *Store
	annot    *Annotator
	clock    clock.Clock

	// index is the optional semantic overlay. Nil ⇒ AI unavailable (TEXT-only).
	// It is kept in sync after every authoritative store.Apply, best-effort.
	index SemanticIndex

	// aiProbe reports whether the AI backends (ollama embeddings, qdrant
	// vector store) are reachable. Defaults to index.Available when an index is
	// wired; an explicit probe (tests) takes precedence.
	aiProbe func(ctx context.Context) (ollama, qdrant bool)

	mu     sync.Mutex
	jobs   map[string]*reindexJob
	jobSeq int

	// readiness caches the vector-index coverage gate so Search never makes a
	// network call on the hot path. refreshReadiness recomputes it after each
	// full reconcile and at startup; Search recomputes lazily past readinessTTL.
	readyMu         sync.Mutex
	readyThreshold  float64
	indexReady      bool
	indexedVectors  int
	expectedVectors int
	readyCheckedAt  time.Time
}

// Deps wires the service. A nil Annotator/Clock defaults to the real ones.
type Deps struct {
	RepoRoot  string
	Store     *Store
	Annotator *Annotator
	Clock     clock.Clock
	Index     SemanticIndex
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
	probe := d.AIProbe
	if probe == nil && d.Index != nil {
		probe = d.Index.Available
	}
	return &Service{
		repoRoot:       d.RepoRoot,
		store:          d.Store,
		annot:          annot,
		clock:          clk,
		index:          d.Index,
		aiProbe:        probe,
		jobs:           map[string]*reindexJob{},
		readyThreshold: loadReadyThreshold(),
	}
}

// loadReadyThreshold reads the coverage threshold from the environment, falling
// back to the default for an empty/malformed/out-of-range value.
func loadReadyThreshold() float64 {
	raw := strings.TrimSpace(os.Getenv(EnvReadyThreshold))
	if raw == "" {
		return DefaultReadyThreshold
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 || v > 1 {
		log.Printf("[security-health] invalid %s=%q, using default %.2f", EnvReadyThreshold, raw, DefaultReadyThreshold)
		return DefaultReadyThreshold
	}
	return v
}

// Search runs the query against the corpus. MODE_AI ranks by vector similarity
// when the semantic index is wired and its backends are up; on any failure (or
// when no index is configured) it degrades to deterministic TEXT search. The
// response reports the mode actually used so callers can render a "(text mode)"
// hint.
func (s *Service) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	if req.Mode == ModeAI && s.index != nil && strings.TrimSpace(req.Query) != "" && s.indexReadyNow(ctx) {
		if resp, ok := s.aiSearch(ctx, req); ok {
			return resp, nil
		}
		// fall through to TEXT on any AI failure (logged in aiSearch)
	}
	hits, err := s.store.Search(ctx, req)
	if err != nil {
		return SearchResponse{}, err
	}
	return SearchResponse{Results: hits, ModeUsed: ModeText}, nil
}

// aiSearch ranks the corpus by vector similarity. It embeds the query, hydrates
// the ANN hits back into records, applies the structured filters in-memory, and
// preserves the vector ranking. Returns ok=false (caller falls back to TEXT) on
// any backend error so a down embedder/qdrant never fails the request.
func (s *Service) aiSearch(ctx context.Context, req SearchRequest) (SearchResponse, bool) {
	limit := req.Limit
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}
	// The index is keyed by package identity (ecosystem|name|version), so each
	// ranked package fans out to every scenario record that uses it — the blast
	// radius a security console exists to surface. Over-fetch *packages* (each
	// fans out) so post-filtering still fills the page.
	scored, err := s.index.Query(ctx, req.Query, limit*4)
	if err != nil {
		log.Printf("[security-health] AI search failed, falling back to TEXT: %v", err)
		return SearchResponse{}, false
	}
	if len(scored) == 0 {
		// Search uses threshold=0, so the ANN returns the nearest points
		// whenever any exist — an empty result means the index isn't populated
		// yet (e.g. the first-reconcile backfill is still running). Fall back to
		// deterministic TEXT rather than hand back zero hits the corpus could
		// answer. (Subsumed by the readiness gate, kept as defense-in-depth.)
		return SearchResponse{}, false
	}
	pkgKeys := make([]string, 0, len(scored))
	for _, ks := range scored {
		pkgKeys = append(pkgKeys, ks.Key)
	}
	byPkg, err := s.store.RecordsByPackages(ctx, pkgKeys)
	if err != nil {
		log.Printf("[security-health] AI search hydrate failed, falling back to TEXT: %v", err)
		return SearchResponse{}, false
	}
	hits := make([]SearchResult, 0, len(scored))
	for _, ks := range scored {
		recs, ok := byPkg[ks.Key]
		if !ok {
			continue // index lags corpus — skip ghost package
		}
		// recs is already sorted by dep_key (deterministic secondary order).
		// Each record inherits its package's vector score.
		for _, r := range recs {
			if !matchesFilters(r, req) {
				continue
			}
			hits = append(hits, SearchResult{Record: r, Score: ks.Score})
		}
	}
	// Records are appended in package-rank order; keep that order and clamp to
	// the requested page.
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return SearchResponse{Results: hits, ModeUsed: ModeAI}, true
}

// indexReadyNow reports whether the cached coverage gate permits AI mode,
// recomputing lazily when the cache is older than readinessTTL so an
// out-of-band Qdrant change still converges without a reconcile.
func (s *Service) indexReadyNow(ctx context.Context) bool {
	s.readyMu.Lock()
	stale := s.readyCheckedAt.IsZero() || s.clock.Now().Sub(s.readyCheckedAt) > readinessTTL
	ready := s.indexReady
	s.readyMu.Unlock()
	if !stale {
		return ready
	}
	s.refreshReadiness(ctx)
	s.readyMu.Lock()
	defer s.readyMu.Unlock()
	return s.indexReady
}

// refreshReadiness recomputes the vector-index coverage gate: ready iff the
// index holds at least ceil(expected * threshold) points. Best-effort — on any
// backend error it marks not-ready (AI degrades to TEXT) and stamps the check
// time so a down backend isn't hammered every request.
func (s *Service) refreshReadiness(ctx context.Context) {
	if s.index == nil {
		s.readyMu.Lock()
		s.indexReady, s.indexedVectors, s.expectedVectors, s.readyCheckedAt = false, 0, 0, s.clock.Now()
		s.readyMu.Unlock()
		return
	}
	idx, err := s.index.CountPoints(ctx)
	if err != nil {
		log.Printf("[security-health] readiness: vector count failed (AI on TEXT): %v", err)
		s.readyMu.Lock()
		s.indexReady, s.readyCheckedAt = false, s.clock.Now()
		s.readyMu.Unlock()
		return
	}
	exp, err := s.store.DistinctPackageCount(ctx)
	if err != nil {
		log.Printf("[security-health] readiness: package count failed (AI on TEXT): %v", err)
		s.readyMu.Lock()
		s.indexReady, s.readyCheckedAt = false, s.clock.Now()
		s.readyMu.Unlock()
		return
	}
	ready := exp > 0 && idx >= int(math.Ceil(float64(exp)*s.readyThreshold))
	s.readyMu.Lock()
	s.indexedVectors, s.expectedVectors, s.indexReady, s.readyCheckedAt = idx, exp, ready, s.clock.Now()
	s.readyMu.Unlock()
}

// matchesFilters applies the structured filters (ecosystem / vulnerable-only /
// name-glob) that the SQL layer would otherwise apply, so AI and TEXT search
// honor the same filter contract.
func matchesFilters(r DependencyRecord, req SearchRequest) bool {
	if req.Ecosystem != EcosystemUnspecified && r.Ecosystem != req.Ecosystem {
		return false
	}
	if req.VulnerableOnly && !r.Vulnerable() {
		return false
	}
	if req.NameGlob != "" {
		if ok, _ := path.Match(req.NameGlob, r.Name); !ok {
			return false
		}
	}
	return true
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
	s.readyMu.Lock()
	indexedVectors, expectedVectors, indexReady := s.indexedVectors, s.expectedVectors, s.indexReady
	s.readyMu.Unlock()
	return Status{
		Available:            true, // TEXT/structured search always works
		Ollama:               ollama,
		Qdrant:               qdrant,
		IndexedCount:         count,
		VulnerableCount:      vulnCount,
		LastReconcileAt:      at,
		LastReconcileOutcome: outcome,
		IndexedVectors:       indexedVectors,
		ExpectedVectors:      expectedVectors,
		IndexReady:           indexReady,
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
	s.syncIndex(ctx, "")
	_ = s.store.SetReconcileState(ctx, s.now(), fmt.Sprintf("reconciled %d records", len(fresh)))
	return nil
}

// EnsureIndex creates the semantic collection at startup (idempotent, no-op
// when no index is wired). Errors are non-fatal: search degrades to TEXT. It
// also primes the readiness cache so a warm restart over an already-populated
// collection serves AI immediately rather than waiting for the first reconcile.
func (s *Service) EnsureIndex(ctx context.Context) error {
	if s.index == nil {
		return nil
	}
	err := s.index.EnsureCollection(ctx)
	s.refreshReadiness(ctx)
	return err
}

// syncIndex pushes the freshly-applied record set into the semantic overlay.
// Best-effort: a down embedder/qdrant logs and leaves search on TEXT rather
// than failing the reconcile that just succeeded against the source of truth.
//
// It only runs on a full-corpus reconcile (scenario==""). A scoped reindex
// holds only one scenario's records, and Sync treats the entire collection as
// the universe — syncing a scoped set would delete every other scenario's
// vectors. Scoped reindexes therefore update only the SQLite corpus (the source
// of truth); the next full reconcile folds the change into the index.
func (s *Service) syncIndex(ctx context.Context, scenario string) {
	if s.index == nil {
		return
	}
	if scenario != "" {
		return // scoped reindex: index refresh deferred to the next full reconcile
	}
	// Embed the deduped package universe (one vector per distinct
	// ecosystem|name|version), not one per scenario row — a CVE belongs to a
	// package+version, not a scenario-usage, so this cuts the embed/upsert
	// workload ~10× on the real corpus.
	pkgs, err := s.store.PackageItems(ctx)
	if err != nil {
		log.Printf("[security-health] semantic index sync skipped (package read failed): %v", err)
		return
	}
	items := make([]aisearch.Item, 0, len(pkgs))
	for _, p := range pkgs {
		items = append(items, aisearch.Item{Key: p.PkgKey(), Text: packageEmbeddingText(p)})
	}
	if up, del, err := s.index.Sync(ctx, items); err != nil {
		log.Printf("[security-health] semantic index sync failed (search stays on TEXT): %v", err)
	} else if up > 0 || del > 0 {
		log.Printf("[security-health] semantic index synced: %d upserted, %d deleted", up, del)
	}
	// Recompute the coverage gate against the freshly-synced index.
	s.refreshReadiness(ctx)
}

// packageEmbeddingText renders a package as the natural-language string whose
// embedding ranks it — name, version, ecosystem, and any known vulnerabilities,
// so semantic queries like "serialization CVE in a Go HTTP library" can match.
// Scenario + source file are deliberately dropped: they are structured-filter /
// TEXT concerns, not good semantic signals, and folding them in would re-key the
// vector per scenario-usage (the duplication this design removes).
func packageEmbeddingText(p PackageItem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s version %s, a %s package.", p.Name, p.Version, ecosystemLabel(p.Ecosystem))
	if len(p.VulnIDs) > 0 {
		fmt.Fprintf(&b, " Known vulnerabilities: %s (maximum severity %s).",
			strings.Join(p.VulnIDs, ", "), severityLabel(p.MaxSeverity))
	} else {
		b.WriteString(" No known vulnerabilities.")
	}
	return b.String()
}

func ecosystemLabel(e Ecosystem) string {
	switch e {
	case EcosystemGo:
		return "Go module"
	case EcosystemNPM:
		return "npm"
	default:
		return "dependency"
	}
}

func severityLabel(s string) string {
	if strings.TrimSpace(s) == "" {
		return "unspecified"
	}
	return s
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
	s.syncIndex(ctx, scenario)
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
