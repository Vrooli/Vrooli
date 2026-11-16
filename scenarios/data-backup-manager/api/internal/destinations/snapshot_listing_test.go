package destinations_test

import (
	"context"
	"testing"

	"data-backup-manager/internal/destinations"
	destmocks "data-backup-manager/internal/destinations/mocks"
	"data-backup-manager/internal/engine"
	enginemocks "data-backup-manager/internal/testutil/mocks"
)

func TestListTargetSnapshotsUsesStableSourceAndReturnsNewestFirst(t *testing.T) {
	ctx := context.Background()
	repo := destmocks.NewFakeRepository()
	if _, err := repo.Create(ctx, destinations.Destination{ID: "dst", Name: "recovery", BackendKind: destinations.BackendS3, Location: "s3://recovery"}); err != nil {
		t.Fatal(err)
	}
	eng := &enginemocks.FakeKopiaEngine{SnapshotListFn: func(_ context.Context, repository, source string) ([]engine.Snapshot, error) {
		if repository != "recovery" || source != "dbm://treasury/wallet" {
			t.Fatalf("SnapshotList(%q,%q)", repository, source)
		}
		return []engine.Snapshot{
			{ID: "older", Source: source, StartTime: "2026-11-14T12:00:00Z", Tags: []string{"dbm.owner:treasury", "dbm.name:wallet"}},
			{ID: "newer", Source: source, StartTime: "2026-11-15T12:00:00Z", Tags: []string{"dbm.owner:treasury", "dbm.name:wallet"}},
		}, nil
	}}
	svc := destinations.NewService(repo, eng, nil, t.TempDir())
	snapshots, err := svc.ListTargetSnapshots(ctx, "dst", "treasury", "wallet")
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 2 || snapshots[0].ID != "newer" || snapshots[1].ID != "older" {
		t.Fatalf("snapshots not newest first: %+v", snapshots)
	}
}

func TestListTargetSnapshotsFailsClosedOnAmbiguousMetadata(t *testing.T) {
	ctx := context.Background()
	repo := destmocks.NewFakeRepository()
	if _, err := repo.Create(ctx, destinations.Destination{ID: "dst", Name: "recovery", BackendKind: destinations.BackendS3, Location: "s3://recovery"}); err != nil {
		t.Fatal(err)
	}
	eng := &enginemocks.FakeKopiaEngine{SnapshotListFn: func(context.Context, string, string) ([]engine.Snapshot, error) {
		return []engine.Snapshot{{ID: "unattributed", Source: "dbm://treasury/wallet", StartTime: "2026-11-15T12:00:00Z"}}, nil
	}}
	svc := destinations.NewService(repo, eng, nil, t.TempDir())
	if _, err := svc.ListTargetSnapshots(ctx, "dst", "treasury", "wallet"); err == nil {
		t.Fatal("snapshot with missing exact owner/name tags was accepted")
	}
}
