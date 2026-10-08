package graph

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// IndexStore tests
// ---------------------------------------------------------------------------

func TestIndexStore_Get_Regenerates(t *testing.T) {
	dir := t.TempDir()
	mb := &mockGraphBuilder{
		graph: Graph{
			Nodes: []Node{{ID: "n1", Type: NodeAgent, Label: "Agent"}},
		},
	}
	s := NewIndexStore(dir, mb)

	idx, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mb.callCount != 1 {
		t.Fatalf("expected Build called once, got %d", mb.callCount)
	}
	if len(idx.Graph.Nodes) != 1 || idx.Graph.Nodes[0].ID != "n1" {
		t.Errorf("unexpected graph: %+v", idx.Graph)
	}
	// File should exist
	if _, err := os.Stat(s.indexPath()); os.IsNotExist(err) {
		t.Error("expected index file to be written")
	}
}

func TestIndexStore_Get_ReadsCached(t *testing.T) {
	dir := t.TempDir()
	mb := &mockGraphBuilder{
		graph: Graph{
			Nodes: []Node{{ID: "n1", Type: NodeAgent}},
		},
	}
	s := NewIndexStore(dir, mb)

	// First call: regenerates
	_, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}
	if mb.callCount != 1 {
		t.Fatalf("expected 1 Build call, got %d", mb.callCount)
	}

	// Second call: returns from in-memory cache, Build NOT called again
	idx, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("second Get: %v", err)
	}
	if mb.callCount != 1 {
		t.Fatalf("expected Build still called once (in-memory cached), got %d", mb.callCount)
	}
	if len(idx.Graph.Nodes) != 1 {
		t.Errorf("expected cached data, got %+v", idx.Graph)
	}
}

func TestIndexStore_Invalidate(t *testing.T) {
	dir := t.TempDir()
	mb := &mockGraphBuilder{
		graph: Graph{
			Nodes: []Node{{ID: "n1", Type: NodeAgent}},
		},
	}
	s := NewIndexStore(dir, mb)

	// Populate cache
	_, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("initial Get: %v", err)
	}
	if mb.callCount != 1 {
		t.Fatalf("expected 1 Build call, got %d", mb.callCount)
	}

	// Invalidate
	s.Invalidate()

	// File should be removed
	if _, err := os.Stat(s.indexPath()); !os.IsNotExist(err) {
		t.Error("expected index file to be removed after Invalidate")
	}

	// Next Get should call Build again (in-memory cache was cleared)
	_, err = s.Get(context.Background())
	if err != nil {
		t.Fatalf("post-invalidate Get: %v", err)
	}
	if mb.callCount != 2 {
		t.Fatalf("expected 2 Build calls after invalidate, got %d", mb.callCount)
	}
}

func TestIndexStore_InvalidateInvalidatesDerivedCaches(t *testing.T) {
	s := NewIndexStore(t.TempDir(), &mockGraphBuilder{})
	dependent := &countingInvalidator{}
	s.SetDependentInvalidators(dependent)
	s.Invalidate()
	if dependent.calls != 1 {
		t.Fatalf("expected dependent cache invalidated once, got %d", dependent.calls)
	}
}

type countingInvalidator struct{ calls int }

func (i *countingInvalidator) Invalidate() { i.calls++ }

func TestIndexStore_Regenerate(t *testing.T) {
	dir := t.TempDir()
	mb := &mockGraphBuilder{
		graph: Graph{
			Nodes: []Node{{ID: "n1", Type: NodeAgent}},
		},
	}
	s := NewIndexStore(dir, mb)

	err := s.Regenerate(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mb.callCount != 1 {
		t.Fatalf("expected 1 Build call, got %d", mb.callCount)
	}

	// Call again - always rebuilds
	err = s.Regenerate(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on second regen: %v", err)
	}
	if mb.callCount != 2 {
		t.Fatalf("expected 2 Build calls, got %d", mb.callCount)
	}
}

func TestIndexStore_BuildError(t *testing.T) {
	dir := t.TempDir()
	mb := &mockGraphBuilder{err: errors.New("build fail")}
	s := NewIndexStore(dir, mb)

	_, err := s.Get(context.Background())
	if err == nil || err.Error() != "build fail" {
		t.Fatalf("expected build error, got: %v", err)
	}
}

func TestIndexStore_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	mb := &mockGraphBuilder{
		graph: Graph{
			Nodes: []Node{{ID: "n1", Type: NodeAgent}},
		},
	}
	s := NewIndexStore(dir, mb)

	// Write corrupt JSON to the index path
	indexDir := filepath.Dir(s.indexPath())
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(s.indexPath(), []byte("{invalid json!!!"), 0o644); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}

	// Get should detect corrupt file and regenerate
	idx, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mb.callCount != 1 {
		t.Fatalf("expected Build called once (regenerate), got %d", mb.callCount)
	}
	if len(idx.Graph.Nodes) != 1 {
		t.Errorf("expected regenerated data, got %+v", idx.Graph)
	}
}

// ---------------------------------------------------------------------------
// Regression tests: in-memory cache performance
// ---------------------------------------------------------------------------

