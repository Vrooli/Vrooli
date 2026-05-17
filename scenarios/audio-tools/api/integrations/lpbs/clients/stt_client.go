package clients

import (
	"context"
	"fmt"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/httpc"
)

// STTClient routes Transcribe through the LPBS audio gateway. Implements
// sttchain.VrooliClient. Today returns Unimplemented until the gateway
// endpoint ships; this seam exists so main.go can wire it once enabled.
type STTClient struct {
	BaseURL string
	Doer    httpc.Doer
}

func NewSTTClient(baseURL string) *STTClient {
	return &STTClient{
		BaseURL: baseURL,
		Doer:    httpc.DefaultDoer(),
	}
}

func (c *STTClient) IsAvailable(ctx context.Context) bool {
	return false // gateway endpoint not yet implemented
}

func (c *STTClient) Model() string { return "lpbs-default" }

func (c *STTClient) Transcribe(ctx context.Context, lpbsToken, userIdentity string, req sttchain.Request) (*sttchain.Result, error) {
	return nil, fmt.Errorf("lpbs-stt: gateway endpoint not implemented (execute/lpbs-audio-gateway-endpoints)")
}
