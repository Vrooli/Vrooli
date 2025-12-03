package browser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"test-genie/internal/uismoke/orchestrator"
)

func TestNewClient(t *testing.T) {
	c := NewClient("http://localhost:4110")

	if c.baseURL != "http://localhost:4110" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "http://localhost:4110")
	}
	if c.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	c := NewClient("http://localhost:4110/")

	if c.baseURL != "http://localhost:4110" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "http://localhost:4110")
	}
}

func TestClient_Health_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pressure" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"pressure":{"running":0}}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	err := c.Health(context.Background())
	if err != nil {
		t.Errorf("Health() error = %v", err)
	}
}

func TestClient_Health_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	err := c.Health(context.Background())
	if err == nil {
		t.Error("Health() expected error for 503 status")
	}
}

func TestClient_ExecuteFunction_Success(t *testing.T) {
	response := map[string]interface{}{
		"data": map[string]interface{}{
			"success": true,
			"console": []interface{}{},
			"network": []interface{}{},
			"handshake": map[string]interface{}{
				"signaled":   true,
				"timedOut":   false,
				"durationMs": 500,
			},
			"html":  "<html></html>",
			"title": "Test Page",
			"url":   "http://localhost:3000",
		},
		"type": "application/json",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/function" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/javascript" {
			t.Errorf("unexpected Content-Type: %s", ct)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	result, err := c.ExecuteFunction(context.Background(), "module.exports = async () => {}")
	if err != nil {
		t.Fatalf("ExecuteFunction() error = %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}
	if !result.Handshake.Signaled {
		t.Error("expected Handshake.Signaled to be true")
	}
}

func TestClient_ExecuteFunction_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.ExecuteFunction(context.Background(), "module.exports = async () => {}")
	if err == nil {
		t.Error("ExecuteFunction() expected error for 500 status")
	}
}

func TestParseResponse_Success(t *testing.T) {
	data := []byte(`{
		"success": true,
		"console": [{"level": "log", "message": "hello"}],
		"network": [],
		"pageErrors": [],
		"handshake": {"signaled": true, "timedOut": false, "durationMs": 500},
		"html": "<html></html>",
		"title": "Test",
		"url": "http://localhost:3000"
	}`)

	result, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}
	if len(result.Console) != 1 {
		t.Errorf("expected 1 console entry, got %d", len(result.Console))
	}
	if !result.Handshake.Signaled {
		t.Error("expected Handshake.Signaled to be true")
	}
}

func TestParseResponse_Wrapped(t *testing.T) {
	data := []byte(`{
		"data": {
			"success": true,
			"handshake": {"signaled": true}
		},
		"type": "application/json"
	}`)

	result, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true after unwrapping")
	}
}

func TestParseResponse_Empty(t *testing.T) {
	_, err := ParseResponse([]byte{})
	if err == nil {
		t.Error("ParseResponse() expected error for empty data")
	}
}

func TestParseResponse_Invalid(t *testing.T) {
	_, err := ParseResponse([]byte("not json"))
	if err == nil {
		t.Error("ParseResponse() expected error for invalid JSON")
	}
}

func TestRemoveScreenshotFromJSON(t *testing.T) {
	data := []byte(`{"success": true, "screenshot": "base64data", "html": "<html>"}`)
	result := removeScreenshotFromJSON(data)

	var obj map[string]interface{}
	if err := json.Unmarshal(result, &obj); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if _, exists := obj["screenshot"]; exists {
		t.Error("screenshot should be removed")
	}
	if _, exists := obj["success"]; !exists {
		t.Error("success should be preserved")
	}
	if _, exists := obj["html"]; !exists {
		t.Error("html should be preserved")
	}
}

func TestPayloadGenerator_Generate(t *testing.T) {
	gen := NewPayloadGenerator()
	payload := gen.Generate("http://localhost:3000", 90*time.Second, 15*time.Second)

	if len(payload) == 0 {
		t.Error("expected non-empty payload")
	}

	// Check that URL is properly JSON-encoded in the payload
	if !containsString(payload, `"http://localhost:3000"`) {
		t.Error("payload should contain JSON-encoded URL")
	}

	// Check for key structural elements
	if !containsString(payload, "module.exports") {
		t.Error("payload should contain module.exports")
	}
	if !containsString(payload, "handshakeCheck") {
		t.Error("payload should contain handshakeCheck function")
	}
	if !containsString(payload, "__vrooliBridgeChildInstalled") {
		t.Error("payload should check for __vrooliBridgeChildInstalled")
	}
}

func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle ||
		len(haystack) > len(needle) && (haystack[:len(needle)] == needle ||
			haystack[len(haystack)-len(needle):] == needle ||
			containsSubstring(haystack, needle)))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestClient_WithHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 5 * time.Second}
	c := NewClient("http://localhost:4110", WithHTTPClient(customClient))

	if c.httpClient != customClient {
		t.Error("httpClient should be set to custom client")
	}
}

func TestClient_WithTimeout(t *testing.T) {
	c := NewClient("http://localhost:4110", WithTimeout(30*time.Second))

	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want %v", c.httpClient.Timeout, 30*time.Second)
	}
}

