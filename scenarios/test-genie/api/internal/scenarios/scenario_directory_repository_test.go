package scenarios

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"test-genie/internal/testsqlite"
)

func TestScenarioDirectoryRepositorySummarizesExecutionEvidence(t *testing.T) {
	db := testsqlite.Open(t)
	now := time.Now().UTC()
	if _, err := db.Exec(`INSERT INTO suite_executions (id, run_id, scenario_name, requested_phases, requested_skip_phases, planned_phases, fail_fast, success, started_at, completed_at) VALUES (?, ?, ?, '[]', '[]', '[]', 0, 0, ?, ?)`, "11111111-1111-1111-1111-111111111111", "run-1", "demo", now.Add(-time.Minute).Format(time.RFC3339), now.Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	summaries, err := NewScenarioDirectoryRepository(db).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].ScenarioName != "demo" || summaries[0].TotalExecutions != 1 || summaries[0].LastExecutionID == nil || summaries[0].LastRunID != "run-1" {
		t.Fatalf("summaries=%+v", summaries)
	}
}

func TestLatestUIChangeIgnoresGeneratedTrees(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "src", "App.tsx")
	generated := filepath.Join(root, "dist", "bundle.js")
	dependency := filepath.Join(root, "node_modules")
	dependencyTarget := t.TempDir()
	for _, path := range []string{source, generated} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(dependencyTarget, dependency); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	old := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	new := old.Add(time.Hour)
	if err := os.Chtimes(source, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(generated, new, new); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{root, filepath.Join(root, "src")} {
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	got, ok, err := latestUIChange(root)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !got.Equal(old) {
		t.Fatalf("latestUIChange = %v, %t; want %v, true", got, ok, old)
	}
}

func TestLatestUIChangeDetectsRemovalThroughParentDirectory(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "deleted.tsx")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	new := old.Add(time.Hour)
	if err := os.Chtimes(file, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(root, new, new); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	// Preserve a deterministic directory timestamp for filesystems with coarse
	// mtime resolution; it models the directory mutation left by deletion.
	if err := os.Chtimes(root, new, new); err != nil {
		t.Fatal(err)
	}
	got, ok, err := latestUIChange(root)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !got.Equal(new) {
		t.Fatalf("latestUIChange after deletion = %v, %t; want %v, true", got, ok, new)
	}
}

func TestScenarioDirectoryRepositoryRetainsMostRecentExperienceVerdict(t *testing.T) {
	db := testsqlite.Open(t)
	older := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	for _, row := range []struct {
		id, runID, phase, status string
		at                       time.Time
	}{
		{"11111111-1111-1111-1111-111111111111", "experience-run", "experience", "passed", older},
		{"22222222-2222-2222-2222-222222222222", "unit-run", "unit", "passed", newer},
	} {
		if _, err := db.Exec(`INSERT INTO suite_executions (id, run_id, scenario_name, requested_phases, requested_skip_phases, planned_phases, fail_fast, success, started_at, completed_at) VALUES (?, ?, 'demo', '[]', '[]', '[]', 0, 1, ?, ?)`, row.id, row.runID, row.at.Add(-time.Minute).Format(time.RFC3339), row.at.Add(5*time.Minute).Format(time.RFC3339)); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO suite_execution_phases (execution_id, ordinal, phase_name, status, completed_at, duration_seconds) VALUES (?, 0, ?, ?, ?, 1)`, row.id, row.phase, row.status, row.at.Format(time.RFC3339)); err != nil {
			t.Fatal(err)
		}
	}
	summaries, err := NewScenarioDirectoryRepository(db).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Fatalf("summaries = %#v", summaries)
	}
	got := summaries[0]
	if got.LastExperienceStatus != "passed" || got.LastExperienceRunID != "experience-run" || got.LastExperienceAt == nil || !got.LastExperienceAt.Equal(older) {
		t.Fatalf("experience verdict = status %q run %q at %v; want passed / experience-run / %v", got.LastExperienceStatus, got.LastExperienceRunID, got.LastExperienceAt, older)
	}
}

func TestScenarioDirectoryRepositoryDoesNotRetainOlderPassOverLatestNotEvaluated(t *testing.T) {
	db := testsqlite.Open(t)
	older := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	for _, row := range []struct {
		id, runID, status string
		at                time.Time
	}{
		{"11111111-1111-1111-1111-111111111111", "passed-run", "passed", older},
		{"22222222-2222-2222-2222-222222222222", "not-evaluated-run", "not_evaluated", newer},
	} {
		if _, err := db.Exec(`INSERT INTO suite_executions (id, run_id, scenario_name, requested_phases, requested_skip_phases, planned_phases, fail_fast, success, started_at, completed_at) VALUES (?, ?, 'demo', '[]', '[]', '[]', 0, 0, ?, ?)`, row.id, row.runID, row.at.Add(-time.Minute).Format(time.RFC3339), row.at.Add(time.Minute).Format(time.RFC3339)); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO suite_execution_phases (execution_id, ordinal, phase_name, status, completed_at, duration_seconds) VALUES (?, 0, 'experience', ?, ?, 1)`, row.id, row.status, row.at.Format(time.RFC3339)); err != nil {
			t.Fatal(err)
		}
	}
	summaries, err := NewScenarioDirectoryRepository(db).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Fatalf("summaries = %#v", summaries)
	}
	got := summaries[0]
	if got.LastExperienceStatus != "not_evaluated" || got.LastExperienceRunID != "not-evaluated-run" || got.LastExperienceAt == nil || !got.LastExperienceAt.Equal(newer) {
		t.Fatalf("latest experience verdict = status %q run %q at %v; want not_evaluated / not-evaluated-run / %v", got.LastExperienceStatus, got.LastExperienceRunID, got.LastExperienceAt, newer)
	}
}

func TestScenarioDirectoryRepositoryComparesFractionalTimestampsAsInstants(t *testing.T) {
	db := testsqlite.Open(t)
	for _, row := range []struct {
		id, runID, status, at string
	}{
		{"11111111-1111-1111-1111-111111111111", "earlier-pass", "passed", "2026-11-01T10:00:00.1Z"},
		{"22222222-2222-2222-2222-222222222222", "later-not-evaluated", "not_evaluated", "2026-11-01T10:00:00.11Z"},
	} {
		if _, err := db.Exec(`INSERT INTO suite_executions (id, run_id, scenario_name, requested_phases, requested_skip_phases, planned_phases, fail_fast, success, started_at, completed_at) VALUES (?, ?, 'demo', '[]', '[]', '[]', 0, 0, ?, ?)`, row.id, row.runID, row.at, row.at); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO suite_execution_phases (execution_id, ordinal, phase_name, status, completed_at, duration_seconds) VALUES (?, 0, 'experience', ?, ?, 1)`, row.id, row.status, row.at); err != nil {
			t.Fatal(err)
		}
	}
	summaries, err := NewScenarioDirectoryRepository(db).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].LastExperienceStatus != "not_evaluated" || summaries[0].LastExperienceRunID != "later-not-evaluated" {
		t.Fatalf("latest fractional-time experience verdict was not selected: %+v", summaries)
	}
}
