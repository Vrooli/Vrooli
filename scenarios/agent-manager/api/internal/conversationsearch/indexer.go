package conversationsearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultIndexPageSize = 250
	// The periodic tick only drains queued changes. Post-commit kicks normally
	// do that sooner; the tick is the safety net for a lost kick.
	defaultChangeInterval = 15 * time.Minute
	// A full rebuild rewrites every staged document and its indexes, so it runs
	// only after a read-only drift check finds the catalog and source disagree.
	defaultDriftCheckInterval = 24 * time.Hour
	defaultDriftCheckDelay    = time.Hour
	initialRepairDelay        = 30 * time.Second
	initialRecoveryRetry      = 5 * time.Second
	// A full or resumed semantic rebuild embeds every eligible document, so its
	// budget scales with the planned corpus instead of a fixed wall clock. The
	// fixed 10-second budget this replaces could never finish a corpus of more
	// than a few hundred documents, which left semantic coverage at 7.6% on
	// 2026-09-09 while every rebuild reported "context deadline exceeded".
	// Measured 2026-09-09: nomic-embed-text embeds 18-36 documents/s one at a
	// time and ~280/s batched, so 250 ms per document is a 5-70x margin that
	// still bounds a wedged embedder.
	semanticRebuildMinimumBudget = 2 * time.Minute
	semanticRebuildPerDocument   = 250 * time.Millisecond
	semanticRebuildMaximumBudget = 12 * time.Hour
	// An incremental semantic pass is not proportional to its changes. Its
	// generation starts as a server-side copy of the whole serving collection
	// and ends with a full validation scroll, an alias promotion and retention
	// cleanup. Observed 2026-11-20: a 12-change pass finished in 44s when Qdrant
	// was idle and exceeded the old 2-minute budget ("stage delete ...: context
	// deadline exceeded") when it was busy; every such failure left that pass's
	// vectors stale until a full rebuild. The fixed overhead covers the copy,
	// validation and promotion; the per-document share covers the embedding.
	semanticIncrementalOverhead = 10 * time.Minute
)

// semanticChangesBudget is the wall-clock budget for one incremental semantic
// pass over n changed documents.
func semanticChangesBudget(n uint64) time.Duration {
	return min(semanticIncrementalOverhead+embeddingShare(n), semanticRebuildMaximumBudget)
}

// embeddingShare is the per-document embedding time for n documents, capped at
// the maximum budget so the multiplication cannot overflow a Duration.
func embeddingShare(n uint64) time.Duration {
	if n > uint64(semanticRebuildMaximumBudget/semanticRebuildPerDocument) {
		return semanticRebuildMaximumBudget
	}
	return time.Duration(n) * semanticRebuildPerDocument
}

// semanticBudgetBasis prefers the documents a rebuild will actually embed over
// every staged document. Tool payloads are never embedded, so the staged count
// inflates a large generation's budget to the ceiling; a failed eligible count
// keeps the conservative staged count.
func (i *Indexer) semanticBudgetBasis(ctx context.Context, generationID string, fallback uint64) uint64 {
	if eligible, err := i.repository.CountSemanticEligibleStaged(ctx, generationID); err == nil && eligible > 0 {
		return eligible
	}
	return fallback
}

// semanticRebuildBudget returns the wall-clock budget for a full semantic
// rebuild of planned documents, clamped to [minimum, maximum].
func semanticRebuildBudget(planned uint64) time.Duration {
	return min(max(semanticRebuildMinimumBudget, embeddingShare(planned)), semanticRebuildMaximumBudget)
}

var errCanonicalDeletionDuringSemanticBuild = errors.New("canonical deletion arrived during semantic build")

var ErrRecoveryPending = errors.New("conversation index recovery pending; new generations are paused")

// autoResumeMaxDocuments bounds crash recovery. Resuming a full rebuild of a
// large corpus automatically on every start can occupy the single SQLite writer
// and the local embedding model for days. Above this size the interrupted
// generation is rolled back and the serving generation continues; an operator
// can start a deliberate reindex when they want one.
const autoResumeMaxDocuments = 10_000

// initialRecoveryTimeout bounds the startup repair scan. A stuck scan must not
// leave the status permanently "recovery pending" or postpone new generations.
const initialRecoveryTimeout = 2 * time.Minute

// statusSnapshotRefreshTimeout bounds the background status refresh. The
// request itself never waits for it.
const statusSnapshotRefreshTimeout = 30 * time.Second

// semanticStatusLegTimeout bounds how long one background status refresh waits
// for the first vector count. The probe itself is capped by
// semanticStatusRefreshTimeout; this leaves room for generation reconciliation.
const semanticStatusLegTimeout = 12 * time.Second

// SemanticCoverageUnknownPrefix starts the degraded-dependency line that says
// semantic coverage has not been observed. Status consumers match it to print
// "unknown" instead of a zero count.
const SemanticCoverageUnknownPrefix = "semantic-coverage: unknown: "

// statusSnapshotCacheTTL serves repeated CLI/UI polls without re-reading the
// projection; the status is observational, not transactional.
const statusSnapshotCacheTTL = 5 * time.Second

type ReindexState string

const (
	ReindexPlanned   ReindexState = "planned"
	ReindexQueued    ReindexState = "queued"
	ReindexRunning   ReindexState = "running"
	ReindexCancelled ReindexState = "cancelled"
	ReindexFailed    ReindexState = "failed"
	ReindexComplete  ReindexState = "complete"
)

type ReindexJob struct {
	ID                 string
	State              ReindexState
	DryRun             bool
	PlannedDocuments   uint64
	ProcessedDocuments uint64
	UpsertedDocuments  uint64
	DeletedDocuments   uint64
	FailedDocuments    uint64
	SourceCheckpoint   string
	ShadowGeneration   string
	ActiveGeneration   string
	StartedAt          time.Time
	UpdatedAt          time.Time
	ErrorCode          string
}

type IndexerOptions struct {
	Source         SourceRepository
	Repository     *SQLiteRepository
	Semantic       SemanticRebuilder
	ChangeInterval time.Duration
	// DriftCheckInterval spaces read-only source/catalog comparisons; the first
	// runs DriftCheckDelay after the owner loop starts.
	DriftCheckInterval time.Duration
	DriftCheckDelay    time.Duration
	PageSize           int
	Clock              func() time.Time
}

type SemanticRebuilder interface {
	Rebuild(context.Context, string) error
}

type SemanticValidatedRebuilder interface {
	RebuildValidated(context.Context, string, func(context.Context) error) error
}

type SemanticChangeRebuilder interface {
	RebuildChanges(context.Context, string, []Document, []string, func(context.Context) error) error
}

