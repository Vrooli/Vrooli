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

	"golang.org/x/sync/errgroup"
)

// SourceBinding parameterizes the reconciler over one logical collection: where
// the source docs come from (Source), how they fan out (Chunker), how each
// chunk's embedding text is composed (Composer), the optional local sparse
// encoder (nil → dense-only), the vector store, and the per-consumer point-ID
// prefix that keeps natural keys collision-free in a shared collection.
type SourceBinding struct {
	Kind     string
	Store    VectorStore
	Source   Source
	Chunker  Chunker
	Composer EmbeddingTextComposer
	Sparse   SparseEncoder
	IDPrefix string
}

func (b SourceBinding) composer() EmbeddingTextComposer {
	if b.Composer != nil {
		return b.Composer
	}
	return NewIdentityComposer()
}

func (b SourceBinding) chunker() Chunker {
	if b.Chunker != nil {
		return b.Chunker
	}
	return NewIdentityChunker()
}

// ErrReconcileBusy is returned by RunOnce when a reconcile is already running.
var ErrReconcileBusy = errors.New("reconcile already in progress")

// Reconciler computes and applies the qdrant work needed to make each binding's
// collection match its source, using two-level drift (§4.1) and a per-tick
// embed budget (§4.2).
type Reconciler struct {
	Embedder    Embedder
	Bindings    []SourceBinding
	Parallelism int
	// MaxEmbedsPerTick caps embeds (upserts) per Apply across all bindings;
	// 0 means unlimited. The overflow is reported as Deferred and re-planned
	// next tick — the first full-repo index never starves Ollama (§4.2).
	MaxEmbedsPerTick int
	Clock            func() time.Time

	mu         sync.Mutex
	running    bool
	cancel     context.CancelFunc
	lastPlan   *DriftReport
	lastResult *ApplyResult
	lastError  string
	canceled   bool
	startedAt  time.Time
	finishedAt time.Time
}

// NewReconciler wires a Reconciler with bounded embed concurrency.
func NewReconciler(embedder Embedder, bindings []SourceBinding, parallelism int) *Reconciler {
	if parallelism <= 0 {
		parallelism = DefaultReconcileParallelism
	}
	if parallelism > MaxReconcileParallelism {
		parallelism = MaxReconcileParallelism
	}
	return &Reconciler{
		Embedder:    embedder,
		Bindings:    bindings,
		Parallelism: parallelism,
		Clock:       time.Now,
	}
}

// Plan walks every binding and reports drift. Read-only.
func (r *Reconciler) Plan(ctx context.Context) (*DriftReport, error) {
	report := &DriftReport{
		PlannedAt:   r.now(),
		Collections: make([]CollectionDriftReport, 0, len(r.Bindings)),
	}
	for _, b := range r.Bindings {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		report.Collections = append(report.Collections, r.planBinding(ctx, b))
	}
	return report, nil
}

func (r *Reconciler) planBinding(ctx context.Context, b SourceBinding) CollectionDriftReport {
	col := CollectionDriftReport{Kind: b.Kind}
	recipe := embedderRecipe(r.Embedder)

	docs, err := b.Source.LoadAll(ctx)
	if err != nil {
		log.Printf("[aisearch] reconciler: LoadAll(%s) failed: %v", b.Kind, err)
		return col
	}
	stored, err := b.Store.ScrollIDs(ctx)
	if err != nil {
		log.Printf("[aisearch] reconciler: ScrollIDs(%s) failed: %v", b.Kind, err)
		return col
	}

	// Group stored points by their source for the source-level skip and ghosts.
	storedBySource := make(map[string][]string, len(stored))
	for pid, item := range stored {
		storedBySource[item.SourceID] = append(storedBySource[item.SourceID], pid)
	}

	composer := b.composer()
	chunker := b.chunker()
	seen := make(map[string]struct{}, len(stored))

	for _, doc := range docs {
		// Level 1: skip an unchanged, fully-indexed file wholesale, before
		// chunking/embedding.
		if existing := storedBySource[doc.ID]; doc.ContentHash != "" && sourceUnchanged(stored, existing, doc.ContentHash) {
			for _, pid := range existing {
				seen[pid] = struct{}{}
			}
			col.UnchangedSources++
			continue
		}

		chunks, err := chunker.Chunk(doc)
		if err != nil {
			log.Printf("[aisearch] reconciler: chunk(%s/%s) failed: %v", b.Kind, doc.ID, err)
			continue
		}
		total := len(chunks)
		for i := range chunks {
			chunk := chunks[i]
			chunk.SourceID = doc.ID
			chunk.Index = i
			pid := PointIDFor(b.IDPrefix, doc.ID, i, total)
			chunk.ID = pid
			seen[pid] = struct{}{}

			text := composer.Compose(chunk)
			payload := buildChunkPayload(chunk, text, doc.ContentHash, total, recipe)
			hash, _ := payload[payloadHashKey].(string)

			existing, ok := stored[pid]
			switch {
			case !ok:
				col.ToUpsert = append(col.ToUpsert, upsertRef(b.Kind, pid, chunk, total, hash, doc.ContentHash))
			case existing.PayloadHash == "":
				col.LegacyCount++
				col.ToUpsert = append(col.ToUpsert, upsertRef(b.Kind, pid, chunk, total, hash, doc.ContentHash))
			case existing.PayloadHash != hash:
				col.ToUpsert = append(col.ToUpsert, upsertRef(b.Kind, pid, chunk, total, hash, doc.ContentHash))
			case existing.SourceHash != doc.ContentHash || existing.ChunkTotal != total:
				// Body unchanged but the parent file changed elsewhere (or was
				// re-split) — refresh source_hash/chunk_total only (no embed) so
				// the source-level gate converges.
				col.ToRefresh = append(col.ToRefresh, RefreshRef{Kind: b.Kind, PointID: pid, SourceHash: doc.ContentHash, Payload: payload})
			default:
				col.UnchangedChunks++
			}
		}
	}

	var ghosts []string
	for pid := range stored {
		if _, ok := seen[pid]; !ok {
			ghosts = append(ghosts, pid)
		}
	}
	sort.Strings(ghosts)
	col.ToDelete = ghosts
	return col
}

