package aisearch

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GenerationMetadata identifies one immutable candidate index generation.
// SourceDigest, Model, and ChunkPolicy make freshness decisions inspectable.
type GenerationMetadata struct {
	ID              string    `json:"id"`
	CreatedAt       time.Time `json:"createdAt"`
	SourceDigest    string    `json:"sourceDigest,omitempty"`
	Model           string    `json:"model,omitempty"`
	ChunkPolicy     string    `json:"chunkPolicy,omitempty"`
	Owner           string    `json:"owner,omitempty"`
	Namespace       string    `json:"namespace,omitempty"`
	Alias           string    `json:"alias,omitempty"`
	ContentIdentity string    `json:"contentIdentity,omitempty"`
	LeaseID         string    `json:"leaseId,omitempty"`
	LeaseExpiresAt  time.Time `json:"leaseExpiresAt,omitempty"`
	Full            bool      `json:"full"`
}

// StoredSourceState is the bounded projection needed to reconcile one source.
// Points contains drift metadata only; vector values never enter the planner.
type StoredSourceState struct {
	SourceHash  string
	Model       string
	ChunkPolicy string
	Points      map[string]ScrollItem
}

// GenerationSourceWrite replaces one source inside a candidate generation.
// ChangedPoints carry new vectors. ReusePointIDs copy unchanged points from the
// active generation without embedding them again.
type GenerationSourceWrite struct {
	SourceID      string
	SourceHash    string
	ChangedPoints []Point
	ReusePointIDs []string
}

// GenerationValidation is returned by the store before alias promotion.
type GenerationValidation struct {
	SourceCount int
	PointCount  int
	Valid       bool
	Detail      string
}

// GenerationStore owns shadow generation lifecycle. A failed or canceled run
// rolls back its candidate and leaves the active generation untouched.
// BeginGeneration starts an empty candidate when metadata.Full is true and a
// copy-on-write view of the active generation when false. StageSource replaces
// the complete source projection, so omitted point IDs are deleted as ghosts.
type GenerationStore interface {
	BeginGeneration(ctx context.Context, metadata GenerationMetadata) error
	LookupActiveSources(ctx context.Context, sourceIDs []string) (map[string]StoredSourceState, error)
	StageSource(ctx context.Context, generationID string, write GenerationSourceWrite) error
	StageDelete(ctx context.Context, generationID, sourceID string) error
	ValidateGeneration(ctx context.Context, generationID string) (GenerationValidation, error)
	PromoteGeneration(ctx context.Context, generationID string) error
	RollbackGeneration(ctx context.Context, generationID string) error
	CleanupGenerations(ctx context.Context, keep int) error
}

// GenerationSourceLookup is an optional resume seam for stores that can read
// source state from an interrupted candidate generation. RunFull uses it to
// avoid re-embedding sources already durably staged before a process restart.
// Stores that do not implement this interface retain the existing behavior.
type GenerationSourceLookup interface {
	LookupGenerationSources(ctx context.Context, generationID string, sourceIDs []string) (map[string]StoredSourceState, error)
}

// GenerationBatchStore is an optional bounded-page write seam. It preserves
// StageSource replacement semantics while allowing remote stores to collapse
// per-source network round trips into one filtered delete and one batch upsert.
type GenerationBatchStore interface {
	StageSources(ctx context.Context, generationID string, writes []GenerationSourceWrite) error
}

// StreamingBinding is the bounded large-corpus counterpart to SourceBinding.
type StreamingBinding struct {
	Kind        string
	Store       GenerationStore
	Source      PagedSource
	Chunker     Chunker
	Composer    EmbeddingTextComposer
	Sparse      SparseEncoder
	IDPrefix    string
	PageSize    int
	Admission   Admission
	EmbedWeight int64
	// BeforePromote lets an adopter re-check source-specific safety invariants
	// (for example, privacy deletions) after validation but before alias cutover.
	BeforePromote func(context.Context) error
	// EmbedConcurrency bounds independent source/chunk embeddings.
	// Zero preserves the historical serial behavior. Admission remains the
	// cross-source/process capacity authority; this only lets a caller use the
	// capacity it has already been granted.
	EmbedConcurrency int
}

