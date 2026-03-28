package graph

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type stubProjector struct {
	mu        sync.Mutex
	calls     map[Lens]int
	responses map[Lens]GraphResponse
	err       error
	block     chan struct{}
}

func (s *stubProjector) Project(_ context.Context, lens Lens) (GraphResponse, error) {
	if s.block != nil {
		<-s.block
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.calls == nil {
		s.calls = make(map[Lens]int)
	}
	s.calls[lens]++
	if s.err != nil {
		return GraphResponse{}, s.err
	}
	if resp, ok := s.responses[lens]; ok {
		return resp, nil
	}
	return NewGraphResponse(lens, nil, nil), nil
}

func TestProjectionCacheCachesPerLens(t *testing.T) {
	projector := &stubProjector{
		responses: map[Lens]GraphResponse{
			LensTopology: NewGraphResponse(LensTopology, []Node{{ID: "n1"}}, nil),
		},
	}
	cache := NewProjectionCache(ProjectionCacheConfig{
		Projector: projector,
		TTL:       time.Minute,
	})

	first, err := cache.Project(context.Background(), LensTopology)
	if err != nil {
		t.Fatalf("first project failed: %v", err)
	}
	second, err := cache.Project(context.Background(), LensTopology)
	if err != nil {
		t.Fatalf("second project failed: %v", err)
	}

	if got := projector.calls[LensTopology]; got != 1 {
		t.Fatalf("expected 1 projector call, got %d", got)
	}
	if first.Meta.GeneratedAt != second.Meta.GeneratedAt {
		t.Fatal("expected cached response to be reused")
	}
}

func TestProjectionCacheInvalidateForcesRebuild(t *testing.T) {
	projector := &stubProjector{
		responses: map[Lens]GraphResponse{
			LensTopology: NewGraphResponse(LensTopology, []Node{{ID: "n1"}}, nil),
		},
	}
	cache := NewProjectionCache(ProjectionCacheConfig{
		Projector: projector,
		TTL:       time.Minute,
	})

	if _, err := cache.Project(context.Background(), LensTopology); err != nil {
		t.Fatalf("first project failed: %v", err)
	}

	cache.Invalidate(LensTopology)
	if _, err := cache.Project(context.Background(), LensTopology); err != nil {
		t.Fatalf("second project failed: %v", err)
	}

	if got := projector.calls[LensTopology]; got != 2 {
		t.Fatalf("expected 2 projector calls after invalidation, got %d", got)
	}
}

func TestProjectionCacheCoalescesConcurrentBuilds(t *testing.T) {
	projector := &stubProjector{
		responses: map[Lens]GraphResponse{
			LensTopology: NewGraphResponse(LensTopology, []Node{{ID: "n1"}}, nil),
		},
		block: make(chan struct{}),
	}
	cache := NewProjectionCache(ProjectionCacheConfig{
		Projector: projector,
		TTL:       time.Minute,
	})

	var wg sync.WaitGroup
	wg.Add(2)

	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			_, err := cache.Project(context.Background(), LensTopology)
			errs <- err
		}()
	}

	close(projector.block)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("project failed: %v", err)
		}
	}
	if got := projector.calls[LensTopology]; got != 1 {
		t.Fatalf("expected 1 projector call, got %d", got)
	}
}

func TestProjectionCacheServesStaleResponseOnRebuildError(t *testing.T) {
	projector := &stubProjector{
		responses: map[Lens]GraphResponse{
			LensTopology: NewGraphResponse(LensTopology, []Node{{ID: "n1"}}, nil),
		},
	}
	cache := NewProjectionCache(ProjectionCacheConfig{
		Projector: projector,
		TTL:       time.Minute,
	})
	cache.now = func() time.Time { return time.Unix(100, 0) }

	initial, err := cache.Project(context.Background(), LensTopology)
	if err != nil {
		t.Fatalf("initial project failed: %v", err)
	}

	projector.err = errors.New("boom")
	cache.now = func() time.Time { return time.Unix(1000, 0) }

	stale, err := cache.Project(context.Background(), LensTopology)
	if err != nil {
		t.Fatalf("expected stale response instead of error: %v", err)
	}
	if stale.Meta.GeneratedAt != initial.Meta.GeneratedAt {
		t.Fatal("expected stale cached response to be served")
	}
}
