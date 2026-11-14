package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrCapacityBusy means no work started and the scheduler may retry admission.
// It must never be returned for a transport error or failed claim rollback.
var ErrCapacityBusy = errors.New("jobs: capacity is busy")

// ErrReleaseFailed means owned effects could not be reconciled. A canceled job
// with this error is failed, never presented as cleaned up.
var ErrReleaseFailed = errors.New("jobs: resource cleanup failed")

// Admission transfers exact resource ownership to the job scheduler. Context
// may include a bound device or prepared worker. Release must stop work before
// relinquishing its claims and must be safe with an already-canceled context.
type Admission struct {
	Context context.Context
	Release func(context.Context) error
	// Meta carries admission facts, including cleanup results. The scheduler
	// merges it only after Release has returned.
	Meta map[string]string
}

type Admitter func(context.Context, Job) (Admission, error)

const (
	admissionRetry = 500 * time.Millisecond
	priorityAge    = 30 * time.Second
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
	timer := time.NewTicker(admissionRetry)
	defer timer.Stop()
	for {
		select {
		case <-m.baseCtx.Done():
			m.cancelPending()
			return
		case <-m.wake:
		case <-timer.C:
		}
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

func (m *Manager) dequeue() *jobEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started || m.baseCtx.Err() != nil {
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
	entry.mu.Unlock()
	if lane == LaneNetwork {
		m.networkActive++
	} else {
		m.localActive++
	}
	return entry
}

func (m *Manager) finishedSlot(entry *jobEntry) {
	m.mu.Lock()
	entry.mu.Lock()
	lane := entry.job.Lane
	entry.mu.Unlock()
	if lane == LaneNetwork {
		m.networkActive--
	} else {
		m.localActive--
	}
	m.mu.Unlock()
	m.signal()
}

func (m *Manager) requeue(entry *jobEntry, message string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if !m.started || m.baseCtx.Err() != nil || entry.job.State != StateAdmitting {
		return false
	}
	if err := entry.job.transition(StateQueued, m.clock.Now(), message); err != nil {
		return false
	}
	if err := m.persistLocked(entry); err != nil {
		return false
	}
	entry.cancel = nil
	entry.retryAt = m.clock.Now().Add(admissionRetry)
	m.pending = append(m.pending, entry)
	return true
}

func (m *Manager) cancelPending() {
	m.mu.Lock()
	pending := m.pending
	m.pending = nil
	m.mu.Unlock()
	for _, entry := range pending {
		entry.mu.Lock()
		id := entry.job.ID
		entry.mu.Unlock()
		_ = m.Cancel(id)
	}
}
