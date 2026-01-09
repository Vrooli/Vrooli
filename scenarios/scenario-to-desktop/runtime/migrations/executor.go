// Package migrations provides migration tracking and execution for the bundle runtime.
package migrations

import (
	"context"
	"fmt"
	"io"

	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/strutil"
	"scenario-to-desktop-runtime/telemetry"
)

// ExecutorConfig holds the configuration for the migration executor.
type ExecutorConfig struct {
	BundlePath string
	AppVersion string
	Tracker    *Tracker
	ProcRunner infra.ProcessRunner
	Telemetry  telemetry.Recorder
}

// EnvRenderer provides template expansion for environment variables and arguments.
type EnvRenderer interface {
	RenderValue(input string) string
	RenderArgs(args []string) []string
}

// LogProvider provides log writers for services.
type LogProvider interface {
	LogWriter(svc manifest.Service) (infra.File, string, error)
}

// Runner defines the interface for migration execution.
// This allows for easy testing with mock implementations.
type Runner interface {
	Run(ctx context.Context, svc manifest.Service, bin manifest.Binary, baseEnv map[string]string) error
	State() State
	SetState(state State)
}

// Ensure Executor implements Runner.
var _ Runner = (*Executor)(nil)

// Executor runs migrations for services.
type Executor struct {
	cfg         ExecutorConfig
	envRenderer EnvRenderer
	logProvider LogProvider
	state       State
}

// NewExecutor creates a new migration executor.
func NewExecutor(cfg ExecutorConfig, envRenderer EnvRenderer, logProvider LogProvider) *Executor {
	return &Executor{
		cfg:         cfg,
		envRenderer: envRenderer,
		logProvider: logProvider,
		state:       NewState(),
	}
}

// SetState sets the current migrations state.
func (e *Executor) SetState(state State) {
	e.state = state
}

// State returns the current migrations state.
func (e *Executor) State() State {
	return e.state
}

// Run executes pending migrations for a service.
// Migrations are run based on their run_on condition: always, first_install, or upgrade.
func (e *Executor) Run(ctx context.Context, svc manifest.Service, bin manifest.Binary, baseEnv map[string]string) error {
	if len(svc.Migrations) == 0 {
		return e.ensureAppVersionRecorded()
	}

	phase := Phase(e.state, e.cfg.AppVersion)
	appliedSet := BuildAppliedSet(e.state.Applied[svc.ID])
	envBase := strutil.CopyStringMap(baseEnv)

	for _, m := range svc.Migrations {
		if err := e.maybeRunMigration(ctx, svc, m, bin, envBase, phase, appliedSet); err != nil {
			return err
		}
	}

	e.state.AppVersion = e.cfg.AppVersion
	return e.cfg.Tracker.Persist(e.state)
}

// maybeRunMigration decides if a migration should run and executes it.
func (e *Executor) maybeRunMigration(
	ctx context.Context,
	svc manifest.Service,
	m manifest.Migration,
	bin manifest.Binary,
	envBase map[string]string,
	phase string,
	appliedSet map[string]bool,
) error {
	if m.Version == "" {
		return fmt.Errorf("migration for service %s missing version", svc.ID)
	}
	if appliedSet[m.Version] {
		return nil
	}

	if !ShouldRun(m, phase) {
		return nil
	}

	if len(m.Command) == 0 {
		return fmt.Errorf("migration %s has no command", m.Version)
	}

	env := strutil.CopyStringMap(envBase)
	for k, v := range m.Env {
		env[k] = e.envRenderer.RenderValue(v)
	}

	if err := e.executeMigration(ctx, svc, m, bin, env); err != nil {
		_ = e.cfg.Telemetry.Record("migration_failed", map[string]interface{}{
			"service_id": svc.ID,
			"version":    m.Version,
			"error":      err.Error(),
		})
		return err
	}

	appliedSet[m.Version] = true
	MarkApplied(&e.state, svc.ID, m.Version)

	_ = e.cfg.Telemetry.Record("migration_applied", map[string]interface{}{
		"service_id": svc.ID,
		"version":    m.Version,
	})

	return e.cfg.Tracker.Persist(e.state)
}

// executeMigration runs a single migration command.
func (e *Executor) executeMigration(ctx context.Context, svc manifest.Service, m manifest.Migration, bin manifest.Binary, env map[string]string) error {
	cmdArgs := e.envRenderer.RenderArgs(m.Command)
	cmdPath := manifest.ResolvePath(e.cfg.BundlePath, cmdArgs[0])
	args := cmdArgs[1:]

	_ = e.cfg.Telemetry.Record("migration_start", map[string]interface{}{
		"service_id": svc.ID,
		"version":    m.Version,
	})

	// Determine working directory.
	workDir := e.cfg.BundlePath
	if bin.CWD != "" {
		workDir = manifest.ResolvePath(e.cfg.BundlePath, bin.CWD)
	}

	// Set up logging.
	logWriter, _, err := e.logProvider.LogWriter(svc)
	if err != nil {
		return err
	}

	var stdout, stderr io.Writer
	if logWriter != nil {
		defer logWriter.Close()
		stdout = logWriter
		stderr = logWriter
	}

	// Start the migration process.
	proc, err := e.cfg.ProcRunner.Start(ctx, cmdPath, args, strutil.EnvMapToList(env), workDir, stdout, stderr)
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
func (e *Executor) ensureAppVersionRecorded() error {
	if e.state.AppVersion == e.cfg.AppVersion {
		return nil
	}
	e.state.AppVersion = e.cfg.AppVersion
	return e.cfg.Tracker.Persist(e.state)
}
