package graph

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestFileExtractionCacheRoundTrip(t *testing.T) {
	cache, err := NewFileExtractionCache(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileExtractionCache: %v", err)
	}
	wantGraph := Graph{Nodes: []Node{{ID: "package:m", Kind: NodeKindPackage, Name: "m"}}}
	wantWarnings := []Warning{{Kind: WarningKindParseError, File: "main.go", Message: "partial"}}
	if err := cache.Put("cache-key", wantGraph, wantWarnings); err != nil {
		t.Fatalf("Put: %v", err)
	}
	gotGraph, gotWarnings, ok := cache.Get("cache-key")
	if !ok {
		t.Fatal("Get reported cache miss")
	}
	if len(gotGraph.Nodes) != 1 || gotGraph.Nodes[0].ID != "package:m" {
		t.Fatalf("graph = %#v", gotGraph)
	}
	if len(gotWarnings) != 1 || gotWarnings[0].Message != "partial" {
		t.Fatalf("warnings = %#v", gotWarnings)
	}
}

func TestFileExtractionCacheServesWarmEntriesFromMemory(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewFileExtractionCache(dir)
	if err != nil {
		t.Fatalf("NewFileExtractionCache: %v", err)
	}
	graph := Graph{Nodes: []Node{{ID: "package:warm"}}}
	if err := cache.Put("warm", graph, nil); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "warm.json")); err != nil {
		t.Fatalf("remove disk entry: %v", err)
	}
	got, _, ok := cache.Get("warm")
	if !ok || len(got.Nodes) != 1 || got.Nodes[0].ID != "package:warm" {
		t.Fatalf("warm Get = %#v, %v; want memory hit", got, ok)
	}
}

func TestFileExtractionCacheBoundsWarmMemoryEntries(t *testing.T) {
	cache, err := NewFileExtractionCache(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileExtractionCache: %v", err)
	}
	for i := 0; i <= defaultExtractionMemoryEntries; i++ {
		key := fmt.Sprintf("entry-%d", i)
		if err := cache.Put(key, Graph{Nodes: []Node{{ID: key}}}, nil); err != nil {
			t.Fatalf("Put(%q): %v", key, err)
		}
	}
	if _, ok := cache.memory["entry-0"]; ok {
		t.Fatal("oldest warm entry survived the memory bound")
	}
	if _, ok := cache.memory[fmt.Sprintf("entry-%d", defaultExtractionMemoryEntries)]; !ok {
		t.Fatal("newest warm entry was evicted")
	}
}

func TestFileExtractionCacheIgnoresCorruptEntry(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewFileExtractionCache(dir)
	if err != nil {
		t.Fatalf("NewFileExtractionCache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("not-json"), 0o644); err != nil {
		t.Fatalf("write corrupt entry: %v", err)
	}
	if _, _, ok := cache.Get("bad"); ok {
		t.Fatal("corrupt entry reported as a hit")
	}
}

func TestFileExtractionCacheEvictsOldEntriesAboveLimit(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewFileExtractionCacheWithLimit(dir, 0)
	if err != nil {
		t.Fatalf("NewFileExtractionCacheWithLimit: %v", err)
	}
	if err := cache.Put("first", Graph{Nodes: []Node{{ID: "first-entry-with-padding"}}}, nil); err != nil {
		t.Fatalf("first Put: %v", err)
	}
	firstInfo, err := os.Stat(filepath.Join(dir, "first.json"))
	if err != nil {
		t.Fatalf("stat first entry: %v", err)
	}
	cache.maxBytes = firstInfo.Size()
	if err := cache.Put("second", Graph{Nodes: []Node{{ID: "second"}}}, nil); err != nil {
		t.Fatalf("second Put: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "first.json")); !os.IsNotExist(err) {
		t.Fatalf("old entry survived disk eviction; stat error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "second.json")); err != nil {
		t.Fatalf("new entry was evicted: %v", err)
	}
}

func TestModuleFingerprintChangesWhenSourceChanges(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/m\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package m\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	opts := LoadOptions{Profile: ExtractionProfileFull}
	first, err := moduleFingerprint(root, opts)
	if err != nil {
		t.Fatalf("first fingerprint: %v", err)
	}
	if err := os.WriteFile(path, []byte("package m\n\nvar Changed = true\n"), 0o644); err != nil {
		t.Fatalf("rewrite main.go: %v", err)
	}
	second, err := moduleFingerprint(root, opts)
	if err != nil {
		t.Fatalf("second fingerprint: %v", err)
	}
	if first == second {
		t.Fatal("fingerprint did not change after source modification")
	}
}
