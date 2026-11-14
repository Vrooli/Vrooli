package aisearch

import (
	"testing"

	pkg "github.com/vrooli/ai-go/search"
)

func TestCollectionForTuningVersionsHybridSchema(t *testing.T) {
	const dense = "scenario-dependency-analyzer-dependencies"
	if got := collectionForTuning(dense, pkg.TuningConfig{Engine: pkg.EngineDense}); got != dense {
		t.Fatalf("dense collection = %q, want %q", got, dense)
	}
	want := dense + HybridCollectionSuffix
	if got := collectionForTuning(dense, pkg.TuningConfig{Engine: pkg.EngineHybrid}); got != want {
		t.Fatalf("hybrid collection = %q, want %q", got, want)
	}
	if got := collectionForTuning(want, pkg.TuningConfig{Engine: pkg.EngineHybrid}); got != want {
		t.Fatalf("already-versioned hybrid collection = %q, want %q", got, want)
	}
}
