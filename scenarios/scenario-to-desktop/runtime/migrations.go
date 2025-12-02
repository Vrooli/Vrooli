package bundleruntime

import (
	"context"
	"fmt"

	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/migrations"
	"scenario-to-desktop-runtime/strutil"
)

// migrationsTracker returns a migrations tracker instance.
func (s *Supervisor) migrationsTracker() *migrations.Tracker {
	return migrations.NewTracker(s.migrationsPath, s.fs)
}

// loadMigrations reads the migrations state from disk.
// Delegates to migrations.Tracker.
func (s *Supervisor) loadMigrations() (MigrationsState, error) {
	tracker := s.migrationsTracker()
	state, err := tracker.Load()
	if err != nil {
		return migrations.NewState(), err
	}
	return state, nil
}

// persistMigrations saves the migrations state to disk.
// Delegates to migrations.Tracker.
func (s *Supervisor) persistMigrations(state MigrationsState) error {
	tracker := s.migrationsTracker()
	return tracker.Persist(state)
}

// runMigrations executes pending migrations for a service.
// Migrations are run based on their run_on condition: always, first_install, or upgrade.
func (s *Supervisor) runMigrations(ctx context.Context, svc manifest.Service, bin manifest.Binary, baseEnv map[string]string) error {
	if len(svc.Migrations) == 0 {
		return s.ensureAppVersionRecorded()
	}

	phase := migrations.Phase(s.migrations, s.opts.Manifest.App.Version)

	state := s.migrations
	appliedSet := migrations.BuildAppliedSet(state.Applied[svc.ID])
	envBase := strutil.CopyStringMap(baseEnv)

	for _, m := range svc.Migrations {
		if err := s.maybeRunMigration(ctx, svc, m, bin, envBase, phase, appliedSet, &state); err != nil {
			return err
		}
	}

	state.AppVersion = s.opts.Manifest.App.Version
	s.migrations = state
	return s.persistMigrations(state)
}

// maybeRunMigration decides if a migration should run and executes it.
func (s *Supervisor) maybeRunMigration(
	ctx context.Context,
	svc manifest.Service,
	m manifest.Migration,
	bin manifest.Binary,
	envBase map[string]string,
	phase string,
	appliedSet map[string]bool,
	state *MigrationsState,
) error {
	if m.Version == "" {
		return fmt.Errorf("migration for service %s missing version", svc.ID)
	}
	if appliedSet[m.Version] {
		return nil
	}

	if !migrations.ShouldRun(m, phase) {
		return nil
	}

	if len(m.Command) == 0 {
		return fmt.Errorf("migration %s has no command", m.Version)
	}

	env := strutil.CopyStringMap(envBase)
	for k, v := range m.Env {
		env[k] = s.renderValue(v)
	}

	if err := s.executeMigration(ctx, svc, m, bin, env); err != nil {
		_ = s.recordTelemetry("migration_failed", map[string]interface{}{
			"service_id": svc.ID,
			"version":    m.Version,
			"error":      err.Error(),
		})
		return err
	}

	appliedSet[m.Version] = true
	migrations.MarkApplied(state, svc.ID, m.Version)

	_ = s.recordTelemetry("migration_applied", map[string]interface{}{
		"service_id": svc.ID,
		"version":    m.Version,
	})

	return s.persistMigrations(*state)
}

// executeMigration runs a single migration command.
func (s *Supervisor) executeMigration(ctx context.Context, svc manifest.Service, m manifest.Migration, bin manifest.Binary, env map[string]string) error {
	cmdArgs := s.renderArgs(m.Command)
	cmdPath := manifest.ResolvePath(s.opts.BundlePath, cmdArgs[0])
	args := cmdArgs[1:]

	_ = s.recordTelemetry("migration_start", map[string]interface{}{
		"service_id": svc.ID,
		"version":    m.Version,
	})

	// Determine working directory.
	workDir := s.opts.BundlePath
	if bin.CWD != "" {
		workDir = manifest.ResolvePath(s.opts.BundlePath, bin.CWD)
	}

	// Set up logging.
	logWriter, _, err := s.logWriter(svc)
	if err != nil {
		return err
	}
	if logWriter != nil {
		defer logWriter.Close()
	}

	// Start the migration process using the injected ProcessRunner.
	proc, err := s.procRunner.Start(ctx, cmdPath, args, strutil.EnvMapToList(env), workDir, logWriter, logWriter)
	if err != nil {
		return fmt.Errorf("start migration %s: %w", m.Version, err)
	}
	if err := proc.Wait(); err != nil {
		return fmt.Errorf("migration %s failed: %w", m.Version, err)
	}
	return nil
}

// ensureAppVersionRecorded updates the migrations state with current app version
// even when no migrations are defined.
func (s *Supervisor) ensureAppVersionRecorded() error {
	state := s.migrations
	if state.AppVersion == s.opts.Manifest.App.Version {
		return nil
	}
	state.AppVersion = s.opts.Manifest.App.Version
	s.migrations = state
	return s.persistMigrations(state)
}
