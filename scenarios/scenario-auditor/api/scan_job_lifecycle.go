package main

import (
	"sync"
	"time"
)

func newSecurityScanLifecycle(initial SecurityScanStatus) *scanJobLifecycle[SecurityScanStatus] {
	return newScanJobLifecycle(initial,
		func(status SecurityScanStatus) string { return status.Status },
		func(status SecurityScanStatus) time.Time { return status.StartedAt },
		func(status *SecurityScanStatus, state, message string) {
			status.Status, status.Message = state, message
		},
		func(status *SecurityScanStatus, elapsed float64) { status.ElapsedSeconds = elapsed },
	)
}

func newStandardsScanLifecycle(initial StandardsScanStatus) *scanJobLifecycle[StandardsScanStatus] {
	return newScanJobLifecycle(initial,
		func(status StandardsScanStatus) string { return status.Status },
		func(status StandardsScanStatus) time.Time { return status.StartedAt },
		func(status *StandardsScanStatus, state, message string) {
			status.Status, status.Message = state, message
		},
		func(status *StandardsScanStatus, elapsed float64) { status.ElapsedSeconds = elapsed },
	)
}

// scanJobLifecycle owns the synchronized status and the common transitions for
// security and standards scans. Domain status DTOs remain distinct so their
// JSON fields and results stay compatible.
type scanJobLifecycle[T any] struct {
	mu          sync.RWMutex
	status      T
	getStatus   func(T) string
	getStarted  func(T) time.Time
	setProgress func(*T, string, string)
	setElapsed  func(*T, float64)
}

func newScanJobLifecycle[T any](initial T, getStatus func(T) string, getStarted func(T) time.Time, setProgress func(*T, string, string), setElapsed func(*T, float64)) *scanJobLifecycle[T] {
	return &scanJobLifecycle[T]{status: initial, getStatus: getStatus, getStarted: getStarted, setProgress: setProgress, setElapsed: setElapsed}
}

func (l *scanJobLifecycle[T]) update(fn func(*T)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if isScanJobTerminal(l.getStatus(l.status)) {
		return
	}
	fn(&l.status)
	l.updateElapsed()
}

func (l *scanJobLifecycle[T]) snapshot() T {
	l.mu.RLock()
	defer l.mu.RUnlock()
	copy := l.status
	if !l.getStarted(copy).IsZero() && l.getStatus(copy) != "pending" {
		l.applyElapsed(&copy, time.Since(l.getStarted(copy)).Seconds())
	}
	return copy
}

func (l *scanJobLifecycle[T]) requestCancel(cancel func(), finishedErr error) error {
	l.mu.Lock()
	status := l.getStatus(l.status)
	if isScanJobTerminal(status) {
		l.mu.Unlock()
		return finishedErr
	}
	if status == "cancelling" {
		l.mu.Unlock()
		return nil
	}
	l.setProgress(&l.status, "cancelling", "Cancellation requested")
	l.updateElapsed()
	l.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (l *scanJobLifecycle[T]) finish(status string, mutate func(*T)) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if isScanJobTerminal(l.getStatus(l.status)) || (status != "completed" && status != "failed" && status != "cancelled") {
		return false
	}
	mutate(&l.status)
	l.updateElapsed()
	return true
}

func (l *scanJobLifecycle[T]) updateElapsed() {
	started := l.getStarted(l.status)
	if !started.IsZero() {
		l.applyElapsed(&l.status, time.Since(started).Seconds())
	}
}

func (l *scanJobLifecycle[T]) applyElapsed(status *T, seconds float64) {
	l.setElapsed(status, seconds)
}

func isScanJobTerminal(status string) bool {
	return status == "completed" || status == "failed" || status == "cancelled"
}
