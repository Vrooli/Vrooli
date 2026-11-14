package content

import (
	"strings"
	"testing"
)

func TestSchemaOwnsContentRuntimeTables(t *testing.T) {
	sql := strings.ToLower(Schema())
	for _, table := range []string{"assets", "feedback_requests", "waitlist_emails"} {
		if !strings.Contains(sql, "create table if not exists "+table) {
			t.Errorf("missing %s", table)
		}
	}
}
