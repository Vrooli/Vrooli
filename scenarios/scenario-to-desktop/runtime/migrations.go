package bundleruntime

import (
	"context"

	"scenario-to-desktop-runtime/manifest"
)

// runMigrations executes pending migrations for a service.
// Delegates to the migrations.Executor.
func (s *Supervisor) runMigrations(ctx context.Context, svc manifest.Service, bin manifest.Binary, baseEnv map[string]string) error {
	if s.migrationExecutor == nil {
		return nil
	}
	err := s.migrationExecutor.Run(ctx, svc, bin, baseEnv)
	if err != nil {
		return err
	}
	// Sync state back to supervisor for compatibility.
	s.migrations = s.migrationExecutor.State()
	return nil
}