type SemanticStatusReporter interface {
	SemanticStatus(context.Context) (documents uint64, collection, layout, model string, err error)
}

type SemanticRollback interface {
	Rollback(context.Context, string) error
}

// Indexer owns the durable catalog/FTS projection lifecycle. Kicks carry
// explicit post-commit change evidence, while every execution compares the
// authoritative source again; the queue is an accelerator, never source truth.
type Indexer struct {
	source        SourceRepository
	repository    *SQLiteRepository
	semantic      SemanticRebuilder
	interval      time.Duration
	driftInterval time.Duration
	driftDelay    time.Duration
	pageSize      int
	clock         func() time.Time

	kick            chan struct{}
	stop            context.CancelFunc
	mu              sync.Mutex
	running         bool
	recoveryPending bool
	recoveryError   string
	// resumeMaxDocuments bounds automatic crash recovery; tests override it to
	// exercise the oversized-generation policy without staging 50k documents.
	resumeMaxDocuments int
	// statusCache keeps the last successful status projection so a status read
	// blocked by the single SQLite writer returns an honest last snapshot
	// instead of hanging the operator's request.
	statusCacheMu    sync.Mutex
	statusCache      ProjectionStatus
	statusCacheAt    time.Time
	statusCacheValid bool
	statusRefreshing bool
	jobs             map[string]*indexJob
	idempotency      map[string]string
	// pruning keeps one detached settled-generation reap at a time. The reap
	// can delete millions of staged rows; it must never hold startup or status
	// behind the single SQLite writer.
	pruning atomic.Bool
}

type indexJob struct {
	ReindexJob
	cancel context.CancelFunc
	// reason names what started this generation, so a full rebuild is
	// attributable in the log instead of anonymous.
	reason string
}

func NewIndexer(options IndexerOptions) (*Indexer, error) {
	if options.Source == nil || options.Repository == nil {
		return nil, errors.New("conversation indexer requires source and repository")
	}
	if options.ChangeInterval <= 0 {
		options.ChangeInterval = defaultChangeInterval
	}
	if options.DriftCheckInterval <= 0 {
		options.DriftCheckInterval = defaultDriftCheckInterval
	}
	if options.DriftCheckDelay <= 0 {
		options.DriftCheckDelay = defaultDriftCheckDelay
	}
	if options.PageSize <= 0 {
		options.PageSize = defaultIndexPageSize
	}
	if options.PageSize > maxSourcePageSize {
		options.PageSize = maxSourcePageSize
	}
	if options.Clock == nil {
		options.Clock = time.Now
	}
	return &Indexer{
		source: options.Source, repository: options.Repository, semantic: options.Semantic, interval: options.ChangeInterval,
		driftInterval: options.DriftCheckInterval, driftDelay: options.DriftCheckDelay, pageSize: options.PageSize, clock: options.Clock, kick: make(chan struct{}, 1), jobs: map[string]*indexJob{}, idempotency: map[string]string{},
		resumeMaxDocuments: autoResumeMaxDocuments,
	}, nil
}

