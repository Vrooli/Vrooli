package audits

import (
	"strings"
	"testing"
)

// TestSchema_Tripwire asserts the embedded SQL contains the expected table
// definition. If someone deletes or renames the table without updating this
// test, it fails — the embed tripwire per SEAMS.md.
func TestSchema_Tripwire(t *testing.T) {
	sql := Schema()
	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS audits") {
		t.Fatalf("schema.sql must contain 'CREATE TABLE IF NOT EXISTS audits'; got:\n%s", sql)
	}
}
