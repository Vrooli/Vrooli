package database

import (
	"regexp"
	"strings"
	"testing"
)

// TestSystemSchemaIsEmpty is the storage-steer §4.1 tripwire at its source.
//
// system.sql holds only comments explaining why it is empty. Comments are not
// statements, so SystemSchema must return "" and let EnsureSchemas skip the
// provider entirely rather than hand a statement-free string to the driver.
func TestSystemSchemaIsEmpty(t *testing.T) {
	if got := SystemSchema(); got != "" {
		t.Errorf("SystemSchema() returned %d bytes; the system home is expected to be empty.\n"+
			"If you genuinely added a cross-cutting object, update this test and say why here.\n"+
			"Domain tables never belong in system.sql — put them in internal/<domain>/schema.sql "+
			"so the domain stays deletable.", len(got))
	}
}

// TestSystemSQLDeclaresNoDomainObjects guards the raw file, not just the
// accessor, so a table added below a comment cannot slip through.
func TestSystemSQLDeclaresNoDomainObjects(t *testing.T) {
	for _, forbidden := range []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"CREATE TABLE", regexp.MustCompile(`(?i)\bCREATE\s+TABLE\b`)},
		{"CREATE INDEX", regexp.MustCompile(`(?i)\bCREATE\s+(UNIQUE\s+)?INDEX\b`)},
		{"CREATE VIEW", regexp.MustCompile(`(?i)\bCREATE\s+(OR\s+REPLACE\s+)?VIEW\b`)},
		{"CREATE TRIGGER", regexp.MustCompile(`(?i)\bCREATE\s+TRIGGER\b`)},
	} {
		if forbidden.pattern.MatchString(systemSQL) {
			t.Errorf("system.sql must declare no domain objects, found %s", forbidden.name)
		}
	}
}

func TestContainsStatement(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty", "", false},
		{"blank lines", "\n\n   \n", false},
		{"comments only", "-- a comment\n\n-- another\n", false},
		{"indented comment", "    -- indented\n", false},
		{"a statement", "-- lead-in\nSELECT 1;\n", true},
		{"statement after blanks", "\n\nCREATE TABLE t (a INT);", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := containsStatement(tc.input); got != tc.want {
				t.Errorf("containsStatement(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// TestSystemSQLExplainsItself keeps the file from decaying into a blank that a
// future reader mistakes for an oversight and "helpfully" fills in.
func TestSystemSQLExplainsItself(t *testing.T) {
	if !strings.Contains(systemSQL, "internal/<domain>/schema.sql") {
		t.Error("system.sql should point readers at where domain tables do belong")
	}
}
