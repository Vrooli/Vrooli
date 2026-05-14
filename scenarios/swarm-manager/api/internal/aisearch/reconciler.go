package aisearch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
)

// ErrReconcileBusy signals that RunOnce was called while a previous RunOnce
// is still in flight. The SyncLoop treats this as success (the in-flight pass
// is doing the work the new tick would have).
var ErrReconcileBusy = errors.New("reconcile already in progress")

// Reconciler is the single owner of the "make qdrant match disk" decision. It
// does no search, no status, and no HTTP — those stay in Service / Handler /
// VectorStore. Plan/Apply/RunOnce form the convergence contract:
//
//	Plan  → DriftReport         (read-only, idempotent; never mutates qdrant)
//	Apply → ApplyResult         (executes a precomputed plan; bounded parallelism)
//	RunOnce → Plan + Apply      (singleton; ErrReconcileBusy on contention)
//
// Decision boundaries (per the implementation plan):
//   - "What work needs doing?"       → Plan returns DriftReport
//   - "Should we re-embed this?"     → composePayloadHash + the inline compare in Plan
//   - "How does the index converge?" → Apply consumes DriftReport
type Reconciler struct {
	Embedder         Embedder
	BacklogStore     VectorStore
	InitiativeStore  VectorStore
	BacklogReader    BacklogReader
	InitiativeReader InitiativeReader
	Parallelism      int

	// Clock is injectable so tests can pin PlannedAt / StartedAt to known
	// values. Defaults to time.Now via now() when nil.
	Clock func() time.Time

	// mu guards the singleton + status fields below. RunOnce is the only
	// caller that mutates running/cancel; Status reads under the same lock.
	mu         sync.Mutex
	running    bool
	cancel     context.CancelFunc
	startedAt  time.Time
	finishedAt time.Time
	canceled   bool
	lastPlan   *DriftReport
	lastResult *ApplyResult
	lastError  string
}

// NewReconciler constructs a Reconciler with sane defaults. Pass nil for
// embedder, store, or reader to disable that side; the reconciler degrades
// (Plan returns an empty per-disabled-side DriftReport without error).
func NewReconciler(
	embedder Embedder,
	backlogStore, initiativeStore VectorStore,
	backlogReader BacklogReader,
	initiativeReader InitiativeReader,
	parallelism int,
) *Reconciler {
	if parallelism < 1 {
		parallelism = DefaultReconcileParallelism
	}
	if parallelism > MaxReconcileParallelism {
		parallelism = MaxReconcileParallelism
	}
	return &Reconciler{
		Embedder:         embedder,
		BacklogStore:     backlogStore,
		InitiativeStore:  initiativeStore,
		BacklogReader:    backlogReader,
		InitiativeReader: initiativeReader,
		Parallelism:      parallelism,
	}
}

func (r *Reconciler) now() time.Time {
	if r.Clock != nil {
		return r.Clock()
	}
	return time.Now()
}

// Plan walks disk and qdrant once each, computes the per-item content
// fingerprint, and returns the structured set of work that would converge
// the index. Read-only; never mutates qdrant.
func (r *Reconciler) Plan(ctx context.Context) (*DriftReport, error) {
	report := &DriftReport{PlannedAt: r.now()}

	if err := r.planBacklog(ctx, report); err != nil {
		return nil, err
	}
	if err := r.planInitiatives(ctx, report); err != nil {
		return nil, err
	}
	return report, nil
}

