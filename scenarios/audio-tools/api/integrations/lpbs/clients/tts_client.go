package clients

import (
	"context"

	"audio-tools/internal/ai/ttschain"
)

type TTSClient struct {
	UnavailableClient
}

func NewTTSClient(baseURL string) *TTSClient {
	return &TTSClient{UnavailableClient: newUnavailableClient(baseURL)}
}

func (c *TTSClient) Synthesize(ctx context.Context, lpbsToken, userIdentity string, req ttschain.Request) (*ttschain.Result, error) {
	return nil, Unimplemented("tts")
}
