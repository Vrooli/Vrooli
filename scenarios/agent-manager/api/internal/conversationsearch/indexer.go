package conversationsearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultIndexPageSize  = 250
	defaultRepairInterval = 15 * time.Minute
	initialRepairDelay    = 30 * time.Second
)

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
	RepairInterval time.Duration
	PageSize       int
	Clock          func() time.Time
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
	source     SourceRepository
	repository *SQLiteRepository
	semantic   SemanticRebuilder
	interval   time.Duration
	pageSize   int
	clock      func() time.Time

	kick        chan struct{}
	stop        context.CancelFunc
	mu          sync.Mutex
	running     bool
	jobs        map[string]*indexJob
	idempotency map[string]string
}

type indexJob struct {
	ReindexJob
	cancel context.CancelFunc
}

func NewIndexer(options IndexerOptions) (*Indexer, error) {
	if options.Source == nil || options.Repository == nil {
		return nil, errors.New("conversation indexer requires source and repository")
	}
	if options.RepairInterval <= 0 {
		options.RepairInterval = defaultRepairInterval
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
		source: options.Source, repository: options.Repository, semantic: options.Semantic, interval: options.RepairInterval,
		pageSize: options.PageSize, clock: options.Clock, kick: make(chan struct{}, 1), jobs: map[string]*indexJob{}, idempotency: map[string]string{},
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
	i.mu.Unlock()
	go func() {
		timer := time.NewTimer(initialRepairDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		go i.loop(ctx)
		i.launchInitial(ctx)
	}()
}

func (i *Indexer) launchInitial(ctx context.Context) {
	if i.recoverInterrupted(ctx) {
		return
	}
	status, err := i.repository.ProjectionStatus(ctx)
	if err == nil && status.ActiveGeneration != "" {
		i.launchChanges(ctx)
		return
	}
	i.launchRepair(ctx)
}

func (i *Indexer) recoverInterrupted(ctx context.Context) bool {
	generations, err := i.repository.BuildingGenerations(ctx)
	if err != nil {
		return false
	}
	resumeIndex := -1
	var resumeCount uint64
	for index, generation := range generations {
		staged, countErr := i.repository.CountStagedGeneration(ctx, generation.GenerationID)
		completeSnapshot := countErr == nil && generation.RecipeVersion == DefaultRecipeVersion && generation.PlannedDocuments > 0 && generation.ProcessedDocuments == generation.PlannedDocuments && staged == generation.PlannedDocuments
		if completeSnapshot && staged > resumeCount {
			resumeIndex, resumeCount = index, staged
		}
	}
	for index, generation := range generations {
		if index == resumeIndex {
			staged := resumeCount
			jobCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
			job := &indexJob{ReindexJob: ReindexJob{
				ID: generation.GenerationID, State: ReindexQueued, PlannedDocuments: staged,
				ProcessedDocuments: staged, UpsertedDocuments: staged, SourceCheckpoint: generation.SourceCheckpoint,
				ShadowGeneration: generation.GenerationID, StartedAt: generation.CreatedAt, UpdatedAt: i.clock().UTC(),
			}, cancel: cancel}
			i.mu.Lock()
			i.jobs[generation.GenerationID] = job
			i.running = true
			i.mu.Unlock()
			go i.resume(jobCtx, generation, staged)
			continue
		}
		if semantic, ok := i.semantic.(SemanticRollback); ok {
			_ = semantic.Rollback(ctx, generation.GenerationID)
		}
		_ = i.repository.RollbackStagedGeneration(ctx, generation.GenerationID, "failed", i.clock().UTC())
	}
	return resumeIndex >= 0
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
	if err := i.rebuildSemantic(ctx, id, 0); err != nil {
		i.failPublishedSemantic(id, fmt.Errorf("resume semantic shadow rebuild: %w", err), ctx)
		return
	}
	if err := i.repository.ActivateGeneration(ctx, id, i.clock().UTC()); err != nil {
		i.rollback(id, err, ctx)
		return
	}
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
	ticker := time.NewTicker(i.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			i.launchRepair(ctx)
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

func (i *Indexer) launchRepair(ctx context.Context) {
	_, _ = i.Reindex(ctx, 0, "", false)
}

func (i *Indexer) launchChanges(ctx context.Context) {
	changes, err := i.repository.PendingChanges(ctx, 1)
	if err != nil || len(changes) == 0 {
		return
	}
	if changes[0].Operation == ChangeRepair {
		i.launchRepair(ctx)
		return
	}
	_, _ = i.reindex(ctx, 0, "", false, true)
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
	return i.reindex(parent, maxDocuments, idempotencyKey, false, false)
}

func (i *Indexer) reindex(parent context.Context, maxDocuments uint64, idempotencyKey string, dryRun, incremental bool) (*ReindexJob, error) {
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
	job := &indexJob{ReindexJob: ReindexJob{ID: id, ShadowGeneration: id, State: ReindexQueued, StartedAt: now, UpdatedAt: now}, cancel: cancel}
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
	changeWatermark, _ := i.repository.MaxPendingChangeSequence(ctx)
	now := i.clock().UTC()
	gen := Generation{GenerationID: id, State: "building", RecipeVersion: DefaultRecipeVersion, CreatedAt: now, UpdatedAt: now}
	if err := i.repository.SaveGeneration(ctx, gen); err != nil {
		i.fail(id, err)
		return
	}
	if incremental {
		i.runIncremental(ctx, id, gen)
		return
	}
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
	if err := i.rebuildSemantic(ctx, id, changeWatermark); err != nil {
		i.failPublishedSemantic(id, fmt.Errorf("semantic shadow rebuild: %w", err), ctx)
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
	i.update(id, func(j *indexJob) { j.State = ReindexComplete; j.ActiveGeneration = id; j.UpdatedAt = i.clock().UTC() })
}

func (i *Indexer) runIncremental(ctx context.Context, id string, gen Generation) {
	changes, err := i.repository.PendingChanges(ctx, maxSourcePageSize)
	if err != nil {
		i.rollback(id, err, ctx)
		return
	}
	if len(changes) == 0 {
		i.rollback(id, errors.New("incremental reconcile had no pending changes"), ctx)
		return
	}
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
			previous, idsErr := i.repository.RunDocumentIDs(ctx, change.SourceRunID)
			if idsErr != nil {
				i.rollback(id, idsErr, ctx)
				return
			}
			deletedSemanticIDs = append(deletedSemanticIDs, previous...)
			documents, loadErr := i.loadRunDocuments(ctx, change.SourceRunID)
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
		i.rollback(id, fmt.Errorf("semantic shadow rebuild: %w", err), ctx)
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
	i.update(id, func(j *indexJob) { j.State, j.ActiveGeneration = ReindexComplete, id })
}

func (i *Indexer) rebuildSemanticChanges(ctx context.Context, generationID string, changeWatermark int64, documents []Document, deletedSourceIDs []string) error {
	semantic, ok := i.semantic.(SemanticChangeRebuilder)
	if !ok {
		return i.rebuildSemantic(ctx, generationID, changeWatermark)
	}
	validate := func(checkCtx context.Context) error {
		deleted, err := i.repository.HasPendingDeletionAfter(checkCtx, changeWatermark)
		if err != nil {
			return err
		}
		if deleted {
			return errors.New("canonical deletion arrived during semantic build")
		}
		return nil
	}
	return semantic.RebuildChanges(ctx, generationID, documents, deletedSourceIDs, validate)
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
			return errors.New("canonical deletion arrived during semantic build")
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
	_ = i.repository.RollbackStagedGeneration(context.WithoutCancel(ctx), id, state, i.clock().UTC())
	_ = i.repository.SaveCheckpoint(context.WithoutCancel(ctx), Checkpoint{SourceName: "canonical", UpdatedAt: i.clock().UTC(), LastErrorCode: cause.Error()})
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
	i.update(id, func(job *indexJob) {
		job.State = ReindexFailed
		job.FailedDocuments++
		job.ErrorCode = cause.Error()
	})
}

func (i *Indexer) fail(id string, err error) {
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
	status, err := i.repository.ProjectionStatus(ctx)
	if err != nil {
		return ProjectionStatus{}, err
	}
	// Counts describe the last reconciled authoritative snapshot. Full source
	// comparison and orphan repair belong to the background index lifecycle;
	// repeating that O(corpus) work in a status request made observability block
	// behind large imports and the single SQLite writer connection.
	if status.PendingChanges > 0 {
		status.DegradedDependencies = append(status.DegradedDependencies, "canonical-projection: pending source changes; counts reflect the last reconciled snapshot")
	}
	if semantic, ok := i.semantic.(SemanticStatusReporter); ok {
		semanticCtx, cancel := context.WithTimeout(ctx, semanticLegTimeout)
		count, collection, layout, model, semanticErr := semantic.SemanticStatus(semanticCtx)
		cancel()
		status.SemanticDocuments, status.CollectionName, status.CollectionLayout, status.EmbeddingModel = count, collection, layout, model
		if semanticErr != nil {
			status.DegradedDependencies = append(status.DegradedDependencies, "semantic: "+semanticErr.Error())
		}
	}
	return status, nil
}