func (b StreamingBinding) pageSize() int {
	size := b.PageSize
	if size <= 0 {
		size = DefaultSourcePageSize
	}
	if size > MaxSourcePageSize {
		size = MaxSourcePageSize
	}
	return size
}

func (b StreamingBinding) chunker() Chunker {
	if b.Chunker != nil {
		return b.Chunker
	}
	return NewIdentityChunker()
}

func (b StreamingBinding) composer() EmbeddingTextComposer {
	if b.Composer != nil {
		return b.Composer
	}
	return NewIdentityComposer()
}

// StreamingResult reports bounded-run evidence without retaining per-source
// plans in memory.
type StreamingResult struct {
	Generation       GenerationMetadata   `json:"generation"`
	Validation       GenerationValidation `json:"validation"`
	Pages            int                  `json:"pages"`
	Sources          int                  `json:"sources"`
	Embedded         int                  `json:"embedded"`
	Reused           int                  `json:"reused"`
	Deleted          int                  `json:"deleted"`
	MaxPageDocuments int                  `json:"maxPageDocuments"`
	Promoted         bool                 `json:"promoted"`
	// CleanupError records a failure to delete retired generations after this
	// generation was promoted. Promotion already succeeded and the alias serves
	// the new generation, so the run is reported as promoted and the failure is
	// surfaced here instead of failing a run whose index is live.
	CleanupError string `json:"cleanupError,omitempty"`
}

// StreamingReconciler builds a shadow generation one bounded source page at a
// time, validates it, then atomically promotes it. The previous generation
// remains the serving generation until promotion succeeds.
type StreamingReconciler struct {
	Embedder Embedder
	Clock    func() time.Time
}

func NewStreamingReconciler(embedder Embedder) *StreamingReconciler {
	return &StreamingReconciler{Embedder: embedder, Clock: time.Now}
}

