package aisearch

import (
	"context"
	"testing"
)

// fakeEmbedder returns a fixed vector regardless of input — the contents
// don't matter for threshold-passthrough verification.
type fakeEmbedder struct{}

func (fakeEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	return []float64{0, 0, 0}, nil
}
func (fakeEmbedder) Available(_ context.Context) bool { return true }

// thresholdSpyStore records the threshold Search receives and returns an
// empty result list. We assert ui-health's configured threshold reaches the
// vector layer unchanged.
type thresholdSpyStore struct {
	gotThreshold float64
	called       bool
}

func (s *thresholdSpyStore) EnsureCollection(context.Context) error { return nil }
func (s *thresholdSpyStore) Upsert(context.Context, string, []float64, map[string]interface{}) error {
	return nil
}
func (s *thresholdSpyStore) Delete(context.Context, string) error          { return nil }
func (s *thresholdSpyStore) BatchDelete(context.Context, []string) error   { return nil }
func (s *thresholdSpyStore) CountPoints(context.Context) (int, error)      { return 0, nil }
func (s *thresholdSpyStore) ScrollIDs(context.Context) (map[string]ScrollItem, error) {
	return nil, nil
}
func (s *thresholdSpyStore) Available(context.Context) bool { return true }
func (s *thresholdSpyStore) Search(_ context.Context, _ []float64, _ int, threshold float64) ([]SearchResult, error) {
	s.gotThreshold = threshold
	s.called = true
	return nil, nil
}

// TestSearchThresholdPassthrough verifies the configured Options.Threshold
// reaches the vector store's Search call. The threshold is how ui-health
// filters out weak embedding matches that would otherwise crowd search
// results when no real match exists.
func TestSearchThresholdPassthrough(t *testing.T) {
	t.Parallel()
	spy := &thresholdSpyStore{}
	svc := NewService(Options{
		Embedder:    fakeEmbedder{},
		VectorStore: spy,
		Threshold:   0.55,
	})
	if _, err := svc.Search(context.Background(), "anything", 10, ModeAI); err != nil {
		t.Fatalf("Search: %v", err)
	}
	if !spy.called {
		t.Fatalf("VectorStore.Search was never invoked")
	}
	if spy.gotThreshold != 0.55 {
		t.Fatalf("threshold passthrough: got %g, want 0.55", spy.gotThreshold)
	}
}
