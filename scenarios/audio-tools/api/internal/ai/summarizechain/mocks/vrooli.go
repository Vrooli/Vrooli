package mocks

import (
	"context"

	"audio-tools/internal/ai/summarizechain"
)

// FakeVrooliClient satisfies summarizechain.VrooliClient.
type FakeVrooliClient struct {
	Available   bool
	Result      *summarizechain.Result
	Err         error
	SummarizeFn func(ctx context.Context, token, identity string, req summarizechain.Request) (*summarizechain.Result, error)
}

func (c *FakeVrooliClient) IsAvailable(context.Context) bool { return c.Available }
func (c *FakeVrooliClient) Model() string                    { return "lpbs-model" }
func (c *FakeVrooliClient) Summarize(ctx context.Context, token, identity string, req summarizechain.Request) (*summarizechain.Result, error) {
	if c.SummarizeFn != nil {
		return c.SummarizeFn(ctx, token, identity, req)
	}
	if c.Err != nil {
		return nil, c.Err
	}
	if c.Result != nil {
		return c.Result, nil
	}
	return &summarizechain.Result{Text: "vrooli-summary"}, nil
}

var _ summarizechain.VrooliClient = (*FakeVrooliClient)(nil)
