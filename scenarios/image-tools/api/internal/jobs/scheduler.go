package jobs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

// ErrCapacityBusy means no work started and the scheduler may retry admission.
// It must never be returned for a transport error or failed claim rollback.
var ErrCapacityBusy = errors.New("jobs: capacity is busy")

// ErrReleaseFailed means owned effects could not be reconciled. A canceled job
// with this error is failed, never presented as cleaned up.
var ErrReleaseFailed = errors.New("jobs: resource cleanup failed")

// ErrReleaseLeaksWork marks a release failure that left running work or files
// behind (a worker that did not stop, temporary output that was not verified
// removed), as opposed to ledger claims whose TTL reclaims them. Such a failure
// fails the job even when its inference succeeded.
var ErrReleaseLeaksWork = errors.New("jobs: release left work or files behind")

// ErrExecutionTimeLimit means the job's own ExecutionTimeoutSeconds elapsed.
// Other deadlines (a capacity call, a lease heartbeat) keep their real cause.
var ErrExecutionTimeLimit = errors.New("jobs: execution time limit exceeded")

// Admission transfers exact resource ownership to the job scheduler. Context
// may include a bound device or prepared worker. Release must stop work before
// relinquishing its claims and must be safe with an already-canceled context.
type Admission struct {
	Context context.Context
	Release func(context.Context) error
	// ReleaseRetryable declares Release idempotent and its claims TTL-recovered:
	// parts that released are not repeated and failed parts may be retried. The
	// manager then retries a failed release with backoff (host capacity locks
	// are shared fleet-wide and briefly contended). If the budget is exhausted
	// after the work succeeded, the job keeps its result and records the
	// unconfirmed release in its meta, unless the failure wraps
	// ErrReleaseLeaksWork. Without this flag, one failed attempt fails the job.
	ReleaseRetryable bool
	// Meta carries admission facts, including cleanup results. The scheduler
	// merges it only after Release has returned.
	Meta map[string]string
}

type Admitter func(context.Context, Job) (Admission, error)

type admissionProgressKey struct{}

// WithAdmissionProgress attaches the reporter an Admitter uses for long
// preparation that happens before any lease is held (for example acquiring a
// model's weights). The scheduler installs it for every admission attempt.
func WithAdmissionProgress(ctx context.Context, report func(progress int, message string)) context.Context {
	if report == nil {
		return ctx
	}
	return context.WithValue(ctx, admissionProgressKey{}, report)
}

// AdmissionProgress returns the admission-phase progress reporter, or a no-op
// when the caller is not the job scheduler.
func AdmissionProgress(ctx context.Context) func(progress int, message string) {
	if report, ok := ctx.Value(admissionProgressKey{}).(func(int, string)); ok && report != nil {
		return report
	}
	return func(int, string) {}
}

const (
	admissionRetry    = 500 * time.Millisecond
	maxAdmissionRetry = 16 * time.Second
	priorityAge       = 30 * time.Second
)

func (m *Manager) acquire(ctx context.Context, job Job) (admission Admission, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("job admission panicked: %v", recovered)
		}
	}()
	if m.admit == nil {
		return Admission{Context: ctx}, nil
	}
	return m.admit(ctx, job)
}

func (m *Manager) signal() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
}

// schedule is the only dispatch loop. It shares priority and aging across all
// operations and retains separate local/network ceilings, not FIFO lanes.
func (m *Manager) schedule() {
	defer m.wg.Done()
	timer := m.clock.NewTicker(admissionRetry)
	defer timer.Stop()
	for {
		select {
		case <-m.baseCtx.Done():
			m.cancelPending()
			return
		case <-m.wake:
		case <-timer.C():
		}
		m.rerouteDue()
		for {
			entry := m.dequeue()
			if entry == nil {
				break
			}
			m.wg.Add(1)
			go m.execute(entry)
		}
	}
}

const (
	// rerouteRecheck paces how often a waiting job is offered to the Rerouter.
	rerouteRecheck = time.Second
	// rerouteRetry paces the next offer after the Rerouter wanted to move a job
	// but could not (for example a spend cap), so a refusal is not re-evaluated
	// against the ledger every dispatch tick.
	rerouteRetry = 30 * time.Second
)

// errAdmissionRerouted ends an admission that was cancelled to move its job.
var errAdmissionRerouted = errors.New("admission cancelled to re-route the job")

