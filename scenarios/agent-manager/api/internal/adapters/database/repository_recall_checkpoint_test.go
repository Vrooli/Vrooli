package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptindex"

	"github.com/google/uuid"
)

func TestRecallCheckpointCommitsRejectSuccessorIdentityChanges(t *testing.T) {
	identityChanges := []struct {
		name   string
		mutate func(*transcriptindex.FileRow)
	}{
		{name: "file key", mutate: func(file *transcriptindex.FileRow) { file.Key += "-other" }},
		{name: "run id", mutate: func(file *transcriptindex.FileRow) { file.RunID = uuid.NewString() }},
		{name: "owner", mutate: func(file *transcriptindex.FileRow) { file.Owner = transcriptindex.OwnerRun }},
	}

	for _, operation := range []string{"rebuild", "evict"} {
		t.Run(operation, func(t *testing.T) {
			for _, change := range identityChanges {
				t.Run(change.name, func(t *testing.T) {
					db, expected := seedRecallCheckpoint(t)
					repo := NewTranscriptIndexRepository(db)
					next := expected
					next.RecallBytes = 0
					next.RecallEvicted = operation == "evict"
					if operation == "evict" {
						next.Status = transcriptindex.StatusPending
						next.Error = transcriptindex.ReplacementEffectsPending
					}
					change.mutate(&next)

					callbackCalled := false
					callback := func(*sql.Tx) error {
						callbackCalled = true
						return nil
					}
					var err error
					if operation == "rebuild" {
						err = repo.CommitRecallRebuild(context.Background(), expected, next, callback)
					} else {
						err = repo.CommitRecallEviction(context.Background(), expected, next, callback)
					}
					if err == nil {
						t.Fatalf("accepted successor with changed %s", change.name)
					}
					if callbackCalled {
						t.Fatalf("event-owner callback ran before rejecting changed %s", change.name)
					}
					rows, listErr := repo.ListTranscriptFiles(context.Background())
					if listErr != nil || len(rows) != 1 || rows[0].Key != expected.Key || rows[0].RunID != expected.RunID || rows[0].Owner != transcriptindex.OwnerIndexer || rows[0].Status != transcriptindex.StatusIndexed || rows[0].RecallBytes != expected.RecallBytes {
						t.Fatalf("rejected identity change altered checkpoint: rows=%+v err=%v", rows, listErr)
					}
				})
			}
		})
	}
}

func TestRecallCheckpointCommitsAcceptSameOwnerIdentity(t *testing.T) {
	for _, operation := range []string{"rebuild", "evict"} {
		t.Run(operation, func(t *testing.T) {
			db, expected := seedRecallCheckpoint(t)
			repo := NewTranscriptIndexRepository(db)
			next := expected
			next.RecallBytes = 0
			next.RecallEvicted = operation == "evict"
			if operation == "evict" {
				next.Status = transcriptindex.StatusPending
				next.Error = transcriptindex.ReplacementEffectsPending
			}

			callbackCalled := false
			callback := func(*sql.Tx) error {
				callbackCalled = true
				return nil
			}
			var err error
			if operation == "rebuild" {
				err = repo.CommitRecallRebuild(context.Background(), expected, next, callback)
			} else {
				err = repo.CommitRecallEviction(context.Background(), expected, next, callback)
			}
			if err != nil || !callbackCalled {
				t.Fatalf("valid %s checkpoint failed: callback=%v err=%v", operation, callbackCalled, err)
			}
			rows, err := repo.ListTranscriptFiles(context.Background())
			if err != nil || len(rows) != 1 || rows[0].Key != expected.Key || rows[0].RunID != expected.RunID || rows[0].Owner != transcriptindex.OwnerIndexer || rows[0].RecallBytes != 0 || rows[0].RecallEvicted != (operation == "evict") {
				t.Fatalf("valid %s successor did not persist: rows=%+v err=%v", operation, rows, err)
			}
		})
	}
}

func seedRecallCheckpoint(t *testing.T) (*DB, transcriptindex.FileRow) {
	t.Helper()
	db := setupTestDB(t)
	ctx := context.Background()
	repos := NewRepositories(db, db.log)
	task := &domain.Task{ID: uuid.New(), Title: "recall checkpoint", ScopePath: "."}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{
		ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(),
		Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted,
		ExecutionMode: domain.ExecutionModeImported,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	file := transcriptindex.FileRow{
		Key: "recall-checkpoint", Path: "/transcript", RunID: run.ID.String(),
		Owner: transcriptindex.OwnerIndexer, Status: transcriptindex.StatusIndexed,
		RecallBytes: 128, FirstSeenAt: time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC),
	}
	if err := repos.TranscriptIndex.SaveTranscriptFile(ctx, file, nil); err != nil {
		t.Fatal(err)
	}
	rows, err := repos.TranscriptIndex.ListTranscriptFiles(ctx)
	if err != nil || len(rows) != 1 {
		t.Fatalf("seeded checkpoint rows=%+v err=%v", rows, err)
	}
	return db, rows[0]
}
