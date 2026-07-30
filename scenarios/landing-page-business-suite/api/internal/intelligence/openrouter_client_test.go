package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newMockOpenRouterServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, OpenRouterClient) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server, NewOpenRouterClient(OpenRouterClientOptions{APIKey: "test-api-key", BaseURL: server.URL})
}

// ============================================================================
// Chat Tests
// ============================================================================

func TestOpenRouterClient_Chat_Success(t *testing.T) {
	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("missing or incorrect authorization header")
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "chatcmpl-123",
			"model": "gpt-4",
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello from the test server!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 6,
				"total_tokens":      16,
			},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	})

	ctx := context.Background()
	resp, err := client.Chat(ctx, OpenRouterChatRequest{
		Model:    "gpt-4",
		Messages: []OpenRouterMessage{{Role: "user", Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("Chat() returned error: %v", err)
	}

	if resp.Content != "Hello from the test server!" {
		t.Errorf("expected content 'Hello from the test server!', got %s", resp.Content)
	}
	if resp.Usage.TotalTokens != 16 {
		t.Errorf("expected total tokens 16, got %d", resp.Usage.TotalTokens)
	}
}

func TestOpenRouterClient_Chat_Non200Status_ReturnsProviderError(t *testing.T) {
	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(`{"error": "server error"}`)); err != nil {
			t.Fatalf("write error response: %v", err)
		}
	})

	ctx := context.Background()
	_, err := client.Chat(ctx, OpenRouterChatRequest{
		Model:    "gpt-4",
		Messages: []OpenRouterMessage{{Role: "user", Content: "Hello"}},
	})

	if err == nil {
		t.Error("expected error for 500 status, got nil")
	}
	if !errors.Is(err, ErrProvider) {
		t.Errorf("expected provider domain error, got %v", err)
	}
}

func TestOpenRouterClient_Chat_401Unauthorized_ReturnsError(t *testing.T) {
	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(`{"error": {"message": "Invalid API key"}}`)); err != nil {
			t.Fatalf("write error response: %v", err)
		}
	})

	ctx := context.Background()
	_, err := client.Chat(ctx, OpenRouterChatRequest{
		Model:    "gpt-4",
		Messages: []OpenRouterMessage{{Role: "user", Content: "Hello"}},
	})

	if err == nil {
		t.Error("expected error for 401 status, got nil")
	}
}

func TestOpenRouterClient_Chat_429RateLimited_ReturnsError(t *testing.T) {
	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		if _, err := w.Write([]byte(`{"error": {"message": "Rate limit exceeded"}}`)); err != nil {
			t.Fatalf("write error response: %v", err)
		}
	})

	ctx := context.Background()
	_, err := client.Chat(ctx, OpenRouterChatRequest{
		Model:    "gpt-4",
		Messages: []OpenRouterMessage{{Role: "user", Content: "Hello"}},
	})

	if err == nil {
		t.Error("expected error for 429 status, got nil")
	}
}

func TestOpenRouterClient_Chat_InvalidJSONResponse_ReturnsDecodeError(t *testing.T) {
	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{invalid json`)); err != nil {
			t.Fatalf("write invalid json: %v", err)
		}
	})

	ctx := context.Background()
	_, err := client.Chat(ctx, OpenRouterChatRequest{
		Model:    "gpt-4",
		Messages: []OpenRouterMessage{{Role: "user", Content: "Hello"}},
	})

	if err == nil {
		t.Error("expected decode error, got nil")
	}
}

func TestOpenRouterClient_Chat_EmptyChoices_ReturnsEmptyContent(t *testing.T) {
	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "chatcmpl-123",
			"model":   "gpt-4",
			"choices": []map[string]interface{}{},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 0,
				"total_tokens":      10,
			},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	})

	ctx := context.Background()
	resp, err := client.Chat(ctx, OpenRouterChatRequest{
		Model:    "gpt-4",
		Messages: []OpenRouterMessage{{Role: "user", Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("Chat() returned error: %v", err)
	}

	if resp.Content != "" {
		t.Errorf("expected empty content, got %s", resp.Content)
	}
}

// ============================================================================
// ChatStream Tests
// ============================================================================

func TestOpenRouterClient_ChatStream_Success(t *testing.T) {
	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		chunks := []string{
			`data: {"id":"1","model":"gpt-4","choices":[{"delta":{"content":"Hello"}}]}`,
			`data: {"id":"1","model":"gpt-4","choices":[{"delta":{"content":" World"}}]}`,
			`data: {"id":"1","model":"gpt-4","choices":[{"delta":{"content":"!"}}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			if _, err := w.Write([]byte(chunk + "\n\n")); err != nil {
				t.Fatalf("write stream chunk: %v", err)
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	})

	ctx := context.Background()
	var collected strings.Builder
	usage, err := client.ChatStream(ctx, OpenRouterChatRequest{
		Model:    "gpt-4",
		Messages: []OpenRouterMessage{{Role: "user", Content: "Hi"}},
	}, func(content string) {
		collected.WriteString(content)
	})
	if err != nil {
		t.Fatalf("ChatStream() returned error: %v", err)
	}

	if collected.String() != "Hello World!" {
		t.Errorf("expected collected content 'Hello World!', got %s", collected.String())
	}

	if usage.TotalTokens != 7 {
		t.Errorf("expected total tokens 7, got %d", usage.TotalTokens)
	}
}

func TestOpenRouterClient_ChatStream_Non200Status_ReturnsError(t *testing.T) {
	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		if _, err := w.Write([]byte("Upstream error")); err != nil {
			t.Fatalf("write upstream error: %v", err)
		}
	})

	ctx := context.Background()
	_, err := client.ChatStream(ctx, OpenRouterChatRequest{
		Model:    "gpt-4",
		Messages: []OpenRouterMessage{{Role: "user", Content: "Hi"}},
	}, nil)

	if err == nil {
		t.Error("expected error for 502 status, got nil")
	}
}

