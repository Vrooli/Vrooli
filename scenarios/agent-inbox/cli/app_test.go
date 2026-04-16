package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestAppCommandsAgainstAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"status":"healthy","service":"agent-inbox"}`)
		case r.URL.Path == "/api/v1/chats" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":"chat-1","name":"Inbox Zero","preview":"follow up","model":"anthropic/claude","is_read":false,"is_archived":false,"is_starred":true,"label_ids":["label-1"],"tools_enabled":true,"web_search_enabled":false,"chat_mode":"llm","created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-02T00:00:00Z"}]`)
		case r.URL.Path == "/api/v1/chats" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":"chat-2","name":"New Chat","model":"anthropic/claude","is_read":true,"is_archived":false,"is_starred":false,"label_ids":[],"tools_enabled":true,"web_search_enabled":false,"chat_mode":"llm","created_at":"2025-01-02T00:00:00Z","updated_at":"2025-01-02T00:00:00Z"}`)
		case r.URL.Path == "/api/v1/agent-runs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"runs":[{"run_id":"run-1","task_id":"task-1","status":"running","phase":"coding","progress_percent":25,"created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-01T00:01:00Z"}],"total":1,"has_more":false}`)
		case r.URL.Path == "/api/v1/settings/yolo-mode" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"enabled":true,"success":true}`)
		case r.URL.Path == "/api/v1/settings/yolo-mode" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"enabled":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tests := [][]string{
		{"--api-base", server.URL + "/api/v1", "chat", "list", "--json"},
		{"--api-base", server.URL + "/api/v1", "list", "--json"},
		{"--api-base", server.URL + "/api/v1", "new", "--json"},
		{"--api-base", server.URL + "/api/v1", "agent", "runs", "--json"},
		{"--api-base", server.URL + "/api/v1", "settings", "yolo", "--set", "true", "--json"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args[2:], "_"), func(t *testing.T) {
			app, err := NewApp()
			if err != nil {
				t.Fatalf("NewApp: %v", err)
			}
			withCapturedStdout(t, func() {
				if err := app.Run(args); err != nil {
					t.Fatalf("Run(%v): %v", args, err)
				}
			})
		})
	}
}

func TestStatusHelpDoesNotCallAPI(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	output := captureOutput(t, func() {
		if err := app.Run([]string{"status", "--help"}); err != nil {
			t.Fatalf("Run: %v", err)
		}
	})
	if !strings.Contains(output, "agent-inbox status [--json]") {
		t.Fatalf("expected status help output, got %q", output)
	}
}

func withCapturedStdout(t *testing.T, fn func()) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, r)
		close(done)
	}()

	fn()
	_ = w.Close()
	<-done
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	done := make(chan string, 1)
	go func() {
		var builder strings.Builder
		_, _ = io.Copy(&builder, r)
		done <- builder.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}
