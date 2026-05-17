package clients

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"audio-tools/internal/ai/ttschain"
)

type TTSClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewTTSClient(baseURL string) *TTSClient {
	return &TTSClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *TTSClient) IsAvailable(ctx context.Context) bool { return false }
func (c *TTSClient) Model() string                        { return "lpbs-default" }

func (c *TTSClient) Synthesize(ctx context.Context, lpbsToken, userIdentity string, req ttschain.Request) (*ttschain.Result, error) {
	return nil, fmt.Errorf("lpbs-tts: gateway endpoint not implemented (execute/lpbs-audio-gateway-endpoints)")
}
