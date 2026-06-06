package recovery

import (
	"strings"

	"github.com/vrooli/vrooli/internal/baselinefloor"
)

// MigrateRequest applies a baseline engagement's managed migration scripts to a
// target database — the promote step's schema runner (Baseline Modes §8).
type MigrateRequest struct {
	Scenario string
	Slug     string
	// Engine selects the storage engine; empty defaults to sqlite (the v1 engine).
	Engine string
	// DBPath is the target database file. The universal no-scripts fast path needs
	// no database at all; when scripts ARE present, v1 requires this explicitly
	// (per-scenario DB-path auto-resolution is a documented follow-up).
	DBPath string
	// MigrationsDir overrides the engagement's managed migration folder.
	MigrationsDir string
	// DryRun validates against a throwaway copy without mutating the real database.
	DryRun bool
}

// MigrateOutput reports the run (the --json shape). The embedded MigrationResult
// flattens engine/applied/skipped/fastPath alongside scenario/slug.
type MigrateOutput struct {
	Scenario      string `json:"scenario"`
	Slug          string `json:"slug"`
	MigrationsDir string `json:"migrationsDir"`
	baselinefloor.MigrationResult
}

// Migrate runs the engagement's managed migration scripts against the target
// database. Scripts live in the floor-owned per-engagement migrations folder;
// "no scripts" is the shape-unchanged fast path (no database required). It always
// dry-runs against a throwaway copy before mutating the real database, so an
// incompatible script bounces (live untouched) rather than corrupting it.
func (s Service) Migrate(req MigrateRequest) (MigrateOutput, error) {
	if err := requireRef(req.Scenario, req.Slug); err != nil {
		return MigrateOutput{}, err
	}
	migrationsDir := strings.TrimSpace(req.MigrationsDir)
	if migrationsDir == "" {
		migrationsDir = s.Store.MigrationsPath(req.Scenario, req.Slug)
	}
	scripts, err := baselinefloor.LoadScripts(migrationsDir)
	if err != nil {
		return MigrateOutput{}, err
	}
	engine := baselinefloor.Engine(strings.ToLower(strings.TrimSpace(req.Engine)))
	if engine == "" {
		engine = baselinefloor.EngineSQLite
	}
	result, err := baselinefloor.RunMigrations(engine, strings.TrimSpace(req.DBPath), scripts, baselinefloor.MigrateOptions{DryRun: req.DryRun})
	out := MigrateOutput{Scenario: req.Scenario, Slug: req.Slug, MigrationsDir: migrationsDir, MigrationResult: result}
	if err != nil {
		return out, err
	}
	return out, nil
}
