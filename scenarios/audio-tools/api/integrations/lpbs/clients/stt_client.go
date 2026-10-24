package clients

import (
	"context"

	"audio-tools/internal/ai/sttchain"
)

// STTClient routes Transcribe through the LPBS audio gateway. Implements
// sttchain.VrooliClient. The client is composed at bootstrap, but returns an
// explicit unavailable error until the shared gateway endpoint ships.
type STTClient struct {
	UnavailableClient
}

func NewSTTClient(baseURL string) *STTClient {
	return &STTClient{UnavailableClient: newUnavailableClient(baseURL)}
}

func (c *STTClient) Transcribe(ctx context.Context, lpbsToken, userIdentity string, req sttchain.Request) (*sttchain.Result, error) {
	return nil, Unimplemented("stt")
}
