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

func TestOpenRouterClient_Chat_Success(t *testing.T) { // [REQ:FIN-COST-001]
	_, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("missing or incorrect authorization header")
		}
		var request struct {
			Usage *OpenRouterUsageRequest `json:"usage"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Usage == nil || !request.Usage.Include {
			t.Fatal("expected provider usage request to be enabled")
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
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 6,
				"total_tokens":      16,
				"cost":              0.000123,
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
	if resp.Usage.CostMicros != 123 {
		t.Errorf("expected provider cost 123 micros, got %d", resp.Usage.CostMicros)
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
	if got, want := err.Error(), "AI provider error: status 500"; got != want {
		t.Errorf("error = %q, want %q", got, want)
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

	if !errors.Is(err, ErrProviderRequestRejected) {
		t.Errorf("401 error=%v; want definitive provider rejection classification", err)
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
		if r.URL.Path != "/api/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-api-key" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("HTTP-Referer"); got != "https://vrooli.com" {
			t.Errorf("HTTP-Referer = %q", got)
		}
		if got := r.Header.Get("X-Title"); got != "Vrooli AI Metered Inference" {
			t.Errorf("X-Title = %q", got)
		}
		var request struct {
			Usage *OpenRouterUsageRequest `json:"usage"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Usage == nil || !request.Usage.Include {
			t.Fatal("expected provider usage request to be enabled for streaming")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		chunks := []string{
			`data: {"id":"1","model":"gpt-4","choices":[{"delta":{"content":"Hello"}}]}`,
			`data: {"id":"1","model":"gpt-4","choices":[{"delta":{"content":" World"}}]}`,
			`data: {"id":"1","model":"gpt-4","choices":[{"delta":{"content":"!"}}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7,"cost":0,"cost_details":{"upstream_inference_cost":0.000045}}}`,
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
	if usage.CostMicros != 45 {
		t.Errorf("expected upstream provider cost 45 micros, got %d", usage.CostMicros)
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
	} else if got, want := err.Error(), "AI provider error: status 502: Upstream error"; got != want {
		t.Errorf("error = %q, want %q", got, want)
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

type failingStreamRoundTripper struct{ body string }

func (rt failingStreamRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       &errorAfterBytesReader{data: []byte(rt.body)},
	}, nil
}

type errorAfterBytesReader struct {
	data []byte
	done bool
}

func (r *errorAfterBytesReader) Read(p []byte) (int, error) {
	if !r.done {
		r.done = true
		return copy(p, r.data), nil
	}
	return 0, errors.New("fixture stream read failure")
}

func (r *errorAfterBytesReader) Close() error { return nil }

func TestOpenRouterClient_ChatStream_ReadFailureReturnsReportedUsageWithoutEstimates(t *testing.T) {
	client := NewOpenRouterClient(OpenRouterClientOptions{
		APIKey: "test-api-key", BaseURL: "http://openrouter.test",
		HTTPClient: &http.Client{Transport: failingStreamRoundTripper{body: "data: {\"choices\":[{\"delta\":{\"content\":\"partial response\"}}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":3,\"total_tokens\":15}}\n\n"}},
	})

	var collected strings.Builder
	usage, err := client.ChatStream(context.Background(), OpenRouterChatRequest{Model: "gpt-4"}, func(content string) {
		collected.WriteString(content)
	})
	if err == nil || !strings.Contains(err.Error(), "fixture stream read failure") {
		t.Fatalf("ChatStream() error = %v, want fixture read failure", err)
	}
	if collected.String() != "partial response" {
		t.Fatalf("partial content = %q, want partial response", collected.String())
	}
	if usage == nil || !usage.UsageReported {
		t.Fatalf("usage = %#v, want explicitly reported usage", usage)
	}
	if usage.PromptTokens != 12 || usage.CompletionTokens != 3 || usage.TotalTokens != 15 {
		t.Fatalf("reported usage = %#v, want provider counts 12/3/15", usage)
	}
}

func TestOpenRouterClient_ChatStream_ReadFailureWithoutUsageDoesNotEstimate(t *testing.T) {
	client := NewOpenRouterClient(OpenRouterClientOptions{
		APIKey: "test-api-key", BaseURL: "http://openrouter.test",
		HTTPClient: &http.Client{Transport: failingStreamRoundTripper{body: "data: {\"choices\":[{\"delta\":{\"content\":\"partial response long enough to look billable\"}}]}\n\n"}},
	})

	usage, err := client.ChatStream(context.Background(), OpenRouterChatRequest{Model: "gpt-4"}, nil)
	if err == nil || !strings.Contains(err.Error(), "fixture stream read failure") {
		t.Fatalf("ChatStream() error = %v, want fixture read failure", err)
	}
	if usage == nil || usage.UsageReported {
		t.Fatalf("usage = %#v, want no reported usage", usage)
	}
	if usage.CompletionTokens != 0 || usage.TotalTokens != 0 {
		t.Fatalf("unmeasured usage = %#v, want no token estimate", usage)
	}
}

func TestOpenRouterClient_ChatStreamEventsRoundTripsToolCallsAndUsage(t *testing.T) {
	toolSpec := json.RawMessage(`{"type":"function","function":{"name":"read_file","parameters":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}}}`)
	choice := json.RawMessage(`"auto"`)
	var client OpenRouterClient
	_, client = newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		var request OpenRouterChatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode provider request: %v", err)
		}
		if len(request.Tools) != 1 || string(request.Tools[0]) != string(toolSpec) {
			t.Fatalf("tools = %s, want supplied read_file declaration", request.Tools)
		}
		if string(request.ToolChoice) != string(choice) {
			t.Fatalf("tool_choice = %s, want auto", request.ToolChoice)
		}
		if len(request.Messages) != 2 || request.Messages[1].ToolCallID != "call_1" {
			t.Fatalf("tool result history = %#v, want tool_call_id call_1", request.Messages)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\":\\\"\"}}]}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"src/main.go\\\"}\"}}]}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":40,\"completion_tokens\":8,\"total_tokens\":48}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	})
	eventClient, ok := client.(OpenRouterStreamEventClient)
	if !ok {
		t.Fatal("provider client does not expose tool-call streaming")
	}
	var events []OpenRouterStreamEvent
	usage, err := eventClient.ChatStreamEvents(context.Background(), OpenRouterChatRequest{
		Model: "openai/gpt-4o-mini",
		Messages: []OpenRouterMessage{
			{Role: "assistant", Content: "", ToolCalls: []OpenRouterToolCall{{ID: "call_1", Type: "function", Function: OpenRouterToolFunction{Name: "read_file", Arguments: `{"path":"src/main.go"}`}}}},
			{Role: "tool", Content: `package main`, ToolCallID: "call_1", Name: "read_file"},
		},
		Tools: []json.RawMessage{toolSpec}, ToolChoice: choice,
	}, func(event OpenRouterStreamEvent) { events = append(events, event) })
	if err != nil {
		t.Fatalf("ChatStreamEvents() error: %v", err)
	}
	if len(events) != 2 || len(events[0].ToolCalls) != 1 || events[0].ToolCalls[0].ID != "call_1" || events[0].ToolCalls[0].Function.Name != "read_file" {
		t.Fatalf("tool call deltas = %#v", events)
	}
	if events[0].ToolCalls[0].Function.Arguments != `{"path":"` || events[1].ToolCalls[0].Function.Arguments != `src/main.go"}` {
		t.Fatalf("partial tool arguments = %#v, want concatenable JSON fragments", events)
	}
	if usage == nil || !usage.UsageReported || usage.PromptTokens != 40 || usage.CompletionTokens != 8 {
		t.Fatalf("usage = %#v, want provider-reported 40/8 tokens", usage)
	}
}

func TestOpenRouterClient_ChatAndStreamPropagateNetworkAndCancellationWithoutKeyLeak(t *testing.T) {
	const key = "test-api-key-synthetic-secret"
	type call func(context.Context, OpenRouterClient) error
	calls := map[string]call{
		"chat": func(ctx context.Context, client OpenRouterClient) error {
			_, err := client.Chat(ctx, OpenRouterChatRequest{Model: "fixture/model"})
			return err
		},
		"stream": func(ctx context.Context, client OpenRouterClient) error {
			streamClient, ok := client.(OpenRouterStreamEventClient)
			if !ok {
				return errors.New("stream event interface unavailable")
			}
			_, err := streamClient.ChatStreamEvents(ctx, OpenRouterChatRequest{Model: "fixture/model"}, nil)
			return err
		},
	}
	for name, doCall := range calls {
		t.Run(name+"/network", func(t *testing.T) {
			client := NewOpenRouterClient(OpenRouterClientOptions{
				APIKey: key, BaseURL: "http://openrouter.test",
				HTTPClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
					return nil, errors.New("synthetic network down")
				})},
			})
			err := doCall(context.Background(), client)
			if err == nil || !strings.Contains(err.Error(), "request failed:") {
				t.Fatalf("call error = %v, want request failure", err)
			}
			if strings.Contains(err.Error(), key) {
				t.Fatalf("call leaked API key in error: %v", err)
			}
		})
		t.Run(name+"/cancellation", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			client := NewOpenRouterClient(OpenRouterClientOptions{APIKey: key, BaseURL: "http://openrouter.test"})
			err := doCall(ctx, client)
			if err == nil || !errors.Is(err, context.Canceled) {
				t.Fatalf("call error = %v, want canceled context", err)
			}
			if strings.Contains(err.Error(), key) {
				t.Fatalf("call leaked API key in error: %v", err)
			}
		})
	}
}

