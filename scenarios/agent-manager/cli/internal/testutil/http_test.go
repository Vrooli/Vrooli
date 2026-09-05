package testutil

import (
	"io"
	"net/http"
	"testing"
)

func TestRecordingServerCapturesImmutableHTTPContract(t *testing.T) {
	server := NewRecordingServer(t, `{"ok":true}`)
	response, err := http.Get(server.URL() + "/api/v1/runs?limit=5")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusOK || string(body) != `{"ok":true}` {
		t.Fatalf("response status=%d body=%s", response.StatusCode, body)
	}
	requests := server.Requests()
	if len(requests) != 1 || requests[0] != (Request{Method: http.MethodGet, Path: "/api/v1/runs", Query: "limit=5"}) {
		t.Fatalf("requests=%+v", requests)
	}
	requests[0].Path = "/mutated"
	if server.Requests()[0].Path != "/api/v1/runs" {
		t.Fatal("Requests returned mutable recorder state")
	}
}
