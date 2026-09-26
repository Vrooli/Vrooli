package mocks

import (
	"context"

	"audio-tools/internal/ai/ttschain"
)

// FakeVrooliClient satisfies ttschain.VrooliClient.
type FakeVrooliClient struct {
	Available    bool
	Result       *ttschain.Result
	Err          error
	SynthesizeFn func(ctx context.Context, token, identity string, req ttschain.Request) (*ttschain.Result, error)
}

func (c *FakeVrooliClient) IsAvailable(context.Context) bool { return c.Available }
func (c *FakeVrooliClient) Model() string                    { return "lpbs-tts" }
func (c *FakeVrooliClient) Synthesize(ctx context.Context, token, identity string, req ttschain.Request) (*ttschain.Result, error) {
	if c.SynthesizeFn != nil {
		return c.SynthesizeFn(ctx, token, identity, req)
	}
	if c.Err != nil {
		return nil, c.Err
	}
	if c.Result != nil {
		return c.Result, nil
	}
	return &ttschain.Result{Audio: []byte("vrooli-audio"), ContentType: "audio/mpeg"}, nil
}

var _ ttschain.VrooliClient = (*FakeVrooliClient)(nil)
