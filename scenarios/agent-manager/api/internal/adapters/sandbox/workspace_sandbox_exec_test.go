// Tests for the ExecProcess seam (Phase E of agent-sandbox-completion).
// Pins the wire shape POSTed to workspace-sandbox /sandboxes/{id}/exec
// and verifies the structured 403 (git_verb_blocked) is surfaced as an
// ExecProcessResult.Blocked entry rather than a generic transport error.

package sandbox_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agent-manager/internal/adapters/sandbox"

	"github.com/google/uuid"
)

func TestWorkspaceSandboxProvider_ExecProcess_Success(t *testing.T) {
	sandboxID := uuid.New()
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/exec") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"exitCode": 0,
			"stdout":   "hello\n",
			"stderr":   "",
			"pid":      4242,
		})
	}))
	defer server.Close()

	prov := sandbox.NewWorkspaceSandboxProvider(server.URL)
	res, err := prov.ExecProcess(context.Background(), sandbox.ExecProcessRequest{
		SandboxID:   sandboxID,
		Command:     "echo",
		Args:        []string{"hello"},
		NetworkMode: "localhost",
		TimeoutSec:  30,
	})
	if err != nil {
		t.Fatalf("ExecProcess: %v", err)
	}
	if res.ExitCode != 0 || res.Stdout != "hello\n" || res.PID != 4242 {
		t.Errorf("unexpected result: %+v", res)
	}
	if res.Blocked != nil {
		t.Errorf("Blocked should be nil on success, got %+v", res.Blocked)
	}
	if got["isolationLevel"] != "vrooli-aware" {
		t.Errorf("expected isolationLevel=vrooli-aware, got %v", got["isolationLevel"])
	}
	if got["timeoutSec"].(float64) != 30 {
		t.Errorf("expected timeoutSec=30, got %v", got["timeoutSec"])
	}
}

func TestWorkspaceSandboxProvider_ExecProcess_GitBlockedSurfaces403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":   "git_verb_blocked",
			"verb":    "commit",
			"message": "git verb \"commit\" is not in the protected-mode allowlist",
		})
	}))
	defer server.Close()

	prov := sandbox.NewWorkspaceSandboxProvider(server.URL)
	res, err := prov.ExecProcess(context.Background(), sandbox.ExecProcessRequest{
		SandboxID: uuid.New(),
		Command:   "git",
		Args:      []string{"commit", "-m", "x"},
	})
	if err != nil {
		t.Fatalf("ExecProcess should not return transport error on structured 403; got %v", err)
	}
	if res == nil || res.Blocked == nil {
		t.Fatal("expected Blocked to be populated on 403")
	}
	if res.Blocked.Error != "git_verb_blocked" || res.Blocked.Verb != "commit" {
		t.Errorf("unexpected Blocked: %+v", res.Blocked)
	}
}

func TestWorkspaceSandboxProvider_ExecProcess_FullNetworkOpensNetwork(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{"exitCode": 0})
	}))
	defer server.Close()

	prov := sandbox.NewWorkspaceSandboxProvider(server.URL)
	if _, err := prov.ExecProcess(context.Background(), sandbox.ExecProcessRequest{
		SandboxID:   uuid.New(),
		Command:     "curl",
		Args:        []string{"https://example.com"},
		NetworkMode: "full",
	}); err != nil {
		t.Fatalf("ExecProcess: %v", err)
	}
	if got["allowNetwork"] != true {
		t.Errorf("expected allowNetwork=true for full network mode, got %v", got["allowNetwork"])
	}
}