// RunFull reconciles the complete paged source into a shadow generation.
func (r *StreamingReconciler) RunFull(ctx context.Context, binding StreamingBinding, metadata GenerationMetadata) (*StreamingResult, error) {
	if err := validateStreamingBinding(binding, r.Embedder); err != nil {
		return nil, err
	}
	metadata.Full = true
	if metadata.ID == "" {
		return nil, fmt.Errorf("streaming reconcile: generation id is required")
	}
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = r.now()
	}
	if err := binding.Store.BeginGeneration(ctx, metadata); err != nil {
		return nil, fmt.Errorf("begin generation %q: %w", metadata.ID, err)
	}
	promoted := false
	defer func() {
		if !promoted {
			_ = binding.Store.RollbackGeneration(context.WithoutCancel(ctx), metadata.ID)
		}
	}()

	result := &StreamingResult{Generation: metadata}
	cursor := ""
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		page, err := binding.Source.LoadPage(ctx, PageRequest{Cursor: cursor, Limit: binding.pageSize()})
		if err != nil {
			return result, fmt.Errorf("load source page %q: %w", cursor, err)
		}
		if len(page.Documents) > binding.pageSize() {
			return result, fmt.Errorf("source page returned %d documents, limit is %d", len(page.Documents), binding.pageSize())
		}
		if !page.Done && (page.NextCursor == "" || page.NextCursor == cursor) {
			return result, fmt.Errorf("source page cursor did not advance from %q", cursor)
		}
		if len(page.Documents) == 0 && !page.Done {
			return result, fmt.Errorf("source page %q is empty but not done", cursor)
		}
		if len(page.Documents) > result.MaxPageDocuments {
			result.MaxPageDocuments = len(page.Documents)
		}
		result.Pages++
		ids, err := pageSourceIDs(page.Documents)
		if err != nil {
			return result, err
		}
		remaining := page.Documents
		if resumable, ok := binding.Store.(GenerationSourceLookup); ok {
			candidate, lookupErr := resumable.LookupGenerationSources(ctx, metadata.ID, ids)
			if lookupErr != nil {
				return result, fmt.Errorf("lookup candidate sources: %w", lookupErr)
			}
			remaining = make([]SourceDoc, 0, len(page.Documents))
			for _, document := range page.Documents {
				state := candidate[document.ID]
				// Candidate identity is generation-scoped. BeginGeneration has
				// already verified the collection's model/layout, and the caller
				// rejects recipe-version drift before resuming the same ID.
				state.Model = metadata.Model
				state.ChunkPolicy = metadata.ChunkPolicy
				if sourceStateComplete(state, document.ContentHash, metadata) {
					result.Reused += len(state.Points)
					continue
				}
				remaining = append(remaining, document)
			}
		}
		remainingIDs, err := pageSourceIDs(remaining)
		if err != nil {
			return result, err
		}
		stored, err := binding.Store.LookupActiveSources(ctx, remainingIDs)
		if err != nil {
			return result, fmt.Errorf("lookup active sources: %w", err)
		}
		if len(stored) > len(remainingIDs) {
			return result, fmt.Errorf("lookup returned %d sources for a %d-source page", len(stored), len(remainingIDs))
		}
		embedded, reused, err := r.stageDocuments(ctx, binding, metadata, remaining, stored)
		if err != nil {
			return result, err
		}
		result.Sources += len(page.Documents)
		result.Embedded += embedded
		result.Reused += reused
		if page.Done {
			break
		}
		cursor = page.NextCursor
	}

	validation, err := binding.Store.ValidateGeneration(ctx, metadata.ID)
	if err != nil {
		return result, fmt.Errorf("validate generation %q: %w", metadata.ID, err)
	}
	result.Validation = validation
	if !validation.Valid {
		return result, fmt.Errorf("generation %q is invalid: %s", metadata.ID, validation.Detail)
	}
	if binding.BeforePromote != nil {
		if err := binding.BeforePromote(ctx); err != nil {
			return result, fmt.Errorf("generation %q pre-promotion check: %w", metadata.ID, err)
		}
	}
	if err := binding.Store.PromoteGeneration(ctx, metadata.ID); err != nil {
		return result, fmt.Errorf("promote generation %q: %w", metadata.ID, err)
	}
	promoted = true
	result.Promoted = true
	if err := binding.Store.CleanupGenerations(ctx, retainedGenerations); err != nil {
		result.CleanupError = err.Error()
	}
	return result, nil
}

// retainedGenerations is how many retired generations a store keeps after a
// promotion: the previous serving generation for an operator rollback, plus
// one more so a rollback target is never the generation being retired.
const retainedGenerations = 2

