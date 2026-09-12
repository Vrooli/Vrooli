package clients

import (
	"context"

	"audio-tools/internal/ai/sttchain"
)

// STTClient routes Transcribe through the LPBS audio gateway. Implements
// sttchain.VrooliClient. Today returns Unimplemented until the gateway
// endpoint ships; this seam exists so main.go can wire it once enabled.
type STTClient struct {
	UnavailableClient
}

func NewSTTClient(baseURL string) *STTClient {
	return &STTClient{UnavailableClient: newUnavailableClient(baseURL)}
}

func (c *STTClient) Transcribe(ctx context.Context, lpbsToken, userIdentity string, req sttchain.Request) (*sttchain.Result, error) {
	return nil, Unimplemented("stt")
}
