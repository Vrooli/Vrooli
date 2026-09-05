package remediation

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"test-genie/internal/testsqlite"
)

func TestSQLiteRepositoryEnforcesOneActiveJobAndPersistsSource(t *testing.T) {
	db := testsqlite.Open(t)
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db)
	job := NewJob(Plan{Scenario: "demo", SourceExecutionID: "exec", SourceRunID: "run"}, []string{"afid:1"}, []string{"requirement:alpha", "requirement:beta"}, "", time.Now().UTC())
	job.ID = "job-1"
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	second := job
	second.ID = "job-2"
	if err := repo.Create(context.Background(), second); !errors.Is(err, ErrActiveJob) {
		t.Fatalf("second create err = %v, want active conflict", err)
	}
	stored, err := repo.Get(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Source.SourceRunID != "run" || stored.SelectedFindingIDs[0] != "afid:1" {
		t.Fatalf("round trip = %+v", stored)
	}
	if got, want := stored.SelectedRequirementIDs, []string{"requirement:alpha", "requirement:beta"}; !slices.Equal(got, want) {
		t.Fatalf("selected requirements = %v, want %v", got, want)
	}
	if stored.SourceHash == "" || stored.SelectionHash == "" {
		t.Fatalf("immutable hashes must survive storage: %+v", stored)
	}
}

func TestSQLiteRepositoryRejectsImmutableEvidenceHashMismatch(t *testing.T) {
	db := testsqlite.Open(t)
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db)
	job := NewJob(Plan{Scenario: "demo", SourceExecutionID: "exec", SourceRunID: "run"}, []string{"afid:1"}, nil, "", time.Now().UTC())
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE remediation_jobs SET selection_hash = 'tampered' WHERE id = ?`, job.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(context.Background(), job.ID); err == nil {
		t.Fatal("expected tampered immutable selection evidence to be rejected")
	}
}

func TestSQLiteRepositoryPersistsAppendOnlyAttemptTimeline(t *testing.T) {
	db := testsqlite.Open(t)
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db)
	job := NewJob(Plan{Scenario: "demo", SourceExecutionID: "exec", SourceRunID: "run"}, []string{"afid:1"}, []string{"REQ-1"}, "", time.Now().UTC())
	job.LaunchAttempt = 1
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	first := newAttempt("launch", "prepared", launchIdempotencyKey(job), "code.default", "intent stored", job.CreatedAt)
	second := newAttempt("launch", "accepted", launchIdempotencyKey(job), "code.default", "accepted by Agent Manager", job.CreatedAt.Add(time.Second))
	second.TaskID, second.RunID = "task-1", "run-1"
	for _, attempt := range []Attempt{first, second} {
		if err := repo.AppendAttempt(context.Background(), attempt, job.ID); err != nil {
			t.Fatal(err)
		}
	}
	// A fresh repository instance models a process restart.
	stored, err := NewSQLiteRepository(db).Get(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Attempts) != 2 || stored.Attempts[0].State != "prepared" || stored.Attempts[1].RunID != "run-1" {
		t.Fatalf("attempt timeline after restart = %+v", stored.Attempts)
	}
}