// RunChanges applies one explicit change set through the same shadow
// generation lifecycle. The change set is staged one bounded page at a time
// inside a single candidate generation, so an incremental run costs one
// generation regardless of how many changes it carries. Deletions are
// source-level and cannot leave ghost chunks behind.
func (r *StreamingReconciler) RunChanges(ctx context.Context, binding StreamingBinding, metadata GenerationMetadata, changes ChangeSet) (*StreamingResult, error) {
	if err := validateStreamingBinding(binding, r.Embedder); err != nil {
		return nil, err
	}
	metadata.Full = false
	if metadata.ID == "" {
		return nil, fmt.Errorf("streaming reconcile: generation id is required")
	}
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = r.now()
	}
	if err := binding.Store.BeginGeneration(ctx, metadata); err != nil {
		return nil, fmt.Errorf("begin generation %q: %w", metadata.ID, err)
	}
	promoted := false
	defer func() {
		if !promoted {
			_ = binding.Store.RollbackGeneration(context.WithoutCancel(ctx), metadata.ID)
		}
	}()

	result := &StreamingResult{Generation: metadata}
	pageSize := binding.pageSize()
	for offset := 0; offset < len(changes.Changes) || offset == 0; offset += pageSize {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		end := min(offset+pageSize, len(changes.Changes))
		page := changes.Changes[offset:end]
		result.Pages++
		if len(page) > result.MaxPageDocuments {
			result.MaxPageDocuments = len(page)
		}
		var upserts []SourceDoc
		for _, change := range page {
			switch change.Operation {
			case ChangeUpsert:
				upserts = append(upserts, change.Document)
			case ChangeDelete:
				if strings.TrimSpace(change.SourceID) == "" {
					return result, fmt.Errorf("delete change requires source id")
				}
				if err := binding.Store.StageDelete(ctx, metadata.ID, change.SourceID); err != nil {
					return result, fmt.Errorf("stage delete %q: %w", change.SourceID, err)
				}
				result.Deleted++
			default:
				return result, fmt.Errorf("unsupported change operation %q", change.Operation)
			}
		}
		ids, err := pageSourceIDs(upserts)
		if err != nil {
			return result, err
		}
		stored, err := binding.Store.LookupActiveSources(ctx, ids)
		if err != nil {
			return result, fmt.Errorf("lookup active sources: %w", err)
		}
		embedded, reused, err := r.stageDocuments(ctx, binding, metadata, upserts, stored)
		if err != nil {
			return result, err
		}
		result.Sources += len(upserts)
		result.Embedded += embedded
		result.Reused += reused
		if end >= len(changes.Changes) {
			break
		}
	}
	validation, err := binding.Store.ValidateGeneration(ctx, metadata.ID)
	if err != nil {
		return result, fmt.Errorf("validate generation %q: %w", metadata.ID, err)
	}
	result.Validation = validation
	if !validation.Valid {
		return result, fmt.Errorf("generation %q is invalid: %s", metadata.ID, validation.Detail)
	}
	if binding.BeforePromote != nil {
		if err := binding.BeforePromote(ctx); err != nil {
			return result, fmt.Errorf("generation %q pre-promotion check: %w", metadata.ID, err)
		}
	}
	if err := binding.Store.PromoteGeneration(ctx, metadata.ID); err != nil {
		return result, fmt.Errorf("promote generation %q: %w", metadata.ID, err)
	}
	promoted = true
	result.Promoted = true
	if err := binding.Store.CleanupGenerations(ctx, retainedGenerations); err != nil {
		result.CleanupError = err.Error()
	}
	return result, nil
}

func (r *StreamingReconciler) stageDocument(ctx context.Context, binding StreamingBinding, metadata GenerationMetadata, doc SourceDoc, stored StoredSourceState) (int, int, error) {
	prepared, err := r.prepareDocument(ctx, binding, metadata, doc, stored)
	if err != nil {
		return 0, 0, err
	}
	if err := binding.Store.StageSource(ctx, metadata.ID, prepared.write); err != nil {
		return 0, 0, fmt.Errorf("stage source %q: %w", doc.ID, err)
	}
	return prepared.embedded, prepared.reused, nil
}

type preparedSource struct {
	write    GenerationSourceWrite
	embedded int
	reused   int
}

// stageDocuments embeds independent source documents concurrently, then
// commits their writes to the generation store in source order. This keeps a
// store's mutation contract serial while allowing the common one-document /
// one-chunk corpus shape to use the declared embedding concurrency.
func (r *StreamingReconciler) stageDocuments(ctx context.Context, binding StreamingBinding, metadata GenerationMetadata, docs []SourceDoc, stored map[string]StoredSourceState) (int, int, error) {
	if len(docs) == 0 {
		return 0, 0, nil
	}
	concurrency := binding.EmbedConcurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	if concurrency > len(docs) {
		concurrency = len(docs)
	}
	workerBinding := binding
	workerBinding.EmbedConcurrency = 1
	prepared := make([]preparedSource, len(docs))
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan int)
	var workers sync.WaitGroup
	var firstErr error
	var errOnce sync.Once
	for range concurrency {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				item, err := r.prepareDocument(workerCtx, workerBinding, metadata, docs[index], stored[docs[index].ID])
				if err != nil {
					errOnce.Do(func() { firstErr = err; cancel() })
					continue
				}
				prepared[index] = item
			}
		}()
	}
