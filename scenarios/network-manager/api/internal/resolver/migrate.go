package resolver

import (
	"context"
	"fmt"
)

// Migrate applies the one compatibility migration needed when the resolver
// credential field was renamed from token_ref to credential_ref. It runs
// before api-core/database.EnsureSchemas so an existing SQLite database can
// reach the current declarative schema without losing stored references.
// Fresh databases and already-migrated databases are no-ops.
func Migrate(ctx context.Context, db SQLExecutor) error {
	columns, err := tableColumns(ctx, db)
	if err != nil {
		return err
	}
	if _, hasCredentialRef := columns["credential_ref"]; hasCredentialRef {
		return nil
	}
	if _, hasTokenRef := columns["token_ref"]; !hasTokenRef {
		return nil
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE resolver_backends RENAME COLUMN token_ref TO credential_ref`); err != nil {
		return fmt.Errorf("rename resolver_backends.token_ref to credential_ref: %w", err)
	}
	return nil
}

func tableColumns(ctx context.Context, db SQLExecutor) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(resolver_backends)")
	if err != nil {
		return nil, fmt.Errorf("inspect resolver_backends columns: %w", err)
	}
	defer rows.Close()

	columns := map[string]struct{}{}
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal any
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &primaryKey); err != nil {
			return nil, fmt.Errorf("scan resolver_backends column: %w", err)
		}
		columns[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate resolver_backends columns: %w", err)
	}
	return columns, nil
}
