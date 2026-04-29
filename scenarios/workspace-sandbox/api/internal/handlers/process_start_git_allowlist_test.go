// Tests for the protected-mode git allowlist enforcement on the
// /sandboxes/{id}/processes (StartProcess) handler.
//
// The allowlist is enforced by Exec already; mirroring it on StartProcess
// closes a bypass: an agent process launched via /processes could otherwise
// invoke `git push` directly, sidestepping the /exec guard.
//
// See execute/protected-sandbox-agent-launch.

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/types"
)

func newProtectedSandboxFixture(id uuid.UUID, allowlist []string) *types.Sandbox {
	return &types.Sandbox{
		ID:            id,
		ScopePath:     "/project/src",
		ProjectRoot:   "/project",
		Owner:         "test-agent",
		Status:        types.StatusActive,
		Driver:        "overlayfs",
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

func newStartProcessHandlers(sb *types.Sandbox) *Handlers {
	return &Handlers{
		Service: &mockService{
			getFn: func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
				return sb, nil
			},
		},
		DB:            &mockPinger{},
		DriverSlot:    driver.NewSlot(&mockDriver{available: true}),
		Config:        config.Config{},
	}
}

func startProcessRequest(t *testing.T, sandboxID uuid.UUID, body string) (*Handlers, *httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	req := httptest.NewRequest("POST", "/sandboxes/"+sandboxID.String()+"/processes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": sandboxID.String()})
	rr := httptest.NewRecorder()
	return nil, rr, req
}

// TestStartProcess_BlocksDisallowedGitVerb verifies the allowlist intercepts
// `git push` (a mutating verb) under the default allowlist of read-only verbs.
func TestStartProcess_BlocksDisallowedGitVerb(t *testing.T) {
	id := uuid.New()
	sb := newProtectedSandboxFixture(id, types.DefaultProtectedGitAllowlist())
	h := newStartProcessHandlers(sb)

	body := `{"command": "git", "args": ["push", "--force"]}`
	_, rr, req := startProcessRequest(t, id, body)

	h.StartProcess(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("StartProcess() status = %d, want %d", rr.Code, http.StatusForbidden)
	}

	var denial struct {
		Error   string `json:"error"`
		Verb    string `json:"verb"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &denial); err != nil {
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
	h := newStartProcessHandlers(sb)

	body := `{"command": "git", "args": ["status"]}`
	_, rr, req := startProcessRequest(t, id, body)

	h.StartProcess(rr, req)

	// Should pass allowlist; the response code depends on the rest of
	// StartProcess (e.g., 201 on success or 5xx on driver issues). The
	// only thing this test asserts is that we did NOT return 403 for the
	// allowlist denial.
	if rr.Code == http.StatusForbidden {
		t.Fatalf("StartProcess() returned 403 for allowed verb 'status'; body=%s", rr.Body.String())
	}
}

// TestStartProcess_NonGitCommandBypassesAllowlist confirms the allowlist
// applies only to git invocations.
func TestStartProcess_NonGitCommandBypassesAllowlist(t *testing.T) {
	id := uuid.New()
	sb := newProtectedSandboxFixture(id, []string{"status"}) // strict allowlist
	h := newStartProcessHandlers(sb)

	body := `{"command": "claude", "args": ["--print"]}`
	_, rr, req := startProcessRequest(t, id, body)

	h.StartProcess(rr, req)

	if rr.Code == http.StatusForbidden {
		t.Fatalf("StartProcess() blocked non-git command 'claude' under git allowlist; body=%s", rr.Body.String())
	}
}

// TestStartProcess_EmptyAllowlistDoesNotEnforce confirms a sandbox with no
// configured allowlist (i.e. non-protected) lets all commands through.
func TestStartProcess_EmptyAllowlistDoesNotEnforce(t *testing.T) {
	id := uuid.New()
	sb := newProtectedSandboxFixture(id, nil)
	h := newStartProcessHandlers(sb)

	body := `{"command": "git", "args": ["push"]}`
	_, rr, req := startProcessRequest(t, id, body)

	h.StartProcess(rr, req)

	if rr.Code == http.StatusForbidden {
		t.Fatalf("StartProcess() returned 403 with empty allowlist; allowlist enforcement should be opt-in")
	}
}
