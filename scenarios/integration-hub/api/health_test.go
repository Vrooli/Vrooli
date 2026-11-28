package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The control plane rejects a start whose /health does not echo the identity it
// injected, so the handler must serve it.
func TestHealth_ServesTheLifecycleStartIdentity(t *testing.T) {
	t.Setenv("VROOLI_START_OPERATION_ID", "start-op-test")
	t.Setenv("VROOLI_BUILD_IDENTITY", "sha256:test-build")
	recorder := httptest.NewRecorder()
	healthHandler()(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "healthy" || body["start_operation_id"] != "start-op-test" || body["build_identity"] != "sha256:test-build" {
		t.Fatalf("health = %v", body)
	}
}
