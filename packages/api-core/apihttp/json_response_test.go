package apihttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSONResponseWritesStatusHeadersAndEnvelope(t *testing.T) {
	response := httptest.NewRecorder()
	WriteJSONResponse(response, http.StatusUnauthorized, map[string]any{
		"success": false,
		"error":   "unauthorized",
	})

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	var got map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["success"] != false || got["error"] != "unauthorized" {
		t.Fatalf("response = %#v, want unauthorized failure envelope", got)
	}
}

func TestBearerTokenMiddlewarePreservesExemptionsAndExactTokenCheck(t *testing.T) {
	const token = "fixture-token"
	unauthorizedCalls := 0
	unauthorized := func(w http.ResponseWriter, _ *http.Request) {
		unauthorizedCalls++
		w.WriteHeader(http.StatusUnauthorized)
	}
	passed := 0
	protected := BearerTokenMiddleware(token, unauthorized, "/health", "/docs")(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { passed++ }),
	)

	tests := []struct {
		name       string
		path       string
		authority  string
		wantStatus int
		wantPassed int
		wantDenied int
	}{
		{name: "health exemption", path: "/health", wantStatus: http.StatusOK, wantPassed: 1},
		{name: "docs exemption", path: "/docs", wantStatus: http.StatusOK, wantPassed: 1},
		{name: "valid bearer", path: "/api", authority: "Bearer fixture-token", wantStatus: http.StatusOK, wantPassed: 1},
		{name: "missing bearer", path: "/api", wantStatus: http.StatusUnauthorized, wantDenied: 1},
		{name: "wrong bearer", path: "/api", authority: "Bearer wrong", wantStatus: http.StatusUnauthorized, wantDenied: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			passed, unauthorizedCalls = 0, 0
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			if test.authority != "" {
				request.Header.Set("Authorization", test.authority)
			}
			response := httptest.NewRecorder()
			protected.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if passed != test.wantPassed {
				t.Fatalf("downstream calls = %d, want %d", passed, test.wantPassed)
			}
			if unauthorizedCalls != test.wantDenied {
				t.Fatalf("unauthorized calls = %d, want %d", unauthorizedCalls, test.wantDenied)
			}
		})
	}
}
