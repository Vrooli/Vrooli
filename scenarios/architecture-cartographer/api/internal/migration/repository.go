package migration

import "context"

// Repository is the persistence seam for the migration tracker. Production
// wires the SQLite implementation; tests substitute a fake or a real
// in-temp-dir SQLite handle.
type Repository interface {
	CreateMigration(ctx context.Context, m Migration) error
	GetMigration(ctx context.Context, id string) (Migration, error)
	UpdateMigrationStatus(ctx context.Context, id string, status MigrationStatus) error

	// UpsertFinding inserts or updates a tracked finding keyed by
	// (migration_id, stable_id). first_seen_at is preserved on update.
	UpsertFinding(ctx context.Context, migrationID string, f Finding) error
	GetFinding(ctx context.Context, migrationID, stableID string) (Finding, error)
	ListFindings(ctx context.Context, migrationID string) ([]Finding, error)
}