func upsertRef(kind, pid string, chunk Chunk, total int, hash, sourceHash string) ItemRef {
	return ItemRef{
		Kind:        kind,
		PointID:     pid,
		SourceID:    chunk.SourceID,
		Index:       chunk.Index,
		Total:       total,
		Name:        chunk.SourceID,
		PayloadHash: hash,
		SourceHash:  sourceHash,
		Chunk:       chunk,
	}
}

// sourceUnchanged reports whether a source is fully indexed and current: every
// stored chunk agrees on the source hash AND the stored count equals the
// recorded chunk total (so a budget-deferred partial index never skips).
func sourceUnchanged(stored map[string]ScrollItem, pids []string, wantHash string) bool {
	n := len(pids)
	if n == 0 {
		return false
	}
	for _, pid := range pids {
		it := stored[pid]
		if it.SourceHash != wantHash || it.ChunkTotal != n {
			return false
		}
	}
	return true
}

// Apply executes a plan: embed+upsert (budget-capped), payload refreshes, then
// ghost deletes. Failures are collected, not fatal.
func (r *Reconciler) Apply(ctx context.Context, plan *DriftReport) (*ApplyResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("apply: plan is required")
	}
	result := &ApplyResult{
		StartedAt:   r.now(),
		Collections: make([]CollectionApplyResult, len(plan.Collections)),
	}
	for i, c := range plan.Collections {
		result.Collections[i] = CollectionApplyResult{Kind: c.Kind}
	}
	bindingByKind := make(map[string]SourceBinding, len(r.Bindings))
	for _, b := range r.Bindings {
		bindingByKind[b.Kind] = b
	}

	var errMu sync.Mutex
	addErr := func(e ReconcileError) {
		errMu.Lock()
		result.Errors = append(result.Errors, e)
		errMu.Unlock()
	}

	parallelism := r.Parallelism
	if parallelism <= 0 {
		parallelism = 1
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(parallelism)

	// Flatten upserts across bindings so the per-tick embed budget applies to
	// the whole corpus, then schedule up to the budget; the rest is Deferred.
	type upsertWork struct {
		ci  int
		ref ItemRef
		b   SourceBinding
	}
	var work []upsertWork
	for ci := range plan.Collections {
		c := plan.Collections[ci]
		b, ok := bindingByKind[c.Kind]
		if !ok {
			continue
		}
		for _, ref := range c.ToUpsert {
			work = append(work, upsertWork{ci: ci, ref: ref, b: b})
		}
	}
	budget := r.MaxEmbedsPerTick
	for i := range work {
		if budget > 0 && i >= budget {
			result.Deferred = len(work) - budget
			break
		}
		w := work[i]
		g.Go(func() error {
			if err := gctx.Err(); err != nil {
				addErr(ReconcileError{Kind: w.b.Kind, PointID: w.ref.PointID, Name: w.ref.Name, Op: "embed", Err: err.Error()})
				return nil
			}
			text := w.b.composer().Compose(w.ref.Chunk)
			dense, err := embedDocumentText(gctx, r.Embedder, text)
			if err != nil {
				addErr(ReconcileError{Kind: w.b.Kind, PointID: w.ref.PointID, Name: w.ref.Name, Op: "embed", Err: err.Error()})
				return nil
			}
			point := Point{ID: w.ref.PointID, Dense: dense, Payload: buildChunkPayload(w.ref.Chunk, text, w.ref.SourceHash, w.ref.Total, embedderRecipe(r.Embedder))}
			if w.b.Sparse != nil {
				sv := w.b.Sparse.Encode(text)
				point.Sparse = &sv
			}
			if err := w.b.Store.Upsert(gctx, point); err != nil {
				addErr(ReconcileError{Kind: w.b.Kind, PointID: w.ref.PointID, Name: w.ref.Name, Op: "upsert", Err: err.Error()})
				return nil
			}
			errMu.Lock()
			result.Collections[w.ci].Upserted++
			errMu.Unlock()
			return nil
		})
	}
	_ = g.Wait()

	// Payload-only refreshes (cheap; no embed, not budgeted).
	for ci := range plan.Collections {
		c := plan.Collections[ci]
		b, ok := bindingByKind[c.Kind]
		if !ok {
			continue
		}
		for _, ref := range c.ToRefresh {
			if err := b.Store.SetPayload(ctx, ref.PointID, ref.Payload); err != nil {
				addErr(ReconcileError{Kind: c.Kind, PointID: ref.PointID, Op: "refresh", Err: err.Error()})
				continue
			}
			result.Collections[ci].Refreshed++
		}
	}

	// Ghost deletions.
	for ci := range plan.Collections {
		c := plan.Collections[ci]
		if len(c.ToDelete) == 0 {
			continue
		}
		b, ok := bindingByKind[c.Kind]
		if !ok {
			continue
		}
		if err := b.Store.BatchDelete(ctx, c.ToDelete); err != nil {
			addErr(ReconcileError{Kind: c.Kind, Op: "delete", Err: err.Error()})
			continue
		}
		result.Collections[ci].Deleted = len(c.ToDelete)
	}

	if result.Deferred > 0 {
		log.Printf("[aisearch] reconciler: per-tick embed budget %d reached, deferred %d upserts to next tick", budget, result.Deferred)
	}
	result.FinishedAt = r.now()
	return result, nil
}

