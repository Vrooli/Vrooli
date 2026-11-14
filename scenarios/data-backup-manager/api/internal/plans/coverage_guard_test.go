package plans_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"data-backup-manager/internal/plans"
	"data-backup-manager/internal/plans/mocks"
)

// fakeGuard returns a fixed set of unregistered non-sensitive recommendations.
type fakeGuard struct {
	missing []plans.MissingTarget
	err     error
	calls   int
}

func (g *fakeGuard) UnregisteredDefaultTargets(context.Context) ([]plans.MissingTarget, error) {
	g.calls++
	return g.missing, g.err
}

func validCreate() plans.CreateInput {
	return plans.CreateInput{Name: "nightly", TargetIDs: []string{"t"}, DestinationIDs: []string{"d"}}
}

func TestCreate_BlocksOnIncompleteCoverage(t *testing.T) {
	ctx := context.Background()
	guard := &fakeGuard{missing: []plans.MissingTarget{{Owner: "vrooli", Name: "plans", Locator: "/p"}}}
	svc := plans.NewService(mocks.NewFakeRepository(), guard)

	_, err := svc.Create(ctx, validCreate())
	if err == nil {
		t.Fatal("expected incomplete-coverage rejection")
	}
	var incomplete plans.ErrIncompleteCoverage
	if !errors.As(err, &incomplete) {
		t.Fatalf("error = %v, want ErrIncompleteCoverage", err)
	}
	if len(incomplete.Missing) != 1 {
		t.Fatalf("missing not surfaced: %+v", incomplete.Missing)
	}
	if connect.CodeOf(plans.ToConnectError(err)) != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want failed_precondition", connect.CodeOf(plans.ToConnectError(err)))
	}
}

func TestCreate_AllowIncompleteCoverageBypassesGuard(t *testing.T) {
	ctx := context.Background()
	guard := &fakeGuard{missing: []plans.MissingTarget{{Owner: "vrooli", Name: "plans"}}}
	svc := plans.NewService(mocks.NewFakeRepository(), guard)

	in := validCreate()
	in.AllowIncompleteCoverage = true
	if _, err := svc.Create(ctx, in); err != nil {
		t.Fatalf("allow_incomplete_coverage should bypass guard: %v", err)
	}
	if guard.calls != 0 {
		t.Fatalf("guard should not be consulted when bypassed, calls=%d", guard.calls)
	}
}

func TestCreate_CompleteCoveragePasses(t *testing.T) {
	ctx := context.Background()
	guard := &fakeGuard{missing: nil}
	svc := plans.NewService(mocks.NewFakeRepository(), guard)

	if _, err := svc.Create(ctx, validCreate()); err != nil {
		t.Fatalf("complete coverage should pass: %v", err)
	}
	if guard.calls != 1 {
		t.Fatalf("guard should be consulted once, calls=%d", guard.calls)
	}
}

func TestUpdate_BlocksOnIncompleteCoverage(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	// First create with a guard that allows, then attempt an update that blocks.
	created, err := plans.NewService(repo, nil).Create(ctx, validCreate())
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}
	guard := &fakeGuard{missing: []plans.MissingTarget{{Owner: "vrooli", Name: "plans"}}}
	svc := plans.NewService(repo, guard)

	_, err = svc.Update(ctx, plans.UpdateInput{ID: created.ID, Name: "x", TargetIDs: []string{"t"}, DestinationIDs: []string{"d"}})
	var incomplete plans.ErrIncompleteCoverage
	if !errors.As(err, &incomplete) {
		t.Fatalf("update should block on incomplete coverage, got %v", err)
	}
}

func TestCreate_GuardErrorPropagates(t *testing.T) {
	ctx := context.Background()
	guard := &fakeGuard{err: errors.New("discovery down")}
	svc := plans.NewService(mocks.NewFakeRepository(), guard)
	if _, err := svc.Create(ctx, validCreate()); err == nil {
		t.Fatal("guard error should propagate")
	}
}
