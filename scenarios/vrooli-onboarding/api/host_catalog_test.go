package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHostManifestCatalogReadsDirectoryOnceAndPreservesFirstMatch(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "internal", "tools")
	for name, body := range map[string]string{
		"a-first":  `{"name":"shared","description":"first"}`,
		"b-second": `{"name":"shared","description":"second"}`,
		"c-other":  `{"name":"other","description":"other tool"}`,
	} {
		path := filepath.Join(directory, name)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "tool.json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	readDirs := 0
	readFiles := 0
	items, err := hostItemsWithIO(root, "tools", map[string]hostRequirement{
		"other":  {Name: "other", Required: true},
		"shared": {Name: "shared", Required: false},
	}, map[string]OptInChoice{"shared": {OptedIn: boolPointer(true)}}, func(path string) ([]os.DirEntry, error) {
		readDirs++
		return os.ReadDir(path)
	}, func(path string) ([]byte, error) {
		readFiles++
		return os.ReadFile(path)
	})
	if err != nil {
		t.Fatal(err)
	}
	if readDirs != 1 {
		t.Fatalf("catalog directory reads = %d, want one", readDirs)
	}
	if readFiles != 3 {
		t.Fatalf("manifest reads = %d, want one per scanned catalog entry", readFiles)
	}

	if len(items) != 2 || items[0].Name != "other" || items[1].Name != "shared" {
		t.Fatalf("items = %+v, want sorted catalog projection", items)
	}
	if items[1].Description != "first" || items[1].Status != "opted_in" {
		t.Fatalf("duplicate catalog identity projection = %+v, want first manifest with opted-in state", items[1])
	}
}

func boolPointer(value bool) *bool { return &value }

func TestHostItemsDoNotRequireCatalogForEmptyRequirementSet(t *testing.T) {
	called := false
	items, err := hostItemsWithIO("/absent", "safeguards", nil, nil, func(string) ([]os.DirEntry, error) {
		called = true
		return nil, os.ErrNotExist
	}, os.ReadFile)
	if err != nil {
		t.Fatalf("empty requirement projection error = %v", err)
	}
	if called || len(items) != 0 {
		t.Fatalf("empty projection read catalog=%v items=%v, want no read and empty result", called, items)
	}
}
