package overview

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCommandsRenderOverviewReports(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, `{"dependencies":["postgres"],"count":1}`)
			return
		}
		_, _ = io.WriteString(w, `{"score":0.9,"compatibility":"high","blockers":[]}`)
	}))
	defer server.Close()
	cmd := New(testAPIClient(server.URL))
	if err := cmd.Analyze([]string{"demo"}); err != nil {
		t.Fatalf("analyze table: %v", err)
	}
	if err := cmd.Analyze([]string{"demo", "--format", "json"}); err != nil {
		t.Fatalf("analyze json: %v", err)
	}
	if err := cmd.Fitness([]string{"demo", "--tier", "desktop"}); err != nil {
		t.Fatalf("fitness table: %v", err)
	}
	if err := cmd.Fitness([]string{"demo", "--format", "json"}); err != nil {
		t.Fatalf("fitness json: %v", err)
	}
}
