package remediation

import (
	"context"
	"errors"
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
	job := Job{ID: "job-1", Scenario: "demo", Status: JobStatusCreated, Source: Plan{Scenario: "demo", SourceExecutionID: "exec", SourceRunID: "run"}, SelectedFindingIDs: []string{"afid:1"}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
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
}
