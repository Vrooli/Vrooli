package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"workflow-health/internal/artifacts"
	workflowrun "workflow-health/internal/validationrun"
)

type stubLedger struct {
	active []workflowrun.ActiveRun
	err    error
}

func (s stubLedger) ListActive(context.Context) ([]workflowrun.ActiveRun, error) {
	return s.active, s.err
}

func TestRunArtifactRetentionCycleRefusesWhenActiveRunsUnknown(t *testing.T) {
	root := t.TempDir()
	timeline := filepath.Join(root, "hub", "runs", "r1", "a", "timeline.json")
	if err := os.MkdirAll(filepath.Dir(timeline), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(timeline, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(timeline, old, old); err != nil {
		t.Fatal(err)
	}
	_, err := runArtifactRetentionCycle(context.Background(), stubLedger{err: errors.New("db down")}, root, nil, artifacts.RetentionPolicy{MinAge: time.Hour}, time.Now())
	if err == nil {
		t.Fatal("cycle must fail when the active run set is unreadable")
	}
	if _, statErr := os.Stat(timeline); statErr != nil {
		t.Error("timeline was removed although active runs were unknown")
	}
}

func TestRunArtifactRetentionCycleProtectsActiveRuns(t *testing.T) {
	root := t.TempDir()
	timeline := filepath.Join(root, "hub", "runs", "live", "a", "timeline.json")
	if err := os.MkdirAll(filepath.Dir(timeline), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(timeline, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(timeline, old, old); err != nil {
		t.Fatal(err)
	}
	res, err := runArtifactRetentionCycle(context.Background(), stubLedger{active: []workflowrun.ActiveRun{{ID: "live"}}}, root, nil, artifacts.RetentionPolicy{MinAge: time.Hour}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Pruned) != 0 || res.Protected["active_run"] != 1 {
		t.Errorf("active run must be protected: %+v", res)
	}
}