func TestOpenRouterClient_CurrentEndpointsUsesSharedAuthenticatedTransport(t *testing.T) {
	server, client := newMockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/models/openai/gpt-4o/endpoints" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-api-key" {
			t.Errorf("Authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"data":{"endpoints":[{"provider_name":"fixture","context_length":128000,"pricing":{"prompt":"0.000001","completion":"0.000002"}}]}}`))
	})
	defer server.Close()

	metadataClient, ok := client.(OpenRouterEndpointMetadataClient)
	if !ok {
		t.Fatal("provider client does not expose endpoint metadata")
	}
	endpoints, err := metadataClient.CurrentEndpoints(context.Background(), "openai/gpt-4o")
	if err != nil {
		t.Fatalf("CurrentEndpoints() error: %v", err)
	}
	if len(endpoints) != 1 || endpoints[0].Provider != "fixture" || endpoints[0].ContextTokens != 128000 {
		t.Fatalf("endpoints = %#v", endpoints)
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
	} else if got, want := err.Error(), "key verification failed: status 401"; got != want {
		t.Errorf("VerifyAPIKey() error = %q, want %q", got, want)
	}
	if err != nil && strings.Contains(err.Error(), "test-api-key") {
		t.Errorf("VerifyAPIKey() leaked key in error: %v", err)
	}
}

