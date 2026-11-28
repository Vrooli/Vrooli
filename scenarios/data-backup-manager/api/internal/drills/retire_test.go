package drills_test

import (
	"context"
	"testing"
	"time"

	drills "data-backup-manager/internal/drills"
)

func TestServiceRetireRecordsReasonAndKeepsOriginalFailedEvidence(t *testing.T) {
	repo := newDrillRepo(t)
	finishedAt := time.Date(2026, 8, 19, 17, 32, 23, 0, time.UTC)
	created, err := repo.Create(context.Background(), drills.Drill{
		ID: "orphaned", PlanID: "removed-plan", TargetID: "target", DestinationID: "destination",
		SnapshotID: "snapshot-proof", RestoreID: "restore-proof", Status: drills.StatusFailed,
		Error: "historical failure", NextAction: "repair source", RequestedAt: finishedAt, FinishedAt: finishedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	retiredAt := finishedAt.Add(24 * time.Hour)
	svc := drills.NewService(drills.Deps{Repo: repo, Clock: fixedClock{now: retiredAt}, BaseContext: context.Background()})
	defer svc.Shutdown(context.Background())

	retired, err := svc.Retire(context.Background(), created.ID, "orphaned plan reference")
	if err != nil {
		t.Fatal(err)
	}
	if retired.Status != drills.StatusFailed || retired.SnapshotID != "snapshot-proof" || retired.RestoreID != "restore-proof" || retired.Error != "historical failure" || retired.NextAction != "repair source" || !retired.FinishedAt.Equal(finishedAt) {
		t.Fatalf("retirement changed original evidence: %+v", retired)
	}
	if !retired.RetiredAt.Equal(retiredAt) || retired.RetirementReason != "orphaned plan reference" {
		t.Fatalf("retirement receipt = %+v", retired)
	}
}

func TestServiceRetireRequiresCompletedDrillAndReason(t *testing.T) {
	repo := newDrillRepo(t)
	active, err := repo.Create(context.Background(), drills.Drill{ID: "active", PlanID: "p", TargetID: "t", DestinationID: "d", Status: drills.StatusRequested})
	if err != nil {
		t.Fatal(err)
	}
	svc := drills.NewService(drills.Deps{Repo: repo, Clock: fixedClock{now: time.Now().UTC()}, BaseContext: context.Background()})
	defer svc.Shutdown(context.Background())
	if _, err := svc.Retire(context.Background(), active.ID, "operator cleanup"); err == nil {
		t.Fatal("active drill was retired")
	}
	if _, err := svc.Retire(context.Background(), active.ID, " "); err == nil {
		t.Fatal("empty retirement reason was accepted")
	}
	unchanged, err := repo.Get(context.Background(), active.ID)
	if err != nil || unchanged.Status != drills.StatusRequested || !unchanged.RetiredAt.IsZero() {
		t.Fatalf("rejected retirement mutated active drill: %+v err=%v", unchanged, err)
	}
}
