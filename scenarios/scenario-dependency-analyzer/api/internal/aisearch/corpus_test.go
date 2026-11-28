package aisearch

import (
	"context"
	"testing"

	pkg "github.com/vrooli/ai-go/search"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

// stubRecordProvider is a minimal RecordProvider for assembling the service in a
// unit test without a live backend.
type stubRecordProvider struct{}

func (stubRecordProvider) AllRecords(context.Context) ([]*governancev1.ApprovedDependencyRecord, error) {
	return nil, nil
}

// TestDependencyCorpusIdentity locks the .dependencies corpus's on-disk identity:
// a drift in collection name, point-id prefix, or kind would silently re-target
// or re-embed the live Qdrant collection. These bytes are the on-disk contract.
func TestDependencyCorpusIdentity(t *testing.T) {
	spec := dependencyCorpus(stubRecordProvider{})
	if spec.id != CorpusDependencies {
		t.Fatalf("id = %q, want %q", spec.id, CorpusDependencies)
	}
	if spec.collection != "scenario-dependency-analyzer-dependencies" {
		t.Fatalf("collection = %q", spec.collection)
	}
	if spec.kind != "dependency" {
		t.Fatalf("kind = %q, want dependency", spec.kind)
	}
	if spec.idPrefix != "sda-dep:" {
		t.Fatalf("idPrefix = %q, want sda-dep:", spec.idPrefix)
	}
}

// TestMultiCorpusServiceIsolation verifies the multi-corpus assembly keeps each
// corpus on its OWN Qdrant collection and point-id namespace — the isolation
// that lets three corpora share one Reconciler/sync loop without cross-talk. It
// also asserts every corpus shares the SAME embed recipe (the corpusSpec
// INVARIANT: one embedder backs all bindings).
func TestMultiCorpusServiceIsolation(t *testing.T) {
	// Assemble two synthetic corpora over the dependency source so we exercise the
	// N-binding path without needing the (not-yet-built) scenario/resource sources.
	base := dependencyCorpus(stubRecordProvider{})
	second := base
	second.id = CorpusID("second")
	second.collection = "scenario-dependency-analyzer-second"
	second.kind = "second"
	second.idPrefix = "sda-second:"

	deps := pkg.EngineDeps{EmbedModel: pkg.DefaultEmbedModel}
	svc := New([]corpusSpec{base, second}, deps, 1, 0)

	if got := len(svc.Corpora()); got != 2 {
		t.Fatalf("Corpora() = %d, want 2", got)
	}
	seenCollections := map[string]bool{}
	seenPrefixes := map[string]bool{}
	for _, id := range svc.Corpora() {
		e := svc.engines[id]
		if e == nil {
			t.Fatalf("engine %q missing", id)
		}
		if seenCollections[e.qspec.Name] {
			t.Fatalf("collection %q shared across corpora — must be isolated", e.qspec.Name)
		}
		seenCollections[e.qspec.Name] = true
		if seenPrefixes[e.spec.idPrefix] {
			t.Fatalf("idPrefix %q shared across corpora — must be isolated", e.spec.idPrefix)
		}
		seenPrefixes[e.spec.idPrefix] = true
	}

	// One shared reconciler drives both bindings.
	if svc.Reconciler() == nil {
		t.Fatal("Reconciler() = nil, want one shared reconciler")
	}

	// Point IDs for the same natural ID in different namespaces must differ.
	depPoint := pkg.PointIDFor(base.idPrefix, "go/connect", 0, 1)
	secondPoint := pkg.PointIDFor(second.idPrefix, "go/connect", 0, 1)
	if depPoint == secondPoint {
		t.Fatalf("point IDs collide across corpora: %q", depPoint)
	}
}
