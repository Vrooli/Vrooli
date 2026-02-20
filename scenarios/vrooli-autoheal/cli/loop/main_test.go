package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunTick_RequestsCompactResponse(t *testing.T) {
	t.Helper()

	var gotCompact string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		gotCompact = r.URL.Query().Get("compact")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"status":"ok","summary":{"total":1,"ok":1,"warning":0,"critical":0}}`))
	}))
	defer ts.Close()

	cfg := &Config{TickEndpoint: ts.URL + "/api/v1/tick"}
	result, err := runTick(cfg)
	if err != nil {
		t.Fatalf("runTick() error = %v", err)
	}
	if gotCompact != "true" {
		t.Fatalf("compact query param = %q, want true", gotCompact)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q, want ok", result.Status)
	}
}
