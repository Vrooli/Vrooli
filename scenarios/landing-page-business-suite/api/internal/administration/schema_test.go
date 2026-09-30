package administration

import (
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
