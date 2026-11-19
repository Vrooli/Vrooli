package handlers

import (
	"net/http/httptest"
	"testing"
)

func TestProcessCPUReplayHandlerRejectsMissingTimes(t *testing.T) {
	handler := NewProcessCPUReplayHandler(nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/forensics/process-cpu-replay", nil))
	if response.Code != 400 {
		t.Fatalf("status = %d, want 400 for missing replay window", response.Code)
	}
}
