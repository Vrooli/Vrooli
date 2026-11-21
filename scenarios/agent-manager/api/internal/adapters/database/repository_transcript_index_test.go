package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/transcriptindex"
	"github.com/sirupsen/logrus"
)

func TestTranscriptFileRepositoryRejectsInvalidStatusTransitions(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	db, err := NewConnection(logrus.New())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := NewTranscriptIndexRepository(db)
	ctx := context.Background()
	file := transcriptindex.FileRow{Key: "status", Path: "/status", Harness: transcriptindex.HarnessCodex, Owner: transcriptindex.OwnerIndexer, Status: transcriptindex.StatusPending, FirstSeenAt: time.Now()}
	if err := repo.SaveTranscriptFile(ctx, file, nil); err != nil {
		t.Fatal(err)
	}
	file.Status = transcriptindex.StatusIndexed
	if err := repo.SaveTranscriptFile(ctx, file, nil); err != nil {
		t.Fatalf("valid pending -> indexed transition rejected: %v", err)
	}
	file.Status = "capturing"
	if err := repo.SaveTranscriptFile(ctx, file, nil); err == nil {
		t.Fatal("unknown status transition was accepted")
	}
	rows, err := repo.ListTranscriptFiles(ctx)
	if err != nil || len(rows) != 1 || rows[0].Status != transcriptindex.StatusIndexed {
		t.Fatalf("rejected transition changed persisted row: rows=%+v err=%v", rows, err)
	}
}
