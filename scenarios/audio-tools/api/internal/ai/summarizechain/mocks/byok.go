// Package mocks holds the hoisted test fakes for the summarize chain.
package mocks

import (
	"context"

	"audio-tools/internal/ai/summarizechain"
)

// FakeBYOK satisfies summarizechain.BYOKAdapter.
type FakeBYOK struct {
	IDStr       string
	Available   bool
	Result      *summarizechain.Result
	Err         error
	SummarizeFn func(ctx context.Context, key string, req summarizechain.Request) (*summarizechain.Result, error)
}

func (f *FakeBYOK) ID() string                               { return f.IDStr }
func (f *FakeBYOK) IsAvailable(context.Context, string) bool { return f.Available }
func (f *FakeBYOK) Model() string                            { return "fake-model" }
func (f *FakeBYOK) Summarize(ctx context.Context, key string, req summarizechain.Request) (*summarizechain.Result, error) {
	if f.SummarizeFn != nil {
		return f.SummarizeFn(ctx, key, req)
	}
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Result != nil {
		return f.Result, nil
	}
	return &summarizechain.Result{Text: "byok-summary"}, nil
}

var _ summarizechain.BYOKAdapter = (*FakeBYOK)(nil)
