package targets_test

import (
	"strings"
	"testing"

	"data-backup-manager/internal/targets"
)

// TestSchema_DeclaresTargetsTable is the embed tripwire: it fails if the
// embedded schema.sql stops declaring the targets table or its (owner, name)
// uniqueness, which the idempotent-registration model depends on.
func TestSchema_DeclaresTargetsTable(t *testing.T) {
	sql := targets.Schema()
	for _, want := range []string{
		"CREATE TABLE IF NOT EXISTS targets",
		"UNIQUE (owner, name)",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("schema.sql missing %q", want)
		}
	}
}