func TestOpenRouterClient_ChatStream_SSEParseError_LogsAndContinues(t *testing.T) {
	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		// Send an invalid JSON chunk followed by valid ones
		chunks := []string{
			`data: {invalid json}`,
			`data: {"id":"1","choices":[{"delta":{"content":"Hello"}}]}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			if _, err := w.Write([]byte(chunk + "\n\n")); err != nil {
				t.Fatalf("write stream chunk: %v", err)
			}
		}
	})

	ctx := context.Background()
	var collected strings.Builder
	_, err := client.ChatStream(ctx, OpenRouterChatRequest{
		Model:    "gpt-4",
		Messages: []OpenRouterMessage{{Role: "user", Content: "Hi"}},
	}, func(content string) {
		collected.WriteString(content)
	})
	if err != nil {
		t.Fatalf("ChatStream() returned error: %v", err)
	}

	// Should still get content from the valid chunk
	if collected.String() != "Hello" {
		t.Errorf("expected 'Hello', got %s", collected.String())
	}
}

func TestOpenRouterClient_ChatStream_DoneHandling_EndsStream(t *testing.T) {
	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		chunks := []string{
			`data: {"id":"1","choices":[{"delta":{"content":"Before"}}]}`,
			`data: [DONE]`,
			`data: {"id":"1","choices":[{"delta":{"content":"After"}}]}`, // Should not be received
		}

		for _, chunk := range chunks {
			if _, err := w.Write([]byte(chunk + "\n\n")); err != nil {
				t.Fatalf("write stream chunk: %v", err)
			}
		}
	})

	ctx := context.Background()
	var collected strings.Builder
	_, err := client.ChatStream(ctx, OpenRouterChatRequest{
		Model:    "gpt-4",
		Messages: []OpenRouterMessage{{Role: "user", Content: "Hi"}},
	}, func(content string) {
		collected.WriteString(content)
	})
	if err != nil {
		t.Fatalf("ChatStream() returned error: %v", err)
	}

	if collected.String() != "Before" {
		t.Errorf("expected only 'Before' (stream should end at [DONE]), got %s", collected.String())
	}
}

func TestOpenRouterClient_ChatStream_EmptyStream_ReturnsZeroUsage(t *testing.T) {
	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if _, err := w.Write([]byte("data: [DONE]\n\n")); err != nil {
			t.Fatalf("write stream done: %v", err)
		}
	})

	ctx := context.Background()
	usage, err := client.ChatStream(ctx, OpenRouterChatRequest{
		Model:    "gpt-4",
		Messages: []OpenRouterMessage{{Role: "user", Content: "Hi"}},
	}, nil)
	if err != nil {
		t.Fatalf("ChatStream() returned error: %v", err)
	}

	// Usage should be 0 or estimated
	if usage.PromptTokens != 0 {
		t.Errorf("expected prompt tokens 0, got %d", usage.PromptTokens)
	}
}

func TestOpenRouterClient_ChatStream_TokenEstimation_WhenNotProvided(t *testing.T) {
	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		// Send content without usage stats
		chunks := []string{
			`data: {"id":"1","choices":[{"delta":{"content":"Hello"}}]}`,
			`data: {"id":"1","choices":[{"delta":{"content":" World!"}}]}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			if _, err := w.Write([]byte(chunk + "\n\n")); err != nil {
				t.Fatalf("write stream chunk: %v", err)
			}
		}
	})

	ctx := context.Background()
	var collected strings.Builder
	usage, err := client.ChatStream(ctx, OpenRouterChatRequest{
		Model:    "gpt-4",
		Messages: []OpenRouterMessage{{Role: "user", Content: "Hi"}},
	}, func(content string) {
		collected.WriteString(content)
	})
	if err != nil {
		t.Fatalf("ChatStream() returned error: %v", err)
	}

	// Usage should be estimated (~12 chars / 4 = 3 tokens)
	if usage.CompletionTokens == 0 {
		t.Error("expected estimated completion tokens > 0")
	}
}

