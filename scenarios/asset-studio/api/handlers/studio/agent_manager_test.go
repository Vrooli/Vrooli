package studio

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAgentManagerCommissionerCreatesUntrustedTask(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agent_manager.v1.AgentManagerService/CreateTask" {
			t.Fatal(r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), "untrusted proposal") {
			t.Fatal(string(b))
		}
		_, _ = w.Write([]byte(`{"task":{"id":"task-1"}}`))
	}))
	defer s.Close()
	id, e := (&agentManagerCommissioner{BaseURL: s.URL, Client: s.Client()}).CreateTask(context.Background(), "draft a brief", "operator")
	if e != nil || id != "task-1" {
		t.Fatalf("id=%q err=%v", id, e)
	}
}
