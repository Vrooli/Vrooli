package validations

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCommandsCoverValidationLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/validations":
			_, _ = io.WriteString(w, `{"id":"v1","status":"running"}`)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/review"):
			_, _ = io.WriteString(w, `{"approval_id":"a1","approval_status":"pending"}`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/profiles/"):
			_, _ = io.WriteString(w, `{"id":"v1","video_url":"/video"}`)
		default:
			_, _ = io.WriteString(w, `[{"id":"v1"}]`)
		}
	}))
	defer server.Close()
	cmd := New(testAPIClient(server.URL))

	if err := cmd.Run([]string{"run", "demo", "--record", "--platform", "linux", "--commit", "abc", "--format", "json"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if err := cmd.Run([]string{"status", "v1", "--format", "json"}); err != nil {
		t.Fatalf("status: %v", err)
	}
	if err := cmd.Run([]string{"video", "v1", "--output", "capture.mp4"}); err != nil {
		t.Fatalf("video: %v", err)
	}
	if err := cmd.Run([]string{"review", "v1", "--decision", "approve", "--notes", "looks good"}); err != nil {
		t.Fatalf("review approve: %v", err)
	}
	if err := cmd.Run([]string{"review", "v1", "--decision", "reject", "--format", "json"}); err != nil {
		t.Fatalf("review reject: %v", err)
	}
	if err := cmd.Run([]string{"list", "--profile", "demo", "--commit", "abc", "--format", "json"}); err != nil {
		t.Fatalf("list: %v", err)
	}
}

func TestCommandsRejectInvalidValidationInput(t *testing.T) {
	cmd := New(testAPIClient("http://127.0.0.1:1"))
	cases := [][]string{
		{},
		{"run"},
		{"run", "demo"},
		{"status"},
		{"video"},
		{"review", "v1"},
		{"review", "v1", "--decision", "maybe"},
		{"list"},
		{"unknown"},
	}
	for _, args := range cases {
		if err := cmd.Run(args); err == nil {
			t.Errorf("expected error for %v", args)
		}
	}
}
