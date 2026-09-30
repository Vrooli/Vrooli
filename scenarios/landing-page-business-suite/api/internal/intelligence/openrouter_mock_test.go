package intelligence

import (
	"context"
)

// MockOpenRouterClient is a test-only double for the gateway's provider seam.
type MockOpenRouterClient struct {
	ChatFn       func(context.Context, OpenRouterChatRequest) (*OpenRouterChatResponse, error)
	ChatStreamFn func(context.Context, OpenRouterChatRequest, func(string)) (*OpenRouterUsage, error)
	VerifyFn     func(context.Context) error
}

func (m *MockOpenRouterClient) Chat(ctx context.Context, req OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
	if m.ChatFn != nil {
		return m.ChatFn(ctx, req)
	}
	return &OpenRouterChatResponse{ID: "mock-id", Model: req.Model, Content: "Mock response", Usage: OpenRouterUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}}, nil
}

func (m *MockOpenRouterClient) ChatStream(ctx context.Context, req OpenRouterChatRequest, onChunk func(string)) (*OpenRouterUsage, error) {
	if m.ChatStreamFn != nil {
		return m.ChatStreamFn(ctx, req, onChunk)
	}
	for _, chunk := range []string{"Hello", " world", "!"} {
		if onChunk != nil {
			onChunk(chunk)
		}
	}
	return &OpenRouterUsage{PromptTokens: 10, CompletionTokens: 3, TotalTokens: 13}, nil
}

func (m *MockOpenRouterClient) VerifyAPIKey(ctx context.Context) error {
	if m.VerifyFn != nil {
		return m.VerifyFn(ctx)
	}
	return nil
}

var _ OpenRouterClient = (*MockOpenRouterClient)(nil)
