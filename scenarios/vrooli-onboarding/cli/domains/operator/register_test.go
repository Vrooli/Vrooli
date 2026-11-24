package operator

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	clitest "github.com/vrooli/cli-core/cliapptest"
	operatorstatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate"
	localtest "vrooli-onboarding/cli/internal/testutil"
)

func TestRegisterAndPatchValidation(t *testing.T) {
	group := Register(&cliapp.ScenarioApp{})
	if group.Name != "operator" || len(group.Subcommands) != 5 {
		t.Fatalf("unexpected operator group: %+v", group)
	}
	if err := group.Subcommands[1].Run([]string{"--body-file", "missing.json"}); err == nil {
		t.Fatal("missing patch file should fail")
	}
	if err := group.Subcommands[2].Run([]string{"--mode", "own"}); err == nil {
		t.Fatal("reserved own mode must be refused")
	}
}

func TestGenericOperatorPatchRejectsPostureFieldsBeforeAnyRequest(t *testing.T) {
	for _, field := range []string{"trust_posture", "update_control", "session_mode"} {
		t.Run(field, func(t *testing.T) {
			if _, err := applyOperatorPatch(nil, new(operatorstatev1.OperatorState), map[string]json.RawMessage{field: json.RawMessage(`"guard"`)}); err == nil {
				t.Fatalf("generic patch accepted owner-managed %s", field)
			}
		})
	}
}

func TestCommandsUseV2ReadAndPatchRoutes(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		if r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ready","scenarios":[],"host_safeguards":{}}`))
	}))
	group := Register(core)
	if err := group.Subcommands[0].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	patchPath := filepath.Join(t.TempDir(), "patch.json")
	if err := localtest.WriteJSON(patchPath, map[string]any{"scenarios": map[string]any{"demo": map[string]any{"enabled": true}}}); err != nil {
		t.Fatal(err)
	}
	if err := group.Subcommands[1].Run([]string{"--body-file", patchPath, "--json"}); err != nil {
		t.Fatal(err)
	}
	if err := group.Subcommands[3].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	if err := group.Subcommands[4].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateControlUsesFieldPermissionReadAndWrite(t *testing.T) {
	var paths []string
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		switch r.URL.Path {
		case "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/ReadPermission":
			if request["permissionId"] != "onboarding/install-posture/update-control" || request["owner"] != "vrooli-onboarding" {
				t.Errorf("typed read request=%v", request)
			}
			_, _ = w.Write([]byte(`{"revision":"update-control-r4"}`))
		case "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/WritePermission":
			if request["permissionId"] != "onboarding/install-posture/update-control" || request["expectedRevision"] != "update-control-r4" {
				t.Errorf("typed write binding=%v", request)
			}
			encoded, _ := request["valueJson"].(string)
			value, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil || string(value) != `{"value":"guard"}` {
				t.Errorf("typed update-control payload=%q err=%v", value, err)
			}
			_, _ = w.Write([]byte(`{"receipt":{"id":"receipt-1","resultRevision":"update-control-r5"}}`))
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
		}
	}))
	group := Register(core)
	if err := group.Subcommands[2].Run([]string{"--mode", "guard", "--json"}); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || paths[0] != "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/ReadPermission" || paths[1] != "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/WritePermission" {
		t.Fatalf("update-control permission calls=%v", paths)
	}
}
