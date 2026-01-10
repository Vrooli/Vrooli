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
	if !actions.CanExtractRecommendations {
		t.Fatalf("expected CanExtractRecommendations to be true for default allowlist")
	}
	if !actions.CanRegenerateRecommendations {
		t.Fatalf("expected CanRegenerateRecommendations to be true for default allowlist")
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

	allowed, _ := CanContinueRun(run)
	if !allowed {
		t.Fatalf("expected continuation to be allowed")
	}

	run.Status = RunStatusRunning
	allowed, _ = CanContinueRun(run)
	if allowed {
		t.Fatalf("expected continuation to be disallowed while running")
	}
}
