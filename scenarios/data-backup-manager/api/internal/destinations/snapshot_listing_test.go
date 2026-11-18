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
		if repository != "recovery" || source != "" {
			t.Fatalf("SnapshotList(%q,%q), want a repository-wide inventory filtered by DBM", repository, source)
		}
		return []engine.Snapshot{
			{ID: "older", Source: "dbm://treasury/wallet", StartTime: "2026-11-14T12:00:00Z", Tags: []string{"dbm.owner:treasury", "dbm.name:wallet"}},
			{ID: "newer", Source: "dbm://treasury/wallet", StartTime: "2026-11-15T12:00:00Z", Tags: []string{"dbm.owner:treasury", "dbm.name:wallet"}},
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

func TestListTargetSnapshotsAcceptsLegacySourceIdentity(t *testing.T) {
	ctx := context.Background()
	repo := destmocks.NewFakeRepository()
	if _, err := repo.Create(ctx, destinations.Destination{ID: "dst", Name: "recovery", BackendKind: destinations.BackendS3, Location: "s3://recovery"}); err != nil {
		t.Fatal(err)
	}
	eng := &enginemocks.FakeKopiaEngine{SnapshotListFn: func(context.Context, string, string) ([]engine.Snapshot, error) {
		return []engine.Snapshot{{ID: "legacy", Source: "dbm://treasury/wallet", StartTime: "2026-11-15T12:00:00Z"}}, nil
	}}
	svc := destinations.NewService(repo, eng, nil, t.TempDir())
	snapshots, err := svc.ListTargetSnapshots(ctx, "dst", "treasury", "wallet")
	if err != nil {
		t.Fatalf("legacy source identity should be readable: %v", err)
	}
	if len(snapshots) != 1 || snapshots[0].ID != "legacy" {
		t.Fatalf("legacy snapshots = %+v, want the one matching source", snapshots)
	}
}

func TestListTargetSnapshotsMatchesTaggedAndLegacySourcePaths(t *testing.T) {
	ctx := context.Background()
	repo := destmocks.NewFakeRepository()
	if _, err := repo.Create(ctx, destinations.Destination{ID: "dst", Name: "recovery", BackendKind: destinations.BackendS3, Location: "s3://recovery"}); err != nil {
		t.Fatal(err)
	}
	legacySource := "matthalloran8@swarminator:/home/vrooli/api/dbm:/treasury/wallet"
	eng := &enginemocks.FakeKopiaEngine{SnapshotListFn: func(context.Context, string, string) ([]engine.Snapshot, error) {
		return []engine.Snapshot{
			{ID: "tagged", Source: legacySource, StartTime: "2026-11-15T12:00:00Z", Tags: []string{"dbm.name:wallet", "dbm.owner:treasury"}},
			{ID: "legacy", Source: legacySource, StartTime: "2026-11-14T12:00:00Z"},
			{ID: "wrong-owner", Source: legacySource, StartTime: "2026-11-13T12:00:00Z", Tags: []string{"dbm.owner:other", "dbm.name:wallet"}},
		}, nil
	}}
	svc := destinations.NewService(repo, eng, nil, t.TempDir())
	snapshots, err := svc.ListTargetSnapshots(ctx, "dst", "treasury", "wallet")
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 2 || snapshots[0].ID != "tagged" || snapshots[1].ID != "legacy" {
		t.Fatalf("snapshots = %+v, want tagged and legacy source identities only", snapshots)
	}
}

func TestListTargetSnapshotsFiltersUnrelatedAndMisattributedEntries(t *testing.T) {
	ctx := context.Background()
	repo := destmocks.NewFakeRepository()
	if _, err := repo.Create(ctx, destinations.Destination{ID: "dst", Name: "recovery", BackendKind: destinations.BackendS3, Location: "s3://recovery"}); err != nil {
		t.Fatal(err)
	}
	eng := &enginemocks.FakeKopiaEngine{SnapshotListFn: func(context.Context, string, string) ([]engine.Snapshot, error) {
		return []engine.Snapshot{
			{ID: "other-source", Source: "dbm://other/data", StartTime: "2026-11-15T12:00:00Z"},
			{ID: "wrong-owner-tag", Source: "dbm://treasury/wallet", StartTime: "2026-11-15T12:00:00Z", Tags: []string{"dbm.owner:other", "dbm.name:wallet"}},
			{ID: "wrong-name-tag", Source: "dbm://treasury/wallet", StartTime: "2026-11-15T12:00:00Z", Tags: []string{"dbm.owner:treasury", "dbm.name:other"}},
			{ID: "good", Source: "dbm://treasury/wallet", StartTime: "2026-11-15T12:00:00Z", Tags: []string{"dbm.owner:treasury", "dbm.name:wallet"}},
			{ID: "bad-time", Source: "dbm://treasury/wallet", StartTime: "not-a-time"},
		}, nil
	}}
	svc := destinations.NewService(repo, eng, nil, t.TempDir())
	snapshots, err := svc.ListTargetSnapshots(ctx, "dst", "treasury", "wallet")
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 1 || snapshots[0].ID != "good" {
		t.Fatalf("filtered snapshots = %+v, want only the exact target", snapshots)
	}
}
