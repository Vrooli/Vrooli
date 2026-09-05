package rolepolicy

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateRetainsActiveRevisionAfterInvalidReload(t *testing.T) {
	path := writeCatalog(t, validCatalogJSON)
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	state, err := newState(path, Requirement{Required: true, Reason: "portable roles"}, func() time.Time { return now })
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

func TestCatalogRejectsModelInventoryAndUnknownFields(t *testing.T) {
	_, err := Parse([]byte(`{
  "schemaVersion":1,
  "metadata":{"catalogId":"test","updatedAt":"2026-07-10"},
  "defaultRole":"code.default",
  "runners":{},
  "roles":{"code.default":{"description":"test","intent":"test","candidates":[{"runner":"codex","resourceRole":"code.default"}]}}
}`))
	if err == nil {
		t.Fatal("Parse accepted Agent Manager model inventory")
	}
}

func TestRepositoryCatalogIsStrictAndModelFree(t *testing.T) {
	revision, err := Load(ResolvePath())
	if err != nil {
		t.Fatalf("Load repository role catalog: %v", err)
	}
	catalog := revision.Catalog()
	if catalog.DefaultRole != "code.default" || len(catalog.Roles) != 9 {
		t.Fatalf("catalog = %#v", catalog)
	}
	if _, exists := catalog.Roles["extract.structured"]; exists {
		t.Fatal("agent-manager still owns the typed inference role extract.structured")
	}
}

func writeCatalog(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "role-policy.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

const validCatalogJSON = `{
  "schemaVersion":1,
  "metadata":{"catalogId":"test","updatedAt":"2026-07-10"},
  "defaultRole":"code.default",
  "roles":{"code.default":{"description":"test","intent":"test","candidates":[{"runner":"codex","resourceRole":"code.default"},{"runner":"claude-code","resourceRole":"code.default"}]}}
}`
