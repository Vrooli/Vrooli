package operations

import (
	"strings"
	"testing"
)

func TestSchemaOwnsOperationsRuntimeTables(t *testing.T) {
	sql := strings.ToLower(Schema())
	for _, table := range []string{"usage_records", "credit_reservations", "api_keys", "users", "auth_tokens", "user_sessions"} {
		if !strings.Contains(sql, "create table if not exists "+table) {
			t.Errorf("missing %s", table)
		}
	}
}
