package audit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"architecture-cartographer/internal/audit"
	conflictsmocks "architecture-cartographer/internal/conflicts/mocks"
	"architecture-cartographer/internal/domains"
	domainsmocks "architecture-cartographer/internal/domains/mocks"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/testutil/mocks"
)

func TestRunLimiter_CancellationWhileQueuedReturnsPromptly(t *testing.T) {
	g := newBlockingGraph()
	d := &domainsmocks.FakeService{Map: domains.DerivedDomainMap{
		Authority:           domains.SourceDomainsDoc,
		AuthorityConfidence: domains.ConfidenceHigh,
	}}
	c := &conflictsmocks.FakeService{}
	limiter := audit.NewRunLimiter(1)
	svc := audit.NewService(g, d, c, nil, nil, nil, mocks.NewFakeClock(time.Time{}), audit.WithRunLimiter(limiter))

	firstDone := make(chan error, 1)
	go func() {
		_, err := svc.Run(context.Background(), audit.RunInput{Scenario: "demo"})
		firstDone <- err
	}()

	select {
	case <-g.extractStarted:
	case <-time.After(time.Second):
		t.Fatal("first run did not reach graph extraction")
	}
	if limiter.Active() != 1 {
		t.Fatalf("active=%d want 1", limiter.Active())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := svc.Run(ctx, audit.RunInput{Scenario: "demo"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second run err=%v want deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 250*time.Millisecond {
		t.Fatalf("queued cancellation took %s", elapsed)
	}

	close(g.release)
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("first run err: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("first run did not release")
	}
	if limiter.Active() != 0 || limiter.Queued() != 0 {
		t.Fatalf("limiter state active=%d queued=%d want zero", limiter.Active(), limiter.Queued())
	}
}

type blockingGraph struct {
	extractStarted chan struct{}
	release        chan struct{}
}

func newBlockingGraph() *blockingGraph {
	return &blockingGraph{
		extractStarted: make(chan struct{}),
		release:        make(chan struct{}),
	}
}

func (g *blockingGraph) ExtractGraph(ctx context.Context, in graph.ExtractGraphInput) (graph.GraphSnapshot, bool, error) {
	close(g.extractStarted)
	select {
	case <-g.release:
	case <-ctx.Done():
		return graph.GraphSnapshot{}, false, ctx.Err()
	}
	return graph.GraphSnapshot{
		Scenario:    in.Scenario,
		ContentHash: "hash",
	}, false, nil
}

func (g *blockingGraph) GetSnapshot(context.Context, string) (graph.GraphSnapshot, error) {
	return graph.GraphSnapshot{}, graph.ErrSnapshotNotFound{}
}

func (g *blockingGraph) LatestSnapshotMeta(context.Context, string) (graph.GraphSnapshotMeta, error) {
	return graph.GraphSnapshotMeta{}, graph.ErrSnapshotNotFound{}
}

func (g *blockingGraph) ListSnapshots(context.Context, graph.ListSnapshotsFilter) (graph.SnapshotPage, error) {
	return graph.SnapshotPage{}, nil
}

func (g *blockingGraph) ClearSnapshots(context.Context, string, bool) (int, bool, error) {
	return 0, false, nil
}

var _ graph.Service = (*blockingGraph)(nil)
