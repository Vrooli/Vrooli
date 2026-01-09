package browser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"test-genie/internal/smoke/orchestrator"
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
	viewport := orchestrator.Viewport{Width: 1280, Height: 720}
	payload := gen.Generate("http://localhost:3000", 90*time.Second, 15*time.Second, viewport, nil)

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
		t.Error("payload should check for __vrooliBridgeChildInstalled (default signal)")
	}
}

func TestPayloadGenerator_Generate_CustomSignals(t *testing.T) {
	gen := NewPayloadGenerator()
	customSignals := []string{"myApp.ready", "MY_CUSTOM_READY"}
	viewport := orchestrator.Viewport{Width: 1280, Height: 720}
	payload := gen.Generate("http://localhost:3000", 90*time.Second, 15*time.Second, viewport, customSignals)

	// Should NOT contain default signals
	if containsString(payload, "__vrooliBridgeChildInstalled") {
		t.Error("payload should NOT contain default signals when custom signals are provided")
	}

	// Should contain custom signals
	if !containsString(payload, "myApp.ready") {
		t.Error("payload should contain custom signal 'myApp.ready'")
	}
	if !containsString(payload, "MY_CUSTOM_READY") {
		t.Error("payload should contain custom signal 'MY_CUSTOM_READY'")
	}
}

func TestGenerateSignalCheck_SimpleProperty(t *testing.T) {
	check := generateSignalCheck("READY_FLAG")
	expected := "window.READY_FLAG === true"
	if check != expected {
		t.Errorf("generateSignalCheck(%q) = %q, want %q", "READY_FLAG", check, expected)
	}
}

func TestGenerateSignalCheck_NestedProperty(t *testing.T) {
	check := generateSignalCheck("IframeBridge.ready")
	if !containsString(check, "window.IframeBridge") {
		t.Error("should contain guard for parent object")
	}
	if !containsString(check, "window.IframeBridge.ready === true") {
		t.Error("should check full path")
	}
}

func TestGenerateSignalCheck_MethodCall(t *testing.T) {
	check := generateSignalCheck("IframeBridge.getState().ready")
	if !containsString(check, "typeof") {
		t.Error("should check method type")
	}
	if !containsString(check, "getState") {
		t.Error("should reference the method")
	}
	if !containsString(check, ".ready === true") {
		t.Error("should check the final property")
	}
}

func TestGenerateSignalCheck_EmptyString(t *testing.T) {
	check := generateSignalCheck("")
	// Empty string should return simple window check
	if check != "window. === true" {
		t.Errorf("generateSignalCheck(\"\") = %q, want %q", check, "window. === true")
	}
}

func TestGenerateSignalCheck_DeeplyNestedProperty(t *testing.T) {
	check := generateSignalCheck("app.state.ui.ready")
	// Should guard the first level and check the full path
	if !containsString(check, "window.app") {
		t.Error("should contain guard for parent object 'window.app'")
	}
	if !containsString(check, "window.app.state.ui.ready === true") {
		t.Errorf("should check full path, got: %s", check)
	}
}

func TestGenerateSignalCheck_MethodAtStart(t *testing.T) {
	check := generateSignalCheck("getReady().status")
	// When method is at index 0, should check typeof directly on window
	if !containsString(check, "typeof window.getReady === 'function'") {
		t.Errorf("should check function type on window, got: %s", check)
	}
	if !containsString(check, "window.getReady().status === true") {
		t.Errorf("should call method and check property, got: %s", check)
	}
}

func TestGenerateSignalCheck_MethodWithDeepProperty(t *testing.T) {
	check := generateSignalCheck("bridge.getState().ui.initialized")
	if !containsString(check, "window.bridge") {
		t.Error("should contain guard for parent object")
	}
	if !containsString(check, "typeof window.bridge.getState === 'function'") {
		t.Errorf("should check function type, got: %s", check)
	}
	if !containsString(check, ".ui.initialized === true") {
		t.Errorf("should check nested property after method call, got: %s", check)
	}
}

func TestGenerateSignalCheck_MethodOnly(t *testing.T) {
	// Edge case: method call with no property after (e.g., "obj.isReady()")
	check := generateSignalCheck("obj.isReady()")
	// This is a method call but with no property after - should return empty or handle gracefully
	if !containsString(check, "typeof") {
		t.Errorf("should still generate typeof check, got: %s", check)
	}
}

