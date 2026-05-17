package clients

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"audio-tools/internal/ai/summarizechain"
)

type SummarizeClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewSummarizeClient(baseURL string) *SummarizeClient {
	return &SummarizeClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *SummarizeClient) IsAvailable(ctx context.Context) bool { return false }
func (c *SummarizeClient) Model() string                        { return "lpbs-default" }

func (c *SummarizeClient) Summarize(ctx context.Context, lpbsToken, userIdentity string, req summarizechain.Request) (*summarizechain.Result, error) {
	return nil, fmt.Errorf("lpbs-summarize: gateway endpoint not implemented (execute/lpbs-audio-gateway-endpoints)")
}
