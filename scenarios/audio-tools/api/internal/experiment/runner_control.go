//go:build audio_lab

package experiment

import (
	"context"
	"fmt"
	"time"
)

func (m *Manager) emitQueued(e *entry, ahead int) {
	if ahead < 0 {
		ahead = 0
	}
	message := "queued"
	if ahead == 1 {
		message = "queued; 1 experiment ahead"
	} else if ahead > 1 {
		message = fmt.Sprintf("queued; %d experiments ahead", ahead)
	}
	m.emit(e, StatusQueued, 0, message)
}

func (m *Manager) markDequeued(e *entry) {
	m.removeQueued(e)
	m.emitQueuePositions()
}

func (m *Manager) removeQueued(target *entry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.queued {
		if e == target {
			m.queued = append(m.queued[:i], m.queued[i+1:]...)
			return
		}
	}
}

func (m *Manager) emitQueuePositions() {
	m.mu.Lock()
	queued := append([]*entry(nil), m.queued...)
	m.mu.Unlock()
	for i, e := range queued {
		m.emitQueued(e, i)
	}
}

func (m *Manager) Get(ctx context.Context, id string) (Experiment, error) {
	m.mu.Lock()
	e, ok := m.entries[id]
	m.mu.Unlock()
	if ok {
		e.mu.Lock()
		exp := e.exp
		e.mu.Unlock()
		return exp, nil
	}
	return m.service.GetExperiment(ctx, id)
}

func (m *Manager) Wait(ctx context.Context, id string) (Experiment, error) {
	m.mu.Lock()
	e, ok := m.entries[id]
	m.mu.Unlock()
	if !ok {
		return m.service.GetExperiment(ctx, id)
	}
	select {
	case <-e.done:
		e.mu.Lock()
		exp := e.exp
		e.mu.Unlock()
		return exp, nil
	case <-ctx.Done():
		return Experiment{}, ctx.Err()
	}
}

func (m *Manager) List(ctx context.Context, filter ListFilter) ([]Experiment, error) {
	return m.service.ListExperiments(ctx, filter)
}

func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	e, ok := m.entries[id]
	m.mu.Unlock()
	if !ok {
		exp, err := m.service.GetExperiment(context.Background(), id)
		if err != nil {
			return err
		}
		if exp.Status.Terminal() {
			return nil
		}
		now := m.clock.Now().UTC()
		if err := exp.Transition(StatusCanceled); err != nil {
			return err
		}
		exp.FinishedAt = &now
		return m.service.repo.UpdateExperiment(m.baseCtx, exp)
	}
	e.mu.Lock()
	if e.exp.Status.Terminal() {
		e.mu.Unlock()
		return nil
	}
	if e.exp.Status == StatusRunning && e.cancel != nil {
		cancel := e.cancel
		e.mu.Unlock()
		cancel()
		select {
		case <-e.done:
			return nil
		case <-time.After(cancelWaitTimeout):
			now := m.clock.Now().UTC()
			e.mu.Lock()
			if e.exp.Status.Terminal() {
				e.mu.Unlock()
				return nil
			}
			if err := e.exp.Transition(StatusCanceled); err != nil {
				e.mu.Unlock()
				return err
			}
			e.exp.FinishedAt = &now
			final := e.exp
			e.mu.Unlock()
			if err := m.service.repo.UpdateExperiment(m.baseCtx, final); err != nil {
				m.logger.Printf("experiment %s: persist cancel-timeout status: %v", final.ID, err)
			}
			m.finalizeEntry(e, final)
		}
		return nil
	}
	now := m.clock.Now().UTC()
	if err := e.exp.Transition(StatusCanceled); err != nil {
		e.mu.Unlock()
		return err
	}
	e.exp.FinishedAt = &now
	final := e.exp
	e.mu.Unlock()
	if err := m.service.repo.UpdateExperiment(m.baseCtx, final); err != nil {
		m.logger.Printf("experiment %s: persist cancel status: %v", final.ID, err)
	}
	m.finalizeEntry(e, final)
	m.removeQueued(e)
	m.emitQueuePositions()
	return nil
}

func (m *Manager) Subscribe(id string) (<-chan ProgressEvent, func(), error) {
	m.mu.Lock()
	e, ok := m.entries[id]
	m.mu.Unlock()
	if !ok {
		exp, err := m.service.GetExperiment(context.Background(), id)
		if err != nil {
			return nil, nil, err
		}
		ch := make(chan ProgressEvent, 1)
		ch <- ProgressEvent{ExperimentID: exp.ID, Status: exp.Status, Progress: terminalProgress(exp.Status), Message: finalMessage(exp), At: m.clock.Now().UTC()}
		close(ch)
		return ch, func() {}, nil
	}
	sub := &subscriber{ch: make(chan ProgressEvent, 64)}
	e.mu.Lock()
	if e.exp.Status.Terminal() {
		ev := ProgressEvent{ExperimentID: e.exp.ID, Status: e.exp.Status, Progress: terminalProgress(e.exp.Status), Message: finalMessage(e.exp), At: m.clock.Now().UTC()}
		e.mu.Unlock()
		sub.ch <- ev
		close(sub.ch)
		sub.closed = true
		return sub.ch, func() {}, nil
	}
	if e.last != nil {
		trySend(sub.ch, *e.last)
	}
	e.subs = append(e.subs, sub)
	e.mu.Unlock()

	unsub := func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		for i, s := range e.subs {
			if s == sub {
				e.subs = append(e.subs[:i], e.subs[i+1:]...)
				if !s.closed {
					close(s.ch)
					s.closed = true
				}
				return
			}
		}
	}
	return sub.ch, unsub, nil
}

func terminalProgress(s Status) int {
	if s == StatusSucceeded {
		return 100
	}
	return 0
}

func finalMessage(exp Experiment) string {
	switch exp.Status {
	case StatusSucceeded:
		return "succeeded"
	case StatusCanceled:
		return "canceled"
	case StatusFailed:
		return exp.Error
	default:
		return string(exp.Status)
	}
}
