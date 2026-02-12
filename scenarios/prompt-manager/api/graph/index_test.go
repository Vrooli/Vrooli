package graph

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

	// Second call: reads from file, Build NOT called again
	idx, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("second Get: %v", err)
	}
	if mb.callCount != 1 {
		t.Fatalf("expected Build still called once (cached), got %d", mb.callCount)
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

	// Next Get should call Build again
	_, err = s.Get(context.Background())
	if err != nil {
		t.Fatalf("post-invalidate Get: %v", err)
	}
	if mb.callCount != 2 {
		t.Fatalf("expected 2 Build calls after invalidate, got %d", mb.callCount)
	}
}

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
