package swaps

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCommandsCoverSwapOperations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/swaps/suggest/"):
			_, _ = io.WriteString(w, `[{"from":"postgres","to":"sqlite","reason":"local","impact":"low","score":0.9},{"from":"redis","to":"memory","impact":"medium"}]`)
		case strings.Contains(r.URL.Path, "/swaps/analyze/"):
			_, _ = io.WriteString(w, `{"compatible":true,"impact":"low"}`)
		case strings.Contains(r.URL.Path, "/swaps/cascade/"):
			_, _ = io.WriteString(w, `{"affected":["worker"]}`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/swaps/"):
			_, _ = io.WriteString(w, `{"id":"s1","from":"postgres","to":"sqlite"}`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/profiles/"):
			_, _ = io.WriteString(w, `{"fitness":0.9}`)
		default:
			_, _ = io.WriteString(w, `{}`)
		}
	}))
	defer server.Close()
	cmd := New(testAPIClient(server.URL))
	if err := cmd.Run([]string{"list", "example", "--format", "table"}); err != nil {
		t.Fatalf("list table: %v", err)
	}
	if err := cmd.Run([]string{"list", "example", "--format", "json"}); err != nil {
		t.Fatalf("list json: %v", err)
	}
	if err := cmd.Run([]string{"analyze", "postgres", "sqlite", "--format", "json"}); err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if err := cmd.Run([]string{"cascade", "postgres", "sqlite"}); err != nil {
		t.Fatalf("cascade: %v", err)
	}
	if err := cmd.Run([]string{"info", "s1"}); err != nil {
		t.Fatalf("info: %v", err)
	}
	if err := cmd.Run([]string{"apply", "demo", "postgres", "sqlite"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if err := cmd.Run([]string{"apply", "demo", "postgres", "sqlite", "--show-fitness", "--format", "json"}); err != nil {
		t.Fatalf("apply with fitness: %v", err)
	}
}

func TestCommandsRejectSwapInput(t *testing.T) {
	cmd := New(testAPIClient("http://127.0.0.1:1"))
	for _, args := range [][]string{{}, {"unknown"}, {"list"}, {"analyze", "one"}, {"cascade", "one"}, {"info"}, {"apply", "demo", "from"}} {
		if err := cmd.Run(args); err == nil {
			t.Errorf("expected error for %v", args)
		}
	}
}