func (r *Reconciler) planBacklog(ctx context.Context, report *DriftReport) error {
	if r.BacklogStore == nil || r.BacklogReader == nil {
		return nil
	}

	items, err := r.BacklogReader.LoadAll()
	if err != nil {
		return fmt.Errorf("load backlog items: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	indexed, err := r.BacklogStore.ScrollIDs(ctx)
	if err != nil {
		return fmt.Errorf("scroll backlog index: %w", err)
	}

	diskByID := make(map[string]struct{}, len(items))
	for i := range items {
		item := items[i] // stable address even on Go < 1.22
		id := backlogPointID(item.Kind, item.Name)
		diskByID[id] = struct{}{}

		text := composeBacklogText(item)
		payloadNoHash := buildBacklogPayload(item, "")
		freshHash := composePayloadHash(text, payloadNoHash)

		existing, present := indexed[id]
		switch {
		case !present:
			report.ToUpsertBacklog = append(report.ToUpsertBacklog, ItemRef{
				Kind:        KindBacklogItem,
				PointID:     id,
				Name:        item.Name,
				PayloadHash: freshHash,
				Backlog:     &item,
			})
		case existing.PayloadHash == "":
			// Legacy point from before payload_hash existed → force re-embed once.
			report.LegacyBacklog++
			report.ToUpsertBacklog = append(report.ToUpsertBacklog, ItemRef{
				Kind:        KindBacklogItem,
				PointID:     id,
				Name:        item.Name,
				PayloadHash: freshHash,
				Backlog:     &item,
			})
		case existing.PayloadHash != freshHash:
			report.ToUpsertBacklog = append(report.ToUpsertBacklog, ItemRef{
				Kind:        KindBacklogItem,
				PointID:     id,
				Name:        item.Name,
				PayloadHash: freshHash,
				Backlog:     &item,
			})
		default:
			report.UnchangedBacklog++
		}
	}

	for id := range indexed {
		if _, onDisk := diskByID[id]; !onDisk {
			report.ToDeleteBacklog = append(report.ToDeleteBacklog, id)
		}
	}
	return nil
}

func (r *Reconciler) planInitiatives(ctx context.Context, report *DriftReport) error {
	if r.InitiativeStore == nil || r.InitiativeReader == nil {
		return nil
	}

	inits, err := r.InitiativeReader.List()
	if err != nil {
		return fmt.Errorf("list initiatives: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	indexed, err := r.InitiativeStore.ScrollIDs(ctx)
	if err != nil {
		return fmt.Errorf("scroll initiative index: %w", err)
	}

	diskByID := make(map[string]struct{}, len(inits))
	for i := range inits {
		init := inits[i]
		id := initiativePointID(init.Name)
		diskByID[id] = struct{}{}

		text := composeInitiativeText(init)
		payloadNoHash := buildInitiativePayload(init, "")
		freshHash := composePayloadHash(text, payloadNoHash)

		existing, present := indexed[id]
		switch {
		case !present:
			report.ToUpsertInitiative = append(report.ToUpsertInitiative, ItemRef{
				Kind:        KindInitiative,
				PointID:     id,
				Name:        init.Name,
				PayloadHash: freshHash,
				Initiative:  &init,
			})
		case existing.PayloadHash == "":
			report.LegacyInitiative++
			report.ToUpsertInitiative = append(report.ToUpsertInitiative, ItemRef{
				Kind:        KindInitiative,
				PointID:     id,
				Name:        init.Name,
				PayloadHash: freshHash,
				Initiative:  &init,
			})
		case existing.PayloadHash != freshHash:
			report.ToUpsertInitiative = append(report.ToUpsertInitiative, ItemRef{
				Kind:        KindInitiative,
				PointID:     id,
				Name:        init.Name,
				PayloadHash: freshHash,
				Initiative:  &init,
			})
		default:
			report.UnchangedInitiative++
		}
	}

	for id := range indexed {
		if _, onDisk := diskByID[id]; !onDisk {
			report.ToDeleteInitiative = append(report.ToDeleteInitiative, id)
		}
	}
	return nil
}

// Apply executes a precomputed DriftReport. Upserts run with bounded
// parallelism (Reconciler.Parallelism); deletes go through one BatchDelete
// per collection. Per-item failures collect into ApplyResult.Errors without
// aborting the rest of the pass — the next tick re-evaluates and retries.
//
// Apply does not consult the singleton flag; callers wanting RunOnce semantics
// must use RunOnce. Plan-then-Apply by handler dry-run / "preview then apply"
// flows is intentional and safe.
func (r *Reconciler) Apply(ctx context.Context, plan *DriftReport) (*ApplyResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan is required")
	}
	result := &ApplyResult{StartedAt: r.now()}
	defer func() { result.FinishedAt = r.now() }()

	// Ensure both collections exist before any upsert. EnsureCollection is
	// idempotent so this is safe to call on every Apply.
	if r.BacklogStore != nil && (len(plan.ToUpsertBacklog) > 0 || len(plan.ToDeleteBacklog) > 0) {
		if err := r.BacklogStore.EnsureCollection(ctx); err != nil {
			return result, fmt.Errorf("ensure backlog collection: %w", err)
		}
	}
	if r.InitiativeStore != nil && (len(plan.ToUpsertInitiative) > 0 || len(plan.ToDeleteInitiative) > 0) {
		if err := r.InitiativeStore.EnsureCollection(ctx); err != nil {
			return result, fmt.Errorf("ensure initiative collection: %w", err)
		}
	}

	// --- Upsert phase: bounded-parallel embed-then-upsert. ---
	var mu sync.Mutex
	addError := func(e ReconcileError) {
		mu.Lock()
		defer mu.Unlock()
		result.Errors = append(result.Errors, e)
	}
	bumpUpsert := func(kind EntityKind) {
		mu.Lock()
		defer mu.Unlock()
		switch kind {
		case KindBacklogItem:
			result.UpsertedBacklog++
		case KindInitiative:
			result.UpsertedInitiative++
		}
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(r.Parallelism)
	r.scheduleUpserts(gctx, g, plan.ToUpsertBacklog, r.BacklogStore, addError, bumpUpsert)
	r.scheduleUpserts(gctx, g, plan.ToUpsertInitiative, r.InitiativeStore, addError, bumpUpsert)
	// errgroup.Wait can return ctx.Canceled; that is the cooperative-cancel
	// signal, not a hard failure — partial counts in result are still valid.
	_ = g.Wait()

	// --- Delete phase: one BatchDelete per collection. ---
	if len(plan.ToDeleteBacklog) > 0 && r.BacklogStore != nil {
		if err := r.BacklogStore.BatchDelete(ctx, plan.ToDeleteBacklog); err != nil {
			addError(ReconcileError{Kind: KindBacklogItem, Op: "delete", Err: err.Error()})
		} else {
			result.DeletedBacklog = len(plan.ToDeleteBacklog)
		}
	}
	if len(plan.ToDeleteInitiative) > 0 && r.InitiativeStore != nil {
		if err := r.InitiativeStore.BatchDelete(ctx, plan.ToDeleteInitiative); err != nil {
			addError(ReconcileError{Kind: KindInitiative, Op: "delete", Err: err.Error()})
		} else {
			result.DeletedInitiative = len(plan.ToDeleteInitiative)
		}
	}

	return result, nil
}

// scheduleUpserts queues every ItemRef onto the errgroup. The closure copies
// ref so the loop variable's per-iteration aliasing (Go ≥1.22) is irrelevant —
// the test suite must pass on older Go versions if anyone ever back-ports.
func (r *Reconciler) scheduleUpserts(
	gctx context.Context,
	g *errgroup.Group,
	refs []ItemRef,
	store VectorStore,
	addError func(ReconcileError),
	bumpUpsert func(EntityKind),
) {
	if store == nil {
		return
	}
	for i := range refs {
		ref := refs[i]
		g.Go(func() (err error) {
			// Recover panics inside the per-item goroutine. errgroup does NOT
			// recover panics; an uncaught panic here would crash the API
			// process. Convert to a typed ReconcileError so the surrounding
			// pass continues and the SyncLoop's next tick sees the partial
			// state and retries the failing item.
			defer func() {
				if rec := recover(); rec != nil {
					addError(ReconcileError{
						Kind: ref.Kind, PointID: ref.PointID, Name: ref.Name,
						Op: "embed", Err: fmt.Sprintf("panic: %v", rec),
					})
				}
			}()
			if err := gctx.Err(); err != nil {
				addError(ReconcileError{Kind: ref.Kind, PointID: ref.PointID, Name: ref.Name, Op: "embed", Err: err.Error()})
				return nil
			}
			text, payload := refToTextAndPayload(ref)
			vector, embedErr := r.Embedder.Embed(gctx, text)
			if embedErr != nil {
				addError(ReconcileError{Kind: ref.Kind, PointID: ref.PointID, Name: ref.Name, Op: "embed", Err: embedErr.Error()})
				return nil
			}
			if upsertErr := store.Upsert(gctx, ref.PointID, vector, payload); upsertErr != nil {
				addError(ReconcileError{Kind: ref.Kind, PointID: ref.PointID, Name: ref.Name, Op: "upsert", Err: upsertErr.Error()})
				return nil
			}
			bumpUpsert(ref.Kind)
			return nil
		})
	}
}

// refToTextAndPayload reconstructs the embedding input and the qdrant payload
// (with payload_hash) from an ItemRef. Centralized so the kind switch lives
// in exactly one place.
func refToTextAndPayload(ref ItemRef) (string, map[string]interface{}) {
	switch ref.Kind {
	case KindBacklogItem:
		var item backlog.BacklogItem
		if ref.Backlog != nil {
			item = *ref.Backlog
		}
		text := composeBacklogText(item)
		payload := buildBacklogPayload(item, ref.PayloadHash)
		return text, payload
	case KindInitiative:
		var init initiatives.Initiative
		if ref.Initiative != nil {
			init = *ref.Initiative
		}
		text := composeInitiativeText(init)
		payload := buildInitiativePayload(init, ref.PayloadHash)
		return text, payload
	default:
		return "", nil
	}
}

// RunOnce is the singleton entry point: one Plan + (HasWork ? Apply : noop).
// Returns (nil, nil, ErrReconcileBusy) when a prior RunOnce is still running.
//
// Snapshot of LastPlan / LastResult / LastError is published under r.mu so
// Status() can return a consistent view to the HTTP /status endpoint.
func (r *Reconciler) RunOnce(ctx context.Context) (*DriftReport, *ApplyResult, error) {
	runCtx, finish, ok := r.tryAcquire(ctx)
	if !ok {
		return nil, nil, ErrReconcileBusy
	}
	defer finish()
	return r.runBody(runCtx)
}

// StartAsync is the HTTP handler's entry point: it synchronously acquires the
// singleton (so a follow-up Status() call sees Running=true with no race
// window), then runs Plan+Apply in a goroutine. Returns ErrReconcileBusy if a
// prior RunOnce is still in flight.
//
// The goroutine uses a context derived from context.Background() so the
// reconcile completes (or is canceled via Cancel) regardless of the
// originating HTTP request.
func (r *Reconciler) StartAsync() error {
	runCtx, finish, ok := r.tryAcquire(context.Background())
	if !ok {
		return ErrReconcileBusy
	}
	go func() {
		defer finish()
		_, _, _ = r.runBody(runCtx)
	}()
	return nil
}

// tryAcquire takes the singleton flag and initializes the lifecycle state
// fields atomically. Returns the cancel-bound run context, a finish closure
// to release the singleton, and an ok flag (false → caller must return
// ErrReconcileBusy).
func (r *Reconciler) tryAcquire(ctx context.Context) (context.Context, func(), bool) {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return nil, nil, false
	}
	runCtx, cancel := context.WithCancel(ctx)
	r.running = true
	r.cancel = cancel
	r.canceled = false
	r.startedAt = r.now()
	r.finishedAt = time.Time{}
	r.lastError = ""
	r.mu.Unlock()

	finish := func() {
		r.mu.Lock()
		r.running = false
		r.cancel = nil
		r.finishedAt = r.now()
		r.mu.Unlock()
		cancel()
	}
	return runCtx, finish, true
}

// runBody is the Plan+Apply body, factored out of RunOnce so StartAsync can
// reuse it from a goroutine. Caller must already hold the singleton via
// tryAcquire and arrange for finish() to run.
func (r *Reconciler) runBody(runCtx context.Context) (*DriftReport, *ApplyResult, error) {
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

	if !plan.HasWork() {
		return plan, &ApplyResult{StartedAt: r.now(), FinishedAt: r.now()}, nil
	}

	result, applyErr := r.Apply(runCtx, plan)

	r.mu.Lock()
	r.lastResult = result
	if applyErr != nil {
		r.lastError = applyErr.Error()
	}
	if errors.Is(runCtx.Err(), context.Canceled) {
		r.canceled = true
	}
	r.mu.Unlock()

	return plan, result, applyErr
}

// Cancel signals an in-flight RunOnce to stop. No-op when idle. Cancellation
// is cooperative — the in-flight upserts complete or abort at their next
// context check; partial counts in ApplyResult remain valid.
func (r *Reconciler) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.canceled = true
		r.cancel()
	}
}

// Status returns a snapshot of the most recent reconcile lifecycle state for
// the GET /api/v1/search/ai/reconcile/status endpoint.
func (r *Reconciler) Status() ReconcileStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := ReconcileStatus{
		Running:    r.running,
		StartedAt:  formatReconcileTime(r.startedAt),
		FinishedAt: formatReconcileTime(r.finishedAt),
		LastPlan:   r.lastPlan,
		LastResult: r.lastResult,
		LastError:  r.lastError,
		Canceled:   r.canceled,
	}
	return out
}

func formatReconcileTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
