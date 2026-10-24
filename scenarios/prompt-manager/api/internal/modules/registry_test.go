package modules

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func TestAllSchemasApplyIdempotently(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "schemas.db")+"?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for i := 0; i < 2; i++ {
		if err := database.EnsureSchemas(context.Background(), db, AllSchemas()...); err != nil {
			t.Fatalf("ensure schemas pass %d: %v", i+1, err)
		}
	}
}

func TestSystemSchemaHasNoTables(t *testing.T) {
	schemas := AllSchemas()
	if len(schemas) == 0 {
		t.Fatal("expected schemas")
	}
	system := strings.ToUpper(schemas[0].Schema())
	if strings.Contains(system, "CREATE TABLE") {
		t.Fatalf("system schema should stay cross-cutting and table-free by default:\n%s", schemas[0].Schema())
	}
}
