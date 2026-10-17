package indexcontrol

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"
)

const (
	defaultChangeBatch = 256
	periodicAudit      = 5 * time.Minute
)

type Coordinator struct {
	Jobs       JobStore
	Source     ChangeSource
	Processor  Processor
	Lifecycle  GenerationLifecycle
	Aliases    AliasController
	Validator  GenerationValidator
	Promotions PromotionStore
	Clock      Clock
	BatchSize  int
}

func (coordinator *Coordinator) Reconcile(ctx context.Context, generation string) (Job, error) {
	if err := coordinator.ready(); err != nil {
		return Job{}, err
	}
	if generation == "" {
		var err error
		generation, err = coordinator.Lifecycle.Active(ctx)
		if err != nil {
			return Job{}, err
		}
	}
	now := coordinator.Clock.Now().UTC()
	job := Job{ID: "reconcile-" + strconv.FormatInt(now.UnixNano(), 10), Kind: "reconcile", State: "running", Generation: generation, CreatedAt: now, UpdatedAt: now}
	if err := coordinator.Jobs.Create(ctx, job); err != nil {
		return Job{}, err
	}
	return coordinator.run(ctx, job)
}

func (coordinator *Coordinator) StartShadow(ctx context.Context, generation string) (Job, error) {
	if err := coordinator.ready(); err != nil {
		return Job{}, err
	}
	if generation == "" {
		return Job{}, fmt.Errorf("shadow generation is required")
	}
	if err := coordinator.Lifecycle.BeginShadow(ctx, generation); err != nil {
		return Job{}, err
	}
	job, err := coordinator.Reconcile(ctx, generation)
	if err != nil {
		return job, err
	}
	if err := coordinator.Validator.Validate(ctx, generation); err != nil {
		job.State, job.Error, job.UpdatedAt = "failed", err.Error(), coordinator.Clock.Now().UTC()
		_ = coordinator.Jobs.Update(context.WithoutCancel(ctx), job)
		return job, err
	}
	if err := coordinator.Lifecycle.CompleteShadow(ctx, generation); err != nil {
		return job, err
	}
	return job, nil
}

func (coordinator *Coordinator) run(ctx context.Context, job Job) (Job, error) {
	batchSize := coordinator.BatchSize
	if batchSize <= 0 || batchSize > 4096 {
		batchSize = defaultChangeBatch
	}
	for {
		if err := ctx.Err(); err != nil {
			return coordinator.finishCancelled(job, err)
		}
		current, err := coordinator.Jobs.Get(ctx, job.ID)
		if err != nil {
			return job, err
		}
		if current.CancellationRequested {
			return coordinator.finishCancelled(current, context.Canceled)
		}
		batch, err := coordinator.Source.Changes(ctx, job.Cursor, batchSize)
		if err != nil {
			return coordinator.finishFailed(job, err)
		}
		changes := deduplicateChanges(batch.Changes)
		if len(changes) > batchSize {
			return coordinator.finishFailed(job, fmt.Errorf("change source exceeded batch limit: %d > %d", len(changes), batchSize))
		}
		if len(changes) > 0 {
			processed, err := coordinator.Processor.Apply(ctx, job.Generation, changes)
			if err != nil {
				return coordinator.finishFailed(job, err)
			}
			job.Progress += processed
		}
		job.Cursor = batch.NextCursor
		job.UpdatedAt = coordinator.Clock.Now().UTC()
		if batch.Done {
			job.State = "succeeded"
		}
		if err := coordinator.Jobs.Update(ctx, job); err != nil {
			return job, err
		}
		if batch.Done {
			return job, nil
		}
		if batch.NextCursor == batch.Cursor {
			return coordinator.finishFailed(job, fmt.Errorf("change cursor did not advance"))
		}
	}
}

func (coordinator *Coordinator) Promote(ctx context.Context, generation string) error {
	if err := coordinator.ready(); err != nil {
		return err
	}
	if err := coordinator.Validator.Validate(ctx, generation); err != nil {
		return err
	}
	current, err := coordinator.Lifecycle.Active(ctx)
	if err != nil {
		return err
	}
	now := coordinator.Clock.Now().UTC()
	id := "promote-" + strconv.FormatInt(now.UnixNano(), 10)
	if err := coordinator.Promotions.Prepare(ctx, id, current, generation, now); err != nil {
		return err
	}
	if err := coordinator.Aliases.Promote(ctx, generation); err != nil {
		_ = coordinator.Promotions.Transition(context.WithoutCancel(ctx), id, "failed", err.Error(), coordinator.Clock.Now())
		return err
	}
	if err := coordinator.Promotions.Transition(ctx, id, "alias_promoted", "", coordinator.Clock.Now()); err != nil {
		_ = coordinator.Aliases.Rollback(context.WithoutCancel(ctx), current)
		return err
	}
	if err := coordinator.Lifecycle.Activate(ctx, generation); err != nil {
		_ = coordinator.Aliases.Rollback(context.WithoutCancel(ctx), current)
		_ = coordinator.Promotions.Transition(context.WithoutCancel(ctx), id, "rolled_back", err.Error(), coordinator.Clock.Now())
		return err
	}
	return coordinator.Promotions.Transition(ctx, id, "committed", "", coordinator.Clock.Now())
}

