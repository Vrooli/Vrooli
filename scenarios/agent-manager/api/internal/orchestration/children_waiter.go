// This file waits on a parent run's direct children: the park waiter that
// wakes a parked parent, and the blocking WaitRunChildren read any caller uses.
package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// ProducerChildren parks a parent run until any of its direct child runs
// settles. The await key is the parked parent's own run ID. The park deadline
// doubles as the timer wake, and `run wake --key` is the explicit wake;
// together they are the orchestrator's three wake sources.
const ProducerChildren = "children"

// defaultChildrenPollInterval bounds how quickly a child's settled state
// reaches the parked parent. The wait itself costs no tokens; only the
// resumed turn does.
const defaultChildrenPollInterval = 10 * time.Second

// childReadLimit bounds one read of a parent's direct children.
const childReadLimit = 1000

// Children wait reasons, shared by the park wake payload and WaitRunChildren.
const (
	ChildrenWakeEnded            = "child_run_ended"
	ChildrenWakeTimer            = "timer"
	ChildrenWakeNoActiveChildren = "no_active_children"
)

// ChildRunReader is the narrow run surface the children waiters read.
type ChildRunReader interface {
	GetRun(ctx context.Context, id uuid.UUID) (*domain.Run, error)
	ListRuns(ctx context.Context, opts RunListOptions) ([]*domain.Run, error)
}

// ChildRunSummary is one child row of a children wait or wake.
type ChildRunSummary struct {
	RunID   uuid.UUID
	Status  domain.RunStatus
	EndedAt *time.Time
}

// ChildrenWaitResult is one answer to "which children settled, which are
// still live": the park wake payload and the WaitRunChildren response.
type ChildrenWaitResult struct {
	WakeReason string
	// Ended lists children that settled (terminal, or needs_review with the
	// worker exited) at or after the wait's start instant.
	Ended []ChildRunSummary
	// Active lists children that can still execute (pending, starting,
	// running or parked).
	Active []ChildRunSummary
	// ObservedAt is when the children were read; the next wait passes it as
	// its start instant so no settlement between calls is missed.
	ObservedAt time.Time
}

// observeChildren reads the parent's direct children once and classifies them
// against since.
func observeChildren(ctx context.Context, runs ChildRunReader, parentID uuid.UUID, since, now time.Time) (ChildrenWaitResult, error) {
	children, err := runs.ListRuns(ctx, RunListOptions{ListOptions: ListOptions{Limit: childReadLimit}, ParentRunID: &parentID})
	result := ChildrenWaitResult{ObservedAt: now}
	if err != nil {
		return result, fmt.Errorf("list child runs of %s: %w", parentID, err)
	}
	for _, child := range children {
		switch {
		case child == nil:
		case child.Status.IsSettled() && child.EndedAt != nil && !child.EndedAt.Before(since):
			result.Ended = append(result.Ended, summarizeChild(child))
		case child.Status.IsLive():
			result.Active = append(result.Active, summarizeChild(child))
		}
	}
	if len(result.Ended) > 0 {
		result.WakeReason = ChildrenWakeEnded
	}
	return result, nil
}

func summarizeChild(run *domain.Run) ChildRunSummary {
	return ChildRunSummary{RunID: run.ID, Status: run.Status, EndedAt: run.EndedAt}
}

// payload renders the wake message body. It lists only the children that
// settled since the last wake and the ones still live: the payload becomes a
// user message that Codex keeps verbatim through every compaction, so listing
// every child ever spawned grew the parent's context with each wake.
func (r ChildrenWaitResult) payload() (string, error) {
	type row struct {
		RunID   string `json:"run_id"`
		Status  string `json:"status"`
		EndedAt string `json:"ended_at,omitempty"`
	}
	rows := func(children []ChildRunSummary) []row {
		out := make([]row, 0, len(children))
		for _, child := range children {
			item := row{RunID: child.RunID.String(), Status: string(child.Status)}
			if child.EndedAt != nil {
				item.EndedAt = child.EndedAt.UTC().Format(time.RFC3339)
			}
			out = append(out, item)
		}
		return out
	}
	data, err := json.Marshal(map[string]any{"wake_reason": r.WakeReason, "ended": rows(r.Ended), "active": rows(r.Active)})
	return string(data), err
}

