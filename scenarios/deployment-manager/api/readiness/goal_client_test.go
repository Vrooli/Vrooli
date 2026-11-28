package readiness

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGoalClientRequiresResolver(t *testing.T) {
	_, _, err := (&GoalClient{}).Open(context.Background(), GoalSpec{Name: "readiness/demo/abc"})
	if err == nil {
		t.Fatal("expected unconfigured client refusal")
	}
}

func TestGoalClientUsesCanonicalNameForDeduplicationAndMilestones(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/GetGoal") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"not_found","message":"missing"}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/CreateGoal") {
			_ = json.NewEncoder(w).Encode(map[string]any{"goal": map[string]string{"name": "readiness-demo-abc"}})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/CreateMilestone") {
			_ = json.NewEncoder(w).Encode(map[string]any{"goal": map[string]string{"name": "readiness-demo-abc"}})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &GoalClient{ResolveURL: func(context.Context) (string, error) { return server.URL, nil }, HTTPClient: server.Client()}
	name, deduped, err := client.Open(context.Background(), GoalSpec{
		Name: "readiness/demo/abc", Title: "Readiness", Priority: 0,
		Milestones: []GoalMilestone{{Name: "one", Title: "One", AcceptanceCriteria: []string{"pass"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if deduped || name != "readiness-demo-abc" {
		t.Fatalf("name=%q deduped=%t, want canonical created name", name, deduped)
	}
	if len(paths) != 3 || !strings.HasSuffix(paths[2], "/CreateMilestone") {
		t.Fatalf("unexpected Connect calls: %v", paths)
	}
}
