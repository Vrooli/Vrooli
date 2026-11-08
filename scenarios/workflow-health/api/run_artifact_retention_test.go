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

func TestRunArtifactRetentionCycleRefusesUnreadableCitationFile(t *testing.T) {
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
	citations := t.TempDir()
	note := filepath.Join(citations, "note.md")
	if err := os.WriteFile(note, []byte("unreadable evidence"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(note, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(note, 0o600) })
	if f, err := os.Open(note); err == nil {
		_ = f.Close()
		t.Skip("this test requires a filesystem user that cannot read mode 000 files")
	}
	_, err := runArtifactRetentionCycle(context.Background(), stubLedger{}, root, []string{citations}, artifacts.RetentionPolicy{MinAge: time.Hour}, time.Now())
	if err == nil {
		t.Fatal("cycle must refuse to prune when a citation file cannot be read")
	}
	if _, statErr := os.Stat(timeline); statErr != nil {
		t.Error("timeline was removed although citations were unreadable")
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