func TestGenerateSignalCheck_SingleMethodNoProp(t *testing.T) {
	// Edge case: just a method call at root with nothing after (e.g., "isReady()")
	// This pattern is invalid - method calls need a property to check after them.
	// The code correctly returns empty string because len(parts) < 2.
	check := generateSignalCheck("isReady()")
	if check != "" {
		t.Errorf("generateSignalCheck(\"isReady()\") should return empty (invalid pattern), got: %q", check)
	}
}

func TestGenerateSignalCheck_InvalidMethodPattern(t *testing.T) {
	// Has () but only one part - should return empty
	check := generateSignalCheck("()")
	if check != "" {
		t.Errorf("generateSignalCheck(\"()\") should return empty string, got: %q", check)
	}
}

func TestGenerateHandshakeCheck_EmptySignals(t *testing.T) {
	check := generateHandshakeCheck(nil)
	if check != "false" {
		t.Errorf("generateHandshakeCheck(nil) = %q, want %q", check, "false")
	}
}

func TestGenerateHandshakeCheck_MultipleSignals(t *testing.T) {
	signals := []string{"flag1", "obj.ready"}
	check := generateHandshakeCheck(signals)

	if !containsString(check, "window.flag1 === true") {
		t.Error("should contain first signal check")
	}
	if !containsString(check, "window.obj.ready === true") {
		t.Error("should contain second signal check")
	}
	if !containsString(check, "||") {
		t.Error("should join with OR operator")
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
	viewport := orchestrator.Viewport{Width: 1280, Height: 720}
	payload := gen.Generate("http://localhost:3000", int64(90000), int(15000), viewport, nil)

	if len(payload) == 0 {
		t.Error("expected non-empty payload")
	}
}

func TestPayloadGenerator_Generate_ZeroTimeout(t *testing.T) {
	gen := NewPayloadGenerator()
	viewport := orchestrator.Viewport{Width: 1280, Height: 720}
	payload := gen.Generate("http://localhost:3000", nil, nil, viewport, nil)

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

func TestParseResponse_TruncatedJSON(t *testing.T) {
	// JSON that's been cut off mid-stream
	data := []byte(`{"success": true, "handshake": {"signaled": tr`)
	_, err := ParseResponse(data)
	if err == nil {
		t.Error("ParseResponse() should return error for truncated JSON")
	}
}

func TestParseResponse_WrongFieldTypes(t *testing.T) {
	// success should be bool, but is string
	data := []byte(`{"success": "yes", "handshake": {"signaled": true}}`)
	// Go's json.Unmarshal is lenient - this may or may not error
	// The important thing is it doesn't panic
	result, err := ParseResponse(data)
	if err != nil {
		// If it errors, that's acceptable
		return
	}
	// If it doesn't error, success should be false (zero value) since "yes" != true
	if result.Success {
		t.Error("Success should be false when type is wrong")
	}
}

func TestParseResponse_WrappedInvalidInnerData(t *testing.T) {
	// Valid wrapper but inner data is invalid JSON
	data := []byte(`{"data": "not valid json object", "type": "application/json"}`)
	_, err := ParseResponse(data)
	if err == nil {
		t.Error("ParseResponse() should return error for invalid inner data")
	}
}

func TestParseResponse_WrappedEmptyData(t *testing.T) {
	// Wrapper with empty data field
	data := []byte(`{"data": {}, "type": "application/json"}`)
	result, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	// Should parse but with zero values
	if result.Success {
		t.Error("Success should be false for empty data")
	}
}

func TestParseResponse_NullFields(t *testing.T) {
	data := []byte(`{
		"success": true,
		"console": null,
		"network": null,
		"pageErrors": null,
		"handshake": {"signaled": true},
		"html": null,
		"screenshot": null
	}`)
	result, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Console != nil {
		t.Error("Console should be nil")
	}
}

func TestParseResponse_ExtraUnknownFields(t *testing.T) {
	// JSON with extra fields that aren't in the struct
	data := []byte(`{
		"success": true,
		"unknownField1": "value",
		"anotherUnknown": 123,
		"handshake": {"signaled": true, "extraField": "ignored"}
	}`)
	result, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("ParseResponse() should not error on unknown fields: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestParseResponse_MalformedHandshake(t *testing.T) {
	// Handshake field is wrong type
	data := []byte(`{"success": true, "handshake": "not an object"}`)
	_, err := ParseResponse(data)
	if err == nil {
		t.Error("ParseResponse() should return error for malformed handshake")
	}
}

func TestParseResponse_VeryLargeResponse(t *testing.T) {
	// Test that we handle large responses without panic
	largeHTML := make([]byte, 1024*1024) // 1MB of HTML
	for i := range largeHTML {
		largeHTML[i] = 'a'
	}

	data := []byte(`{"success": true, "handshake": {"signaled": true}, "html": "` + string(largeHTML) + `"}`)
	result, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestParseResponse_UnicodeContent(t *testing.T) {
	data := []byte(`{
		"success": true,
		"handshake": {"signaled": true},
		"html": "<html>こんにちは世界 🌍 émojis</html>",
		"title": "Тест 测试"
	}`)
	result, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if !containsString(result.HTML, "こんにちは") {
		t.Error("HTML should contain Japanese characters")
	}
	if !containsString(result.Title, "Тест") {
		t.Error("Title should contain Cyrillic characters")
	}
}

func TestClient_ExecuteFunction_Timeout(t *testing.T) {
	// Server that takes longer than the timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond) // Respond slowly
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "handshake": {"signaled": true}}`))
	}))
	defer server.Close()

	// Create client with very short timeout
	c := NewClient(server.URL, WithTimeout(50*time.Millisecond))

	start := time.Now()
	_, err := c.ExecuteFunction(context.Background(), "module.exports = async () => {}")
	elapsed := time.Since(start)

	if err == nil {
		t.Error("ExecuteFunction() should return error on timeout")
	}

	// Verify the timeout was enforced (should be around 50ms, not 500ms)
	if elapsed >= 400*time.Millisecond {
		t.Errorf("Request took %v, expected timeout around 50ms", elapsed)
	}
}

func TestClient_Health_Timeout(t *testing.T) {
	// Server that takes longer than the timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithTimeout(50*time.Millisecond))

	start := time.Now()
	err := c.Health(context.Background())
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Health() should return error on timeout")
	}

	if elapsed >= 400*time.Millisecond {
		t.Errorf("Request took %v, expected timeout around 50ms", elapsed)
	}
}

func TestClient_ExecuteFunction_ContextTimeout(t *testing.T) {
	// Server that responds slowly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient(server.URL) // Default timeout is 120s

	// Use context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := c.ExecuteFunction(ctx, "module.exports = async () => {}")
	elapsed := time.Since(start)

	if err == nil {
		t.Error("ExecuteFunction() should return error when context times out")
	}

	if elapsed >= 400*time.Millisecond {
		t.Errorf("Request took %v, expected context timeout around 50ms", elapsed)
	}
}

func TestClient_DefaultTimeout(t *testing.T) {
	c := NewClient("http://localhost:4110")

	if c.httpClient.Timeout != 120*time.Second {
		t.Errorf("Default timeout = %v, want %v", c.httpClient.Timeout, 120*time.Second)
	}
}

func TestPayloadGenerator_Generate_CustomViewport(t *testing.T) {
	gen := NewPayloadGenerator()

	// Test with custom viewport dimensions
	customViewport := orchestrator.Viewport{Width: 1920, Height: 1080}
	payload := gen.Generate("http://localhost:3000", 90*time.Second, 15*time.Second, customViewport, nil)

	// Check that custom viewport dimensions are in the payload
	if !containsString(payload, "width: 1920") {
		t.Error("payload should contain custom viewport width 1920")
	}
	if !containsString(payload, "height: 1080") {
		t.Error("payload should contain custom viewport height 1080")
	}

	// Verify it doesn't contain default dimensions
	if containsString(payload, "width: 1280") {
		t.Error("payload should NOT contain default viewport width when custom is specified")
	}
}

func TestPayloadGenerator_Generate_MobileViewport(t *testing.T) {
	gen := NewPayloadGenerator()

	// Test with mobile viewport dimensions
	mobileViewport := orchestrator.Viewport{Width: 375, Height: 812}
	payload := gen.Generate("http://localhost:3000", 90*time.Second, 15*time.Second, mobileViewport, nil)

	// Check that mobile viewport dimensions are in the payload
	if !containsString(payload, "width: 375") {
		t.Error("payload should contain mobile viewport width 375")
	}
	if !containsString(payload, "height: 812") {
		t.Error("payload should contain mobile viewport height 812")
	}
}