send:
	for index := range docs {
		select {
		case jobs <- index:
		case <-workerCtx.Done():
			break send
		}
	}
	close(jobs)
	workers.Wait()
	if firstErr != nil {
		return 0, 0, firstErr
	}
	if err := ctx.Err(); err != nil {
		return 0, 0, err
	}
	var embedded, reused int
	if batch, ok := binding.Store.(GenerationBatchStore); ok {
		writes := make([]GenerationSourceWrite, len(prepared))
		for index, item := range prepared {
			writes[index] = item.write
			embedded += item.embedded
			reused += item.reused
		}
		if err := batch.StageSources(ctx, metadata.ID, writes); err != nil {
			return 0, 0, fmt.Errorf("stage %d sources: %w", len(writes), err)
		}
		return embedded, reused, nil
	}
	for index, item := range prepared {
		if err := binding.Store.StageSource(ctx, metadata.ID, item.write); err != nil {
			return embedded, reused, fmt.Errorf("stage source %q: %w", docs[index].ID, err)
		}
		embedded += item.embedded
		reused += item.reused
	}
	return embedded, reused, nil
}

func (r *StreamingReconciler) prepareDocument(ctx context.Context, binding StreamingBinding, metadata GenerationMetadata, doc SourceDoc, stored StoredSourceState) (preparedSource, error) {
	if strings.TrimSpace(doc.ID) == "" {
		return preparedSource{}, fmt.Errorf("source document id is required")
	}
	if sourceStateComplete(stored, doc.ContentHash, metadata) {
		ids := make([]string, 0, len(stored.Points))
		for id := range stored.Points {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		write := GenerationSourceWrite{SourceID: doc.ID, SourceHash: doc.ContentHash, ReusePointIDs: ids}
		return preparedSource{write: write, reused: len(ids)}, nil
	}
	chunks, err := binding.chunker().Chunk(doc)
	if err != nil {
		return preparedSource{}, fmt.Errorf("chunk %q: %w", doc.ID, err)
	}
	write := GenerationSourceWrite{SourceID: doc.ID, SourceHash: doc.ContentHash}
	composer := binding.composer()
	recipe := embedderRecipe(r.Embedder)
	tasks := make([]pendingEmbedding, 0, len(chunks))
	for i := range chunks {
		if err := ctx.Err(); err != nil {
			return preparedSource{}, err
		}
		chunk := chunks[i]
		chunk.SourceID = doc.ID
		chunk.Index = i
		chunk.ID = PointIDFor(binding.IDPrefix, doc.ID, i, len(chunks))
		text := composer.Compose(chunk)
		payload := buildChunkPayload(chunk, text, doc.ContentHash, len(chunks), recipe)
		payloadHash, _ := payload[payloadHashKey].(string)
		if previous, ok := stored.Points[chunk.ID]; ok && previous.PayloadHash == payloadHash {
			write.ReusePointIDs = append(write.ReusePointIDs, chunk.ID)
			continue
		}
		point := Point{ID: chunk.ID, Payload: payload}
		if binding.Sparse != nil {
			sparse := binding.Sparse.Encode(text)
			point.Sparse = &sparse
		}
		tasks = append(tasks, pendingEmbedding{chunkIndex: i, text: text, point: point})
	}
	if err := r.embedPending(ctx, binding, doc.ID, tasks); err != nil {
		return preparedSource{}, err
	}
	for _, task := range tasks {
		write.ChangedPoints = append(write.ChangedPoints, task.point)
	}
	return preparedSource{write: write, embedded: len(write.ChangedPoints), reused: len(write.ReusePointIDs)}, nil
}

type pendingEmbedding struct {
	chunkIndex int
	text       string
	point      Point
}

func (r *StreamingReconciler) embedPending(ctx context.Context, binding StreamingBinding, sourceID string, tasks []pendingEmbedding) error {
	if len(tasks) == 0 {
		return nil
	}
	concurrency := binding.EmbedConcurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	if concurrency > len(tasks) {
		concurrency = len(tasks)
	}
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan int)
	var workers sync.WaitGroup
	var firstErr error
	var errOnce sync.Once
	recordError := func(err error) {
		errOnce.Do(func() {
			firstErr = err
			cancel()
		})
	}
	for range concurrency {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				release := func() {}
				if binding.Admission != nil {
					weight := binding.EmbedWeight
					if weight <= 0 {
						weight = 1
					}
					var err error
					release, err = binding.Admission.Acquire(workerCtx, weight)
					if err != nil {
						recordError(fmt.Errorf("admit embed for %q: %w", sourceID, err))
						continue
					}
				}
				dense, err := embedDocumentText(workerCtx, r.Embedder, tasks[index].text)
				release()
				if err != nil {
					recordError(fmt.Errorf("embed %q chunk %d: %w", sourceID, tasks[index].chunkIndex, err))
					continue
				}
				tasks[index].point.Dense = dense
			}
		}()
	}
