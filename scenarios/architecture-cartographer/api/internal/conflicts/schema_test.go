package conflicts

import (
	"strings"
	"testing"
)

func TestSchema_NotEmpty(t *testing.T) {
	if Schema() == "" {
		t.Fatal("conflicts.Schema() returned empty; check go:embed wiring")
	}
}

func TestSchema_ContainsConflictsTable(t *testing.T) {
	if !strings.Contains(Schema(), "CREATE TABLE IF NOT EXISTS conflicts") {
		t.Fatalf("conflicts.Schema() missing conflicts table")
	}
}
