package clients

import (
	"context"
	"fmt"

	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/httpc"
)

type TTSClient struct {
	BaseURL string
	Doer    httpc.Doer
}

func NewTTSClient(baseURL string) *TTSClient {
	return &TTSClient{
		BaseURL: baseURL,
		Doer:    httpc.DefaultDoer(),
	}
}

func (c *TTSClient) IsAvailable(ctx context.Context) bool { return false }
func (c *TTSClient) Model() string                        { return "lpbs-default" }

func (c *TTSClient) Synthesize(ctx context.Context, lpbsToken, userIdentity string, req ttschain.Request) (*ttschain.Result, error) {
	return nil, fmt.Errorf("lpbs-tts: gateway endpoint not implemented (execute/lpbs-audio-gateway-endpoints)")
}
