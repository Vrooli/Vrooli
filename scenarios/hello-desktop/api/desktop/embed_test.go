package desktop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewExposesAPIHandlerAndCleanLifecycle(t *testing.T) {
	handler, lifecycle, err := New()
	if err != nil || handler == nil || lifecycle == nil {
		t.Fatalf("New() = (%v, %v, %v)", handler, lifecycle, err)
	}
	ctx := context.Background()
	if err := lifecycle.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := lifecycle.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := lifecycle.Start(ctx); err == nil {
		t.Fatal("lifecycle restarted after stop")
	}
}

func TestEmbeddedHandlerPreservesFixtureJourneyRoutes(t *testing.T) {
	handler, _, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/health", "/api/greet?name=Embed%20Test", "/api/test/last-greeting?name=Embed%20Test"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s", path, response.Code, response.Body.String())
		}
		var payload map[string]string
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("GET %s returned invalid JSON: %v", path, err)
		}
	}
}
