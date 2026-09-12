package modules

import (
	"os"
	"regexp"
	"strings"
	"testing"

	localdb "knowledge-observatory/internal/database"
)

// TestSystemSchemaDeclaresNoDomainObjects is the tripwire required by
// storage-steer §4.1.
//
// A domain table declared in a central file can never be removed by deleting
// the domain's folder: the definition survives, the table is orphaned, and it
// is recreated on every boot. This test fails the moment anyone adds a table or
// an index to the system home instead of to internal/<domain>/schema.sql.
func TestSystemSchemaDeclaresNoDomainObjects(t *testing.T) {
	sql := localdb.SystemSchema()

	forbidden := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"CREATE TABLE", regexp.MustCompile(`(?i)\bCREATE\s+TABLE\b`)},
		{"CREATE INDEX", regexp.MustCompile(`(?i)\bCREATE\s+(UNIQUE\s+)?INDEX\b`)},
		{"CREATE VIEW", regexp.MustCompile(`(?i)\bCREATE\s+(OR\s+REPLACE\s+)?VIEW\b`)},
	}
	for _, f := range forbidden {
		if f.pattern.MatchString(sql) {
			t.Errorf("system.sql must declare no domain objects, found %s. "+
				"Move it to internal/<domain>/schema.sql so the domain stays deletable.", f.name)
		}
	}
}

// TestNoCentralSchemaFileSurvives proves the central schema file is gone. Its
// existence was the deletability bug this restructure removed; a reintroduced
// copy would silently take ownership back.
func TestNoCentralSchemaFileSurvives(t *testing.T) {
	for _, path := range []string{"../../../api/schema.sql", "../../../schema.sql"} {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("central schema file %s must not exist; every table is owned by a domain", path)
		}
	}
}

// TestAllSchemasAreNonEmptyAndIdempotent asserts every registered domain
// actually ships DDL, and that the DDL is safe to apply on every boot.
//
// EnsureSchemas runs unconditionally at startup, so a bare CREATE TABLE would
// fail the second boot. ALTER TABLE is forbidden outright (storage-steer §4.1):
// it is a syntax error on SQLite and a migration concern, not a schema concern.
func TestAllSchemasAreNonEmptyAndIdempotent(t *testing.T) {
	providers := AllSchemas()
	if len(providers) < 2 {
		t.Fatalf("expected the system home plus every domain, got %d providers", len(providers))
	}

	createTable := regexp.MustCompile(`(?i)\bCREATE\s+TABLE\b`)
	guardedTable := regexp.MustCompile(`(?i)\bCREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\b`)
	createIndex := regexp.MustCompile(`(?i)\bCREATE\s+(UNIQUE\s+)?INDEX\b`)
	guardedIndex := regexp.MustCompile(`(?i)\bCREATE\s+(UNIQUE\s+)?INDEX\s+IF\s+NOT\s+EXISTS\b`)
	alter := regexp.MustCompile(`(?i)\bALTER\s+TABLE\b`)

	for i, p := range providers {
		sql := p.Schema()
		// The system home is allowed to be empty; a domain is not.
		if i > 0 && strings.TrimSpace(sql) == "" {
			t.Errorf("provider %d declares no schema", i)
			continue
		}
		if alter.MatchString(sql) {
			t.Errorf("provider %d contains ALTER TABLE; schema.sql declares state, migrations change it", i)
		}
		if n := len(createTable.FindAllString(sql, -1)); n != len(guardedTable.FindAllString(sql, -1)) {
			t.Errorf("provider %d has an unguarded CREATE TABLE; EnsureSchemas runs on every boot", i)
		}
		if n := len(createIndex.FindAllString(sql, -1)); n != len(guardedIndex.FindAllString(sql, -1)) {
			t.Errorf("provider %d has an unguarded CREATE INDEX; EnsureSchemas runs on every boot", i)
		}
	}
}
