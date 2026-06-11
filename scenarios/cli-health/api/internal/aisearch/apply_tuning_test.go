package aisearch

import (
	"context"
	"errors"
	"strings"
	"testing"

	pkg "github.com/vrooli/ai-go/search"
)

// errEnsureStore wraps a fake store but fails EnsureCollection — modelling the
// schema-guard rejection of a structural index-time change (e.g. dense↔hybrid).
type errEnsureStore struct {
	*fakeVectorStore
	ensureErr error
}

func (s errEnsureStore) EnsureCollection(_ context.Context, _ pkg.CollectionSpec) error {
	return s.ensureErr
}

// tunedTestService builds a Service whose rebuild closure returns an engine over
// `next` (ignoring the tuning value — we are exercising the SWAP mechanics, not
// the factor wiring, which NewServiceForTuning owns and the live test covers).
func tunedTestService(disc DiscoverySource, cur, next *engine) *Service {
	return &Service{eng: cur, rebuild: func(pkg.TuningConfig) *engine { return next }}
}

func TestApplyTuningRequiresTunedConstructor(t *testing.T) {
	// A Service built from explicit components (NewService) has no factor builder,
	// so it cannot rebuild itself for a tuning — ApplyTuning must say so clearly
	// rather than silently no-op.
	svc := newTestService(sampleCorpus(), &fakeEmbedder{available: true}, newFakeStore())
	_, _, _, err := svc.ApplyTuning(context.Background(), pkg.CommandCorpusTuning())
	if err == nil || !strings.Contains(err.Error(), "not tuning-rebuildable") {
		t.Fatalf("want a not-tuning-rebuildable error, got %v", err)
	}
}

func TestApplyTuningSwapsEngineAndReembeds(t *testing.T) {
	disc := sampleCorpus()
	store1, store2 := newFakeStore(), newFakeStore()
	eng1 := buildEngine(Options{Embedder: &fakeEmbedder{available: true}, VectorStore: store1, Discovery: disc, Parallelism: 2})
	eng2 := buildEngine(Options{Embedder: &fakeEmbedder{available: true}, VectorStore: store2, Discovery: disc, Parallelism: 2})
	svc := tunedTestService(disc, eng1, eng2)

	// Build the initial index into store1.
	job1, err := svc.Reindex(context.Background(), "", false)
	if err != nil {
		t.Fatalf("initial reindex: %v", err)
	}
	waitJob(t, svc, job1.ID)
	if len(store1.points) == 0 {
		t.Fatalf("initial reindex populated nothing in store1")
	}
	if len(store2.points) != 0 {
		t.Fatalf("store2 should be empty before the apply, got %d", len(store2.points))
	}

	// Apply a (notionally index-time) tuning: the engine must swap to eng2 and the
	// corpus must re-embed into store2 — proving the live recipe apply re-embeds
	// without a process restart.
	jobID, _, _, err := svc.ApplyTuning(context.Background(), pkg.CommandCorpusTuning())
	if err != nil {
		t.Fatalf("ApplyTuning: %v", err)
	}
	waitJob(t, svc, jobID)

	if svc.current() != eng2 {
		t.Fatalf("ApplyTuning did not swap the live engine")
	}
	if len(store2.points) != len(store1.points) {
		t.Fatalf("re-embed into the swapped store incomplete: store2=%d want %d", len(store2.points), len(store1.points))
	}
	// The forwarders must now resolve against the swapped engine.
	if svc.Reconciler() != eng2.svc.Reconciler() {
		t.Fatalf("Reconciler() did not follow the swap (sync loop would reconcile the stale engine)")
	}
}

func TestApplyTuningStructuralMismatchDoesNotSwap(t *testing.T) {
	disc := sampleCorpus()
	store1 := newFakeStore()
	eng1 := buildEngine(Options{Embedder: &fakeEmbedder{available: true}, VectorStore: store1, Discovery: disc, Parallelism: 2})
	// The rebuilt engine's store rejects EnsureCollection (schema guard) — a
	// structural change like dense↔hybrid. ApplyTuning must surface the error and
	// leave the live engine untouched (no auto-drop, no half-applied swap).
	badStore := errEnsureStore{fakeVectorStore: newFakeStore(), ensureErr: errors.New("collection schema mismatch")}
	eng2 := buildEngine(Options{Embedder: &fakeEmbedder{available: true}, VectorStore: badStore, Discovery: disc, Parallelism: 2})
	svc := tunedTestService(disc, eng1, eng2)

	_, _, _, err := svc.ApplyTuning(context.Background(), pkg.CommandCorpusTuning())
	if err == nil || !strings.Contains(err.Error(), "ensure collection") {
		t.Fatalf("want an ensure-collection error, got %v", err)
	}
	if svc.current() != eng1 {
		t.Fatalf("ApplyTuning swapped the engine despite a structural mismatch")
	}
}
