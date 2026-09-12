package bootstrap

import (
	"context"
	"fmt"

	localdb "audio-tools/internal/database"
	"audio-tools/internal/modules"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"

	// modernc.org/sqlite registers the pure-Go sqlite driver under the
	// "sqlite" name; api-core's database.Connect resolves it by driver name.
	_ "modernc.org/sqlite"
)

// OpenDB resolves the on-disk SQLite path, opens it via api-core's
// database.Connect, ensures the audio-tools schemas, and returns the
// live handle plus the resolved DSN for diagnostics.
func OpenDB(ctx context.Context, env Env) (*database.RoutedDB, string, error) {
	dsn, err := resolveDSN(env)
	if err != nil {
		return nil, "", err
	}
	db, err := database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return nil, dsn, fmt.Errorf("connect sqlite: %w", err)
	}
	// Apply additive repairs before EnsureSchemas performs its declared-shape
	// drift check. Otherwise an existing table missing a newly declared column
	// fails startup before the forward migration gets a chance to add it.
	if err := localdb.ApplyMigrations(ctx, db.Primary()); err != nil {
		_ = db.Close()
		return nil, dsn, fmt.Errorf("pre-schema migrations: %w", err)
	}
	if err := database.EnsureSchemas(ctx, db.Primary(), modules.AllSchemas()...); err != nil {
		_ = db.Close()
		return nil, dsn, fmt.Errorf("ensure schemas: %w", err)
	}
	// Repeat after schema creation so a fresh database sees the same completed
	// migration set. Duplicate-column outcomes are deliberately idempotent.
	if err := localdb.ApplyMigrations(ctx, db.Primary()); err != nil {
		_ = db.Close()
		return nil, dsn, fmt.Errorf("apply migrations: %w", err)
	}
	return db, dsn, nil
}

// resolveDSN returns the DSN for audio-tools' own database, resolved from the
// scenario's identity rather than from the environment.
func resolveDSN(Env) (string, error) {
	return storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "audio-tools"})
}
