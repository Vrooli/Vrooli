package fakedriver

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestStartSessionServerLifecycleContract(t *testing.T) {
	server := StartSessionServer(t, "session-123")
	response, err := server.Client().Post(server.URL+"/session/start", "application/json", nil)
	if err != nil {
		t.Fatalf("start session request: %v", err)
	}
	var started struct {
		SessionID      string `json:"session_id"`
		ActualViewport struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"actual_viewport"`
	}
	if err := json.NewDecoder(response.Body).Decode(&started); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || started.SessionID != "session-123" || started.ActualViewport.Width != 1280 || started.ActualViewport.Height != 720 {
		t.Fatalf("start response = status %d, body %+v", response.StatusCode, started)
	}

	response, err = server.Client().Post(server.URL+"/session/session-123/close", "application/json", nil)
	if err != nil {
		t.Fatalf("close session request: %v", err)
	}
	var closed struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(response.Body).Decode(&closed); err != nil {
		t.Fatalf("decode close response: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || !closed.Success {
		t.Fatalf("close response = status %d, body %+v", response.StatusCode, closed)
	}
}
