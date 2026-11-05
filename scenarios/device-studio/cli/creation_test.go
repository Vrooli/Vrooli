package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreationCLICallsDisclosedLocalJobRoutes(t *testing.T) {
	var starts, gets, cancels int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/projects/project-1/geometry-creations":
			starts++
			var request map[string]any
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode start body: %v", err)
			}
			if request["confirm_local_ai"] != true || request["brief"] != "compact device" || request["reference_media_ids"] == nil {
				t.Errorf("unexpected start request: %#v", request)
			}
			_, _ = io.WriteString(w, `{"job_id":"job-1","project_id":"project-1","candidate_id":"creation-test","state":"queued","version":1,"review_status":"unreviewed"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/geometry-creations/job-1":
			gets++
			_, _ = io.WriteString(w, `{"job_id":"job-1","project_id":"project-1","candidate_id":"creation-test","state":"running","version":4,"review_status":"unreviewed"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/geometry-creations/job-1/cancel":
			cancels++
			var request map[string]int64
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request["version"] != 4 {
				t.Errorf("cancel did not use observed version: %#v %v", request, err)
			}
			_, _ = io.WriteString(w, `{"job_id":"job-1","project_id":"project-1","candidate_id":"creation-test","state":"cancel_requested","version":5,"review_status":"unreviewed"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("DEVICE_STUDIO_API_BASE", server.URL)
	t.Setenv("API_BASE_URL", "")
	t.Setenv("VITE_API_BASE_URL", "")
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"creation", "start", "--project-id", "project-1", "--brief", "compact device", "--width-mm", "78", "--height-mm", "148", "--depth-mm", "12", "--reference-media-ids", "media-1,media-2", "--confirm-local-ai"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"creation", "get", "job-1"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"creation", "cancel", "job-1", "--expected-version", "4"}); err != nil {
		t.Fatal(err)
	}
	if starts != 1 || gets != 1 || cancels != 1 {
		t.Fatalf("route calls start=%d get=%d cancel=%d", starts, gets, cancels)
	}
	if err := app.Run([]string{"creation", "start", "--project-id", "project-1", "--brief", "compact device", "--width-mm", "78", "--height-mm", "148", "--depth-mm", "12", "--reference-media-ids", "media-1"}); err == nil {
		t.Fatal("CLI must require explicit local AI consent")
	}
	if starts != 1 {
		t.Fatalf("missing consent dispatched a request: %d", starts)
	}
}
