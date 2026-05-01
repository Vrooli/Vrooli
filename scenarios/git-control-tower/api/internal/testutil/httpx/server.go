package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func NewServer(t *testing.T, routes map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	for path, handler := range routes {
		mux.HandleFunc(path, handler)
	}
	return NewHandlerServer(t, mux)
}

func NewHandlerServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func TestClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

func AssertMethod(t *testing.T, r *http.Request, want string) {
	t.Helper()

	if r.Method != want {
		t.Errorf("expected %s, got %s", want, r.Method)
	}
}

func DecodeJSON[T any](t *testing.T, r *http.Request) T {
	t.Helper()

	var value T
	if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
		t.Errorf("decode request: %v", err)
	}
	return value
}

func WriteJSON(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Errorf("encode response: %v", err)
	}
}
