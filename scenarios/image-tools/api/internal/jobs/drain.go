package jobs

import (
	"context"
	"errors"
	"sort"
	"time"
)

// Draining for a restart.
//
// The process is replaced while the job table survives. Work that has computed
// nothing yet (queued, or waiting inside admission for capacity or model
// weights) is not worth holding the replacement for: it goes back on the queue,
// persisted, and the next process runs it. Only jobs actually computing are
// given drain time, and Active reports them with their progress.

// errPausedForDrain ends an admission that began just as a drain started.
var errPausedForDrain = errors.New("job paused for a restart before it started computing")

// ActiveJob is a job that holds the drain: it is computing, or being canceled.
type ActiveJob struct {
	ID        string
	Operation string
	State     State
	Progress  int
	Message   string
	StartedAt time.Time
}

// entryResumable reports whether the entry can be started again by a later
// process. The caller holds entry.mu.
func (m *Manager) entryResumable(entry *jobEntry) bool {
	return entry.runner == nil && !entry.job.TestMode && m.canResume(entry.job.Operation)
}

// entryRestartable reports whether the entry may be stopped mid-run and started
// again with no loss. The caller holds entry.mu.
func (m *Manager) entryRestartable(entry *jobEntry) bool {
	return m.entryResumable(entry) && m.canRestart(entry.job.Operation)
}

// Quiesce stops the scheduler from starting work and puts every resumable job
// that is inside admission, and every restartable job that is running, back on
// the queue. Other jobs that are computing keep running; they are reported by
// Active until they finish. It is safe to call repeatedly and is undone by
// Resume.
func (m *Manager) Quiesce(context.Context) error {
	// The flag is set before the scan, so a job that begins admission later
	// sees it and stays queued, and one that began earlier is found below.
	m.draining.Store(true)
	m.mu.Lock()
	var cancels []context.CancelFunc
	for _, entry := range m.entries {
		entry.mu.Lock()
		pausable := (entry.job.State == StateAdmitting && m.entryResumable(entry)) ||
			(entry.job.State == StateRunning && m.entryRestartable(entry))
		if pausable && entry.cancel != nil && entry.move == nil && !entry.pause {
			entry.pause = true
			cancels = append(cancels, entry.cancel)
		}
		entry.mu.Unlock()
	}
	m.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
	return nil
}

// Resume undoes Quiesce when a prepared replacement is abandoned.
func (m *Manager) Resume(context.Context) error {
	m.draining.Store(false)
	m.signal()
	return nil
}

// pauseForDrain puts an admitting job whose admission was cancelled for the
// drain back on the queue. It reports false when the job is no longer
// admitting (it was canceled meanwhile) or the new state cannot be recorded.
func (m *Manager) pauseForDrain(entry *jobEntry) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry.mu.Lock()
	defer entry.mu.Unlock()
	entry.pause = false
	if entry.job.State != StateAdmitting && entry.job.State != StateRunning {
		return false
	}
	previous := entry.job
	if err := entry.job.transition(StateQueued, m.clock.Now(), "paused for a restart; it resumes automatically afterwards"); err != nil {
		return false
	}
	entry.job.Progress, entry.job.StartedAt = 0, nil
	if err := m.persistLocked(entry); err != nil {
		entry.job = previous
		return false
	}
	entry.cancel = nil
	entry.retryAt = time.Time{}
	entry.capacityRefusals = 0
	m.pending = append(m.pending, entry)
	return true
}

// Active lists the jobs that hold the drain, longest running first. A job that
// is still inside admission counts until it has been returned to the queue,
// because it holds capacity leases until then; one that cannot be resumed also
// counts, since it would be lost.
func (m *Manager) Active() []ActiveJob {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []ActiveJob
	for _, entry := range m.entries {
		entry.mu.Lock()
		j := entry.job
		holds := j.State == StateRunning || j.State == StateCanceling ||
			(j.State == StateAdmitting && (entry.pause || !m.entryResumable(entry)))
		entry.mu.Unlock()
		if !holds {
			continue
		}
		a := ActiveJob{ID: j.ID, Operation: j.Operation, State: j.State, Progress: j.Progress, Message: j.Message}
		if j.StartedAt != nil {
			a.StartedAt = *j.StartedAt
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, k int) bool { return out[i].StartedAt.Before(out[k].StartedAt) })
	return out
}

// releaseResumable closes the in-memory wait of queued jobs a later process
// will run, once the scheduler has stopped. Their stored state stays queued.
func (m *Manager) releaseResumable() {
	m.mu.Lock()
	left := m.pending
	m.pending = nil
	m.mu.Unlock()
	for _, entry := range left {
		entry.mu.Lock()
		job := entry.job
		entry.mu.Unlock()
		if !job.State.Terminal() {
			m.finalizeEntry(entry, job)
		}
	}
}