// rerouteDue offers every local job that has not started running to the
// Rerouter: queued jobs, and jobs inside admission (waiting on capacity or
// preparing a model, which is where long local waits actually happen). It runs
// on the dispatch goroutine, the only place jobs leave the pending queue, so a
// queued job offered here cannot start admission while its move is applied.
func (m *Manager) rerouteDue() {
	if m.reroute == nil {
		return
	}
	now := m.clock.Now()
	m.mu.Lock()
	var due []*jobEntry
	for _, entry := range m.entries {
		entry.mu.Lock()
		waiting := entry.job.State == StateQueued || (entry.job.State == StateAdmitting && entry.cancel != nil && entry.move == nil)
		eligible := waiting && entry.job.Lane != LaneNetwork && !entry.rerouteAt.After(now)
		entry.mu.Unlock()
		if eligible {
			due = append(due, entry)
		}
	}
	m.mu.Unlock()
	for _, entry := range due {
		m.rerouteEntry(entry, now)
	}
}

func (m *Manager) rerouteEntry(entry *jobEntry, now time.Time) {
	entry.mu.Lock()
	job, values := entry.job, entry.routeValues
	entry.mu.Unlock()
	ctx := valueOverlayContext{Context: m.baseCtx, values: values}
	move, err := m.runReroute(ctx, job)
	next := now.Add(rerouteRecheck)
	if err != nil {
		log.Printf("jobs: reroute of %s declined: %v", job.ID, err)
		next = now.Add(rerouteRetry)
	}
	if move == nil {
		entry.mu.Lock()
		entry.rerouteAt = next
		entry.mu.Unlock()
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	entry.mu.Lock()
	defer entry.mu.Unlock()
	entry.rerouteAt = next
	if entry.job.State == StateAdmitting {
		// Claim the job and cancel its admission. execute releases the leases
		// and then applies the move; startRunning refuses a claimed job, so it
		// cannot begin running in between.
		if entry.cancel == nil || entry.move != nil {
			return
		}
		entry.move = move
		log.Printf("jobs: cancelling admission of %s to re-route it: %s", job.ID, move.Message)
		entry.cancel()
		return
	}
	pending := false
	for _, queued := range m.pending {
		if queued == entry {
			pending = true
			break
		}
	}
	if !pending || entry.job.State != StateQueued {
		return
	}
	previous := entry.job
	entry.job.Payload, entry.job.Lane, entry.job.EstimatedSeconds = move.Payload, normalizeLane(move.Lane), move.EstimatedSeconds
	entry.job.Message = move.Message
	if err := m.persistLocked(entry); err != nil {
		// The move was not recorded, so it must not be applied: a restart or a
		// reader would otherwise disagree with what runs.
		entry.job.Payload, entry.job.Lane, entry.job.EstimatedSeconds, entry.job.Message = previous.Payload, previous.Lane, previous.EstimatedSeconds, previous.Message
		log.Printf("jobs: reroute of %s not recorded: %v", job.ID, err)
	}
}

// applyAdmissionMove re-queues a job whose admission was cancelled for a
// re-route, on its new payload and lane, after its leases were released. It
// reports false (leaving the job unchanged) when the move cannot be recorded or
// the job was cancelled meanwhile.
func (m *Manager) applyAdmissionMove(entry *jobEntry) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry.mu.Lock()
	defer entry.mu.Unlock()
	return m.applyMoveLocked(entry)
}

// applyMoveLocked is applyAdmissionMove for a caller holding m.mu and entry.mu.
func (m *Manager) applyMoveLocked(entry *jobEntry) bool {
	move := entry.move
	entry.move = nil
	if move == nil || !m.started || m.baseCtx.Err() != nil || (entry.job.State != StateAdmitting && entry.job.State != StateRunning) {
		return false
	}
	previous := entry.job
	entry.job.Payload, entry.job.Lane, entry.job.EstimatedSeconds = move.Payload, normalizeLane(move.Lane), move.EstimatedSeconds
	if len(move.Meta) > 0 {
		merged := make(map[string]string, len(previous.Meta)+len(move.Meta))
		for k, v := range previous.Meta {
			merged[k] = v
		}
		for k, v := range move.Meta {
			merged[k] = v
		}
		entry.job.Meta = merged
	}
	entry.job.StartedAt, entry.job.Progress = nil, 0
	if err := entry.job.transition(StateQueued, m.clock.Now(), move.Message); err != nil {
		entry.job = previous
		return false
	}
	if err := m.persistLocked(entry); err != nil {
		entry.job = previous
		log.Printf("jobs: reroute of %s not recorded: %v", entry.job.ID, err)
		return false
	}
	entry.cancel = nil
	entry.retryAt = time.Time{}
	entry.rerouteAt = m.clock.Now().Add(rerouteRecheck)
	entry.capacityRefusals = 0
	m.pending = append(m.pending, entry)
	return true
}

