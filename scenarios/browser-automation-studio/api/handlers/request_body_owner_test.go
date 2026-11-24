package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/browser-automation-studio/internal/httpjson"
)

func TestRecordingAndHistoryCallbacksShareBoundedJSONBodyOwner(t *testing.T) {
	handler, recordMode, tempDir, _ := createTestHandlerWithRecordMode(t)
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	oversizedSessionID := strings.Repeat("x", int(httpjson.MaxBodyBytes())+1)
	body := fmt.Sprintf(`{"session_id":%q}`, oversizedSessionID)

	tests := []struct {
		name    string
		invoke  func(http.ResponseWriter, *http.Request)
		wantHit func() bool
	}{
		{
			name:    "start live recording",
			invoke:  handler.StartLiveRecording,
			wantHit: func() bool { return recordMode.StartRecordingCalled },
		},
		{
			name:   "history callback",
			invoke: handler.HistoryCallback,
			wantHit: func() bool {
				return false // The callback has no browser-session side effect before decoding.
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
			response := httptest.NewRecorder()
			tc.invoke(response, req)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
			}
			if tc.wantHit() {
				t.Fatal("oversized request reached its downstream caller")
			}
		})
	}
}
