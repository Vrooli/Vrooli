package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	applydomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/apply"
)

func TestApplyReplayEligibilityRequiresPositiveEvidence(t *testing.T) {
	prior := []applyRun{{
		ID: "old-local", Target: "local", Status: applyRunInterrupted,
		Items: []applyItemResult{
			{applyItem: applyItem{ID: "resource:db"}, Outcome: applyItemApplying},
			{applyItem: applyItem{ID: "tool:git"}, Outcome: applyItemFailed},
			{applyItem: applyItem{ID: "scenario:done"}, Outcome: applyItemApplied},
		},
	}, {
		ID: "other-target", Target: "remote", Items: []applyItemResult{{
			applyItem: applyItem{ID: "resource:db"}, Outcome: applyItemIndeterminate,
		}},
	}}

	blocked := applyReplayBlockers(prior, "local", []applydomain.Item{
		{ID: "resource:db", ObservedState: applyStateUnknown},
		{ID: "tool:git", ObservedState: applyStatePending},
	})
	if strings.Join(blocked, ",") != "resource:db,tool:git" {
		t.Fatalf("unresolved same-target effects = %v, want sorted IDs", blocked)
	}

	resolved := applyReplayBlockers(prior, "local", []applydomain.Item{
		{ID: "resource:db", ObservedState: applyStateSatisfied},
		{ID: "tool:git", ObservedState: "not_applicable"},
	})
	if len(resolved) != 0 {
		t.Fatalf("positive observations did not release retry: %v", resolved)
	}
	removed := applyReplayBlockers(prior, "local", nil)
	if len(removed) != 0 {
		t.Fatalf("deselected uncertain items still block unrelated work: %v", removed)
	}
}

func TestApplyRecoveryProjectionAndReplayShareDurableUncertainty(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("HOME", t.TempDir())
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z"}`)

	run := applyRun{
		ID: "interrupted-resource-run", Target: "local", Status: applyRunApplying,
		RunnerPID: 4242, Heartbeat: time.Now().UTC().Add(-2 * staleApplyHeartbeat).Format(time.RFC3339),
		Items: []applyItemResult{{
			applyItem: applyItem{ID: "resource:postgres", Kind: "resource", Name: "postgres"},
			Outcome:   applyItemApplying,
		}},
	}
	if err := persistApplyRun(run); err != nil {
		t.Fatal(err)
	}

	previousAlive := processAliveObserved
	t.Cleanup(func() { processAliveObserved = previousAlive })
	processAliveObserved = func(int) (bool, error) { return false, nil }
	projected, err := (&Server{}).getApplyRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("project interrupted run: %v", err)
	}
	if projected.Status != string(applyRunInterrupted) || len(projected.Steps) != 1 || projected.Steps[0].State != string(applyItemIndeterminate) {
		t.Fatalf("recovery projection = %+v, want interrupted with one indeterminate step", projected)
	}
	if err := checkApplyReplayEligibility("local", []applydomain.Item{{ID: "resource:postgres", ObservedState: applyStateUnknown}}); err == nil || !strings.Contains(err.Error(), "target state is unknown") {
		t.Fatalf("unknown readback retry error = %v, want unresolved effect blocked", err)
	}
	if err := checkApplyReplayEligibility("local", []applydomain.Item{{ID: "resource:postgres", ObservedState: applyStateSatisfied}}); err != nil {
		t.Fatalf("positive readback should release replay: %v", err)
	}
	durable, err := loadPersistedApplyRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if durable.Status != applyRunApplying || durable.Items[0].Outcome != applyItemApplying {
		t.Fatalf("read projection rewrote journal: %q/%q, want durable uncertainty preserved", durable.Status, durable.Items[0].Outcome)
	}
	misfiledPath, err := applyRunPath("misfiled-copy")
	if err != nil {
		t.Fatal(err)
	}
	misfiledBytes, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(misfiledPath, misfiledBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkApplyReplayEligibility("local", nil); err == nil || !strings.Contains(err.Error(), "unreadable") {
		t.Fatalf("misfiled journal replay error = %v, want fail-closed identity rejection", err)
	}
	if err := os.Remove(misfiledPath); err != nil {
		t.Fatal(err)
	}

	processAliveObserved = func(int) (bool, error) { return false, os.ErrPermission }
	projected, err = (&Server{}).getApplyRun(context.Background(), run.ID)
	if err != nil || projected.Status != string(applyRunApplying) {
		t.Fatalf("unavailable process evidence projection = %+v, %v; want active run", projected, err)
	}
	if err := checkApplyReplayEligibility("local", nil); err == nil || !strings.Contains(err.Error(), "still owns this target") {
		t.Fatalf("unavailable process evidence admission = %v, want ownership block", err)
	}
}

func TestCheckApplyReplayEligibilityFailsClosedOnCorruptJournal(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("HOME", t.TempDir())
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z"}`)
	run := applyRun{
		ID: "old-uncertain", Target: "local", Status: applyRunIndeterminate,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Items:     []applyItemResult{{applyItem: applyItem{ID: "scenario:writer"}, Outcome: applyItemApplying}},
	}
	if err := persistApplyRun(run); err != nil {
		t.Fatal(err)
	}
	plan := []applydomain.Item{{ID: "scenario:writer", ObservedState: applyStateUnknown}}
	if err := checkApplyReplayEligibility("local", plan); err == nil || !strings.Contains(err.Error(), "retry blocked") {
		t.Fatalf("unknown effect retry error = %v, want blocked", err)
	}
	if err := checkApplyReplayEligibility("local", []applydomain.Item{{ID: "scenario:writer", ObservedState: applyStateSatisfied}}); err != nil {
		t.Fatalf("positive target observation should release retry: %v", err)
	}

	previousAlive := processAliveObserved
	processAliveObserved = func(int) (bool, error) { return false, os.ErrPermission }
	t.Cleanup(func() { processAliveObserved = previousAlive })
	live := applyRun{ID: "live-run", Target: "local", Status: applyRunApplying, RunnerPID: 4242, Heartbeat: time.Now().UTC().Add(-2 * staleApplyHeartbeat).Format(time.RFC3339)}
	if err := persistApplyRun(live); err != nil {
		t.Fatal(err)
	}
	if err := checkApplyReplayEligibility("local", nil); err == nil || !strings.Contains(err.Error(), "still owns this target") {
		t.Fatalf("active-run replay eligibility = %v, want serialization error", err)
	}
	livePath, err := applyRunPath(live.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(livePath); err != nil {
		t.Fatal(err)
	}

	path, err := applyRunPath("corrupt")
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, path, "{")
	if err := checkApplyReplayEligibility("local", nil); err == nil || !strings.Contains(err.Error(), "unreadable") {
		t.Fatalf("corrupt journal eligibility = %v, want fail-closed unreadable error", err)
	}
}
