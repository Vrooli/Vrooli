package intelligence

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/lpbsidentity"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"landing-page-business-suite-api/internal/intelligence"
)

type allowLimiter struct{}

func (allowLimiter) Allow(string) bool { return true }

type meteredInferenceStub struct {
	response *intelligence.AIResponse
}

func (g meteredInferenceStub) ExecuteChat(_ context.Context, _ string, _ intelligence.AIRequest) (*intelligence.AIResponse, error) {
	return g.response, nil
}

func (meteredInferenceStub) ExecuteChatStream(context.Context, string, intelligence.AIRequest, http.ResponseWriter) error {
	return nil
}
func (meteredInferenceStub) GetAvailableModels() []string      { return nil }
func (meteredInferenceStub) HealthCheck(context.Context) error { return nil }

type callerInferenceStub struct {
	called bool
	err    error
}

func (s *callerInferenceStub) ExecuteInferenceForCaller(_ context.Context, user, caller string, _ intelligence.InferenceRequest) (*intelligence.AIResponse, error) {
	s.called = user == "actor@example.test" && caller == "web-console"
	if s.err != nil {
		return nil, s.err
	}
	return &intelligence.AIResponse{Content: "safe result"}, nil
}
func (*callerInferenceStub) ExecuteInferenceStreamForCaller(context.Context, string, string, intelligence.InferenceRequest, http.ResponseWriter) error {
	return nil
}
func (*callerInferenceStub) ExecuteChat(context.Context, string, intelligence.AIRequest) (*intelligence.AIResponse, error) {
	return nil, nil
}
func (*callerInferenceStub) ExecuteChatStream(context.Context, string, intelligence.AIRequest, http.ResponseWriter) error {
	return nil
}
func (*callerInferenceStub) GetAvailableModels() []string      { return nil }
func (*callerInferenceStub) HealthCheck(context.Context) error { return nil }

var (
	_ intelligence.MeteredInferenceProvider = meteredInferenceStub{}
	_ Limiter                               = allowLimiter{}
)