// Start is deliberately non-blocking. Initial reconciliation runs in the
// background and therefore cannot delay API health.
func (i *Indexer) Start(parent context.Context) {
	if i == nil {
		return
	}
	i.mu.Lock()
	if i.stop != nil {
		i.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	i.stop = cancel
	i.recoveryPending = true
	i.mu.Unlock()
	go func() {
		timer := time.NewTimer(initialRepairDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		// Recovery must admit or drain abandoned generations before kicks can
		// launch another writer against the same SQLite projection.
		i.loop(ctx)
	}()
}

func (i *Indexer) launchInitial(ctx context.Context) error {
	i.mu.Lock()
	i.recoveryPending = true
	i.mu.Unlock()
	// The legacy projection upgrade runs before recovery so no generation can
	// publish into the catalog while it is being copied.
	recoveryCtx, cancelRecovery := context.WithTimeout(ctx, initialRecoveryTimeout)
	defer cancelRecovery()
	var resumed bool
	_, err := i.repository.UpgradeLegacyProjection(recoveryCtx)
	if err != nil {
		err = fmt.Errorf("upgrade legacy conversation projection: %w", err)
	} else {
		resumed, err = i.recoverInterrupted(recoveryCtx)
	}
	var active bool
	if err == nil && !resumed {
		// Admission needs an authoritative read; observational status tolerates
		// missing metadata and must not turn a database failure into a fresh build.
		active, err = i.repository.HasActiveGeneration(recoveryCtx)
	}
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		diagnostic := fmt.Sprintf("%v: %v", ErrRecoveryPending, err)
		i.mu.Lock()
		i.recoveryError = diagnostic
		i.mu.Unlock()
		diagnosticCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_ = i.repository.SetRecoveryError(diagnosticCtx, diagnostic, i.clock().UTC())
		return err
	}
	i.mu.Lock()
	i.recoveryPending, i.recoveryError = false, ""
	i.mu.Unlock()
	diagnosticCtx, cancel := context.WithTimeout(ctx, time.Second)
	_ = i.repository.SetRecoveryError(diagnosticCtx, "", i.clock().UTC())
	cancel()
	// Staged rows stranded by an earlier process (retired, or failed with an
	// interrupted rollback) drain here rather than waiting for an activation.
	i.pruneSettledGenerations(ctx)
	if resumed {
		return nil
	}
	if active {
		// A recipe change alters every document, so it is the one serving state
		// that needs a full rebuild at startup rather than a change drain.
		if recipe, recipeErr := i.repository.ActiveGenerationRecipe(recoveryCtx); recipeErr == nil && recipe != DefaultRecipeVersion {
			i.launchRepair(ctx, fmt.Sprintf("startup: serving recipe %q is not %q", recipe, DefaultRecipeVersion))
			return nil
		}
		i.launchChanges(ctx)
		return nil
	}
	i.launchRepair(ctx, "startup: no active generation")
	return nil
}

// recoverInterrupted settles generations an earlier process left building. They
// are recorded as cancelled, not failed: the process stopped, nothing about the
// generation was wrong, and counting every restart as a failed generation made
// the catalog read as a failing index.
func (i *Indexer) recoverInterrupted(ctx context.Context) (bool, error) {
	generations, err := i.repository.BuildingGenerations(ctx)
	if err != nil {
		return false, fmt.Errorf("read interrupted generations: %w", err)
	}
	resumeIndex := -1
	var resumeCount uint64
	for index, generation := range generations {
		staged, countErr := i.repository.CountStagedGeneration(ctx, generation.GenerationID)
		if countErr != nil {
			return false, fmt.Errorf("inspect interrupted generation %s: %w", generation.GenerationID, countErr)
		}
		completeSnapshot := generation.RecipeVersion == DefaultRecipeVersion && generation.PlannedDocuments > 0 && generation.ProcessedDocuments == generation.PlannedDocuments && staged == generation.PlannedDocuments
		if completeSnapshot && staged > resumeCount {
			resumeIndex, resumeCount = index, staged
		}
	}
	if resumeIndex >= 0 && int(resumeCount) > i.resumeMaxDocuments {
		// Do not auto-resume an unbounded rebuild. Roll this generation back so
		// it cannot hold the writer or the embedding model hostage on every
		// restart; the active generation keeps serving the existing vectors.
		candidate := generations[resumeIndex]
		if semantic, ok := i.semantic.(SemanticRollback); ok {
			// The candidate never served. A Qdrant alias conflict here must not
			// block SQLite recovery forever; the orphaned candidate collection
			// remains visible to generation retention for cleanup. Bound the
			// best-effort attempt so recovery never waits on a slow Qdrant.
			rollbackCtx, cancelRollback := context.WithTimeout(ctx, 5*time.Second)
			_ = semantic.Rollback(rollbackCtx, candidate.GenerationID)
			cancelRollback()
		}
		if err := i.repository.MarkGenerationSettled(ctx, candidate.GenerationID, "cancelled", i.clock().UTC()); err != nil {
			return false, fmt.Errorf("settle oversized interrupted generation %s: %w", candidate.GenerationID, err)
		}
		// Reap its staged rows in the background. The generation never served,
		// so the startup gate must not wait for a 1.24M-row delete.
		go func(generationID string) {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			_ = i.repository.CleanupGenerationDocuments(cleanupCtx, generationID)
		}(candidate.GenerationID)
		resumeIndex, resumeCount = -1, 0
	}
	for index, generation := range generations {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if index == resumeIndex {
			continue
		}
		if semantic, ok := i.semantic.(SemanticRollback); ok {
			if err := semantic.Rollback(ctx, generation.GenerationID); err != nil {
				return false, fmt.Errorf("rollback semantic generation %s: %w", generation.GenerationID, err)
			}
		}
		if err := i.repository.RollbackStagedGeneration(ctx, generation.GenerationID, "cancelled", i.clock().UTC()); err != nil {
			return false, fmt.Errorf("cleanup interrupted generation %s: %w", generation.GenerationID, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if resumeIndex < 0 {
		return false, nil
	}
	// Resume publication only after abandoned-shadow cleanup has yielded its
	// writer; otherwise bootstrap and recovery compete with each other.
	generation := generations[resumeIndex]
	jobCtx, cancel := context.WithCancel(ctx)
	job := &indexJob{ReindexJob: ReindexJob{
		ID: generation.GenerationID, State: ReindexQueued, PlannedDocuments: resumeCount,
		ProcessedDocuments: resumeCount, UpsertedDocuments: resumeCount, SourceCheckpoint: generation.SourceCheckpoint,
		ShadowGeneration: generation.GenerationID, StartedAt: generation.CreatedAt, UpdatedAt: i.clock().UTC(),
	}, cancel: cancel}
	i.mu.Lock()
	i.jobs[generation.GenerationID] = job
	i.running = true
	i.mu.Unlock()
	go i.resume(jobCtx, generation, resumeCount)

	return true, nil
}

func (i *Indexer) resume(ctx context.Context, generation Generation, staged uint64) {
	id := generation.GenerationID
	defer func() {
		i.mu.Lock()
		i.running = false
		i.mu.Unlock()
		checkCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if changes, err := i.repository.PendingChanges(checkCtx, 1); err == nil && len(changes) > 0 {
			i.Kick()
		}
	}()
	i.update(id, func(job *indexJob) { job.State = ReindexRunning })
	if err := i.repository.PublishStagedGeneration(ctx, id, staged); err != nil {
		i.rollback(id, err, ctx)
		return
	}
	// The interrupted process may have died at any point after its original
	// watermark. Keeping all queued changes is conservative; deletion safety is
	// checked across the entire remaining queue before vector alias promotion.
	semanticCtx, cancelSemantic := context.WithTimeout(ctx, semanticRebuildBudget(i.semanticBudgetBasis(ctx, id, staged)))
	semanticErr := i.rebuildSemantic(semanticCtx, id, 0)
	cancelSemantic()
	if semanticErr != nil {
		cause := fmt.Errorf("resume semantic shadow rebuild: %w", semanticErr)
		if errors.Is(semanticErr, errCanonicalDeletionDuringSemanticBuild) || ctx.Err() != nil {
			i.failPublishedSemantic(id, cause, ctx)
			return
		}
		if !i.activateLexicalAfterSemanticFailure(id, cause, ctx) {
			return
		}
		return
	}
	if err := i.repository.ActivateGeneration(ctx, id, i.clock().UTC()); err != nil {
		i.rollback(id, err, ctx)
		return
	}
	_ = i.repository.ClearIndexError(ctx, i.clock().UTC())
	i.pruneSettledGenerations(ctx)
	i.update(id, func(job *indexJob) {
		job.State, job.ActiveGeneration, job.UpdatedAt = ReindexComplete, id, i.clock().UTC()
	})
}

func (i *Indexer) Stop() {
	if i == nil {
		return
	}
	i.mu.Lock()
	cancel := i.stop
	i.stop = nil
	i.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (i *Indexer) Kick() {
	if i == nil {
		return
	}
	select {
	case i.kick <- struct{}{}:
	default:
	}
}

// Notify is called only after the canonical mutation committed. Privacy
// deletions are removed from FTS immediately; vector/corpus repair remains
// asynchronous and coalesced.
func (i *Indexer) Notify(ctx context.Context, operation ChangeOperation, runID, eventID string) error {
	if i == nil {
		return nil
	}
	now := i.clock().UTC()
	switch operation {
	case ChangeDeleteEvent:
		if _, err := i.repository.DeleteSourceEvent(ctx, runID, eventID); err != nil {
			return err
		}
	case ChangeDeleteRun:
		if _, err := i.repository.DeleteRun(ctx, runID); err != nil {
			return err
		}
	}
	if err := i.repository.EnqueueChange(ctx, operation, runID, eventID, now); err != nil {
		return err
	}
	i.Kick()
	return nil
}

func (i *Indexer) loop(ctx context.Context) {
	// Keep recovery in the same owner loop as normal work. Read or cleanup
	// failure retains the admission gate; neither kicks nor the repair ticker
	// can create a competing generation while this retry is pending.
	for {
		if err := i.launchInitial(ctx); err == nil {
			break
		}
		timer := time.NewTimer(initialRecoveryRetry)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
	ticker := time.NewTicker(i.interval)
	defer ticker.Stop()
	drift := time.NewTimer(i.driftDelay)
	defer drift.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			i.launchChanges(ctx)
		case <-drift.C:
			i.checkDrift(ctx)
			drift.Reset(i.driftInterval)
		case <-i.kick:
			timer := time.NewTimer(250 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			for {
				select {
				case <-i.kick:
					continue
				default:
				}
				break
			}
			i.launchChanges(ctx)
		}
	}
}

func (i *Indexer) launchRepair(ctx context.Context, reason string) {
	_, _ = i.reindex(ctx, 0, "", false, false, reason)
}

func (i *Indexer) launchChanges(ctx context.Context) {
	changes, err := i.repository.PendingChanges(ctx, 1)
	if err != nil || len(changes) == 0 {
		return
	}
	if changes[0].Operation == ChangeRepair {
		i.launchRepair(ctx, "queued repair marker")
		return
	}
	_, _ = i.reindex(ctx, 0, "", false, true, "queued changes")
}

func (i *Indexer) Plan(ctx context.Context, maxDocuments uint64) (*ReindexJob, error) {
	count, _, _, err := i.scan(ctx, maxDocuments, nil)
	if err != nil {
		return nil, err
	}
	now := i.clock().UTC()
	return &ReindexJob{ID: newGenerationID(now), State: ReindexPlanned, DryRun: true, PlannedDocuments: count, StartedAt: now, UpdatedAt: now}, nil
}

func (i *Indexer) Reindex(parent context.Context, maxDocuments uint64, idempotencyKey string, dryRun bool) (*ReindexJob, error) {
	if dryRun {
		return i.Plan(parent, maxDocuments)
	}
	// Non-dry reindex is asynchronous: counting here would walk the complete
	// canonical corpus once before the actual staged walk and delay background
	// startup without improving safety. The running job records the authoritative
	// planned count when its single snapshot traversal completes.
	return i.reindex(parent, maxDocuments, idempotencyKey, false, false, "operator reindex")
}

func (i *Indexer) reindex(parent context.Context, maxDocuments uint64, idempotencyKey string, dryRun, incremental bool, reason string) (*ReindexJob, error) {
	if dryRun {
		return i.Plan(parent, maxDocuments)
	}
	i.mu.Lock()
	if idempotencyKey != "" {
		if id := i.idempotency[idempotencyKey]; id != "" {
			job := cloneJob(i.jobs[id])
			i.mu.Unlock()
			return job, nil
		}
	}
	if i.recoveryPending {
		i.mu.Unlock()
		return nil, ErrRecoveryPending
	}
	if i.running {
		for _, existing := range i.jobs {
			if existing.State == ReindexQueued || existing.State == ReindexRunning {
				job := cloneJob(existing)
				i.mu.Unlock()
				return job, nil
			}
		}
	}
	now := i.clock().UTC()
	id := newGenerationID(now)
	jobCtx, cancel := context.WithCancel(context.WithoutCancel(parent))
	job := &indexJob{ReindexJob: ReindexJob{ID: id, ShadowGeneration: id, State: ReindexQueued, StartedAt: now, UpdatedAt: now}, cancel: cancel, reason: reason}
	i.jobs[id] = job
	if idempotencyKey != "" {
		i.idempotency[idempotencyKey] = id
	}
	i.running = true
	result := cloneJob(job)
	i.mu.Unlock()
	go i.run(jobCtx, id, maxDocuments, incremental)
	return result, nil
}

func (i *Indexer) RunOnce(ctx context.Context, maxDocuments uint64) (*ReindexJob, error) {
	now := i.clock().UTC()
	id := newGenerationID(now)
	job := &indexJob{ReindexJob: ReindexJob{ID: id, ShadowGeneration: id, State: ReindexQueued, StartedAt: now, UpdatedAt: now}}
	i.mu.Lock()
	i.jobs[id] = job
	i.running = true
	i.mu.Unlock()
	i.run(ctx, id, maxDocuments, false)
	result, _ := i.Status(id)
	if result.State == ReindexFailed {
		return result, errors.New(result.ErrorCode)
	}
	if result.State == ReindexCancelled {
		return result, context.Canceled
	}
	return result, nil
}

func (i *Indexer) run(ctx context.Context, id string, maxDocuments uint64, incremental bool) {
	defer func() {
		i.mu.Lock()
		i.running = false
		i.mu.Unlock()
		checkCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if changes, err := i.repository.PendingChanges(checkCtx, 1); err == nil && len(changes) > 0 {
			i.Kick()
		}
	}()
	i.update(id, func(j *indexJob) { j.State = ReindexRunning })
	var changes []ProjectionChange
	if incremental {
		// A kick or tick can find work that another drain already took. Read the
		// queue before minting a generation so an empty drain leaves no failed
		// row behind.
		var err error
		if changes, err = i.repository.PendingChanges(ctx, maxSourcePageSize); err != nil {
			i.fail(id, err)
			return
		}
		if len(changes) == 0 {
			i.update(id, func(j *indexJob) { j.State, j.UpdatedAt = ReindexComplete, i.clock().UTC() })
			return
		}
	}
	changeWatermark, _ := i.repository.MaxPendingChangeSequence(ctx)
	now := i.clock().UTC()
	gen := Generation{GenerationID: id, State: "building", RecipeVersion: DefaultRecipeVersion, CreatedAt: now, UpdatedAt: now}
	if err := i.repository.SaveGeneration(ctx, gen); err != nil {
		i.fail(id, err)
		return
	}
	if incremental {
		log.Printf("conversation search: generation %s incremental: %d queued change(s)", id, len(changes))
		i.runIncremental(ctx, id, gen, changes)
		return
	}
	log.Printf("conversation search: generation %s FULL rebuild: %s", id, i.jobReason(id))
	if err := i.repository.BeginStagedGeneration(ctx, id); err != nil {
		i.fail(id, err)
		return
	}
	snapshot, err := i.sourceSnapshot(ctx)
	if err != nil {
		i.rollback(id, err, ctx)
		return
	}
	count, digest, checkpoint, err := i.scanSnapshot(ctx, maxDocuments, snapshot, func(document Document) error {
		if err := i.repository.StageDocument(ctx, id, document); err != nil {
			return err
		}
		i.update(id, func(j *indexJob) {
			j.ProcessedDocuments++
			j.UpsertedDocuments++
			j.SourceCheckpoint = checkpointFor(document)
		})
		if current, ok := i.Status(id); ok && current.ProcessedDocuments%uint64(i.pageSize) == 0 {
			progress := gen
			progress.ProcessedDocuments = current.ProcessedDocuments
			progress.SourceCheckpoint = current.SourceCheckpoint
			progress.UpdatedAt = i.clock().UTC()
			if err := i.repository.SaveGeneration(ctx, progress); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		i.rollback(id, err, ctx)
		return
	}
	i.update(id, func(j *indexJob) { j.PlannedDocuments = count; j.SourceCheckpoint = checkpoint })
	gen.PlannedDocuments, gen.ProcessedDocuments, gen.SourceCheckpoint, gen.UpdatedAt = count, count, checkpoint, i.clock().UTC()
	if err := i.repository.SaveGeneration(ctx, gen); err != nil {
		i.rollback(id, err, ctx)
		return
	}
	// Snapshot-capable sources pin append-only event membership. Joined run
	// metadata may legitimately change while a page walk is in progress, and
	// requiring a second byte-identical scan would starve an active corpus.
	// Sources without a snapshot seam retain the conservative two-scan check.
	if snapshot == nil {
		verifyCount, verifyDigest, _, verifyErr := i.scanSnapshot(ctx, maxDocuments, snapshot, nil)
		if verifyErr != nil || verifyCount != count || verifyDigest != digest {
			if verifyErr == nil {
				verifyErr = fmt.Errorf("canonical source mutated during build")
			}
			i.rollback(id, verifyErr, ctx)
			return
		}
	}
	staged, err := i.repository.CountStagedGeneration(ctx, id)
	if err != nil || staged != count {
		if err == nil {
			err = fmt.Errorf("shadow validation mismatch: source=%d staged=%d", count, staged)
		}
		i.rollback(id, err, ctx)
		return
	}
	if err := i.repository.PublishStagedGeneration(ctx, id, count); err != nil {
		i.rollback(id, err, ctx)
		return
	}
	_ = i.repository.MarkChangesProcessed(context.WithoutCancel(ctx), changeWatermark, i.clock().UTC())
	_ = i.repository.SaveCheckpoint(context.WithoutCancel(ctx), Checkpoint{SourceName: "canonical", SourceCursor: checkpoint, SourceFingerprint: digest, UpdatedAt: i.clock().UTC()})
	semanticCtx, cancelSemantic := context.WithTimeout(ctx, semanticRebuildBudget(i.semanticBudgetBasis(ctx, id, count)))
	semanticStarted := i.clock()
	semanticErr := i.rebuildSemantic(semanticCtx, id, changeWatermark)
	cancelSemantic()
	// The shared reconciler also validates, promotes the alias and prunes
	// retired collections inside this call, so its duration is the only
	// evidence of where a slow post-embedding step went.
	log.Printf("conversation search: generation %s semantic leg took %s (failed=%t)", id, i.clock().Sub(semanticStarted).Round(time.Second), semanticErr != nil)
	if semanticErr != nil {
		cause := fmt.Errorf("semantic shadow rebuild: %w", semanticErr)
		// A canonical deletion means the snapshot is no longer safe to serve.
		// Other semantic failures are an optional-leg degradation: the lexical
		// snapshot is complete and must become searchable even when embeddings
		// or the vector store are unavailable.
		if errors.Is(semanticErr, errCanonicalDeletionDuringSemanticBuild) || ctx.Err() != nil {
			i.failPublishedSemantic(id, cause, ctx)
			return
		}
		if !i.activateLexicalAfterSemanticFailure(id, cause, ctx) {
			return
		}
		return
	}
	// Changes committed after the second canonical scan do not invalidate the
	// verified snapshot: they remain above changeWatermark and are reconciled by
	// the incremental queue after promotion. Requiring global quiescence here
	// starves first-time indexing on an actively written conversation store.
	if _, err := i.repository.MaxPendingChangeSequence(ctx); err != nil {
		i.rollback(id, err, ctx)
		return
	}
	if err := ctx.Err(); err != nil {
		i.rollback(id, err, ctx)
		return
	}
	if err := i.repository.ActivateGeneration(ctx, id, i.clock().UTC()); err != nil {
		i.rollback(id, err, ctx)
		return
	}
	_ = i.repository.ClearIndexError(ctx, i.clock().UTC())
	i.pruneSettledGenerations(ctx)
	i.update(id, func(j *indexJob) { j.State = ReindexComplete; j.ActiveGeneration = id; j.UpdatedAt = i.clock().UTC() })
}

func (i *Indexer) runIncremental(ctx context.Context, id string, gen Generation, changes []ProjectionChange) {
	if err := i.repository.BeginStagedGeneration(ctx, id); err != nil {
		i.rollback(id, err, ctx)
		return
	}
	checkpoint := ""
	var semanticDocuments []Document
	var deletedSemanticIDs []string
	for _, change := range changes {
		if err := ctx.Err(); err != nil {
			i.rollback(id, err, ctx)
			return
		}
		switch change.Operation {
		case ChangeUpsertRun:
			// Only previously embedded (prose) documents need a vector delete.
			previous, idsErr := i.repository.SemanticDocumentIDs(ctx, change.SourceRunID, change.SourceEventID)
			if idsErr != nil {
				i.rollback(id, idsErr, ctx)
				return
			}
			deletedSemanticIDs = append(deletedSemanticIDs, previous...)
			documents, loadErr := i.loadChangedDocuments(ctx, change.SourceRunID, change.SourceEventID)
			if loadErr != nil {
				i.rollback(id, loadErr, ctx)
				return
			}
			for _, document := range documents {
				if err := i.repository.StageDocument(ctx, id, document); err != nil {
					i.rollback(id, err, ctx)
					return
				}
				checkpoint = checkpointFor(document)
			}
			semanticDocuments = append(semanticDocuments, documents...)
		case ChangeDeleteEvent:
			previous, idsErr := i.repository.TombstonedDocumentIDs(ctx, change.SourceRunID, change.SourceEventID)
			if idsErr != nil {
				i.rollback(id, idsErr, ctx)
				return
			}
			deletedSemanticIDs = append(deletedSemanticIDs, previous...)
		case ChangeDeleteRun:
			previous, idsErr := i.repository.TombstonedDocumentIDs(ctx, change.SourceRunID, "")
			if idsErr != nil {
				i.rollback(id, idsErr, ctx)
				return
			}
			deletedSemanticIDs = append(deletedSemanticIDs, previous...)
		case ChangeRepair:
			i.rollback(id, errors.New("repair marker requires a full reconcile"), ctx)
			i.Kick()
			return
		}
	}
	if err := i.repository.ApplyStagedChanges(ctx, id, changes); err != nil {
		i.rollback(id, err, ctx)
		return
	}
	_, count, _, err := i.repository.CountCoverage(ctx)
	if err != nil {
		i.rollback(id, err, ctx)
		return
	}
	i.update(id, func(j *indexJob) {
		j.PlannedDocuments, j.ProcessedDocuments, j.UpsertedDocuments, j.SourceCheckpoint = count, count, count, checkpoint
	})
	gen.PlannedDocuments, gen.ProcessedDocuments, gen.SourceCheckpoint, gen.UpdatedAt = count, count, checkpoint, i.clock().UTC()
	if err := i.repository.SaveGeneration(ctx, gen); err != nil {
		i.rollback(id, err, ctx)
		return
	}
	changeWatermark := changes[len(changes)-1].Sequence
	if err := i.rebuildSemanticChanges(ctx, id, changeWatermark, semanticDocuments, deletedSemanticIDs); err != nil {
		// ApplyStagedChanges already committed the lexical projection. An
		// optional semantic leg must not make a newly imported conversation
		// invisible or leave its canonical change permanently pending. Keep
		// lexical search available, record the semantic degradation, and let a
		// later repair rebuild the optional leg. Changes after this watermark
		// (including deletions) remain queued for the next incremental pass.
		if semantic, ok := i.semantic.(SemanticRollback); ok {
			_ = semantic.Rollback(context.WithoutCancel(ctx), id)
		}
		gen.State = "failed"
		gen.FailedDocuments++
		gen.UpdatedAt = i.clock().UTC()
		if saveErr := i.repository.SaveGeneration(context.WithoutCancel(ctx), gen); saveErr != nil {
			i.fail(id, saveErr)
			return
		}
		if markErr := i.repository.MarkChangesProcessed(context.WithoutCancel(ctx), changeWatermark, i.clock().UTC()); markErr != nil {
			i.fail(id, markErr)
			return
		}
		_ = i.repository.ClearTombstones(context.WithoutCancel(ctx), changes)
		logGenerationFailure(id, "semantic leg failed; lexical applied", err)
		i.update(id, func(j *indexJob) {
			j.State = ReindexFailed
			j.FailedDocuments++
			j.ErrorCode = fmt.Sprintf("semantic shadow rebuild: %v", err)
		})
		return
	}
	if err := i.repository.MarkChangesProcessed(context.WithoutCancel(ctx), changeWatermark, i.clock().UTC()); err != nil {
		i.fail(id, err)
		return
	}
	_ = i.repository.ClearTombstones(context.WithoutCancel(ctx), changes)
	if err := i.repository.ActivateGeneration(ctx, id, i.clock().UTC()); err != nil {
		i.rollback(id, err, ctx)
		return
	}
	_ = i.repository.ClearIndexError(ctx, i.clock().UTC())
	i.pruneSettledGenerations(ctx)
	i.update(id, func(j *indexJob) { j.State, j.ActiveGeneration = ReindexComplete, id })
}

func (i *Indexer) rebuildSemanticChanges(ctx context.Context, generationID string, changeWatermark int64, documents []Document, deletedSourceIDs []string) error {
	semantic, ok := i.semantic.(SemanticChangeRebuilder)
	if !ok {
		return i.rebuildSemantic(ctx, generationID, changeWatermark)
	}
	// A document's content class is fixed by its source event, so a changed
	// document that is not semantically indexable was never embedded and its
	// prior version has nothing to delete. Sending those deletes anyway made
	// every tool-payload change copy the whole serving vector collection into a
	// new generation; on 2026-09-14 thousands of compacted tool events stalled
	// the indexer behind Qdrant. Tombstoned deletions are not among the changed
	// documents and always pass through.
	neverEmbedded := make(map[string]struct{})
	indexable := 0
	for _, document := range documents {
		if semanticIndexableContent(document.ContentClass) {
			indexable++
		} else {
			neverEmbedded[document.DocumentID] = struct{}{}
		}
	}
	deletes := make([]string, 0, len(deletedSourceIDs))
	for _, id := range deletedSourceIDs {
		if _, skip := neverEmbedded[id]; !skip {
			deletes = append(deletes, id)
		}
	}
	if indexable == 0 && len(deletes) == 0 {
		return nil
	}
	// Bounded like a full rebuild: an unbounded leg held the single owner loop,
	// the change queue, and fenced compaction behind a slow vector store.
	//
	// There is deliberately no "a deletion arrived after the watermark" gate
	// here, unlike a full rebuild. An incremental candidate starts as a copy of
	// the serving collection and changes only what this pass's own changes say;
	// a deletion queued after the watermark stays pending and the next pass
	// removes it from both legs, so promoting adds no exposure the lexical leg
	// (already applied above) does not have. The gate failed about half of all
	// passes on 2026-11-20 (about 40 runs are deleted per hour, a pass takes one
	// to two minutes), and each failure left that pass's vectors stale until a
	// full rebuild.
	semanticCtx, cancel := context.WithTimeout(ctx, semanticChangesBudget(uint64(indexable+len(deletes))))
	defer cancel()
	return semantic.RebuildChanges(semanticCtx, generationID, documents, deletes, nil)
}

func (i *Indexer) rebuildSemantic(ctx context.Context, generationID string, changeWatermark int64) error {
	if i.semantic == nil {
		return nil
	}
	validate := func(checkCtx context.Context) error {
		deleted, err := i.repository.HasPendingDeletionAfter(checkCtx, changeWatermark)
		if err != nil {
			return err
		}
		if deleted {
			return errCanonicalDeletionDuringSemanticBuild
		}
		return nil
	}
	if semantic, ok := i.semantic.(SemanticValidatedRebuilder); ok {
		return semantic.RebuildValidated(ctx, generationID, validate)
	}
	if err := i.semantic.Rebuild(ctx, generationID); err != nil {
		return err
	}
	return validate(ctx)
}

func (i *Indexer) loadRunDocuments(ctx context.Context, runID string) ([]Document, error) {
	if source, ok := i.source.(RunDocumentSource); ok {
		return source.LoadRunDocuments(ctx, runID)
	}
	var documents []Document
	var cursor *SourceCursor
	for {
		page, err := i.source.LoadSourcePage(ctx, cursor, i.pageSize)
		if err != nil {
			return nil, err
		}
		for _, document := range page.Documents {
			if document.SourceRunID == runID {
				documents = append(documents, document)
			}
		}
		if page.NextCursor == nil {
			break
		}
		cursor = page.NextCursor
	}
	return documents, nil
}

func (i *Indexer) loadChangedDocuments(ctx context.Context, runID, eventID string) ([]Document, error) {
	if eventID != "" {
		if source, ok := i.source.(EventDocumentSource); ok {
			return source.LoadEventDocuments(ctx, runID, eventID)
		}
	}
	return i.loadRunDocuments(ctx, runID)
}

func (i *Indexer) scan(ctx context.Context, maxDocuments uint64, visit func(Document) error) (uint64, string, string, error) {
	snapshot, err := i.sourceSnapshot(ctx)
	if err != nil {
		return 0, "", "", err
	}
	return i.scanSnapshot(ctx, maxDocuments, snapshot, visit)
}

func (i *Indexer) sourceSnapshot(ctx context.Context) (*SourceCursor, error) {
	if source, ok := i.source.(SnapshotSource); ok {
		return source.SnapshotCursor(ctx)
	}
	return nil, nil
}

func (i *Indexer) scanSnapshot(ctx context.Context, maxDocuments uint64, snapshot *SourceCursor, visit func(Document) error) (uint64, string, string, error) {
	hash := sha256.New()
	var count uint64
	cursor := snapshot
	checkpoint := ""
	for {
		page, err := i.source.LoadSourcePage(ctx, cursor, i.pageSize)
		if err != nil {
			return count, "", checkpoint, err
		}
		for _, document := range page.Documents {
			if maxDocuments > 0 && count >= maxDocuments {
				return count, "", checkpoint, fmt.Errorf("canonical corpus exceeds max_documents=%d", maxDocuments)
			}
			count++
			_, _ = hash.Write([]byte(document.DocumentID))
			_, _ = hash.Write([]byte{0})
			_, _ = hash.Write([]byte(document.ContentHash))
			checkpoint = checkpointFor(document)
			if visit != nil {
				if err := visit(document); err != nil {
					return count, "", checkpoint, err
				}
			}
		}
		if page.NextCursor == nil {
			break
		}
		cursor = page.NextCursor
	}
	return count, hex.EncodeToString(hash.Sum(nil)), checkpoint, nil
}

func (i *Indexer) rollback(id string, cause error, ctx context.Context) {
	state := "failed"
	jobState := ReindexFailed
	if errors.Is(cause, context.Canceled) {
		state, jobState = "cancelled", ReindexCancelled
	}
	cleanupCtx, cancelCleanup := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	cleanupErr := i.repository.RollbackStagedGeneration(cleanupCtx, id, state, i.clock().UTC())
	cancelCleanup()
	if cleanupErr != nil {
		cause = errors.Join(cause, fmt.Errorf("generation cleanup incomplete; retained for recovery: %w", cleanupErr))
	}
	_ = i.repository.SaveCheckpoint(context.WithoutCancel(ctx), Checkpoint{SourceName: "canonical", UpdatedAt: i.clock().UTC(), LastErrorCode: cause.Error()})
	logGenerationFailure(id, state, cause)
	i.update(id, func(j *indexJob) { j.State = jobState; j.FailedDocuments++; j.ErrorCode = cause.Error() })
}

func (i *Indexer) failPublishedSemantic(id string, cause error, ctx context.Context) {
	if semantic, ok := i.semantic.(SemanticRollback); ok {
		_ = semantic.Rollback(context.WithoutCancel(ctx), id)
	}
	if generation, err := i.repository.LoadGeneration(context.WithoutCancel(ctx), id); err == nil {
		generation.State = "ready"
		generation.FailedDocuments++
		generation.UpdatedAt = i.clock().UTC()
		_ = i.repository.SaveGeneration(context.WithoutCancel(ctx), generation)
	}
	_ = i.repository.SaveCheckpoint(context.WithoutCancel(ctx), Checkpoint{SourceName: "canonical", UpdatedAt: i.clock().UTC(), LastErrorCode: cause.Error()})
	logGenerationFailure(id, "published generation left unactivated", cause)
	i.update(id, func(job *indexJob) {
		job.State = ReindexFailed
		job.FailedDocuments++
		job.ErrorCode = cause.Error()
	})
}

// activateLexicalAfterSemanticFailure keeps the validated lexical projection
// available when the optional semantic leg cannot finish. The generation is
// intentionally marked degraded so operators can distinguish this from a
// fully hybrid-ready promotion.
func (i *Indexer) activateLexicalAfterSemanticFailure(id string, cause error, ctx context.Context) bool {
	if semantic, ok := i.semantic.(SemanticRollback); ok {
		_ = semantic.Rollback(context.WithoutCancel(ctx), id)
	}
	if err := i.repository.ActivateGeneration(context.WithoutCancel(ctx), id, i.clock().UTC()); err != nil {
		i.failPublishedSemantic(id, fmt.Errorf("activate lexical snapshot after semantic failure: %w", err), ctx)
		return false
	}
	i.pruneSettledGenerations(ctx)
	if generation, err := i.repository.LoadGeneration(context.WithoutCancel(ctx), id); err == nil {
		generation.State = "active"
		generation.FailedDocuments++
		generation.UpdatedAt = i.clock().UTC()
		if saveErr := i.repository.SaveGeneration(context.WithoutCancel(ctx), generation); saveErr != nil {
			i.fail(id, saveErr)
			return false
		}
	}
	_ = i.repository.SaveCheckpoint(context.WithoutCancel(ctx), Checkpoint{SourceName: "canonical", UpdatedAt: i.clock().UTC(), LastErrorCode: cause.Error()})
	logGenerationFailure(id, "lexical activated without vectors", cause)
	i.update(id, func(job *indexJob) {
		job.State = ReindexFailed
		job.FailedDocuments++
		job.ActiveGeneration = id
		job.ErrorCode = cause.Error()
	})
	return true
}

// logGenerationFailure leaves the cause of a failed generation in the process
// log. The job record is in memory and the catalog row carries no reason, so
// without this a restart erased why a generation failed.
func logGenerationFailure(id, stage string, cause error) {
	log.Printf("conversation search: generation %s failed (%s): %v", id, stage, cause)
}

func (i *Indexer) fail(id string, err error) {
	logGenerationFailure(id, "indexer error", err)
	i.update(id, func(j *indexJob) { j.State = ReindexFailed; j.ErrorCode = err.Error(); j.FailedDocuments++ })
}

func (i *Indexer) update(id string, change func(*indexJob)) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if job := i.jobs[id]; job != nil {
		change(job)
		job.UpdatedAt = i.clock().UTC()
	}
}

func (i *Indexer) jobReason(id string) string {
	i.mu.Lock()
	defer i.mu.Unlock()
	if job := i.jobs[id]; job != nil && job.reason != "" {
		return job.reason
	}
	return "unattributed"
}

func (i *Indexer) Status(id string) (*ReindexJob, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	job := i.jobs[id]
	return cloneJob(job), job != nil
}

func (i *Indexer) Cancel(id string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	job := i.jobs[id]
	if job == nil || job.cancel == nil || (job.State != ReindexQueued && job.State != ReindexRunning) {
		return false
	}
	job.cancel()
	return true
}

func cloneJob(job *indexJob) *ReindexJob {
	if job == nil {
		return nil
	}
	out := job.ReindexJob
	return &out
}

func checkpointFor(d Document) string {
	return fmt.Sprintf("%s|%s|%d", d.OccurredAt.UTC().Format(time.RFC3339Nano), d.SourceRunID, d.EventSequence)
}

var generationSequence atomic.Uint64

func newGenerationID(now time.Time) string {
	return fmt.Sprintf("conversation-%d-%d", now.UnixNano(), generationSequence.Add(1))
}

// SortedJobs exposes bounded deterministic evidence for status/debugging.
func (i *Indexer) SortedJobs() []ReindexJob {
	i.mu.Lock()
	defer i.mu.Unlock()
	out := make([]ReindexJob, 0, len(i.jobs))
	for _, job := range i.jobs {
		out = append(out, job.ReindexJob)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].StartedAt.Before(out[b].StartedAt) })
	return out
}

func (i *Indexer) StatusSnapshot(ctx context.Context) (ProjectionStatus, error) {
	// Never block an operator status request on the single SQLite writer.
	// Serve the last successful projection and refresh it once in the
	// background. The first request after a restart returns a warming note
	// instead of hanging; the next returns real data.
	i.statusCacheMu.Lock()
	if i.statusCacheValid && time.Since(i.statusCacheAt) < statusSnapshotCacheTTL {
		cached := i.statusCache
		i.statusCacheMu.Unlock()
		return cached, nil
	}
	if !i.statusRefreshing {
		i.statusRefreshing = true
		go i.refreshStatusSnapshot()
	}
	cached, ok := i.statusCache, i.statusCacheValid
	i.statusCacheMu.Unlock()
	if ok {
		return cached, nil
	}
	warming := ProjectionStatus{LastErrorCode: "status warming up; retry in a moment"}
	i.mu.Lock()
	if i.recoveryPending {
		reason := i.recoveryError
		if reason == "" {
			reason = ErrRecoveryPending.Error()
		}
		warming.LastErrorCode = reason
		warming.DegradedDependencies = append(warming.DegradedDependencies, "canonical-projection: "+reason)
	}
	i.mu.Unlock()
	return warming, nil
}

func (i *Indexer) refreshStatusSnapshot() {
	ctx, cancel := context.WithTimeout(context.Background(), statusSnapshotRefreshTimeout)
	defer cancel()
	status, err := i.readStatusSnapshot(ctx)
	i.statusCacheMu.Lock()
	if err == nil {
		i.statusCache, i.statusCacheAt, i.statusCacheValid = status, time.Now(), true
	}
	i.statusRefreshing = false
	i.statusCacheMu.Unlock()
}

func (i *Indexer) readStatusSnapshot(ctx context.Context) (ProjectionStatus, error) {
	status, err := i.repository.ProjectionStatus(ctx)
	if err != nil {
		return ProjectionStatus{}, err
	}
	i.mu.Lock()
	pending, recoveryError := i.recoveryPending, i.recoveryError
	i.mu.Unlock()
	if pending {
		if recoveryError == "" {
			recoveryError = ErrRecoveryPending.Error()
		}
		status.LastErrorCode = recoveryError
		status.DegradedDependencies = append(status.DegradedDependencies, "canonical-projection: "+recoveryError)
	}
	// Counts describe the last reconciled authoritative snapshot. Full source
	// comparison and orphan repair belong to the background index lifecycle;
	// repeating that O(corpus) work in a status request made observability block
	// behind large imports and the single SQLite writer connection.
	if status.PendingChanges > 0 {
		status.DegradedDependencies = append(status.DegradedDependencies, "canonical-projection: pending source changes; counts reflect the last reconciled snapshot")
	}
	if semantic, ok := i.semantic.(SemanticStatusReporter); ok {
		// This runs in the background refresh, never on a caller's request, so
		// the first vector count may take its full probe budget instead of the
		// short retrieval budget. Abandoning it early is what reported a serving
		// index as 0%.
		semanticCtx, cancel := context.WithTimeout(ctx, semanticStatusLegTimeout)
		count, collection, layout, model, semanticErr := semantic.SemanticStatus(semanticCtx)
		cancel()
		status.CollectionName, status.CollectionLayout, status.EmbeddingModel = collection, layout, model
		switch {
		case errors.Is(semanticErr, ErrSemanticCoverageUnknown):
			// No count was ever observed: say so instead of reporting zero.
			status.SemanticCoverageUnknown = true
			status.DegradedDependencies = append(status.DegradedDependencies, SemanticCoverageUnknownPrefix+semanticErr.Error())
		case semanticErr != nil:
			// A failed later probe keeps the last observed count beside the
			// explicit degradation; a genuinely empty leg reads as 0 with it.
			status.SemanticDocuments = count
			status.DegradedDependencies = append(status.DegradedDependencies, "semantic: "+semanticErr.Error())
		default:
			status.SemanticDocuments = count
		}
		if eligible, eligibleErr := i.repository.CountSemanticEligible(ctx); eligibleErr == nil {
			status.SemanticEligibleDocuments = eligible
		}
	}
	// Report generation health explicitly. A recovered or unverified catalog must
	// never present as healthy just because the lexical leg is complete.
	if reporter, ok := i.semantic.(SemanticGenerationHealthReporter); ok {
		if line := reporter.GenerationHealthSnapshot().Degradation(); line != "" {
			status.DegradedDependencies = append(status.DegradedDependencies, line)
		}
	}
	return status, nil
}

func (i *Indexer) pruneSettledGenerations(ctx context.Context) {
	if i == nil || i.repository == nil || !i.pruning.CompareAndSwap(false, true) {
		return
	}
	// Detached: a reap of stranded staged rows competes with live readers for
	// the single SQLite writer, and startup/status must not wait for it. The
	// guard keeps concurrent call sites from starting overlapping reaps.
	go func() {
		defer i.pruning.Store(false)
		pruneCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Minute)
		defer cancel()
		_, _ = i.repository.PruneSettledGenerations(pruneCtx)
		// Candidate bookkeeping settles here too: activation is the moment a
		// generation demonstrably finished, so a long-lived thread that only
		// runs on rebuilds keeps the status path free of maintenance writes.
		_, _ = i.repository.SettleStaleQdrantCandidateRecords(pruneCtx, i.clock().UTC())
	}()
}
