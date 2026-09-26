package evidence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Migrate applies additive receipt columns that cannot be expressed as an
// idempotent CREATE TABLE statement. The schema bootstrap creates the column
// for new databases; this path keeps existing scenario databases readable.
func Migrate(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}) error {
	for _, column := range []string{"producer_execution_id TEXT NOT NULL DEFAULT ''", "final_url TEXT NOT NULL DEFAULT ''", "redirect_urls TEXT NOT NULL DEFAULT '[]'", "etag TEXT NOT NULL DEFAULT ''", "last_modified TEXT NOT NULL DEFAULT ''"} {
		_, err := db.ExecContext(ctx, `ALTER TABLE evidence_receipts ADD COLUMN `+column)
		if err == nil {
			continue
		}
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "duplicate column") || strings.Contains(lower, "already exists") {
			continue
		}
		return fmt.Errorf("add evidence receipt column: %w", err)
	}
	return nil
}
