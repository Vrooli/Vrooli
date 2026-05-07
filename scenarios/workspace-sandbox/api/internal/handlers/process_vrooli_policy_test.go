package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"workspace-sandbox/internal/runtime"
)

func TestExec_BlocksDestructiveVrooliMaintenance(t *testing.T) {
	id := uuid.New()
	sb := newProtectedSandboxFixture(id, nil)
	live := newLive(t, protectedService(sb))

	resp, body := live.DoJSON(t, "POST", sandboxesPath(id, "/exec"),
		`{"command": "/usr/bin/env", "args": ["vrooli", "cleanup", "locks"]}`)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Exec status = %d, want 403; body=%s", resp.StatusCode, body)
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
