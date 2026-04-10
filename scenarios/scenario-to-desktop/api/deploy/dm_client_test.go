package deploy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestDMServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func newTestDMClient(serverURL string) *DMClient {
	return NewDMClientWithResolver(
		func(_ context.Context) (string, error) { return serverURL, nil },
		http.DefaultClient,
	)
}

func TestDMClient_CreateApproval_Success(t *testing.T) {
	server := newTestDMServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/profiles/prof-1/approvals" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["git_commit_hash"] != "abc123" {
			t.Errorf("expected commit hash abc123, got %s", body["git_commit_hash"])
		}
		if body["platform"] != "linux" {
			t.Errorf("expected platform linux, got %s", body["platform"])
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Approval{
			ID:            "appr-1",
			ProfileID:     "prof-1",
			GitCommitHash: "abc123",
			Platform:      "linux",
			Status:        "pending",
		})
	})
	defer server.Close()

	client := newTestDMClient(server.URL)
	approval, err := client.CreateApproval(context.Background(), "prof-1", "abc123", "linux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approval.ID != "appr-1" {
		t.Errorf("expected ID appr-1, got %s", approval.ID)
	}
	if approval.Status != "pending" {
		t.Errorf("expected status pending, got %s", approval.Status)
	}
}

func TestDMClient_CreateApproval_409_Idempotent(t *testing.T) {
	server := newTestDMServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(Approval{
			ID:       "existing-1",
			Status:   "pending",
			Platform: "linux",
		})
	})
	defer server.Close()

	client := newTestDMClient(server.URL)
	approval, err := client.CreateApproval(context.Background(), "prof-1", "abc123", "linux")
	if err != nil {
		t.Fatalf("409 should not return error: %v", err)
	}
	if approval.ID != "existing-1" {
		t.Errorf("expected existing approval ID, got %s", approval.ID)
	}
}

func TestDMClient_CreateApproval_500(t *testing.T) {
	server := newTestDMServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	})
	defer server.Close()

	client := newTestDMClient(server.URL)
	_, err := client.CreateApproval(context.Background(), "prof-1", "abc123", "linux")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestDMClient_CreateApproval_ConnectionRefused(t *testing.T) {
	client := NewDMClientWithResolver(
		func(_ context.Context) (string, error) { return "http://127.0.0.1:1", nil },
		&http.Client{},
	)
	_, err := client.CreateApproval(context.Background(), "prof-1", "abc123", "linux")
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestDMClient_CheckReleaseGate_Ready(t *testing.T) {
	server := newTestDMServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("commit") != "abc123" {
			t.Errorf("expected commit=abc123, got %s", r.URL.Query().Get("commit"))
		}
		_ = json.NewEncoder(w).Encode(ReleaseGateStatus{
			Ready: true,
			Platforms: []PlatformGateStatus{
				{Platform: "linux", Status: "approved", Ready: true},
				{Platform: "win", Status: "approved", Ready: true},
			},
		})
	})
	defer server.Close()

	client := newTestDMClient(server.URL)
	status, err := client.CheckReleaseGate(context.Background(), "prof-1", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.Ready {
		t.Error("expected gate to be ready")
	}
	if len(status.Platforms) != 2 {
		t.Errorf("expected 2 platforms, got %d", len(status.Platforms))
	}
}

func TestDMClient_CheckReleaseGate_Blocked(t *testing.T) {
	server := newTestDMServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(ReleaseGateStatus{
			Ready: false,
			Platforms: []PlatformGateStatus{
				{Platform: "linux", Status: "approved", Ready: true},
				{Platform: "win", Status: "pending", Ready: false},
			},
		})
	})
	defer server.Close()

	client := newTestDMClient(server.URL)
	status, err := client.CheckReleaseGate(context.Background(), "prof-1", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Ready {
		t.Error("expected gate to be blocked")
	}
	blocked := status.BlockedPlatforms()
	if len(blocked) != 1 || blocked[0] != "win" {
		t.Errorf("expected [win] blocked, got %v", blocked)
	}
}

func TestDMClient_CheckReleaseGate_500(t *testing.T) {
	server := newTestDMServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	defer server.Close()

	client := newTestDMClient(server.URL)
	_, err := client.CheckReleaseGate(context.Background(), "prof-1", "abc123")
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestDMClient_ListApprovals(t *testing.T) {
	server := newTestDMServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode([]Approval{
			{ID: "a1", Platform: "linux", Status: "approved"},
			{ID: "a2", Platform: "win", Status: "pending"},
		})
	})
	defer server.Close()

	client := newTestDMClient(server.URL)
	approvals, err := client.ListApprovals(context.Background(), "prof-1", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(approvals) != 2 {
		t.Errorf("expected 2 approvals, got %d", len(approvals))
	}
}

func TestReleaseGateStatus_BlockedPlatforms(t *testing.T) {
	status := &ReleaseGateStatus{
		Ready: false,
		Platforms: []PlatformGateStatus{
			{Platform: "linux", Ready: true},
			{Platform: "win", Ready: false},
			{Platform: "mac", Ready: false},
		},
	}
	blocked := status.BlockedPlatforms()
	if len(blocked) != 2 {
		t.Errorf("expected 2 blocked, got %d", len(blocked))
	}
}
