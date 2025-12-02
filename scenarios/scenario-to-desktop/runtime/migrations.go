package bundleruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"scenario-to-desktop-runtime/manifest"
)

// migrationsState tracks applied migrations per service and the current app version.
type migrationsState struct {
	AppVersion string              `json:"app_version"`
	Applied    map[string][]string `json:"applied"` // service ID -> list of applied migration versions
}

// loadMigrations reads the migrations state from disk.
// Returns an empty state if the file doesn't exist.
func (s *Supervisor) loadMigrations() (migrationsState, error) {
	state := migrationsState{
		Applied: map[string][]string{},
	}
	data, err := os.ReadFile(s.migrationsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return state, nil
		}
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	if state.Applied == nil {
		state.Applied = map[string][]string{}
	}
	return state, nil
}

// persistMigrations saves the migrations state to disk.
func (s *Supervisor) persistMigrations(state migrationsState) error {
	if err := os.MkdirAll(filepath.Dir(s.migrationsPath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(s.migrationsPath, data, 0o600)
}

// runMigrations executes pending migrations for a service.
// Migrations are run based on their run_on condition: always, first_install, or upgrade.
func (s *Supervisor) runMigrations(ctx context.Context, svc manifest.Service, bin manifest.Binary, baseEnv map[string]string) error {
	if len(svc.Migrations) == 0 {
		return s.ensureAppVersionRecorded()
	}

	phase := s.installPhase()
	state := s.migrations

	appliedSet := buildAppliedSet(state.Applied[svc.ID])
	envBase := copyStringMap(baseEnv)

	for _, m := range svc.Migrations {
		if err := s.maybeRunMigration(ctx, svc, m, bin, envBase, phase, appliedSet, &state); err != nil {
			return err
		}
	}

	state.AppVersion = s.opts.Manifest.App.Version
	s.migrations = state
	return s.persistMigrations(state)
}

// buildAppliedSet creates a lookup set from applied version list.
func buildAppliedSet(versions []string) map[string]bool {
	set := make(map[string]bool)
	for _, v := range versions {
		set[v] = true
	}
	return set
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
	state *migrationsState,
) error {
	if m.Version == "" {
		return fmt.Errorf("migration for service %s missing version", svc.ID)
	}
	if appliedSet[m.Version] {
		return nil
	}

	if !shouldRunMigration(m, phase) {
		return nil
	}

	if len(m.Command) == 0 {
		return fmt.Errorf("migration %s has no command", m.Version)
	}

	env := copyStringMap(envBase)
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
	if state.Applied[svc.ID] == nil {
		state.Applied[svc.ID] = []string{}
	}
	state.Applied[svc.ID] = append(state.Applied[svc.ID], m.Version)

	_ = s.recordTelemetry("migration_applied", map[string]interface{}{
		"service_id": svc.ID,
		"version":    m.Version,
	})

	return s.persistMigrations(*state)
}

// shouldRunMigration checks if a migration should run based on its run_on condition.
func shouldRunMigration(m manifest.Migration, phase string) bool {
	runOn := strings.TrimSpace(m.RunOn)
	if runOn == "" {
		runOn = "always"
	}
	switch runOn {
	case "always":
		return true
	case "first_install":
		return phase == "first_install"
	case "upgrade":
		return phase == "upgrade"
	default:
		return false
	}
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

	cmd := exec.CommandContext(ctx, cmdPath, args...)
	cmd.Env = envMapToList(env)

	if bin.CWD != "" {
		cmd.Dir = manifest.ResolvePath(s.opts.BundlePath, bin.CWD)
	} else {
		cmd.Dir = s.opts.BundlePath
	}

	logWriter, _, err := s.logWriter(svc)
	if err != nil {
		return err
	}
	if logWriter != nil {
		defer logWriter.Close()
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start migration %s: %w", m.Version, err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("migration %s failed: %w", m.Version, err)
	}
	return nil
}

// installPhase determines if this is a first install, upgrade, or current version.
func (s *Supervisor) installPhase() string {
	if s.migrations.AppVersion == "" {
		return "first_install"
	}
	if s.migrations.AppVersion != s.opts.Manifest.App.Version {
		return "upgrade"
	}
	return "current"
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
