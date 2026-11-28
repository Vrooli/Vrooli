package httpmw

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAllowedOrigins(t *testing.T) {
	if got := AllowedOrigins(" https://console.example, http://localhost:* "); len(got) != 2 || got[0] != "https://console.example" || got[1] != "http://localhost:*" {
		t.Fatalf("parsed origins = %#v", got)
	}
	if !OriginAllowed("http://localhost:3000", AllowedOrigins("")) || OriginAllowed("https://example.com", AllowedOrigins("")) {
		t.Fatal("default origin policy mismatch")
	}
}

func TestCORSPreflightAndOriginPolicy(t *testing.T) {
	nextCalled := false
	handler := CORS(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true }))
	request := httptest.NewRequest(http.MethodOptions, "/", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || nextCalled {
		t.Fatalf("preflight status=%d nextCalled=%t", recorder.Code, nextCalled)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("allow origin=%q", got)
	}
}