func (coordinator *Coordinator) Rollback(ctx context.Context, generation string) error {
	if err := coordinator.ready(); err != nil {
		return err
	}
	current, err := coordinator.Lifecycle.Active(ctx)
	if err != nil {
		return err
	}
	if err := coordinator.Aliases.Rollback(ctx, generation); err != nil {
		return err
	}
	if err := coordinator.Lifecycle.Rollback(ctx, generation); err != nil {
		_ = coordinator.Aliases.Promote(context.WithoutCancel(ctx), current)
		return err
	}
	return nil
}

func (coordinator *Coordinator) Cancel(ctx context.Context, id string) error {
	if err := coordinator.ready(); err != nil {
		return err
	}
	return coordinator.Jobs.RequestCancel(ctx, id, coordinator.Clock.Now().UTC())
}

func (coordinator *Coordinator) Cleanup(context.Context) error { return nil }

func (coordinator *Coordinator) finishCancelled(job Job, cause error) (Job, error) {
	job.State, job.CancellationRequested, job.Error, job.UpdatedAt = "cancelled", true, cause.Error(), coordinator.Clock.Now().UTC()
	if err := coordinator.Jobs.Update(context.Background(), job); err != nil {
		return job, errors.Join(cause, err)
	}
	return job, cause
}

func (coordinator *Coordinator) finishFailed(job Job, cause error) (Job, error) {
	job.State, job.Error, job.UpdatedAt = "failed", cause.Error(), coordinator.Clock.Now().UTC()
	if err := coordinator.Jobs.Update(context.Background(), job); err != nil {
		return job, errors.Join(cause, err)
	}
	return job, cause
}

func (coordinator *Coordinator) ready() error {
	if coordinator == nil || coordinator.Jobs == nil || coordinator.Source == nil || coordinator.Processor == nil || coordinator.Lifecycle == nil || coordinator.Aliases == nil || coordinator.Validator == nil || coordinator.Promotions == nil || coordinator.Clock == nil {
		return fmt.Errorf("reconciliation coordinator dependencies are incomplete")
	}
	return nil
}

func deduplicateChanges(changes []Change) []Change {
	byPath := make(map[string]Change, len(changes))
	for _, change := range changes {
		if change.Path != "" {
			byPath[change.Path] = change
		}
	}
	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	result := make([]Change, 0, len(paths))
	for _, path := range paths {
		result = append(result, byPath[path])
	}
	return result
}

var _ Reconciler = (*Coordinator)(nil)

type bufferedEvent struct {
	change Change
	first  time.Time
	due    time.Time
}

type Debouncer struct {
	mu       sync.Mutex
	events   map[string]bufferedEvent
	delay    time.Duration
	maxDelay time.Duration
}

func NewDebouncer(delay, maxDelay time.Duration) *Debouncer {
	if delay <= 0 {
		delay = 500 * time.Millisecond
	}
	if maxDelay <= 0 || maxDelay > 10*time.Second {
		maxDelay = 10 * time.Second
	}
	return &Debouncer{events: map[string]bufferedEvent{}, delay: delay, maxDelay: maxDelay}
}

func (debouncer *Debouncer) Add(now time.Time, change Change) {
	debouncer.mu.Lock()
	defer debouncer.mu.Unlock()
	event, exists := debouncer.events[change.Path]
	if !exists {
		event.first = now
	}
	event.change = change
	event.due = now.Add(debouncer.delay)
	if latest := event.first.Add(debouncer.maxDelay); event.due.After(latest) {
		event.due = latest
	}
	debouncer.events[change.Path] = event
}

func (debouncer *Debouncer) Ready(now time.Time, limit int) []Change {
	debouncer.mu.Lock()
	defer debouncer.mu.Unlock()
	if limit <= 0 || limit > 4096 {
		limit = defaultChangeBatch
	}
	paths := make([]string, 0, len(debouncer.events))
	for path, event := range debouncer.events {
		if !event.due.After(now) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	if len(paths) > limit {
		paths = paths[:limit]
	}
	result := make([]Change, 0, len(paths))
	for _, path := range paths {
		result = append(result, debouncer.events[path].change)
		delete(debouncer.events, path)
	}
	return result
}

func AuditDue(last, now time.Time) bool {
	return last.IsZero() || !now.Before(last.Add(periodicAudit))
}