// TestIndexStore_InMemoryCache_SkipsDiskOnSubsequentGets verifies that
// after the first Get(), subsequent calls return from the in-memory cache
// without touching the disk. This is the core performance fix: every
// GET /api/v1/graph no longer re-reads and re-parses the JSON file.
func TestIndexStore_InMemoryCache_SkipsDiskOnSubsequentGets(t *testing.T) {
	dir := t.TempDir()
	mb := &mockGraphBuilder{
		graph: Graph{
			Nodes: []Node{{ID: "n1", Type: NodeSkill}},
			Edges: []Edge{{From: "n1", To: "n2", Kind: EdgeCLIRead}},
		},
	}
	s := NewIndexStore(dir, mb)

	// First Get: populates both disk and in-memory cache.
	first, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}

	// Delete the on-disk file. If Get() relies on disk, this would force a rebuild.
	_ = os.Remove(s.indexPath())

	// Second Get: must return from in-memory cache without rebuilding.
	second, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("second Get: %v", err)
	}
	if mb.callCount != 1 {
		t.Fatalf("expected Build called once (in-memory cache should avoid disk), got %d", mb.callCount)
	}
	// Must be the same pointer (in-memory cache returns the cached reference).
	if first != second {
		t.Error("expected same *GraphIndex pointer from in-memory cache")
	}
}

// TestIndexStore_InMemoryCache_ConcurrentGets verifies that concurrent
// Get() calls don't cause races or duplicate builds.
func TestIndexStore_InMemoryCache_ConcurrentGets(t *testing.T) {
	dir := t.TempDir()
	mb := &mockGraphBuilder{
		graph: Graph{
			Nodes: []Node{{ID: "n1", Type: NodeAgent}},
		},
	}
	s := NewIndexStore(dir, mb)

	// Warm the cache.
	_, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("warm-up Get: %v", err)
	}

	// Fire 50 concurrent Get() calls.
	const goroutines = 50
	var wg sync.WaitGroup
	errs := make([]error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = s.Get(context.Background())
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: unexpected error: %v", i, err)
		}
	}

	// Only the warm-up call should have triggered Build.
	if mb.callCount != 1 {
		t.Fatalf("expected 1 Build call (all concurrent Gets served from cache), got %d", mb.callCount)
	}
}

// TestIndexStore_Regenerate_UpdatesInMemoryCache verifies that Regenerate()
// updates the in-memory cache so subsequent Get() calls return the new data.
func TestIndexStore_Regenerate_UpdatesInMemoryCache(t *testing.T) {
	dir := t.TempDir()
	mb := &mockGraphBuilder{
		graph: Graph{
			Nodes: []Node{{ID: "n1", Type: NodeAgent}},
		},
	}
	s := NewIndexStore(dir, mb)

	// Populate cache.
	_, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("initial Get: %v", err)
	}

	// Change builder output and regenerate.
	mb.graph = Graph{
		Nodes: []Node{{ID: "n2", Type: NodeSkill}},
	}
	if err := s.Regenerate(context.Background()); err != nil {
		t.Fatalf("Regenerate: %v", err)
	}

	// Get should return the new data from in-memory cache (no disk read needed).
	_ = os.Remove(s.indexPath()) // remove disk to prove in-memory is used
	idx, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("post-regenerate Get: %v", err)
	}
	if len(idx.Graph.Nodes) != 1 || idx.Graph.Nodes[0].ID != "n2" {
		t.Errorf("expected regenerated data (n2), got %+v", idx.Graph)
	}
	if mb.callCount != 2 {
		t.Fatalf("expected 2 Build calls (initial + regenerate), got %d", mb.callCount)
	}
}

// TestIndexStore_DiskFallback_WhenNoInMemoryCache verifies that when
// the process is cold (no in-memory cache) but a valid disk file exists,
// Get() loads from disk without calling Build.
func TestIndexStore_DiskFallback_WhenNoInMemoryCache(t *testing.T) {
	dir := t.TempDir()
	mb := &mockGraphBuilder{
		graph: Graph{
			Nodes: []Node{{ID: "n1", Type: NodeAgent}},
		},
	}

	// Store 1: build and persist to disk.
	s1 := NewIndexStore(dir, mb)
	_, err := s1.Get(context.Background())
	if err != nil {
		t.Fatalf("s1 Get: %v", err)
	}
	if mb.callCount != 1 {
		t.Fatalf("expected 1 Build call, got %d", mb.callCount)
	}

	// Store 2: simulates a new process (cold in-memory cache, disk exists).
	s2 := NewIndexStore(dir, mb)
	idx, err := s2.Get(context.Background())
	if err != nil {
		t.Fatalf("s2 Get: %v", err)
	}
	// Build should NOT have been called again — loaded from disk.
	if mb.callCount != 1 {
		t.Fatalf("expected Build still at 1 (loaded from disk), got %d", mb.callCount)
	}
	if len(idx.Graph.Nodes) != 1 || idx.Graph.Nodes[0].ID != "n1" {
		t.Errorf("expected disk-loaded data, got %+v", idx.Graph)
	}
}
