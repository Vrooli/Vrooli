package commerce

import (
	"strings"
	"testing"
)

func TestOperationsSchemaOwnsCommerceRuntimeTables(t *testing.T) {
	sql := strings.ToLower(OperationsSchema())
	for _, table := range []string{"usage_records", "usage_events", "credit_reservations", "api_keys", "users", "auth_tokens", "user_sessions"} {
		if !strings.Contains(sql, "create table if not exists "+table) {
			t.Errorf("missing %s", table)
		}
	}
	for _, column := range []string{"cost_micros", "prompt_tokens", "completion_tokens", "provider"} {
		if !strings.Contains(sql, column) {
			t.Errorf("usage_events schema missing provider cost column %s", column)
		}
	}
}
