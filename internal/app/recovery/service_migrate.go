package recovery

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	// no database at all. When scripts ARE present and DBPath is empty, the SQLite
	// engine auto-resolves the live database from the scenario's variant-aware data
	// dir (the InstanceKey storage SSOT) following the react-vite convention
	// "<data-dir>/<scenario>.db", falling back to the sole *.db file in that dir;
	// a non-conventional scenario passes this explicitly to override.
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
	// DBPathAutoResolved is true when the SQLite database path was derived from the
	// scenario's variant-aware data dir rather than supplied via --db-path. The
	// resolved path itself is reported in the embedded MigrationResult.Database.
	DBPathAutoResolved bool `json:"dbPathAutoResolved"`
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

	// DB-path resolution. The no-scripts fast path needs no database, so only
	// resolve when there is something to apply. An explicit --db-path always wins;
	// otherwise the SQLite engine auto-resolves the live database from the
	// scenario's variant-aware data dir so promote does not require the operator to
	// know each scenario's on-disk db location (Baseline Modes §8, part-15 follow-up).
	dbPath := strings.TrimSpace(req.DBPath)
	autoResolved := false
	if len(scripts) > 0 && dbPath == "" && engine == baselinefloor.EngineSQLite {
		resolved, resErr := s.resolveLiveSQLiteDB(req.Scenario)
		if resErr != nil {
			return MigrateOutput{Scenario: req.Scenario, Slug: req.Slug, MigrationsDir: migrationsDir}, resErr
		}
		dbPath = resolved
		autoResolved = true
	}

	result, err := baselinefloor.RunMigrations(engine, dbPath, scripts, baselinefloor.MigrateOptions{DryRun: req.DryRun})
	out := MigrateOutput{
		Scenario:           req.Scenario,
		Slug:               req.Slug,
		MigrationsDir:      migrationsDir,
		DBPathAutoResolved: autoResolved,
		MigrationResult:    result,
	}
	if err != nil {
		return out, err
	}
	return out, nil
}

// resolveLiveSQLiteDB derives the live SQLite database file for a scenario from
// the InstanceKey storage SSOT (the same data dir `recovery namespace` reports),
// so a baseline promote can migrate the live schema without the operator naming
// the file. It resolves the LIVE variant deliberately: a shadow's copied database
// is validation-only and is never promoted (Baseline Modes §8).
//
// Resolution within the data dir:
//  1. the react-vite convention "<data-dir>/<scenario>.db" when that file exists;
//  2. otherwise the sole "*.db" file in the data dir.
//
// A non-conventional layout — no .db file, several of them, or an unresolvable
// data dir — is a clear, actionable error pointing at --db-path rather than a
// guess, so the runner never migrates an orphan database the scenario never reads.
func (s Service) resolveLiveSQLiteDB(scenario string) (string, error) {
	ns, err := s.Namespace(NamespaceRequest{Scenario: scenario})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(ns.DataDir) == "" {
		return "", fmt.Errorf("recovery: cannot auto-resolve the SQLite database for %q (data dir unresolvable); pass --db-path", scenario)
	}

	canonical := filepath.Join(ns.DataDir, scenario+".db")
	if info, statErr := os.Stat(canonical); statErr == nil && !info.IsDir() {
		return canonical, nil
	}

	matches, globErr := filepath.Glob(filepath.Join(ns.DataDir, "*.db"))
	if globErr != nil {
		return "", fmt.Errorf("recovery: scan %q for a SQLite database: %w", ns.DataDir, globErr)
	}
	files := matches[:0]
	for _, m := range matches {
		if info, statErr := os.Stat(m); statErr == nil && !info.IsDir() {
			files = append(files, m)
		}
	}
	switch len(files) {
	case 1:
		return files[0], nil
	case 0:
		return "", fmt.Errorf("recovery: no SQLite database found under %q for %q; pass --db-path", ns.DataDir, scenario)
	default:
		sort.Strings(files)
		return "", fmt.Errorf("recovery: multiple SQLite databases under %q (%s); pass --db-path to choose one",
			ns.DataDir, strings.Join(baseNames(files), ", "))
	}
}

// baseNames returns the file names (without directories) for an error message.
func baseNames(paths []string) []string {
	names := make([]string, len(paths))
	for i, p := range paths {
		names[i] = filepath.Base(p)
	}
	return names
}