func (m *Manager) runReroute(ctx context.Context, job Job) (move *Reroute, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			move, err = nil, fmt.Errorf("job reroute panicked: %v", recovered)
		}
	}()
	return m.reroute(ctx, job)
}

func (m *Manager) dequeue() *jobEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started || m.baseCtx.Err() != nil || m.draining.Load() {
		return nil
	}
	now := m.clock.Now()
	best, bestRank := -1, -1
	for i := 0; i < len(m.pending); {
		entry := m.pending[i]
		entry.mu.Lock()
		job, retryAt := entry.job, entry.retryAt
		entry.mu.Unlock()
		if job.State.Terminal() {
			m.pending = append(m.pending[:i], m.pending[i+1:]...)
			continue
		}
		available := m.localActive < m.cpuN
		if job.Lane == LaneNetwork {
			available = m.networkActive < m.netN
		}
		rank := priorityRank(job.Priority) + int(now.Sub(job.CreatedAt)/priorityAge)
		if available && !retryAt.After(now) && rank > bestRank {
			best, bestRank = i, rank
		}
		i++
	}
	if best < 0 {
		return nil
	}
	entry := m.pending[best]
	m.pending = append(m.pending[:best], m.pending[best+1:]...)
	entry.mu.Lock()
	lane := entry.job.Lane
	entry.slotNetwork = lane == LaneNetwork
	entry.mu.Unlock()
	if lane == LaneNetwork {
		m.networkActive++
	} else {
		m.localActive++
	}
	return entry
}

// finishedSlot returns a dispatch slot. A local job that reached a terminal
// state has released its capacity leases, so queued local work is retried now
// instead of after its capacity backoff. A requeued job freed nothing.
func (m *Manager) finishedSlot(entry *jobEntry) {
	m.mu.Lock()
	entry.mu.Lock()
	// The slot is the ceiling this execution took, not the job's current lane:
	// a re-route out of admission changes the lane while the slot is held. A
	// moved job freed local capacity just as a finished one did.
	network := entry.slotNetwork
	released := entry.job.State.Terminal() || (entry.job.State == StateQueued && entry.job.Lane == LaneNetwork && !network)
	entry.mu.Unlock()
	if network {
		m.networkActive--
	} else {
		m.localActive--
		if released {
			for _, waiting := range m.pending {
				waiting.mu.Lock()
				if waiting.job.Lane != LaneNetwork {
					waiting.retryAt = time.Time{}
				}
				waiting.mu.Unlock()
			}
		}
	}
	m.mu.Unlock()
	m.signal()
}

// capacityBackoff doubles the admission retry delay after each consecutive
// capacity refusal, up to maxAdmissionRetry. Each attempt consults the shared
// host broker, so a saturated host must not be polled at a fixed short rate.
func capacityBackoff(refusals int) time.Duration {
	delay := admissionRetry
	for i := 0; i < refusals && delay < maxAdmissionRetry; i++ {
		delay *= 2
	}
	return min(delay, maxAdmissionRetry)
}

func (m *Manager) requeue(entry *jobEntry, message string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if !m.started || m.baseCtx.Err() != nil || entry.job.State != StateAdmitting {
		return false
	}
	if entry.move != nil && m.applyMoveLocked(entry) {
		// A re-route claimed the job while this refusal was being handled: it
		// rejoins the queue on its new route instead of retrying locally.
		return true
	}
	if err := entry.job.transition(StateQueued, m.clock.Now(), message); err != nil {
		return false
	}
	if err := m.persistLocked(entry); err != nil {
		return false
	}
	entry.cancel = nil
	entry.retryAt = m.clock.Now().Add(capacityBackoff(entry.capacityRefusals))
	entry.capacityRefusals++
	m.pending = append(m.pending, entry)
	return true
}

func (m *Manager) cancelPending() {
	m.mu.Lock()
	pending := m.pending
	m.pending = nil
	m.mu.Unlock()
	var keep []*jobEntry
	for _, entry := range pending {
		entry.mu.Lock()
		id, resumable := entry.job.ID, m.entryResumable(entry)
		entry.mu.Unlock()
		if resumable {
			// Queued work that a later process can run stays queued.
			keep = append(keep, entry)
			continue
		}
		_ = m.Cancel(id)
	}
	if len(keep) > 0 {
		m.mu.Lock()
		m.pending = append(m.pending, keep...)
		m.mu.Unlock()
	}
}