func TestOpenRouterClient_VerifyAPIKey_NetworkError_ReturnsError(t *testing.T) {
	client := NewOpenRouterClient(OpenRouterClientOptions{
		APIKey: "test-api-key", BaseURL: "http://openrouter.test",
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("synthetic network down")
		})},
	})
	err := client.VerifyAPIKey(context.Background())
	if err == nil || !strings.Contains(err.Error(), "connectivity check failed: Get \"http://openrouter.test/api/v1/auth/key\"") {
		t.Fatalf("VerifyAPIKey() error = %v, want connectivity failure", err)
	}
	if strings.Contains(err.Error(), "test-api-key") {
		t.Errorf("VerifyAPIKey() leaked key in error: %v", err)
	}
}

func TestOpenRouterClient_VerifyAPIKey_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := NewOpenRouterClient(OpenRouterClientOptions{APIKey: "test-api-key", BaseURL: "http://openrouter.test"})
	err := client.VerifyAPIKey(ctx)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("VerifyAPIKey() error = %v, want canceled-context error", err)
	}
	if strings.Contains(err.Error(), "test-api-key") {
		t.Errorf("VerifyAPIKey() leaked key in error: %v", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

// ============================================================================
// NewOpenRouterClient Tests
// ============================================================================

func TestNewOpenRouterClient_DefaultBaseURLAndTimeout(t *testing.T) {
	client := NewOpenRouterClient(OpenRouterClientOptions{
		APIKey: "test-key",
	})

	// Type assert to access internal fields
	httpClient := client.(*httpOpenRouterClient)

	if httpClient.shared.BaseURL() != "https://openrouter.ai" {
		t.Errorf("expected default base URL 'https://openrouter.ai', got %s", httpClient.shared.BaseURL())
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
	client := NewOpenRouterClient(OpenRouterClientOptions{APIKey: "test-key", BaseURL: "https://proxy.example.test/", Referer: "https://app.example.test", Title: "Example Metered Inference"}).(*httpOpenRouterClient)
	if client.shared.BaseURL() != "https://proxy.example.test" || client.referer != "https://app.example.test" || client.title != "Example Metered Inference" {
		t.Fatalf("client = %#v", client)
	}
}

// ============================================================================
// Shared transport header tests
// ============================================================================

func TestOpenRouterClient_ChatUsesSharedTransportHeaders(t *testing.T) {
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
	if capturedReq.Header.Get("X-Title") != "Vrooli AI Metered Inference" {
		t.Error("expected X-Title header")
	}
}
