package mocks

import (
	"context"

	"audio-tools/internal/ai/sttchain"
)

// FakeVrooliClient satisfies sttchain.VrooliClient. Tests configure
// Available and Result/Err for canned behavior, or TranscribeFn for
// per-call control.
type FakeVrooliClient struct {
	Available    bool
	Result       *sttchain.Result
	Err          error
	TranscribeFn func(ctx context.Context, token, identity string, req sttchain.Request) (*sttchain.Result, error)
}

func (c *FakeVrooliClient) IsAvailable(context.Context) bool { return c.Available }
func (c *FakeVrooliClient) Model() string                    { return "lpbs-model" }
func (c *FakeVrooliClient) Transcribe(ctx context.Context, token, identity string, req sttchain.Request) (*sttchain.Result, error) {
	if c.TranscribeFn != nil {
		return c.TranscribeFn(ctx, token, identity, req)
	}
	if c.Err != nil {
		return nil, c.Err
	}
	if c.Result != nil {
		return c.Result, nil
	}
	return &sttchain.Result{Text: "vrooli", Tier: sttchain.TierVrooli}, nil
}

var _ sttchain.VrooliClient = (*FakeVrooliClient)(nil)
