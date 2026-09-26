package mocks

import (
	"context"
	"sync/atomic"

	"signal-inbox/internal/inference"
)

// FakeInference supplies deterministic embeddings and classifications to domain tests.
type FakeInference struct {
	EmbedOut      []float64
	EmbedErr      error
	ClassifyOut   string
	ClassifyErr   error
	EmbedCalls    atomic.Int64
	ClassifyCalls atomic.Int64
}

func (f *FakeInference) Embed(_ context.Context, _ string, _ inference.EmbeddingTask) ([]float64, error) {
	f.EmbedCalls.Add(1)
	return f.EmbedOut, f.EmbedErr
}

func (f *FakeInference) Classify(context.Context, string) (string, error) {
	f.ClassifyCalls.Add(1)
	return f.ClassifyOut, f.ClassifyErr
}

var _ inference.Client = (*FakeInference)(nil)
