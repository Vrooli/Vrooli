package restores

import (
	"strings"
	"testing"
)

// TestSchema_Tripwire asserts the embedded SQL contains the expected table
// definition. If someone deletes or renames the table without updating this
// test, the test fails at compile-time via the embed directive or at runtime
// here. This is the embed tripwire per SEAMS.md.
func TestSchema_Tripwire(t *testing.T) {
	sql := Schema()
	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS restores") {
		t.Fatalf("schema.sql must contain 'CREATE TABLE IF NOT EXISTS restores'; got:\n%s", sql)
	}
}
