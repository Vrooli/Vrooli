// Tests for the protected-mode git allowlist enforcement on the
// /sandboxes/{id}/processes (StartProcess) handler.
//
// The allowlist is enforced by Exec already; mirroring it on StartProcess
// closes a bypass: an agent process launched via /processes could otherwise
// invoke `git push` directly, sidestepping the /exec guard.
//
// See execute/protected-sandbox-agent-launch.
//
// Round 4 Phase 3 migrated these tests off httptest.ResponseRecorder onto
// the live-HTTP harness so the request flows through the production
// middleware chain.

package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/runtime"
	"workspace-sandbox/internal/testutil/mocks/sandboxiface"
	"workspace-sandbox/internal/types"
)

func newProtectedSandboxFixture(id uuid.UUID, allowlist []string) *types.Sandbox {
	return &types.Sandbox{
		ID:            id,
		ScopePath:     "/project/src",
		ProjectRoot:   "/project",
		Owner:         "test-agent",
		Status:        types.StatusActive,
		DriverID:      "overlayfs-userns",
		DriverVersion: "1.0",
		CreatedAt:     time.Now(),
		MergedDir:     "/tmp/sandbox/" + id.String() + "/merged",
		Behavior: types.SandboxBehavior{
			Protected: types.ProtectedConfig{
				GitAllowlist: allowlist,
			},
		},
	}
}

func protectedService(sb *types.Sandbox) *sandboxiface.FakeService {
	return &sandboxiface.FakeService{
		GetFn: func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
			return sb, nil
		},
	}
}

// TestStartProcess_BlocksDisallowedGitVerb verifies the allowlist intercepts
// `git push` (a mutating verb) under the default allowlist of read-only verbs.
func TestStartProcess_BlocksDisallowedGitVerb(t *testing.T) {
	id := uuid.New()
	sb := newProtectedSandboxFixture(id, types.DefaultProtectedGitAllowlist())
	live := newLive(t, protectedService(sb))

	resp, body := live.DoJSON(t, "POST", sandboxesPath(id, "/processes"),
		`{"command": "git", "args": ["push", "--force"]}`)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("StartProcess status = %d, want 403; body=%s", resp.StatusCode, body)
	}

	var denial struct {
		Error   string `json:"error"`
		Verb    string `json:"verb"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &denial); err != nil {
		t.Fatalf("decode denial body: %v", err)
	}
	if denial.Error != "git_verb_blocked" {
		t.Errorf("denial.Error = %q, want git_verb_blocked", denial.Error)
	}
	if denial.Verb != "push" {
		t.Errorf("denial.Verb = %q, want push", denial.Verb)
	}
	if denial.Message == "" {
		t.Error("denial.Message should be non-empty (operator-friendly text)")
	}
}

// TestStartProcess_AllowsListedGitVerb verifies the allowlist does NOT block
// `git status` under the default allowlist.
func TestStartProcess_AllowsListedGitVerb(t *testing.T) {
	id := uuid.New()
	sb := newProtectedSandboxFixture(id, types.DefaultProtectedGitAllowlist())
	live := newLive(t, protectedService(sb))

	resp, body := live.DoJSON(t, "POST", sandboxesPath(id, "/processes"),
		`{"command": "git", "args": ["status"]}`)

	// Should pass allowlist; the response code depends on the rest of
	// StartProcess (e.g., 201 on success or 5xx on driver issues). The
	// only thing this test asserts is that we did NOT return 403 for the
	// allowlist denial.
	if resp.StatusCode == http.StatusForbidden {
		t.Fatalf("StartProcess returned 403 for allowed verb 'status'; body=%s", body)
	}
}

// TestStartProcess_NonGitCommandBypassesAllowlist confirms the allowlist
// applies only to git invocations.
func TestStartProcess_NonGitCommandBypassesAllowlist(t *testing.T) {
	id := uuid.New()
	sb := newProtectedSandboxFixture(id, []string{"status"})
	live := newLive(t, protectedService(sb))

	resp, body := live.DoJSON(t, "POST", sandboxesPath(id, "/processes"),
		`{"command": "claude", "args": ["--print"]}`)
	if resp.StatusCode == http.StatusForbidden {
		t.Fatalf("StartProcess blocked non-git command 'claude' under git allowlist; body=%s", body)
	}
}

// TestStartProcess_EmptyAllowlistDoesNotEnforce confirms a sandbox with no
// configured allowlist (i.e. non-protected) lets all commands through.
func TestStartProcess_EmptyAllowlistDoesNotEnforce(t *testing.T) {
	id := uuid.New()
	sb := newProtectedSandboxFixture(id, nil)
	live := newLive(t, protectedService(sb))

	resp, body := live.DoJSON(t, "POST", sandboxesPath(id, "/processes"),
		`{"command": "git", "args": ["push"]}`)
	if resp.StatusCode == http.StatusForbidden {
		t.Fatalf("StartProcess returned 403 with empty allowlist; allowlist enforcement should be opt-in; body=%s", body)
	}
}

func TestStartProcess_BlocksDestructiveVrooliMaintenance(t *testing.T) {
	id := uuid.New()
	sb := newProtectedSandboxFixture(id, nil)
	live := newLive(t, protectedService(sb))

	resp, body := live.DoJSON(t, "POST", sandboxesPath(id, "/processes"),
		`{"command": "bash", "args": ["-lc", "vrooli cleanup orphans &> /tmp/o.txt; tail -10 /tmp/o.txt"]}`)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("StartProcess status = %d, want 403; body=%s", resp.StatusCode, body)
	}

	var denial struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &denial); err != nil {
		t.Fatalf("decode denial body: %v", err)
	}
	if denial.Error != runtime.VrooliPolicyDestructiveMaintenanceBlocked {
		t.Errorf("denial.Error = %q, want %q", denial.Error, runtime.VrooliPolicyDestructiveMaintenanceBlocked)
	}
	if denial.Message == "" {
		t.Error("denial.Message should be non-empty")
	}
}
