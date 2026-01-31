package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func TestDo_SetsClientSourceHeader(t *testing.T) {
	// Create a test server that captures headers
	var receivedHeaders http.Header
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok": true}`)
	}))
	defer ts.Close()

	// Create a minimal context with the test server URL
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:           "test-app",
		Version:        "1.0.0",
		AllowAnonymous: true,
	})
	if err != nil {
		t.Fatalf("Failed to create scenario app: %v", err)
	}

	// Override the API base to point to our test server
	core.APIOverride = ts.URL

	ctx := &appctx.Context{
		Name:    "test-app",
		Version: "1.0.0",
		Core:    core,
	}

	// Make a request
	statusCode, _, err := Do(ctx, "GET", "/test", nil, nil, nil)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("Do() status = %d, want %d", statusCode, http.StatusOK)
	}

	// Verify X-Client-Source header was set
	clientSource := receivedHeaders.Get("X-Client-Source")
	if clientSource != "cli" {
		t.Errorf("X-Client-Source header = %q, want %q", clientSource, "cli")
	}
}

func TestDo_SetsAuthorizationHeader(t *testing.T) {
	var receivedHeaders http.Header
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Set token via environment variable (the way ScenarioApp expects it)
	const testTokenEnv = "TEST_APP_API_TOKEN"
	os.Setenv(testTokenEnv, "test-token-123")
	defer os.Unsetenv(testTokenEnv)

	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:           "test-app",
		Version:        "1.0.0",
		AllowAnonymous: true,
		TokenEnvVars:   []string{testTokenEnv},
	})
	if err != nil {
		t.Fatalf("Failed to create scenario app: %v", err)
	}
	core.APIOverride = ts.URL

	ctx := &appctx.Context{
		Name:         "test-app",
		Version:      "1.0.0",
		Core:         core,
		TokenEnvVars: []string{testTokenEnv},
	}

	_, _, err = Do(ctx, "GET", "/test", nil, nil, nil)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	auth := receivedHeaders.Get("Authorization")
	if auth != "Bearer test-token-123" {
		t.Errorf("Authorization header = %q, want %q", auth, "Bearer test-token-123")
	}
}

func TestDo_SetsContentTypeForBody(t *testing.T) {
	var receivedHeaders http.Header
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:           "test-app",
		Version:        "1.0.0",
		AllowAnonymous: true,
	})
	if err != nil {
		t.Fatalf("Failed to create scenario app: %v", err)
	}
	core.APIOverride = ts.URL

	ctx := &appctx.Context{
		Name:    "test-app",
		Version: "1.0.0",
		Core:    core,
	}

	// Request with body
	body := []byte(`{"key": "value"}`)
	_, _, err = Do(ctx, "POST", "/test", nil, body, nil)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	contentType := receivedHeaders.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type header = %q, want %q", contentType, "application/json")
	}
}

func TestDo_CustomHeaders(t *testing.T) {
	var receivedHeaders http.Header
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:           "test-app",
		Version:        "1.0.0",
		AllowAnonymous: true,
	})
	if err != nil {
		t.Fatalf("Failed to create scenario app: %v", err)
	}
	core.APIOverride = ts.URL

	ctx := &appctx.Context{
		Name:    "test-app",
		Version: "1.0.0",
		Core:    core,
	}

	customHeaders := map[string]string{
		"X-Custom-Header": "custom-value",
	}
	_, _, err = Do(ctx, "GET", "/test", nil, nil, customHeaders)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	customHeader := receivedHeaders.Get("X-Custom-Header")
	if customHeader != "custom-value" {
		t.Errorf("X-Custom-Header = %q, want %q", customHeader, "custom-value")
	}

	// X-Client-Source should still be set
	clientSource := receivedHeaders.Get("X-Client-Source")
	if clientSource != "cli" {
		t.Errorf("X-Client-Source header = %q, want %q", clientSource, "cli")
	}
}

func TestDo_NilContext(t *testing.T) {
	_, _, err := Do(nil, "GET", "/test", nil, nil, nil)
	if err == nil {
		t.Error("Do(nil context) should return error")
	}
}

func TestDo_EmptyAPIBase(t *testing.T) {
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:           "test-app",
		Version:        "1.0.0",
		AllowAnonymous: true,
	})
	if err != nil {
		t.Fatalf("Failed to create scenario app: %v", err)
	}
	// Don't set API base - leave it empty

	ctx := &appctx.Context{
		Name:    "test-app",
		Version: "1.0.0",
		Core:    core,
	}

	_, _, err = Do(ctx, "GET", "/test", nil, nil, nil)
	if err == nil {
		t.Error("Do() with empty API base should return error")
	}
}
