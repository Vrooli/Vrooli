package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// writeProjectScopeFixture builds a repository whose root manifest declares a
// host-owned credential and a host-owned safeguard, which is the case no
// scenario directory can own.
func writeProjectScopeFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	storageRoot := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "service.json"), `{
  "service": {"name": "vrooli", "description": "Project scope"},
  "credentials": {"descriptors": [
    {"logical_id": "vrooli/remote-desktop", "field": "username", "label": "Remote desktop username", "description": "Username consumed by the host remote-desktop provider.", "required": false},
    {"logical_id": "vrooli/remote-desktop", "field": "password", "label": "Remote desktop password", "description": "Password consumed by the host remote-desktop provider.", "required": false}
  ]},
  "hostTools": [{"name": "jq", "required": true, "reason": "JSON parsing"}],
  "hostSafeguards": [{"name": "workspace_sandbox_userns", "required": true, "reason": "Workspace sandbox restart"}]
}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{
  "service": {"name": "alpha", "description": "Alpha", "system_required": true},
  "hostTools": [{"name": "tmux", "required": true, "reason": "session support"}]
}`)
	writeFixtureFile(t, filepath.Join(root, "internal", "tools", "jq", "tool.json"), `{"name":"jq","description":"JSON parser","commands":["jq"]}`)
	writeFixtureFile(t, filepath.Join(root, "internal", "tools", "tmux", "tool.json"), `{"name":"tmux","description":"terminal multiplexer","commands":["tmux"]}`)
	writeFixtureFile(t, filepath.Join(root, "internal", "safeguards", "workspace_sandbox_userns", "safeguard.json"), `{"name":"workspace_sandbox_userns","description":"Workspace sandbox user namespaces","privilege":"elevated","verificationCheck":{"files":["/etc/apparmor.d/vrooli-workspace-sandbox"]}}`)
	writeFixtureFile(t, filepath.Join(storageRoot, "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-25T00:00:00Z","scenarios":{"alpha":{"enabled":true}}}`)
	if err := os.MkdirAll(filepath.Join(root, "resources"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("VROOLI_STORAGE_ROOT", storageRoot)
	stubExternalReadinessProbes(t)
	return root, storageRoot
}

func credentialAddresses(t *testing.T, body []byte) map[string]credentialReadiness {
	t.Helper()
	var payload struct {
		Credentials []credentialReadiness `json:"credentials"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	byAddress := make(map[string]credentialReadiness, len(payload.Credentials))
	for _, credential := range payload.Credentials {
		byAddress[credential.LogicalID+":"+credential.Field] = credential
	}
	return byAddress
}

// [REQ:ONB-CRED-PROJECT-SCOPE]
// A host-owned credential has no scenario directory, so the project manifest is
// its only possible owner. If the wizard cannot show it, the operator has to
// leave the flow to provision it, which is the defect this asserts against.
func TestV2CredentialsIncludeProjectScope(t *testing.T) {
	writeProjectScopeFixture(t)
	w := doGet(t, NewServer(), "/api/v2/credentials")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	byAddress := credentialAddresses(t, w.Body.Bytes())
	for _, field := range []string{"username", "password"} {
		credential, found := byAddress["vrooli/remote-desktop:"+field]
		if !found {
			t.Fatalf("credential surface is missing vrooli/remote-desktop:%s: %s", field, w.Body.String())
		}
		if credential.Resource != "project" {
			t.Fatalf("owner = %q, want project", credential.Resource)
		}
		if credential.Label == "" || credential.Description == "" {
			t.Fatalf("declared label and description did not reach the card: %+v", credential)
		}
		if credential.Required {
			t.Fatalf("required = true, want the declared false for %s", field)
		}
	}
}

// [REQ:ONB-CRED-PROJECT-SCOPE]
func TestV2ReadinessIncludesProjectScope(t *testing.T) {
	writeProjectScopeFixture(t)
	w := doGet(t, NewServer(), "/api/v2/readiness")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	byAddress := credentialAddresses(t, w.Body.Bytes())
	if _, found := byAddress["vrooli/remote-desktop:username"]; !found {
		t.Fatalf("readiness credentials are missing the project scope: %s", w.Body.String())
	}
}

// [REQ:ONB-TIER-BUNDLE-COMPLETENESS]
// A desktop bundle stages no repository-root manifest by design, so it must
// inherit no project-scope credential. Asserting it keeps the exclusion from
// being widened silently by a later packaging change.
func TestBundleModeExcludesProjectScopeCredentials(t *testing.T) {
	bundle, storageRoot := writeBundleFixture(t, true)
	writeFixtureFile(t, filepath.Join(bundle, "catalog", ".vrooli", "service.json"), `{
  "service": {"name": "vrooli", "description": "Project scope that must not be inherited"},
  "credentials": {"descriptors": [{"logical_id": "vrooli/remote-desktop", "field": "username", "required": false}]}
}`)
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("BUNDLE_ROOT", bundle)
	t.Setenv("VROOLI_STORAGE_ROOT", storageRoot)
	stubExternalReadinessProbes(t)

	for _, endpoint := range []string{"/api/v2/credentials", "/api/v2/readiness"} {
		w := doGet(t, NewServer(), endpoint)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d: %s", endpoint, w.Code, w.Body.String())
		}
		if _, found := credentialAddresses(t, w.Body.Bytes())["vrooli/remote-desktop:username"]; found {
			t.Fatalf("GET %s inherited a project-scope credential in bundle mode: %s", endpoint, w.Body.String())
		}
	}
}
