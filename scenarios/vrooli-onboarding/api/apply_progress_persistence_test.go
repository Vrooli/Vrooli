package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestApplyDoesNotStartItemWhenApplyingIntentCannotBePersisted(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("HOME", t.TempDir())
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z"}`)

	fake := &recordingApplyExecutor{}
	previousExecutor := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previousExecutor })

	run := applyRun{
		ID:        "apply-intent-write-fails",
		Target:    "local",
		Status:    applyRunPending,
		StartedAt: operatorStateNow().UTC().Format(time.RFC3339),
		Items: []applyItemResult{{
			applyItem: applyItem{ID: "resource:postgres", Kind: "resource", Name: "postgres"},
			Outcome:   applyItemPending,
		}},
	}
	if err := persistApplyRun(run); err != nil {
		t.Fatalf("persist initial run: %v", err)
	}

	previousPersist := persistApplyRunUpdate
	persistApplyRunUpdate = func(next applyRun) error {
		if next.Status == applyRunApplying && next.Items[0].Outcome == applyItemApplying {
			return errors.New("injected durable write failure")
		}
		return persistApplyRun(next)
	}
	t.Cleanup(func() { persistApplyRunUpdate = previousPersist })

	err := executeApplyRun(context.Background(), run)
	if err == nil {
		t.Fatal("executeApplyRun succeeded despite losing the in-flight intent write")
	}
	if calls := fake.snapshotCalls(); len(calls) != 0 {
		t.Fatalf("host effects started without durable applying intent: %v", calls)
	}

	persisted, loadErr := loadPersistedApplyRun(run.ID)
	if loadErr != nil {
		t.Fatalf("load run after refusal: %v", loadErr)
	}
	if persisted.Status != applyRunApplying || persisted.Items[0].Outcome != applyItemPending {
		t.Fatalf("persisted recovery state = %q/%q, want applying/pending", persisted.Status, persisted.Items[0].Outcome)
	}
}
