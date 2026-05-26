package restores_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"data-backup-manager/internal/restores"
	restoresmocks "data-backup-manager/internal/restores/mocks"
	"data-backup-manager/internal/sources"
	sourcesmocks "data-backup-manager/internal/sources/mocks"
	"data-backup-manager/internal/testutil/mocks"
)

func newVerifyService(t *testing.T, scratchRoot string, eng *mocks.FakeKopiaEngine) restores.Service {
	t.Helper()
	clk := mocks.NewFakeClock(time.Time{})
	capt := &sourcesmocks.FakeCapturer{SourceKind: sources.KindFilesystem}
	registry := sources.NewRegistry(capt)
	return restores.NewService(restores.Deps{
		Repo: restores.NewSQLiteRepository(newRestoresDB(t), clk),
		Targets: &restoresmocks.FakeTargetLookup{
			Targets: map[string]restores.TargetForRestore{
				"t1": {ID: "t1", Kind: sources.KindFilesystem, Locator: "loc"},
			},
		},
		Destinations: &restoresmocks.FakeDestinationLookup{
			Destinations: map[string]restores.DestinationForRestore{
				"dst-1": {ID: "dst-1", Name: "nightly"},
			},
		},
		Engine:      eng,
		Sources:     registry,
		Clock:       clk,
		ScratchRoot: scratchRoot,
	})
}

// TestVerify_RecordsLastVerified is the happy-path test: VerifyTarget with a
// default FakeKopiaEngine (verify succeeds). Asserts:
//   - status == verified
//   - last_verified_at is set (non-zero)
//   - checksum is non-empty
//   - scratch dir no longer exists (cleaned up)
func TestVerify_RecordsLastVerified(t *testing.T) {
	ctx := context.Background()
	scratchRoot := t.TempDir()
	eng := &mocks.FakeKopiaEngine{}

	svc := newVerifyService(t, scratchRoot, eng)
	rec, err := svc.VerifyTarget(ctx, "t1", "dst-1", "snap-1")
	if err != nil {
		t.Fatalf("VerifyTarget: %v", err)
	}

	if rec.Status != restores.RestoreVerified {
		t.Errorf("status = %s, want verified", rec.Status)
	}
	if rec.LastVerifiedAt.IsZero() {
		t.Error("last_verified_at must not be zero on success")
	}
	if rec.Checksum == "" {
		t.Error("checksum must not be empty on success")
	}

	// Scratch dir must have been cleaned up.
	entries, _ := os.ReadDir(scratchRoot)
	for _, e := range entries {
		if e.IsDir() {
			// Any leftover subdirectory is a leak.
			t.Errorf("scratch dir leaked: %s", filepath.Join(scratchRoot, e.Name()))
		}
	}

	// Invariants hold.
	if err := restores.CheckInvariants(rec); err != nil {
		t.Errorf("CheckInvariants: %v", err)
	}
}

// TestVerify_MismatchNotVerified is the load-bearing safety test: when
// SnapshotVerify returns an error, the status must be failed, last_verified_at
// must be ZERO, and the scratch dir must still be cleaned up.
func TestVerify_MismatchNotVerified(t *testing.T) {
	ctx := context.Background()
	scratchRoot := t.TempDir()
	eng := &mocks.FakeKopiaEngine{}
	// Program the engine to fail verification.
	eng.SnapshotVerifyFn = func(_ context.Context, _, _ string, _ int) error {
		return errors.New("verify: content mismatch")
	}

	svc := newVerifyService(t, scratchRoot, eng)
	rec, err := svc.VerifyTarget(ctx, "t1", "dst-1", "snap-bad")
	// A verify failure records the failure as a Restore row; no error is returned
	// to the caller — the failure is encoded in the record's status.
	if err != nil {
		t.Fatalf("VerifyTarget returned unexpected error: %v", err)
	}

	// CRITICAL: status must be failed, NOT verified.
	if rec.Status != restores.RestoreFailed {
		t.Errorf("status = %s, want failed (a false 'verified' is OT-P0-006)", rec.Status)
	}
	// CRITICAL: last_verified_at must be ZERO on failure.
	if !rec.LastVerifiedAt.IsZero() {
		t.Errorf("last_verified_at must be zero on verify failure, got %v", rec.LastVerifiedAt)
	}

	// Scratch dir must still be cleaned up even on failure.
	entries, _ := os.ReadDir(scratchRoot)
	for _, e := range entries {
		if e.IsDir() {
			t.Errorf("scratch dir leaked on failure: %s", filepath.Join(scratchRoot, e.Name()))
		}
	}

	// Invariants hold for the failed record.
	if err := restores.CheckInvariants(rec); err != nil {
		t.Errorf("CheckInvariants on failed record: %v", err)
	}
}
