package openai

import (
	"context"
	"net/http"
)

// ChatRaw sends a chat completion and returns its response body without
// decoding choices. Consumers that own provider-specific response shapes can
// share the common OpenAI-compatible transport while retaining their decoder.
func (c *Client) ChatRaw(ctx context.Context, request ChatRequest) ([]byte, error) {
	body, err := marshalRequest(request)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, http.MethodPost, "/chat/completions", body)
}
