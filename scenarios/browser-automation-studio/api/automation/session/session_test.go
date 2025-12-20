package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/driver"
)

func TestCloseWithArtifacts(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/session/sess-123/close", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":     true,
			"video_paths": []string{"/tmp/video-1.webm", "/tmp/video-2.webm"},
		})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL)
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	sess := &Session{
		id:     "sess-123",
		mode:   ModeExecution,
		client: client,
	}

	resp, err := sess.CloseWithArtifacts(context.Background())
	if err != nil {
		t.Fatalf("close: %v", err)
	}

	if resp == nil || len(resp.VideoPaths) != 2 {
		t.Fatalf("expected 2 video paths, got %#v", resp)
	}
}