// RunOnce composes Plan + Apply with singleton semantics.
func (r *Reconciler) RunOnce(ctx context.Context) (*DriftReport, *ApplyResult, error) {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return nil, nil, ErrReconcileBusy
	}
	runCtx, cancel := context.WithCancel(ctx)
	r.running = true
	r.cancel = cancel
	r.canceled = false
	r.startedAt = r.now()
	r.lastError = ""
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.running = false
		r.cancel = nil
		r.finishedAt = r.now()
		r.mu.Unlock()
		cancel()
	}()

	plan, err := r.Plan(runCtx)
	if err != nil {
		r.mu.Lock()
		r.lastError = err.Error()
		r.mu.Unlock()
		return nil, nil, err
	}
	r.mu.Lock()
	r.lastPlan = plan
	r.mu.Unlock()

	apply, err := r.Apply(runCtx, plan)
	r.mu.Lock()
	r.lastResult = apply
	if err != nil {
		r.lastError = err.Error()
	}
	r.mu.Unlock()
	return plan, apply, err
}

// Cancel aborts an in-flight RunOnce.
func (r *Reconciler) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.canceled = true
		r.cancel()
	}
}

// Status returns a snapshot of the reconciler's last-known state.
func (r *Reconciler) Status() ReconcileStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := ReconcileStatus{
		Running:    r.running,
		LastPlan:   r.lastPlan,
		LastResult: r.lastResult,
		LastError:  r.lastError,
		Canceled:   r.canceled,
	}
	if !r.startedAt.IsZero() {
		st.StartedAt = r.startedAt.Format(time.RFC3339)
	}
	if !r.finishedAt.IsZero() && !r.running {
		st.FinishedAt = r.finishedAt.Format(time.RFC3339)
	}
	return st
}

func (r *Reconciler) now() time.Time {
	if r.Clock != nil {
		return r.Clock()
	}
	return time.Now()
}

// buildChunkPayload assembles the qdrant payload for one chunk: the source
// metadata, the retrievable body, the grouping/drift fields, and the two
// reconciler hashes (payload_hash excludes both hashes; source_hash is stored
// for the source-level gate).
func buildChunkPayload(chunk Chunk, embeddingText, sourceHash string, total int, recipe string) map[string]any {
	p := make(map[string]any, len(chunk.Meta)+6)
	for k, v := range chunk.Meta {
		p[k] = v
	}
	p[bodyKey] = chunk.Body
	p[sourceIDKey] = chunk.SourceID
	p[chunkIndexKey] = chunk.Index
	p[chunkTotalKey] = total
	// Fold the embedding recipe into the hashed text so a model/prefix change
	// re-embeds the corpus. An empty recipe keeps the legacy hash byte-identical.
	hashText := embeddingText
	if recipe != "" {
		hashText = recipe + "\x00" + embeddingText
	}
	p[payloadHashKey] = composePayloadHash(hashText, p)
	if strings.TrimSpace(sourceHash) != "" {
		p[sourceHashKey] = sourceHash
	}
	return p
}