// ============================================================================
// VerifyAPIKey Tests
// ============================================================================

func TestOpenRouterClient_VerifyAPIKey_Success(t *testing.T) {
	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/key" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("missing or incorrect authorization header")
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status": "valid"}`)); err != nil {
			t.Fatalf("write status response: %v", err)
		}
	})

	ctx := context.Background()
	err := client.VerifyAPIKey(ctx)
	if err != nil {
		t.Fatalf("VerifyAPIKey() returned error: %v", err)
	}
}

func TestOpenRouterClient_VerifyAPIKey_InvalidKey_ReturnsError(t *testing.T) {
	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(`{"error": "Invalid API key"}`)); err != nil {
			t.Fatalf("write error response: %v", err)
		}
	})

	ctx := context.Background()
	err := client.VerifyAPIKey(ctx)
	if err == nil {
		t.Error("expected error for invalid key, got nil")
	}
}

func TestOpenRouterClient_VerifyAPIKey_NetworkError_ReturnsError(t *testing.T) {
	// Create a client pointing to a non-existent server
	client := NewOpenRouterClient(OpenRouterClientOptions{
		APIKey:  "test-api-key",
		BaseURL: "http://localhost:59999", // Non-existent port
		Timeout: 100 * time.Millisecond,
	})

	ctx := context.Background()
	err := client.VerifyAPIKey(ctx)
	if err == nil {
		t.Error("expected network error, got nil")
	}
}

// ============================================================================
// NewOpenRouterClient Tests
// ============================================================================

func TestNewOpenRouterClient_DefaultBaseURLAndTimeout(t *testing.T) {
	client := NewOpenRouterClient(OpenRouterClientOptions{
		APIKey: "test-key",
	})

	// Type assert to access internal fields
	httpClient := client.(*httpOpenRouterClient)

	if httpClient.baseURL != "https://openrouter.ai" {
		t.Errorf("expected default base URL 'https://openrouter.ai', got %s", httpClient.baseURL)
	}

	if httpClient.httpClient.Timeout != 120*time.Second {
		t.Errorf("expected default timeout 120s, got %v", httpClient.httpClient.Timeout)
	}
}

func TestNewOpenRouterClient_CustomHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 30 * time.Second}

	client := NewOpenRouterClient(OpenRouterClientOptions{
		APIKey:     "test-key",
		HTTPClient: customClient,
	})

	httpClient := client.(*httpOpenRouterClient)

	if httpClient.httpClient.Timeout != 30*time.Second {
		t.Errorf("expected custom timeout 30s, got %v", httpClient.httpClient.Timeout)
	}
}

func TestNewOpenRouterClient_CustomAttribution(t *testing.T) {
	client := NewOpenRouterClient(OpenRouterClientOptions{APIKey: "test-key", BaseURL: "https://proxy.example.test/", Referer: "https://app.example.test", Title: "Example Gateway"}).(*httpOpenRouterClient)
	if client.baseURL != "https://proxy.example.test" || client.referer != "https://app.example.test" || client.title != "Example Gateway" {
		t.Fatalf("client = %#v", client)
	}
}

// ============================================================================
// setHeaders Tests
// ============================================================================

func TestOpenRouterClient_SetHeaders_AllHeadersSet(t *testing.T) {
	var capturedReq *http.Request

	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "1",
			"model":   "gpt-4",
			"choices": []map[string]interface{}{},
			"usage":   map[string]int{},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	})

	ctx := context.Background()
	if _, err := client.Chat(ctx, OpenRouterChatRequest{
		Model:    "gpt-4",
		Messages: []OpenRouterMessage{{Role: "user", Content: "Hi"}},
	}); err != nil {
		t.Fatalf("Chat() returned error: %v", err)
	}

	if capturedReq.Header.Get("Content-Type") != "application/json" {
		t.Error("expected Content-Type header")
	}
	if capturedReq.Header.Get("Authorization") != "Bearer test-api-key" {
		t.Error("expected Authorization header")
	}
	if capturedReq.Header.Get("HTTP-Referer") != "https://vrooli.com" {
		t.Error("expected HTTP-Referer header")
	}
	if capturedReq.Header.Get("X-Title") != "Vrooli AI Gateway" {
		t.Error("expected X-Title header")
	}
}
