package httpx

import (
	"net/http"
	"strings"
	"testing"
)

func TestServerHelpersHandleJSONRoundTrip(t *testing.T) {
	type requestBody struct {
		Name string `json:"name"`
	}
	type responseBody struct {
		OK bool `json:"ok"`
	}

	server := NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/test": func(w http.ResponseWriter, r *http.Request) {
			AssertMethod(t, r, http.MethodPost)
			req := DecodeJSON[requestBody](t, r)
			if req.Name != "git-control-tower" {
				t.Fatalf("Name = %q, want git-control-tower", req.Name)
			}
			WriteJSON(t, w, http.StatusAccepted, responseBody{OK: true})
		},
	})

	res, err := TestClient().Post(server.URL+"/api/v1/test", "application/json", strings.NewReader(`{"name":"git-control-tower"}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusAccepted)
	}
	if got := res.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}
}

func TestNewHandlerServerRegistersCleanup(t *testing.T) {
	server := NewHandlerServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	res, err := TestClient().Get(server.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusNoContent)
	}
}
