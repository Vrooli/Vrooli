package plans_test

import (
	"strings"
	"testing"

	"data-backup-manager/internal/plans"
)

// TestSchema_DeclaresPlansTables is the embed tripwire: it fails if the
// embedded schema.sql stops declaring the plans table or its membership tables.
func TestSchema_DeclaresPlansTables(t *testing.T) {
	sql := plans.Schema()
	for _, want := range []string{
		"CREATE TABLE IF NOT EXISTS plans",
		"plan_targets",
		"plan_destinations",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("schema.sql missing %q", want)
		}
	}
}
