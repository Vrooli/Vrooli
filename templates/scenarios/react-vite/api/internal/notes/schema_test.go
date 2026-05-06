package notes

import (
	"strings"
	"testing"
)

// TestSchema_NotEmpty is a tripwire for go:embed silently dropping the
// file (e.g., wrong path, missing build tag). Real apply-and-query
// coverage lives in sqlite_test.go.
func TestSchema_NotEmpty(t *testing.T) {
	if Schema() == "" {
		t.Fatal("notes.Schema() returned empty; check go:embed wiring")
	}
}

// TestSchema_ContainsNotesTable pins the embedded contents to the
// canonical CRUD reference table; if a future edit removes the table
// definition this catches it before sqlite_test runs.
func TestSchema_ContainsNotesTable(t *testing.T) {
	if !strings.Contains(Schema(), "CREATE TABLE IF NOT EXISTS notes") {
		t.Fatalf("notes.Schema() missing notes table; got: %s", Schema())
	}
}

func TestSchema_ContainsAttachmentsTable(t *testing.T) {
	if !strings.Contains(Schema(), "CREATE TABLE IF NOT EXISTS attachments") {
		t.Fatalf("notes.Schema() missing attachments table; got: %s", Schema())
	}
}
