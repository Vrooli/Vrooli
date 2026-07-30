package intelligence

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"landing-page-business-suite-api/internal/intelligence"
)

type allowLimiter struct{}

func (allowLimiter) Allow(string) bool { return true }

type gatewayStub struct {
	response *intelligence.AIResponse
}

func (g gatewayStub) ExecuteChat(_ context.Context, _ string, _ intelligence.AIRequest) (*intelligence.AIResponse, error) {
	return g.response, nil
}

func (gatewayStub) ExecuteChatStream(context.Context, string, intelligence.AIRequest, http.ResponseWriter) error {
	return nil
}
func (gatewayStub) GetAvailableModels() []string      { return nil }
func (gatewayStub) HealthCheck(context.Context) error { return nil }

var (
	_ intelligence.Gateway = gatewayStub{}
	_ Limiter              = allowLimiter{}
)

func TestChatTranslatesAuthenticatedRequest(t *testing.T) {
	handler := New(Dependencies{
		Service:         gatewayStub{response: &intelligence.AIResponse{Content: "hello"}},
		UserRateLimiter: allowLimiter{},
		IPRateLimiter:   allowLimiter{},
		IPKeyFunc:       func(*http.Request) string { return "127.0.0.1" },
		UserIdentity:    func(context.Context) string { return "customer@example.com" },
		WriteJSONError:  writeTestError,
		Log:             func(string, map[string]interface{}) {},
		LogError:        func(string, map[string]interface{}) {},
	})

	body := []byte(`{"model":"openai/gpt-4o","messages":[{"role":"user","content":"hello"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.Chat()(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var got intelligence.AIResponse
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Content != "hello" {
		t.Fatalf("content=%q, want hello", got.Content)
	}
}

func TestChatRejectsUnauthenticatedRequest(t *testing.T) {
	handler := New(Dependencies{
		Service:         gatewayStub{},
		UserRateLimiter: allowLimiter{},
		IPRateLimiter:   allowLimiter{},
		IPKeyFunc:       func(*http.Request) string { return "127.0.0.1" },
		UserIdentity:    func(context.Context) string { return "" },
		WriteJSONError:  writeTestError,
		Log:             func(string, map[string]interface{}) {},
		LogError:        func(string, map[string]interface{}) {},
	})
	response := httptest.NewRecorder()
	handler.Chat()(response, httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func writeTestError(w http.ResponseWriter, status int, message, errorType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message, "type": errorType})
}

type connectGatewayStub struct {
	request intelligence.AIRequest
	err     error
}

func (g *connectGatewayStub) ExecuteChat(_ context.Context, _ string, request intelligence.AIRequest) (*intelligence.AIResponse, error) {
	g.request = request
	if g.err != nil {
		return nil, g.err
	}
	return &intelligence.AIResponse{ID: "chat-1", Model: request.Model, Content: "response"}, nil
}

func (*connectGatewayStub) ExecuteChatStream(context.Context, string, intelligence.AIRequest, http.ResponseWriter) error {
	return nil
}
func (*connectGatewayStub) GetAvailableModels() []string      { return []string{"provider/model"} }
func (*connectGatewayStub) HealthCheck(context.Context) error { return nil }

func TestConnectChatTranslatesTypedRequest(t *testing.T) {
	gateway := &connectGatewayStub{}
	handler := NewConnectHandler(Dependencies{
		Service:         gateway,
		UserRateLimiter: allowLimiter{},
		IPRateLimiter:   allowLimiter{},
		UserIdentity:    func(context.Context) string { return "customer@example.com" },
		Log:             func(string, map[string]interface{}) {},
		LogError:        func(string, map[string]interface{}) {},
	})

	request := connect.NewRequest(&lpbsv1.ChatRequest{
		Model:     "provider/model",
		Messages:  []*lpbsv1.AIMessage{{Role: "user", Content: "hello"}},
		MaxTokens: 128,
		Metadata:  &lpbsv1.AIMetadata{AppBundleKey: "bundle", Operation: "summarize"},
	})
	request.Header().Set("X-Forwarded-For", "203.0.113.10, 10.0.0.1")
	response, err := handler.Chat(context.Background(), request)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if response.Msg.GetContent() != "response" {
		t.Fatalf("content=%q", response.Msg.GetContent())
	}
	if gateway.request.Metadata.AppBundleKey != "bundle" || gateway.request.Metadata.Operation != "summarize" {
		t.Fatalf("metadata=%+v", gateway.request.Metadata)
	}
}

func TestConnectChatMapsCreditExhaustion(t *testing.T) {
	handler := NewConnectHandler(Dependencies{
		Service:         &connectGatewayStub{err: intelligence.ErrInsufficientCredits},
		UserRateLimiter: allowLimiter{},
		IPRateLimiter:   allowLimiter{},
		UserIdentity:    func(context.Context) string { return "customer@example.com" },
		Log:             func(string, map[string]interface{}) {},
		LogError:        func(string, map[string]interface{}) {},
	})
	_, err := handler.Chat(context.Background(), connect.NewRequest(&lpbsv1.ChatRequest{
		Model: "provider/model", Messages: []*lpbsv1.AIMessage{{Role: "user", Content: "hello"}},
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("code=%s err=%v", connect.CodeOf(err), err)
	}
}

func TestClientIPFromHeaders(t *testing.T) {
	headers := http.Header{"X-Forwarded-For": []string{"203.0.113.10, 10.0.0.1"}}
	if got := clientIPFromHeaders(headers); got != "203.0.113.10" {
		t.Fatalf("client IP=%q", got)
	}
	if got := clientIPFromHeaders(http.Header{}); got != "unknown" {
		t.Fatalf("fallback=%q", got)
	}
}
