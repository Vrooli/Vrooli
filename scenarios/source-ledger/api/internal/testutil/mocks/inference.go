package mocks

import (
	"context"
	"sync"

	"source-ledger/internal/inference"
)

// FakeInference is a controllable inference.Client for domain tests.
type FakeInference struct {
	mu           sync.Mutex
	EmbedOut     []float64
	EmbedErr     error
	ClassifyOut  string
	ClassifyErr  error
	SummarizeOut string
	SummarizeErr error
	EmbedTasks   []inference.EmbeddingTask
}

func (f *FakeInference) Embed(_ context.Context, _ string, task inference.EmbeddingTask) ([]float64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.EmbedTasks = append(f.EmbedTasks, task)
	return append([]float64(nil), f.EmbedOut...), f.EmbedErr
}

func (f *FakeInference) Classify(context.Context, string) (string, error) {
	return f.ClassifyOut, f.ClassifyErr
}

func (f *FakeInference) Summarize(context.Context, string) (string, error) {
	return f.SummarizeOut, f.SummarizeErr
}

var _ inference.Client = (*FakeInference)(nil)
