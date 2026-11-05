package destinations_test

import (
	"strings"
	"testing"

	"data-backup-manager/internal/destinations"
)

// TestSchema_DeclaresDestinationsTable is the embed tripwire: it fails if the
// embedded schema.sql stops declaring the destinations table, which the domain
// model depends on.
func TestSchema_DeclaresDestinationsTable(t *testing.T) {
	sql := destinations.Schema()
	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS destinations") {
		t.Error("schema.sql missing \"CREATE TABLE IF NOT EXISTS destinations\"")
	}
}
