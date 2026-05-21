package conflicts_test

import (
	"context"
	"errors"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/mocks"
)

func newSvc(detectors ...conflicts.Detector) (*mocks.FakeRepository, conflicts.Service) {
	repo := &mocks.FakeRepository{}
	reg := conflicts.NewRegistry(detectors...)
	resolvers := conflicts.NewResolverRegistry()
	return repo, conflicts.NewService(repo, reg, resolvers)
}

func TestService_UpsertConflicts_AssignsScenarioAndStatus(t *testing.T) {
	_, svc := newSvc()
	got, err := svc.UpsertConflicts(context.Background(), "demo", []conflicts.Conflict{{Type: "cycle"}})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if got[0].Scenario != "demo" {
		t.Fatalf("want scenario demo, got %q", got[0].Scenario)
	}
	if got[0].Status != conflicts.ResolutionStatusDetected {
		t.Fatalf("want status detected, got %q", got[0].Status)
	}
}

func TestService_AssignConflict_TransitionsAndUpdates(t *testing.T) {
	repo, svc := newSvc()
	seeded, _ := svc.UpsertConflicts(context.Background(), "demo", []conflicts.Conflict{{Type: "cycle"}})
	id := seeded[0].ID
	got, dry, err := svc.AssignConflict(context.Background(), id, "graph", "ok", false)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if dry {
		t.Fatal("dry-run should be false")
	}
	if got.AssignedDomain != "graph" || got.Status != conflicts.ResolutionStatusAssigned {
		t.Fatalf("unexpected: %+v", got)
	}
	if repo.UpdateCalls.Load() != 1 {
		t.Fatalf("UpdateStatus should be called once, got %d", repo.UpdateCalls.Load())
	}
}

func TestService_AssignConflict_DryRunDoesNotPersist(t *testing.T) {
	repo, svc := newSvc()
	seeded, _ := svc.UpsertConflicts(context.Background(), "demo", []conflicts.Conflict{{Type: "cycle"}})
	got, dry, err := svc.AssignConflict(context.Background(), seeded[0].ID, "graph", "ok", true)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if !dry || got.Status != conflicts.ResolutionStatusAssigned {
		t.Fatalf("unexpected: %+v dry=%v", got, dry)
	}
	if repo.UpdateCalls.Load() != 0 {
		t.Fatal("dry-run should not touch repo")
	}
}

func TestService_AssignConflict_RejectsEmptyDomain(t *testing.T) {
	_, svc := newSvc()
	seeded, _ := svc.UpsertConflicts(context.Background(), "demo", []conflicts.Conflict{{Type: "cycle"}})
	_, _, err := svc.AssignConflict(context.Background(), seeded[0].ID, "", "ok", false)
	var typed conflicts.ErrInvalidAssignment
	if !errors.As(err, &typed) {
		t.Fatalf("want ErrInvalidAssignment, got %v", err)
	}
}

func TestService_ResolveConflict_RejectsCommittedTransition(t *testing.T) {
	repo, svc := newSvc()
	c := conflicts.Conflict{ID: "c-1", Scenario: "demo", Type: "cycle", Status: conflicts.ResolutionStatusCommitted}
	if _, err := repo.UpsertConflict(context.Background(), c); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, _, _, err := svc.ResolveConflict(context.Background(), "c-1", "", false, false)
	var typed conflicts.ErrInvalidTransition
	if !errors.As(err, &typed) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
}

func TestService_ResolveConflict_ForceMovesToForceResolved(t *testing.T) {
	_, svc := newSvc()
	seeded, _ := svc.UpsertConflicts(context.Background(), "demo", []conflicts.Conflict{{Type: "cycle"}})
	got, _, _, err := svc.ResolveConflict(context.Background(), seeded[0].ID, "skipping", true, false)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Status != conflicts.ResolutionStatusForceResolved {
		t.Fatalf("want force_resolved, got %q", got.Status)
	}
}

func TestRegistry_DetectAll_AlphabeticalOrder(t *testing.T) {
	zd := &mocks.FakeDetector{NameValue: "zeta", Conflicts: []conflicts.Conflict{{Type: "z"}}}
	ad := &mocks.FakeDetector{NameValue: "alpha", Conflicts: []conflicts.Conflict{{Type: "a"}}}
	reg := conflicts.NewRegistry(zd, ad)
	out, err := reg.DetectAll(context.Background(), conflicts.DetectInput{})
	if err != nil {
		t.Fatalf("DetectAll: %v", err)
	}
	if out[0].Type != "a" || out[1].Type != "z" {
		t.Fatalf("registry order broken: %+v", out)
	}
}
