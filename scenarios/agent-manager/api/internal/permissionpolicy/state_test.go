package permissionpolicy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const validCatalogJSON = `{
  "schemaVersion": 1,
  "metadata": {"catalogId": "test", "updatedAt": "2026-07-10"},
  "targetScopes": ["user"],
  "rules": [{
    "id": "deny-root",
    "action": "deny",
    "matcher": {"kind": "bash", "pattern": "rm -rf /"},
    "rationale": "Protect the root filesystem.",
    "owner": "test-owner",
    "targetScope": "user",
    "requiresHardEnforcement": false
  }]
}`

func TestStateRetainsActiveRevisionAfterInvalidReload(t *testing.T) {
	path := writeCatalog(t, validCatalogJSON)
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	state, err := newState(path, Requirement{Required: true, Reason: "portable global permissions"}, func() time.Time { return now })
	if err != nil {
		t.Fatalf("newState: %v", err)
	}
	active := state.Active()
	if active == nil {
		t.Fatal("active revision is nil")
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"metadata":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := state.Reload(); err == nil {
		t.Fatal("Reload succeeded for invalid catalog")
	}
	if got := state.Active(); got == nil || got.Digest() != active.Digest() {
		t.Fatalf("active revision changed after failed reload: %#v", got)
	}
	status := state.Status()
	if !status.Ready || status.LastReloadAttempt == nil || status.LastReloadAttempt.Succeeded {
		t.Fatalf("status = %#v", status)
	}
}

func TestParseIsStrictAndUsesExactBytesForDigest(t *testing.T) {
	first, err := Parse([]byte(validCatalogJSON))
	if err != nil {
		t.Fatalf("Parse first catalog: %v", err)
	}
	second, err := Parse([]byte("\n" + validCatalogJSON))
	if err != nil {
		t.Fatalf("Parse second catalog: %v", err)
	}
	if first.Digest() == second.Digest() {
		t.Fatalf("digest must identify exact declared bytes: %q", first.Digest())
	}
	if _, err := Parse([]byte(strings.Replace(validCatalogJSON, `"rules": [`, `"unexpected": true, "rules": [`, 1))); err == nil {
		t.Fatal("Parse accepted unknown field")
	}
}

func TestCatalogRequiresAuditablePortableRules(t *testing.T) {
	for name, mutate := range map[string]func(string) string{
		"missing rationale": func(value string) string {
			return strings.Replace(value, `"rationale": "Protect the root filesystem.",`, "", 1)
		},
		"missing owner": func(value string) string { return strings.Replace(value, `"owner": "test-owner",`, "", 1) },
		"invalid scope": func(value string) string {
			return strings.Replace(value, `"targetScope": "user"`, `"targetScope": "workspace"`, 1)
		},
		"unsupported matcher": func(value string) string {
			return strings.Replace(value, `"kind": "bash"`, `"kind": "tool"`, 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(mutate(validCatalogJSON))); err == nil {
				t.Fatal("Parse accepted invalid permission catalog")
			}
		})
	}
}

func TestResourceDocumentIsScopedAndExcludesAuditFields(t *testing.T) {
	revision, err := Parse([]byte(validCatalogJSON))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	document, err := revision.Catalog().ResourceDocument("user")
	if err != nil {
		t.Fatalf("ResourceDocument: %v", err)
	}
	if document.Scope != "user" || len(document.Rules) != 1 || document.Rules[0].ID != "deny-root" {
		t.Fatalf("document = %#v", document)
	}
	if _, err := revision.Catalog().ResourceDocument("workspace"); err == nil {
		t.Fatal("ResourceDocument accepted unsupported scope")
	}
}

func TestRepositoryCatalogIsStrict(t *testing.T) {
	revision, err := Load(ResolvePath())
	if err != nil {
		t.Fatalf("Load repository permission catalog: %v", err)
	}
	catalog := revision.Catalog()
	if catalog.Metadata.CatalogID != "agent-manager-coding-permission-policy" || len(catalog.Rules) != 1 {
		t.Fatalf("catalog = %#v", catalog)
	}
}

func TestCatalogAllowsAnExplicitEmptyManagedScope(t *testing.T) {
	revision, err := Parse([]byte(`{
  "schemaVersion": 1,
  "metadata": {"catalogId": "empty", "updatedAt": "2026-07-10"},
  "targetScopes": ["user"],
  "rules": []
}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	document, err := revision.Catalog().ResourceDocument("user")
	if err != nil || len(document.Rules) != 0 {
		t.Fatalf("document = %#v, err = %v", document, err)
	}
}

func writeCatalog(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "permission-policy.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
