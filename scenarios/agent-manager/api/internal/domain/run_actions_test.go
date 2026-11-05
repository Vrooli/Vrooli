package domain

import (
	"testing"
	"time"
)

func TestRunActionsFor_DefaultAllowlist(t *testing.T) {
	run := &Run{
		Tag:    "scenario-to-cloud-investigation",
		Status: RunStatusComplete,
	}

	actions := RunActionsFor(run, RunActionContext{})

	if !actions.CanApplyInvestigation {
		t.Fatalf("expected CanApplyInvestigation to be true for default allowlist")
	}
}

func TestRunActionsFor_StatusGates(t *testing.T) {
	run := &Run{
		Tag:    "investigation",
		Status: RunStatusRunning,
	}

	actions := RunActionsFor(run, RunActionContext{})

	if actions.CanDelete {
		t.Fatalf("expected CanDelete to be false for running runs")
	}
	if !actions.CanStop {
		t.Fatalf("expected CanStop to be true for running runs")
	}
	if actions.CanApplyInvestigation {
		t.Fatalf("expected CanApplyInvestigation to be false for non-complete runs")
	}
}

func TestCanContinueRun(t *testing.T) {
	now := time.Now()
	run := &Run{
		Status:    RunStatusComplete,
		SessionID: "session-123",
		StartedAt: &now,
	}

	allowed, reason := CanContinueRun(run)
	if !allowed {
		t.Fatalf("expected continuation to be allowed")
	}
	if reason != "" {
		t.Fatalf("expected empty reason when allowed, got %q", reason)
	}

	run.Status = RunStatusRunning
	allowed, reason = CanContinueRun(run)
	if allowed {
		t.Fatalf("expected continuation to be disallowed while running")
	}
	if reason == "" {
		t.Fatalf("expected non-empty reason when disallowed due to status")
	}

	run.Status = RunStatusComplete
	run.SessionID = ""
	allowed, reason = CanContinueRun(run)
	if allowed {
		t.Fatalf("expected continuation to be disallowed without session ID")
	}
	if reason == "" {
		t.Fatalf("expected non-empty reason when disallowed due to missing session ID")
	}
}

func TestRunActionsFor_ContinueReason(t *testing.T) {
	// Running run with session ID: reason should explain it's in progress.
	run := &Run{
		Status:    RunStatusRunning,
		SessionID: "sess-1",
	}
	actions := RunActionsFor(run, RunActionContext{})
	if actions.CanContinue {
		t.Fatal("expected CanContinue to be false for running run")
	}
	if actions.CanContinueReason == "" {
		t.Fatal("expected CanContinueReason to be populated for running run")
	}

	// Completed run without session ID: reason should explain missing session.
	run.Status = RunStatusComplete
	run.SessionID = ""
	actions = RunActionsFor(run, RunActionContext{})
	if actions.CanContinue {
		t.Fatal("expected CanContinue to be false without session ID")
	}
	if actions.CanContinueReason == "" {
		t.Fatal("expected CanContinueReason to be populated without session ID")
	}

	// Completed run with session ID: continuation allowed, no reason.
	run.SessionID = "sess-1"
	actions = RunActionsFor(run, RunActionContext{})
	if !actions.CanContinue {
		t.Fatal("expected CanContinue to be true")
	}
	if actions.CanContinueReason != "" {
		t.Fatalf("expected empty CanContinueReason when allowed, got %q", actions.CanContinueReason)
	}

	// Failed run with session ID (e.g. after timeout): continuation allowed.
	run.Status = RunStatusFailed
	run.SessionID = "sess-timeout"
	actions = RunActionsFor(run, RunActionContext{})
	if !actions.CanContinue {
		t.Fatal("expected CanContinue to be true for failed run with session ID")
	}
	if actions.CanContinueReason != "" {
		t.Fatalf("expected empty CanContinueReason for failed run with session ID, got %q", actions.CanContinueReason)
	}

	// Completed runner turn with failed sandbox finalization: continuation is
	// allowed because finalization is not runner activity.
	run.Status = RunStatusComplete
	run.SessionID = "sess-finalization-failed"
	run.FinalizationStatus = RunFinalizationStatusFailed
	run.FinalizationError = "checkpoint unavailable"
	actions = RunActionsFor(run, RunActionContext{})
	if !actions.CanContinue {
		t.Fatal("expected CanContinue to be true when only finalization failed")
	}
	if actions.FinalizationWarning == "" {
		t.Fatal("expected finalization warning to be populated")
	}
	if !actions.CanRetryFinalization {
		t.Fatal("expected CanRetryFinalization when finalization failed")
	}
}