func TestExtractArtifacts_Success(t *testing.T) {
	screenshotData := []byte{0x89, 0x50, 0x4E, 0x47}
	screenshotB64 := "iVBORw0=" // Valid base64 for PNG header

	response := &orchestrator.BrowserResponse{
		Screenshot: screenshotB64,
		Console: []orchestrator.ConsoleEntry{
			{Level: "log", Message: "test"},
		},
		Network: []orchestrator.NetworkEntry{
			{URL: "http://localhost/api", ErrorText: "failed"},
		},
		PageErrors: []orchestrator.PageError{
			{Message: "TypeError"},
		},
		HTML: "<html></html>",
		Raw:  json.RawMessage(`{"success": true}`),
	}

	artifacts, err := ExtractArtifacts(response)
	if err != nil {
		t.Fatalf("ExtractArtifacts() error = %v", err)
	}

	if len(artifacts.Screenshot) == 0 {
		t.Error("Screenshot should be decoded")
	}
	if len(artifacts.Console) == 0 {
		t.Error("Console should be marshaled")
	}
	if len(artifacts.Network) == 0 {
		t.Error("Network should be marshaled")
	}
	if len(artifacts.PageErrors) == 0 {
		t.Error("PageErrors should be marshaled")
	}
	if string(artifacts.HTML) != "<html></html>" {
		t.Errorf("HTML = %q, want %q", string(artifacts.HTML), "<html></html>")
	}
	if len(artifacts.Raw) == 0 {
		t.Error("Raw should be set")
	}

	// Suppress unused variable warning
	_ = screenshotData
}

func TestExtractArtifacts_InvalidBase64Screenshot(t *testing.T) {
	response := &orchestrator.BrowserResponse{
		Screenshot: "not valid base64!!!",
	}

	_, err := ExtractArtifacts(response)
	if err == nil {
		t.Error("ExtractArtifacts() should return error for invalid base64")
	}
}

func TestExtractArtifacts_EmptyResponse(t *testing.T) {
	response := &orchestrator.BrowserResponse{}

	artifacts, err := ExtractArtifacts(response)
	if err != nil {
		t.Fatalf("ExtractArtifacts() error = %v", err)
	}

	if artifacts.Screenshot != nil {
		t.Error("Screenshot should be nil for empty response")
	}
	if artifacts.Console != nil {
		t.Error("Console should be nil for empty response")
	}
}

func TestToHandshakeResult(t *testing.T) {
	raw := &orchestrator.HandshakeRaw{
		Signaled:   true,
		TimedOut:   false,
		DurationMs: 500,
		Error:      "",
	}

	result := ToHandshakeResult(raw)

	if !result.Signaled {
		t.Error("Signaled should be true")
	}
	if result.TimedOut {
		t.Error("TimedOut should be false")
	}
	if result.DurationMs != 500 {
		t.Errorf("DurationMs = %d, want 500", result.DurationMs)
	}
	if result.Error != "" {
		t.Errorf("Error = %q, want empty", result.Error)
	}
}

func TestToHandshakeResult_Timeout(t *testing.T) {
	raw := &orchestrator.HandshakeRaw{
		Signaled:   false,
		TimedOut:   true,
		DurationMs: 15000,
		Error:      "Timeout waiting for handshake",
	}

	result := ToHandshakeResult(raw)

	if result.Signaled {
		t.Error("Signaled should be false")
	}
	if !result.TimedOut {
		t.Error("TimedOut should be true")
	}
	if result.Error != "Timeout waiting for handshake" {
		t.Errorf("Error = %q, want %q", result.Error, "Timeout waiting for handshake")
	}
}

func TestClient_ExecuteFunction_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := c.ExecuteFunction(ctx, "module.exports = async () => {}")
	if err == nil {
		t.Error("ExecuteFunction() should return error for cancelled context")
	}
}

func TestClient_Health_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := c.Health(ctx)
	if err == nil {
		t.Error("Health() should return error for cancelled context")
	}
}

func TestPayloadGenerator_Generate_IntegerTimeouts(t *testing.T) {
	gen := NewPayloadGenerator()
	payload := gen.Generate("http://localhost:3000", int64(90000), int(15000))

	if len(payload) == 0 {
		t.Error("expected non-empty payload")
	}
}

func TestPayloadGenerator_Generate_ZeroTimeout(t *testing.T) {
	gen := NewPayloadGenerator()
	payload := gen.Generate("http://localhost:3000", nil, nil)

	if len(payload) == 0 {
		t.Error("expected non-empty payload")
	}
}

func TestParseResponse_WithPageErrors(t *testing.T) {
	data := []byte(`{
		"success": false,
		"pageErrors": [
			{"message": "TypeError: Cannot read property 'foo' of undefined", "timestamp": "2024-01-01T00:00:00Z"}
		],
		"error": "Page threw errors"
	}`)

	result, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}

	if result.Success {
		t.Error("Success should be false")
	}
	if len(result.PageErrors) != 1 {
		t.Errorf("expected 1 page error, got %d", len(result.PageErrors))
	}
	if result.Error != "Page threw errors" {
		t.Errorf("Error = %q, want %q", result.Error, "Page threw errors")
	}
}

func TestParseResponse_WithNetworkFailures(t *testing.T) {
	data := []byte(`{
		"success": false,
		"network": [
			{"url": "http://api.example.com/data", "method": "GET", "status": 500, "errorText": "Internal Server Error"}
		]
	}`)

	result, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}

	if len(result.Network) != 1 {
		t.Errorf("expected 1 network entry, got %d", len(result.Network))
	}
	if result.Network[0].URL != "http://api.example.com/data" {
		t.Errorf("Network[0].URL = %q, want %q", result.Network[0].URL, "http://api.example.com/data")
	}
}

func TestRemoveScreenshotFromJSON_InvalidJSON(t *testing.T) {
	data := []byte("not valid json")
	result := removeScreenshotFromJSON(data)

	// Should return original data if JSON is invalid
	if string(result) != string(data) {
		t.Error("should return original data for invalid JSON")
	}
}