type childrenWaiter struct {
	runs     ChildRunReader
	interval time.Duration
	now      func() time.Time
}

// NewChildrenWaiter builds the waiter for ProducerChildren. A zero interval
// uses the default poll interval.
func NewChildrenWaiter(runs ChildRunReader, interval time.Duration) Waiter {
	if interval <= 0 {
		interval = defaultChildrenPollInterval
	}
	return &childrenWaiter{runs: runs, interval: interval, now: time.Now}
}

func (w *childrenWaiter) Producer() string { return ProducerChildren }

// Wait returns once a child run has settled that the parent has not yet been
// told about: one that ended after the parent's previous wake, or after the
// parent started when it has never been woken. A child that finishes while the
// parent is still deciding to park therefore wakes it immediately, and an
// agent-manager restart neither misses nor repeats a child. On deadline it
// returns the current child status with the context error, so the timer wake
// still carries the evidence.
func (w *childrenWaiter) Wait(ctx context.Context, key string) (string, error) {
	parentID, err := uuid.Parse(strings.TrimSpace(key))
	if err != nil {
		return "", domain.NewValidationError("key", fmt.Sprintf("children await key %q must be the parked parent run ID", key))
	}
	since := w.now()
	if parent, err := w.runs.GetRun(ctx, parentID); err == nil && parent != nil {
		since = unreportedSince(parent, since)
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		observed, err := observeChildren(ctx, w.runs, parentID, since, w.now())
		if err != nil && ctx.Err() == nil {
			return "", err
		}
		if observed.WakeReason == ChildrenWakeEnded {
			return observed.payload()
		}
		select {
		case <-ctx.Done():
			observed.WakeReason = ChildrenWakeTimer
			payload, _ := observed.payload()
			return payload, ctx.Err()
		case <-ticker.C:
		}
	}
}

// unreportedSince is the instant after which a child's end has not been
// delivered to the parent: its previous wake, else its start, else creation.
func unreportedSince(parent *domain.Run, fallback time.Time) time.Time {
	switch {
	case parent.LastAwaitResolvedAt != nil:
		return *parent.LastAwaitResolvedAt
	case parent.StartedAt != nil:
		return *parent.StartedAt
	case !parent.CreatedAt.IsZero():
		return parent.CreatedAt
	default:
		return fallback
	}
}

// MaxRunWaitTimeout bounds one blocking run or children wait. A caller that
// wants to wait longer calls again; each call is a fresh read, so an Agent
// Manager restart ends only the call in flight.
const MaxRunWaitTimeout = 15 * time.Minute

// Run wait outcomes.
const (
	RunWaitTerminal    = "terminal"
	RunWaitNeedsReview = "needs_review"
	RunWaitTimedOut    = "timed_out"
)

// RunWaitResult is the run as last read and why the wait returned.
type RunWaitResult struct {
	Run     *domain.Run
	Outcome string
}

// runWaitBackoff spaces the reads of a blocking wait: 1s, 2s, 4s, 8s, then
// every 10s (the children park cadence). A run that settles quickly answers
// quickly; a long wait costs one indexed read per 10 seconds.
type runWaitBackoff struct{ next, max time.Duration }

func (o *Orchestrator) newRunWaitBackoff() *runWaitBackoff {
	first := o.runWaitPoll
	if first <= 0 {
		first = time.Second
	}
	return &runWaitBackoff{next: first, max: 10 * first}
}

func (b *runWaitBackoff) delay() time.Duration {
	d := b.next
	b.next = min(2*b.next, b.max)
	return d
}

func validateRunWaitTimeout(timeout time.Duration) error {
	if timeout <= 0 || timeout > MaxRunWaitTimeout {
		return domain.NewValidationErrorWithHint("timeout_seconds", fmt.Sprintf("wait timeout must be between 1s and %s", MaxRunWaitTimeout), "call again to wait longer")
	}
	return nil
}

