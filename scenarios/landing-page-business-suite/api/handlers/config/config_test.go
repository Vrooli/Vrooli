package config

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLandingAppliesTestDelayBeforeLoading(t *testing.T) {
	sequence := []string{}
	handler := Landing(Dependencies{
		TestMode: func(context.Context) bool { return true },
		Sleep: func(delay time.Duration) {
			if delay != time.Second {
				t.Errorf("delay = %s, want 1s", delay)
			}
			sequence = append(sequence, "sleep")
		},
		Delay: time.Second,
		Get: func(context.Context, string) (any, error) {
			sequence = append(sequence, "get")
			return map[string]string{"ok": "true"}, nil
		},
		WriteJSON: func(http.ResponseWriter, any) {},
	})
	handler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/landing-config", nil))
	if len(sequence) != 2 || sequence[0] != "sleep" || sequence[1] != "get" {
		t.Fatalf("sequence = %v, want [sleep get]", sequence)
	}
}
