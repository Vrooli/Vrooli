package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestReleaseComputeClaimsFreesCapacityButKeepsWorkerAndDisk proves the safety
// review can run without the job's own GPU/RAM claim still counted against the
// host, while the manager keeps the worker finish and disk reservation for its
// final release (calling it twice must not double-release).
func TestReleaseComputeClaimsFreesCapacityButKeepsWorkerAndDisk(t *testing.T) {
	g := &leaseGroup{}
	var released, finished, stopped int
	g.releases = []func(context.Context) error{func(context.Context) error { released++; return nil }}
	g.finish = []func(context.Context) error{func(context.Context) error { finished++; return nil }}
	g.stops = []func(){func() { stopped++ }}

	ctx := withComputeRelease(context.Background(), g.releaseCompute)
	if err := releaseComputeClaims(ctx); err != nil {
		t.Fatalf("early release: %v", err)
	}
	if released != 1 || stopped != 1 {
		t.Fatalf("early release: released=%d stopped=%d, want 1 and 1", released, stopped)
	}
	if finished != 0 {
		t.Fatalf("early release must not finish the worker, got %d", finished)
	}
	if err := g.release(context.Background()); err != nil {
		t.Fatalf("final release: %v", err)
	}
	if released != 1 || finished != 1 {
		t.Fatalf("final release: released=%d finished=%d, want 1 and 1", released, finished)
	}
}

func TestReleaseComputeClaimsWithoutAHookIsANoop(t *testing.T) {
	if err := releaseComputeClaims(context.Background()); err != nil {
		t.Fatalf("no hook: %v", err)
	}
}

func TestReleaseComputeClaimsReportsAFailedRelease(t *testing.T) {
	g := &leaseGroup{releases: []func(context.Context) error{
		func(context.Context) error { return errors.New("broker unavailable") },
	}}
	err := releaseComputeClaims(withComputeRelease(context.Background(), g.releaseCompute))
	if err == nil || !strings.Contains(err.Error(), "broker unavailable") {
		t.Fatalf("a failed release must be reported, got %v", err)
	}
	if g.releases[0] == nil {
		t.Fatal("a failed release must stay retryable for the manager's final release")
	}
}
