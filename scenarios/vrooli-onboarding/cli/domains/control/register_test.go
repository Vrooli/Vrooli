package control

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

func TestControlCommandsReadApplyAndExport(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/ReadPermission" {
			var request struct {
				PermissionID string `json:"permissionId"`
				Owner        string `json:"owner"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"permissionId": request.PermissionID, "owner": request.Owner, "revision": "r1"})
			return
		}
		if r.URL.Path == "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/WritePermission" {
			var request struct {
				PermissionID     string `json:"permissionId"`
				Owner            string `json:"owner"`
				ExpectedRevision string `json:"expectedRevision"`
				IdempotencyKey   string `json:"idempotencyKey"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"receipt": map[string]any{
				"permissionId": request.PermissionID, "owner": request.Owner, "actorId": "operator-1",
				"expectedRevision": request.ExpectedRevision, "resultRevision": "r2",
				"idempotencyKey": request.IdempotencyKey, "outcome": "committed",
			}})
			return
		}
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		if r.Method == http.MethodPatch || r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		_, _ = w.Write([]byte(`{"scenarios":[],"tools":[],"safeguards":[]}`))
	}))
	groups := CommandGroups(core)
	if err := groups[0].Commands[0].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	subs := SubcommandGroups(core)
	if err := subs[0].Subcommands[0].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	if err := subs[2].Subcommands[0].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	if err := subs[2].Subcommands[1].Run([]string{"--name", "safe", "--key", "mode", "--value-json", `"strict"`}); err != nil {
		t.Fatal(err)
	}
	if err := subs[2].Subcommands[2].Run([]string{"--subject", "operator@example.test"}); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "union.json")
	if err := subs[1].Subcommands[0].Run([]string{"--output", output}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatal(err)
	}
}

func TestControlRejectsIncompleteSafeguardConfig(t *testing.T) {
	if err := setConfig(nil, nil); err == nil {
		t.Fatal("incomplete safeguard config should fail before using the API")
	}
}
