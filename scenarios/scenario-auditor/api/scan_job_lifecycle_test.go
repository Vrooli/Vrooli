package main

import (
	"errors"
	"testing"
	"time"
)

type testScanStatus struct {
	State   string
	Message string
	Started time.Time
	Elapsed float64
}

func newTestScanLifecycle(state string) *scanJobLifecycle[testScanStatus] {
	return newScanJobLifecycle(testScanStatus{State: state, Started: time.Now()},
		func(s testScanStatus) string { return s.State },
		func(s testScanStatus) time.Time { return s.Started },
		func(s *testScanStatus, state, message string) { s.State, s.Message = state, message },
		func(s *testScanStatus, elapsed float64) { s.Elapsed = elapsed },
	)
}

func TestScanJobLifecycleRejectsInvalidAndTerminalTransitions(t *testing.T) {
	lifecycle := newTestScanLifecycle("running")
	if lifecycle.finish("paused", func(s *testScanStatus) { s.State = "paused" }) {
		t.Fatal("finish accepted a nonterminal state")
	}
	if got := lifecycle.snapshot().State; got != "running" {
		t.Fatalf("state after invalid transition = %q, want running", got)
	}
	if !lifecycle.finish("completed", func(s *testScanStatus) { s.State = "completed" }) {
		t.Fatal("finish rejected a valid terminal transition")
	}
	if lifecycle.finish("failed", func(s *testScanStatus) { s.State = "failed" }) {
		t.Fatal("finish allowed a terminal state to be overwritten")
	}
	if got := lifecycle.snapshot().State; got != "completed" {
		t.Fatalf("state after terminal transition = %q, want completed", got)
	}
}

func TestScanJobLifecycleCancelIsIdempotentAndRejectsFinishedJobs(t *testing.T) {
	calls := 0
	lifecycle := newTestScanLifecycle("running")
	finishedErr := errors.New("domain-specific finished error")
	cancel := func() { calls++ }
	if err := lifecycle.requestCancel(cancel, finishedErr); err != nil {
		t.Fatalf("first cancel: %v", err)
	}
	if err := lifecycle.requestCancel(cancel, finishedErr); err != nil {
		t.Fatalf("repeated cancel: %v", err)
	}
	if calls != 1 {
		t.Fatalf("cancel called %d times, want 1", calls)
	}
	lifecycle.finish("cancelled", func(s *testScanStatus) { s.State = "cancelled" })
	if err := lifecycle.requestCancel(cancel, finishedErr); err != finishedErr {
		t.Fatalf("cancel after finish = %v, want original domain error", err)
	}
}

func TestSecurityAndStandardsFinishedCancelErrorsRemainDistinct(t *testing.T) {
	security := &SecurityScanJob{lifecycle: newSecurityScanLifecycle(SecurityScanStatus{Status: "completed"})}
	standards := &StandardsScanJob{lifecycle: newStandardsScanLifecycle(StandardsScanStatus{Status: "completed"})}
	if err := security.requestCancel(); err != errSecurityScanFinished {
		t.Fatalf("security finished cancel error = %v, want %v", err, errSecurityScanFinished)
	}
	if err := standards.requestCancel(); err != errStandardsScanFinished {
		t.Fatalf("standards finished cancel error = %v, want %v", err, errStandardsScanFinished)
	}
}