// blockUntil reads with backoff until check reports done or the timeout
// elapses. It returns timedOut when the bound elapsed first. Cancelling ctx
// returns its error. Waiting only reads: it never changes the observed runs.
func (o *Orchestrator) blockUntil(ctx context.Context, timeout time.Duration, check func() (bool, error)) (timedOut bool, err error) {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	backoff := o.newRunWaitBackoff()
	for {
		done, err := check()
		if err != nil || done {
			return false, err
		}
		timer := time.NewTimer(backoff.delay())
		select {
		case <-waitCtx.Done():
			timer.Stop()
			if ctx.Err() != nil {
				return false, ctx.Err()
			}
			if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				return true, nil
			}
			return false, waitCtx.Err()
		case <-timer.C:
		}
	}
}

// repositoryChildReader reads runs straight from the repository. Waiters only
// classify status, so they skip the action and session-link hydration that
// GetRun and ListRuns add for display on every poll.
type repositoryChildReader struct{ o *Orchestrator }

// StatusReader is the reader the children park waiter polls with.
func (o *Orchestrator) StatusReader() ChildRunReader { return repositoryChildReader{o: o} }

func (r repositoryChildReader) GetRun(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	run, err := r.o.runs.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, domain.NewNotFoundError("Run", id)
	}
	return run, nil
}

func (r repositoryChildReader) ListRuns(ctx context.Context, opts RunListOptions) ([]*domain.Run, error) {
	return r.o.runs.List(ctx, opts.repositoryFilter())
}

// WaitRun blocks until the run settles or timeout elapses. Any caller may
// wait; it need not be a run, parked, or launched by Agent Manager.
func (o *Orchestrator) WaitRun(ctx context.Context, id uuid.UUID, timeout time.Duration) (*RunWaitResult, error) {
	if err := validateRunWaitTimeout(timeout); err != nil {
		return nil, err
	}
	reader := repositoryChildReader{o: o}
	var observed domain.RunStatus
	timedOut, err := o.blockUntil(ctx, timeout, func() (bool, error) {
		run, err := reader.GetRun(ctx, id)
		if err != nil {
			return false, err
		}
		observed = run.Status
		return observed.IsSettled(), nil
	})
	if err != nil {
		return nil, err
	}
	// Return the display form the run read APIs return.
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	result := &RunWaitResult{Run: run, Outcome: RunWaitTerminal}
	switch {
	case timedOut:
		result.Outcome = RunWaitTimedOut
	case observed == domain.RunStatusNeedsReview:
		result.Outcome = RunWaitNeedsReview
	}
	return result, nil
}

// WaitRunChildren blocks until a direct child of parentID settles at or after
// since (the call's start when nil), no child is live, or timeout elapses. It
// returns the same classification a parked parent's children wake carries.
func (o *Orchestrator) WaitRunChildren(ctx context.Context, parentID uuid.UUID, since *time.Time, timeout time.Duration) (*ChildrenWaitResult, error) {
	if err := validateRunWaitTimeout(timeout); err != nil {
		return nil, err
	}
	reader := repositoryChildReader{o: o}
	if _, err := reader.GetRun(ctx, parentID); err != nil {
		return nil, err
	}
	from := o.now()
	if since != nil {
		from = *since
	}
	var last ChildrenWaitResult
	timedOut, err := o.blockUntil(ctx, timeout, func() (bool, error) {
		observed, err := observeChildren(ctx, reader, parentID, from, o.now())
		if err != nil {
			return false, err
		}
		if observed.WakeReason == "" && len(observed.Active) == 0 {
			observed.WakeReason = ChildrenWakeNoActiveChildren
		}
		last = observed
		return observed.WakeReason != "", nil
	})
	if err != nil {
		return nil, err
	}
	if timedOut {
		last.WakeReason = ChildrenWakeTimer
	}
	return &last, nil
}
