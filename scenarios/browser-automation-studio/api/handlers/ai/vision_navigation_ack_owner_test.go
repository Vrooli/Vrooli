package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestVisionNavigationCallbackCallersShareAcknowledgementOwner(t *testing.T) {
	handler := NewVisionNavigationHandler(logrus.New())
	cases := []struct {
		name string
		body string
		want map[string]any
	}{
		{name: "step event", body: `{"stepNumber":1}`, want: map[string]any{"received": true, "abort": false}},
		{name: "completion event", body: `{"status":"completed"}`, want: map[string]any{"received": true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/ai-navigate/callback", strings.NewReader(tc.body))
			response := httptest.NewRecorder()
			handler.HandleAINavigateCallback(response, req)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
			}
			if got := response.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			var got map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode acknowledgement: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("acknowledgement = %#v, want %#v", got, tc.want)
			}
			for key, want := range tc.want {
				if got[key] != want {
					t.Errorf("acknowledgement[%q] = %#v, want %#v", key, got[key], want)
				}
			}
		})
	}
}
