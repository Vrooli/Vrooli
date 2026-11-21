package runtimeapp

import (
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestRebaselineProgressStateFormatDescribesItsFiles(t *testing.T) {
	store, err := rebaselineProgressStateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != rebaselineProgressSchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s; a deliberate shape change must bump Progress.Version and re-pin", got, rebaselineProgressSchemaDigest)
	}
	// The descriptor must name the same files the transition writer uses, or a
	// forced transition would snapshot a path nothing writes.
	home := t.TempDir()
	paths, err := preparedRebaselineStatePathsForHome(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.StatePaths) != 2 {
		t.Fatalf("state paths = %v, want the progress and pause files", store.StatePaths)
	}
	for _, declared := range store.StatePaths {
		if !endsWith(declared, "root-rebaseline/progress.json") && !endsWith(declared, "root-rebaseline/pause.json") {
			t.Fatalf("unexpected state path %q (writer uses %+v)", declared, paths)
		}
	}
}

func endsWith(value, suffix string) bool {
	return len(value) >= len(suffix) && value[len(value)-len(suffix):] == suffix
}
