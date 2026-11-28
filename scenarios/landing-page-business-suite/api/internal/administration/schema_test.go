package administration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchemaOwnsAdministrationRuntimeTables(t *testing.T) {
	sql := strings.ToLower(Schema())
	for _, table := range []string{"admin_users", "admin_sessions", "remote_profiles"} {
		if !strings.Contains(sql, "create table if not exists "+table) {
			t.Errorf("missing %s", table)
		}
	}
}

func TestLegacyEmailTransportConstraintChangeLivesOutsideRuntimeSchema(t *testing.T) {
	runtime := strings.ToLower(Schema())
	if strings.Contains(runtime, "alter table") || strings.Contains(runtime, "drop table") {
		t.Fatal("administration runtime schema must not perform migrations")
	}
	migrationPath := filepath.Join("..", "..", "operator-migrations", "20261119_email_provider_transport_registry.sql")
	migration, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read operator migration: %v", err)
	}
	statement := strings.ToLower(string(migration))
	if !strings.Contains(statement, "alter table email_providers drop constraint if exists email_providers_transport_check") {
		t.Fatal("one-shot operator migration does not remove the legacy transport allowlist")
	}
}
