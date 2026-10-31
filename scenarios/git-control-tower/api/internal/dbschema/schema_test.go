package dbschema

import (
	"strings"
	"testing"
)

func TestSchemaContainsAuditAndRepositoryTables(t *testing.T) {
	schema := Schema()
	for _, table := range []string{"git_audit_log", "git_repos"} {
		if !strings.Contains(schema, table) {
			t.Fatalf("schema is missing %q", table)
		}
	}
}
