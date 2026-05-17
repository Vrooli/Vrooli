package clients

import (
	"context"
	"fmt"

	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/httpc"
)

type SummarizeClient struct {
	BaseURL string
	Doer    httpc.Doer
}

func NewSummarizeClient(baseURL string) *SummarizeClient {
	return &SummarizeClient{
		BaseURL: baseURL,
		Doer:    httpc.DefaultDoer(),
	}
}

func (c *SummarizeClient) IsAvailable(ctx context.Context) bool { return false }
func (c *SummarizeClient) Model() string                        { return "lpbs-default" }

func (c *SummarizeClient) Summarize(ctx context.Context, lpbsToken, userIdentity string, req summarizechain.Request) (*summarizechain.Result, error) {
	return nil, fmt.Errorf("lpbs-summarize: gateway endpoint not implemented (execute/lpbs-audio-gateway-endpoints)")
}