send:
	for index := range tasks {
		select {
		case jobs <- index:
		case <-workerCtx.Done():
			break send
		}
	}
	close(jobs)
	workers.Wait()
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}

func sourceStateComplete(stored StoredSourceState, sourceHash string, metadata GenerationMetadata) bool {
	if sourceHash == "" || stored.SourceHash != sourceHash || stored.Model != metadata.Model || stored.ChunkPolicy != metadata.ChunkPolicy || len(stored.Points) == 0 {
		return false
	}
	count := len(stored.Points)
	for _, point := range stored.Points {
		if point.SourceHash != sourceHash || point.ChunkTotal != count {
			return false
		}
	}
	return true
}

func validateStreamingBinding(binding StreamingBinding, embedder Embedder) error {
	if binding.Store == nil || binding.Source == nil || embedder == nil {
		return fmt.Errorf("streaming reconcile requires store, source, and embedder")
	}
	if strings.TrimSpace(binding.Kind) == "" {
		return fmt.Errorf("streaming reconcile requires binding kind")
	}
	return nil
}

func pageSourceIDs(docs []SourceDoc) ([]string, error) {
	ids := make([]string, 0, len(docs))
	seen := make(map[string]struct{}, len(docs))
	for _, doc := range docs {
		id := strings.TrimSpace(doc.ID)
		if id == "" {
			return nil, fmt.Errorf("source document id is required")
		}
		if _, exists := seen[id]; exists {
			return nil, fmt.Errorf("source page contains duplicate id %q", id)
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *StreamingReconciler) now() time.Time {
	if r.Clock != nil {
		return r.Clock()
	}
	return time.Now()
}

// PagedSourceAdapter intentionally adapts the legacy LoadAll contract for
// bounded small corpora. It materializes once, then serves stable pages. Large
// corpora must implement PagedSource directly.
type PagedSourceAdapter struct {
	Source Source

	mu     sync.Mutex
	loaded bool
	docs   []SourceDoc
	err    error
}

func NewPagedSourceAdapter(source Source) *PagedSourceAdapter {
	return &PagedSourceAdapter{Source: source}
}

func (a *PagedSourceAdapter) LoadPage(ctx context.Context, request PageRequest) (SourcePage, error) {
	if a == nil || a.Source == nil {
		return SourcePage{}, fmt.Errorf("paged source adapter requires source")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.loaded {
		a.docs, a.err = a.Source.LoadAll(ctx)
		a.loaded = true
	}
	if a.err != nil {
		return SourcePage{}, a.err
	}
	start := 0
	if request.Cursor != "" {
		parsed, err := strconv.Atoi(request.Cursor)
		if err != nil || parsed < 0 {
			return SourcePage{}, fmt.Errorf("invalid page cursor %q", request.Cursor)
		}
		start = parsed
	}
	if start > len(a.docs) {
		return SourcePage{}, fmt.Errorf("page cursor %d exceeds corpus size %d", start, len(a.docs))
	}
	limit := request.Limit
	if limit <= 0 {
		limit = DefaultSourcePageSize
	}
	if limit > MaxSourcePageSize {
		limit = MaxSourcePageSize
	}
	end := start + limit
	if end > len(a.docs) {
		end = len(a.docs)
	}
	page := SourcePage{Documents: append([]SourceDoc(nil), a.docs[start:end]...), Done: end == len(a.docs)}
	if !page.Done {
		page.NextCursor = strconv.Itoa(end)
	}
	return page, nil
}
