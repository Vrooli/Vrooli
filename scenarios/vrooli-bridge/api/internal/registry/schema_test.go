package registry

import (
	"strings"
	"testing"
)

// TestSchema_EmbedsNodesTable is the embed-content tripwire: it proves schema.go
// embeds schema.sql (not an empty string) and that the nodes table DDL is
// present, so a botched embed fails here rather than at boot.
func TestSchema_EmbedsNodesTable(t *testing.T) {
	s := Schema()
	if !strings.Contains(s, "CREATE TABLE IF NOT EXISTS nodes") {
		t.Fatalf("schema does not declare the nodes table:\n%s", s)
	}
	if !strings.Contains(s, "revoked_at") {
		t.Fatalf("schema missing revoked_at column:\n%s", s)
	}
}
