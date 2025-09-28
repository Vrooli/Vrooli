package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestSubmitIssueToTrackerReturnsIssueID(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/issues" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}
		if payload["title"] == "" {
			t.Fatalf("expected title field to be populated")
		}

		resp := map[string]interface{}{
			"success": true,
			"message": "created",
			"data": map[string]interface{}{
				"issue_id": "issue-abc",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	if len(parts) < 3 {
		t.Fatalf("unexpected server URL: %s", server.URL)
	}
	portStr := parts[len(parts)-1]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	result, err := submitIssueToTracker(context.Background(), port, map[string]interface{}{"title": "test"})
	if err != nil {
		t.Fatalf("submitIssueToTracker returned error: %v", err)
	}
	if result == nil || result.IssueID != "issue-abc" {
		t.Fatalf("expected issue-abc, got %#v", result)
	}
}

func TestSubmitIssueToTrackerParsesNestedIssueID(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success": true,
			"message": "created",
			"data": map[string]interface{}{
				"issue": map[string]interface{}{
					"id": "issue-nested",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	if len(parts) < 3 {
		t.Fatalf("unexpected server URL: %s", server.URL)
	}
	portStr := parts[len(parts)-1]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	result, err := submitIssueToTracker(context.Background(), port, map[string]interface{}{"title": "test"})
	if err != nil {
		t.Fatalf("submitIssueToTracker returned error: %v", err)
	}
	if result.IssueID != "issue-nested" {
		t.Fatalf("expected issue-nested, got %q", result.IssueID)
	}
}
