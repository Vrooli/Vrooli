package graph

import (
	"context"
	"fmt"
	"testing"
	"time"

	"architecture-cartographer/internal/clock"
)

// newRetentionService wires the real service over a real database.
func newRetentionService(t *testing.T) (Service, func(scenario, hash string, at time.Time, size int)) {
	t.Helper()
	db, _ := newRetentionDB(t)
	repo := NewSQLiteRepository(db, clock.System{})
	svc := NewService(repo, clock.System{})
	insert := func(scenario, hash string, at time.Time, size int) {
		insertSnapshot(t, db, scenario, hash, at, size)
	}
	return svc, insert
}

// TestPreviewSnapshotRetention_ReportsNonZeroAboveTheFloor is the owner-side
// estimate storage-manager consumes.
//
// Before this existed, storage-manager could not see the 73 GB that filled the
// disk, because there was no way to ask architecture-cartographer what was
// safe to drop.
func TestPreviewSnapshotRetention_ReportsNonZeroAboveTheFloor(t *testing.T) {
	svc, insert := newRetentionService(t)
	ctx := context.Background()

	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 8; i++ {
		insert("alpha", fmt.Sprintf("a-%d", i), base.Add(time.Duration(i)*time.Hour), 1000)
	}
	for i := 0; i < 2; i++ {
		insert("beta", fmt.Sprintf("b-%d", i), base.Add(time.Duration(i)*time.Hour), 500)
	}

	preview, err := svc.PreviewSnapshotRetention(ctx, 3)
	if err != nil {
		t.Fatalf("PreviewSnapshotRetention: %v", err)
	}

	if preview.ReclaimableRows != 5 {
		t.Errorf("ReclaimableRows = %d, want 5 (alpha has 8, keeping 3)", preview.ReclaimableRows)
	}
	if preview.ReclaimableBytes != 5*1000 {
		t.Errorf("ReclaimableBytes = %d, want %d", preview.ReclaimableBytes, 5*1000)
	}
	if preview.TotalSnapshots != 10 {
		t.Errorf("TotalSnapshots = %d, want 10", preview.TotalSnapshots)
	}
	if preview.KeepPerScenario != 3 {
		t.Errorf("KeepPerScenario = %d, want 3", preview.KeepPerScenario)
	}

	// The worst offender must lead, so an operator reading one line during an
	// incident reads the right one.
	if len(preview.Scenarios) == 0 || preview.Scenarios[0].Scenario != "alpha" {
		t.Fatalf("scenarios = %+v, want alpha first", preview.Scenarios)
	}
	if preview.Scenarios[0].ReclaimableCount != 5 {
		t.Errorf("alpha reclaimable = %d, want 5", preview.Scenarios[0].ReclaimableCount)
	}
	// A scenario already at the floor reports nothing reclaimable.
	for _, s := range preview.Scenarios {
		if s.Scenario == "beta" && s.ReclaimableCount != 0 {
			t.Errorf("beta reclaimable = %d, want 0 (it holds 2, below the floor of 3)", s.ReclaimableCount)
		}
	}
}

// TestPreviewSnapshotRetention_ReportsZeroAtTheFloor asserts a bounded table
// advertises nothing to reclaim, so storage-manager does not offer a no-op.
func TestPreviewSnapshotRetention_ReportsZeroAtTheFloor(t *testing.T) {
	svc, insert := newRetentionService(t)
	ctx := context.Background()

	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 8; i++ {
		insert("alpha", fmt.Sprintf("a-%d", i), base.Add(time.Duration(i)*time.Hour), 1000)
	}

	if _, err := svc.ApplySnapshotRetention(ctx, 3); err != nil {
		t.Fatalf("ApplySnapshotRetention: %v", err)
	}

	preview, err := svc.PreviewSnapshotRetention(ctx, 3)
	if err != nil {
		t.Fatalf("PreviewSnapshotRetention: %v", err)
	}
	if preview.ReclaimableBytes != 0 || preview.ReclaimableRows != 0 {
		t.Errorf("at the floor preview reports %d bytes / %d rows, want 0/0",
			preview.ReclaimableBytes, preview.ReclaimableRows)
	}
	if preview.TotalSnapshots != 3 {
		t.Errorf("TotalSnapshots = %d, want 3", preview.TotalSnapshots)
	}
}

// TestPreviewSnapshotRetention_ReportsPayloadNotFileSize asserts the estimate
// describes reclaimable payload rather than the whole database.
//
// graph_snapshots shares its file with fourteen other tables. Reporting file
// size would claim reclaimable space that is not, and would invite exactly the
// truncate-the-file approach the ownership boundary exists to prevent.
func TestPreviewSnapshotRetention_ReportsPayloadNotFileSize(t *testing.T) {
	svc, insert := newRetentionService(t)
	ctx := context.Background()

	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	const payload = 4096
	for i := 0; i < 5; i++ {
		insert("alpha", fmt.Sprintf("a-%d", i), base.Add(time.Duration(i)*time.Hour), payload)
	}

	preview, err := svc.PreviewSnapshotRetention(ctx, 3)
	if err != nil {
		t.Fatalf("PreviewSnapshotRetention: %v", err)
	}
	// Exactly two rows of payload, not the file, not the table.
	if preview.ReclaimableBytes != 2*payload {
		t.Errorf("ReclaimableBytes = %d, want exactly %d (two rows of payload)", preview.ReclaimableBytes, 2*payload)
	}
}

// TestApplySnapshotRetention_MatchesItsPreview asserts the estimate is honest:
// applying removes what the preview advertised.
func TestApplySnapshotRetention_MatchesItsPreview(t *testing.T) {
	svc, insert := newRetentionService(t)
	ctx := context.Background()

	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 9; i++ {
		insert("alpha", fmt.Sprintf("a-%d", i), base.Add(time.Duration(i)*time.Hour), 777)
	}

	preview, err := svc.PreviewSnapshotRetention(ctx, 2)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	result, err := svc.ApplySnapshotRetention(ctx, 2)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	if result.RowsRemoved != preview.ReclaimableRows {
		t.Errorf("apply removed %d rows, preview advertised %d", result.RowsRemoved, preview.ReclaimableRows)
	}
	if result.BytesReclaimed != preview.ReclaimableBytes {
		t.Errorf("apply reclaimed %d bytes, preview advertised %d", result.BytesReclaimed, preview.ReclaimableBytes)
	}
}