func TestChatTranslatesAuthenticatedRequest(t *testing.T) {
	handler := New(Dependencies{
		Service:         meteredInferenceStub{response: &intelligence.AIResponse{Content: "hello"}},
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
		Service:         meteredInferenceStub{},
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

func TestValidateInferenceRequestAcceptsToolConversationAndBoundsTools(t *testing.T) {
	request := intelligence.InferenceRequest{
		Role: "code.default",
		Messages: []intelligence.AIMessage{
			{Role: "assistant", ToolCalls: []intelligence.OpenRouterToolCall{{ID: "call_1", Type: "function", Function: intelligence.OpenRouterToolFunction{Name: "read_file", Arguments: `{"path":"a.go"}`}}}},
			{Role: "tool", ToolCallID: "call_1", Name: "read_file", Content: "package main"},
		},
		Tools:      []json.RawMessage{json.RawMessage(`{"type":"function","function":{"name":"read_file"}}`)},
		ToolChoice: json.RawMessage(`"auto"`),
	}
	if err := ValidateInferenceRequest(&request); err != nil {
		t.Fatalf("valid tool conversation rejected: %v", err)
	}
	request.Tools = make([]json.RawMessage, 65)
	for i := range request.Tools {
		request.Tools[i] = json.RawMessage(`{}`)
	}
	if err := ValidateInferenceRequest(&request); err == nil {
		t.Fatal("request with more than 64 tools was accepted")
	}
}

func TestInferenceRejectsBusinessAccountWithoutMembership(t *testing.T) {
	resolverCalled := false
	handler := New(Dependencies{
		Service:         meteredInferenceStub{},
		UserRateLimiter: allowLimiter{},
		IPRateLimiter:   allowLimiter{},
		IPKeyFunc:       func(*http.Request) string { return "127.0.0.1" },
		UserID:          func(context.Context) string { return "user-1" },
		UserIdentity:    func(context.Context) string { return "customer@example.com" },
		ResolveBusinessAccount: func(_ context.Context, userID, userEmail, requestedID string) (string, error) {
			resolverCalled = true
			if userID != "user-1" || userEmail != "customer@example.com" || requestedID != "acct-other" {
				t.Fatalf("unexpected account resolution input: %q %q %q", userID, userEmail, requestedID)
			}
			return "", errors.New("not a member")
		},
		WriteJSONError: writeTestError,
		Log:            func(string, map[string]interface{}) {},
		LogError:       func(string, map[string]interface{}) {},
	})
	body := []byte(`{"role":"coding","business_account_id":"acct-other","messages":[{"role":"user","content":"hello"}]}`)
	response := httptest.NewRecorder()
	handler.Inference()(response, httptest.NewRequest(http.MethodPost, "/api/v1/ai/inference", bytes.NewReader(body)))
	if response.Code != http.StatusForbidden || !resolverCalled {
		t.Fatalf("status=%d resolverCalled=%t body=%s", response.Code, resolverCalled, response.Body.String())
	}
}

func TestStandardInferenceUsesVerifiedScenarioRoleAndDefaultHumanAccount(t *testing.T) {
	service := &callerInferenceStub{}
	verifiedCaller := false
	resolvedAccount := false
	handler := New(Dependencies{
		Service:         service,
		UserRateLimiter: allowLimiter{}, IPRateLimiter: allowLimiter{}, IPKeyFunc: func(*http.Request) string { return "ip" },
		UserID:       func(context.Context) string { return "actor-1" },
		UserIdentity: func(context.Context) string { return "actor@example.test" },
		ResolveBusinessAccount: func(_ context.Context, userID, identity, requested string) (string, error) {
			resolvedAccount = userID == "actor-1" && identity == "actor@example.test" && requested == ""
			return "acct-default", nil
		},
		ResolveCreditCallerScenario: func(_ context.Context, authorization string) (string, error) {
			verifiedCaller = authorization == "Bearer synthetic-scenario-principal"
			if !verifiedCaller {
				return "", errors.New("synthetic principal refused")
			}
			return "web-console", nil
		},
		WriteJSONError: writeTestError, Log: func(string, map[string]interface{}) {}, LogError: func(string, map[string]interface{}) {},
	})
	body := []byte(`{"role":"author.generator","messages":[{"role":"user","content":"synthetic prompt"}],"metadata":{"gateway_root_operation_id":"0123456789abcdef0123456789abcdef","app_bundle_key":"caller-controlled"}}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/inference", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer synthetic-human-session")
	request.Header.Set(lpbsidentity.CreditCallerAuthorizationHeader, "Bearer synthetic-scenario-principal")
	response := httptest.NewRecorder()
	handler.Inference()(response, request)
	if response.Code != http.StatusOK || !verifiedCaller || !resolvedAccount || !service.called {
		t.Fatalf("status=%d verifiedCaller=%t resolvedAccount=%t body=%s", response.Code, verifiedCaller, resolvedAccount, response.Body.String())
	}
}

func TestStandardInferenceRejectsUnverifiedCallerBeforeReservation(t *testing.T) {
	service := &callerInferenceStub{}
	handler := New(Dependencies{
		Service:         service,
		UserRateLimiter: allowLimiter{}, IPRateLimiter: allowLimiter{}, IPKeyFunc: func(*http.Request) string { return "ip" },
		UserID: func(context.Context) string { return "actor-1" }, UserIdentity: func(context.Context) string { return "actor@example.test" },
		ResolveBusinessAccount:      func(context.Context, string, string, string) (string, error) { return "acct-default", nil },
		ResolveCreditCallerScenario: func(context.Context, string) (string, error) { return "", errors.New("signature refused") },
		WriteJSONError:              writeTestError, Log: func(string, map[string]interface{}) {}, LogError: func(string, map[string]interface{}) {},
	})
	body := []byte(`{"role":"author.generator","messages":[{"role":"user","content":"synthetic prompt"}],"metadata":{"gateway_root_operation_id":"0123456789abcdef0123456789abcdef"}}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/inference", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer synthetic-human-session")
	request.Header.Set(lpbsidentity.CreditCallerAuthorizationHeader, "Bearer forged")
	response := httptest.NewRecorder()
	handler.Inference()(response, request)
	if response.Code != http.StatusForbidden || service.called {
		t.Fatalf("status=%d called=%t body=%s", response.Code, service.called, response.Body.String())
	}
}

func TestStandardInferenceSurfacesUncertainOutcomeAndRootOperation(t *testing.T) {
	rootOperationID := "0123456789abcdef0123456789abcdef"
	service := &callerInferenceStub{err: errors.Join(intelligence.ErrInferenceOutcomeUncertain, intelligence.ErrInferenceReservationRetentionFailed)}
	handler := New(Dependencies{
		Service:         service,
		UserRateLimiter: allowLimiter{}, IPRateLimiter: allowLimiter{}, IPKeyFunc: func(*http.Request) string { return "ip" },
		UserID: func(context.Context) string { return "actor-1" }, UserIdentity: func(context.Context) string { return "actor@example.test" },
		ResolveBusinessAccount:      func(context.Context, string, string, string) (string, error) { return "acct-default", nil },
		ResolveCreditCallerScenario: func(context.Context, string) (string, error) { return "web-console", nil },
		WriteJSONError:              writeTestError, Log: func(string, map[string]interface{}) {}, LogError: func(string, map[string]interface{}) {},
	})
	body := []byte(`{"role":"author.generator","messages":[{"role":"user","content":"synthetic"}],"metadata":{"gateway_root_operation_id":"` + rootOperationID + `"}}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/inference", bytes.NewReader(body))
	request.Header.Set(lpbsidentity.CreditCallerAuthorizationHeader, "Bearer signed-synthetic-principal")
	response := httptest.NewRecorder()
	handler.Inference()(response, request)
	var envelope map[string]string
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Code != http.StatusBadGateway || response.Header().Get("X-Vrooli-Root-Operation-ID") != rootOperationID || envelope["type"] != "outcome_uncertain" || !service.called {
		t.Fatalf("status=%d root=%q envelope=%#v serviceCalled=%t", response.Code, response.Header().Get("X-Vrooli-Root-Operation-ID"), envelope, service.called)
	}
}

func writeTestError(w http.ResponseWriter, status int, message, errorType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message, "type": errorType})
}

type connectMeteredInferenceStub struct {
	request intelligence.AIRequest
	err     error
}

func (g *connectMeteredInferenceStub) ExecuteChat(_ context.Context, _ string, request intelligence.AIRequest) (*intelligence.AIResponse, error) {
	g.request = request
	if g.err != nil {
		return nil, g.err
	}
	return &intelligence.AIResponse{ID: "chat-1", Model: request.Model, Content: "response"}, nil
}

func (*connectMeteredInferenceStub) ExecuteChatStream(context.Context, string, intelligence.AIRequest, http.ResponseWriter) error {
	return nil
}
func (*connectMeteredInferenceStub) GetAvailableModels() []string      { return []string{"provider/model"} }
func (*connectMeteredInferenceStub) HealthCheck(context.Context) error { return nil }

func TestConnectChatTranslatesTypedRequest(t *testing.T) {
	gateway := &connectMeteredInferenceStub{}
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
		Service:         &connectMeteredInferenceStub{err: intelligence.ErrInsufficientCredits},
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

func TestBillingAuthorityErrorsAreActionableAndDistinct(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{intelligence.ErrBillingAccountRequired, 403, "billing_account_required"},
		{intelligence.ErrBillingScopeUnavailable, 503, "billing_scope_unavailable"},
		{intelligence.ErrInsufficientCredits, 402, "insufficient_credits"},
	} {
		w := httptest.NewRecorder()
		WriteError(Dependencies{WriteJSONError: writeTestError, LogError: func(string, map[string]interface{}) {}}, w, tc.err)
		if w.Code != tc.status || !bytes.Contains(w.Body.Bytes(), []byte(tc.code)) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body)
		}
		if tc.status != 402 && !bytes.Contains(w.Body.Bytes(), []byte(tc.err.Error())) {
			t.Fatalf("missing actionable message: %s", w.Body)
		}
		ce := connectAIError(tc.err)
		if tc.status != 402 && !bytes.Contains([]byte(ce.Error()), []byte(tc.code)) {
			t.Fatalf("connect error=%v", ce)
		}
	}
}

func TestHTTPAdmissionRejectsMissingVerifiedBillingScopeBeforeSideEffects(t *testing.T) {
	for _, account := range []string{"", "acct-selected"} {
		for _, route := range []string{"chat", "stream", "inference", "inference-stream"} {
			t.Run(route+"/"+account, func(t *testing.T) {
				h := New(Dependencies{
					Service:         intelligence.NewMeteredInferenceService(intelligence.MeteredInferenceServiceOptions{}),
					UserRateLimiter: allowLimiter{}, IPRateLimiter: allowLimiter{}, IPKeyFunc: func(*http.Request) string { return "ip" },
					UserID: func(context.Context) string { return "user" }, UserIdentity: func(context.Context) string { return "actor@example.test" },
					ResolveBusinessAccount: func(_ context.Context, user, actor, id string) (string, error) {
						if user != "user" || actor != "actor@example.test" {
							t.Fatal("lost principal")
						}
						return id, nil
					},
					WriteJSONError: writeTestError, Log: func(string, map[string]interface{}) {}, LogError: func(string, map[string]interface{}) {},
				})
				payload := map[string]any{"business_account_id": account, "messages": []map[string]string{{"role": "user", "content": "hello"}}, "metadata": map[string]string{"app_bundle_key": "caller-controlled"}}
				if route == "inference" || route == "inference-stream" {
					payload["role"] = "code.default"
				} else {
					payload["model"] = "openai/gpt-4o"
				}
				body, _ := json.Marshal(payload)
				r := httptest.NewRequest("POST", "/api/v1/ai/"+route, bytes.NewReader(body))
				w := httptest.NewRecorder()
				switch route {
				case "chat":
					h.Chat()(w, r)
				case "stream":
					h.Stream()(w, r)
				case "inference":
					h.Inference()(w, r)
				case "inference-stream":
					h.InferenceStream()(w, r)
				}
				status, code := 503, "billing_scope_unavailable"
				if account == "" {
					status, code = 403, "billing_account_required"
				}
				if w.Code != status || !bytes.Contains(w.Body.Bytes(), []byte(code)) {
					t.Fatalf("status=%d body=%s", w.Code, w.Body)
				}
				if w.Header().Get("Content-Type") == "text/event-stream" {
					t.Fatal("stream started before admission")
				}
			})
		}
	}
}
