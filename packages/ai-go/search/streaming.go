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
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	SourceDigest string    `json:"sourceDigest,omitempty"`
	Model        string    `json:"model,omitempty"`
	ChunkPolicy  string    `json:"chunkPolicy,omitempty"`
	Full         bool      `json:"full"`
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
		stored, err := binding.Store.LookupActiveSources(ctx, ids)
		if err != nil {
			return result, fmt.Errorf("lookup active sources: %w", err)
		}
		if len(stored) > len(ids) {
			return result, fmt.Errorf("lookup returned %d sources for a %d-source page", len(stored), len(ids))
		}
		for _, doc := range page.Documents {
			embedded, reused, err := r.stageDocument(ctx, binding, metadata, doc, stored[doc.ID])
			if err != nil {
				return result, err
			}
			result.Sources++
			result.Embedded += embedded
			result.Reused += reused
		}
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
	if err := binding.Store.PromoteGeneration(ctx, metadata.ID); err != nil {
		return result, fmt.Errorf("promote generation %q: %w", metadata.ID, err)
	}
	promoted = true
	result.Promoted = true
	if err := binding.Store.CleanupGenerations(ctx, 2); err != nil {
		return result, fmt.Errorf("cleanup generations after promotion: %w", err)
	}
	return result, nil
}

// RunChanges applies one bounded explicit change set through the same shadow
// generation lifecycle. Deletions are source-level and cannot leave ghost
// chunks behind.
func (r *StreamingReconciler) RunChanges(ctx context.Context, binding StreamingBinding, metadata GenerationMetadata, changes ChangeSet) (*StreamingResult, error) {
	if err := validateStreamingBinding(binding, r.Embedder); err != nil {
		return nil, err
	}
	if len(changes.Changes) > binding.pageSize() {
		return nil, fmt.Errorf("change set contains %d changes, limit is %d", len(changes.Changes), binding.pageSize())
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

	result := &StreamingResult{Generation: metadata, Pages: 1, MaxPageDocuments: len(changes.Changes)}
	var upserts []SourceDoc
	for _, change := range changes.Changes {
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
	for _, doc := range upserts {
		embedded, reused, err := r.stageDocument(ctx, binding, metadata, doc, stored[doc.ID])
		if err != nil {
			return result, err
		}
		result.Sources++
		result.Embedded += embedded
		result.Reused += reused
	}
	validation, err := binding.Store.ValidateGeneration(ctx, metadata.ID)
	if err != nil {
		return result, fmt.Errorf("validate generation %q: %w", metadata.ID, err)
	}
	result.Validation = validation
	if !validation.Valid {
		return result, fmt.Errorf("generation %q is invalid: %s", metadata.ID, validation.Detail)
	}
	if err := binding.Store.PromoteGeneration(ctx, metadata.ID); err != nil {
		return result, fmt.Errorf("promote generation %q: %w", metadata.ID, err)
	}
	promoted = true
	result.Promoted = true
	if err := binding.Store.CleanupGenerations(ctx, 2); err != nil {
		return result, fmt.Errorf("cleanup generations after promotion: %w", err)
	}
	return result, nil
}

func (r *StreamingReconciler) stageDocument(ctx context.Context, binding StreamingBinding, metadata GenerationMetadata, doc SourceDoc, stored StoredSourceState) (int, int, error) {
	if strings.TrimSpace(doc.ID) == "" {
		return 0, 0, fmt.Errorf("source document id is required")
	}
	if sourceStateComplete(stored, doc.ContentHash, metadata) {
		ids := make([]string, 0, len(stored.Points))
		for id := range stored.Points {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		write := GenerationSourceWrite{SourceID: doc.ID, SourceHash: doc.ContentHash, ReusePointIDs: ids}
		if err := binding.Store.StageSource(ctx, metadata.ID, write); err != nil {
			return 0, 0, fmt.Errorf("stage unchanged source %q: %w", doc.ID, err)
		}
		return 0, len(ids), nil
	}
	chunks, err := binding.chunker().Chunk(doc)
	if err != nil {
		return 0, 0, fmt.Errorf("chunk %q: %w", doc.ID, err)
	}
	write := GenerationSourceWrite{SourceID: doc.ID, SourceHash: doc.ContentHash}
	composer := binding.composer()
	recipe := embedderRecipe(r.Embedder)
	for i := range chunks {
		if err := ctx.Err(); err != nil {
			return 0, 0, err
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
		release := func() {}
		if binding.Admission != nil {
			weight := binding.EmbedWeight
			if weight <= 0 {
				weight = 1
			}
			release, err = binding.Admission.Acquire(ctx, weight)
			if err != nil {
				return 0, 0, fmt.Errorf("admit embed for %q: %w", doc.ID, err)
			}
		}
		dense, embedErr := embedDocumentText(ctx, r.Embedder, text)
		release()
		if embedErr != nil {
			return 0, 0, fmt.Errorf("embed %q chunk %d: %w", doc.ID, i, embedErr)
		}
		point := Point{ID: chunk.ID, Dense: dense, Payload: payload}
		if binding.Sparse != nil {
			sparse := binding.Sparse.Encode(text)
			point.Sparse = &sparse
		}
		write.ChangedPoints = append(write.ChangedPoints, point)
	}
	if err := binding.Store.StageSource(ctx, metadata.ID, write); err != nil {
		return 0, 0, fmt.Errorf("stage source %q: %w", doc.ID, err)
	}
	return len(write.ChangedPoints), len(write.ReusePointIDs), nil
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
