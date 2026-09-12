package clients

import (
	"context"

	"audio-tools/internal/ai/summarizechain"
)

type SummarizeClient struct {
	UnavailableClient
}

func NewSummarizeClient(baseURL string) *SummarizeClient {
	return &SummarizeClient{UnavailableClient: newUnavailableClient(baseURL)}
}

func (c *SummarizeClient) Summarize(ctx context.Context, lpbsToken, userIdentity string, req summarizechain.Request) (*summarizechain.Result, error) {
	return nil, Unimplemented("summarize")
}
