package lifecycle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/shell"
)

// setupCheckCache memoizes SetupNeeded results within a single top-level Start
// invocation. Dependency trees frequently re-evaluate the same scenario's setup
// conditions (once for the reuse decision, once before running the setup phase,
// and once per parent that shares the dependency), each re-walking the same
// replace dirs. The cache is created per Start and threaded through the start
// path; nil means "no memoization" (each call recomputes).
type setupCheckCache map[string]setupCheckResult

type setupCheckResult struct {
	needed  bool
	reasons []string
}

// forceSetupFor reports whether the scenario identified by slug should be
// force-rebuilt. A blank ForceSetupScenario means "force every scenario in this
// start"; a specific value forces only that scenario (and its self-restart).
// This is the single definition of force-setup scope, shared by the top-level
// start path and the dependency loop.
func forceSetupFor(opts StartOptions, slug string) bool {
	return opts.ForceSetup && (opts.ForceSetupScenario == "" || opts.ForceSetupScenario == slug)
}

// setupNeededCached returns SetupNeeded, memoized through cache when non-nil.
// The cache key includes the variant and the force flag so a forced and an
// unforced evaluation of the same scenario never alias.
func (r *Runner) setupNeededCached(item scenario.Scenario, force bool, cache setupCheckCache) (bool, []string, error) {
	return r.evaluateSetupChecksCached(item, force, false, cache)
}

// freshnessStaleCached evaluates only the freshness checks (binaries/ui-bundle),
// memoized through cache when non-nil. This is the verb the reuse decision
// consults: a running, healthy instance is only stopped+rebuilt when a
// freshness check is stale, never because a provisioning check (data, files,
// directories, resources, dependencies) reports "not yet populated".
func (r *Runner) freshnessStaleCached(item scenario.Scenario, force bool, cache setupCheckCache) (bool, []string, error) {
	return r.evaluateSetupChecksCached(item, force, true, cache)
}

func (r *Runner) evaluateSetupChecksCached(item scenario.Scenario, force, freshnessOnly bool, cache setupCheckCache) (bool, []string, error) {
	if cache == nil {
		return r.evaluateSetupChecks(item, force, freshnessOnly)
	}
	key := fmt.Sprintf("%s@%s|force=%t|fresh=%t", item.Slug, item.Variant, force, freshnessOnly)
	if cached, ok := cache[key]; ok {
		return cached.needed, append([]string(nil), cached.reasons...), nil
	}
	needed, reasons, err := r.evaluateSetupChecks(item, force, freshnessOnly)
	if err != nil {
		return false, nil, err
	}
	cache[key] = setupCheckResult{needed: needed, reasons: append([]string(nil), reasons...)}
	return needed, append([]string(nil), reasons...), nil
}

// SetupNeeded reports whether the scenario's setup phase must run before/at
// start. It evaluates every declared setup condition — both provisioning
// (ensure-if-missing) and freshness (rebuild-if-changed). It is the gate for
// running the setup phase on an actual (re)start, NOT the gate for bouncing a
// healthy running process (that is freshnessStaleCached).
func (r *Runner) SetupNeeded(item scenario.Scenario, force bool) (bool, []string, error) {
	return r.evaluateSetupChecks(item, force, false)
}

// isFreshnessCheck reports whether a setup condition type is a freshness verb
// (rebuild-if-content-changed) as opposed to a provisioning verb
// (ensure-if-missing). Only freshness checks may restart a running healthy
// process; provisioning checks (resources, dependencies, data, files,
// directories) ensure state exists and must never trigger a bounce. The "cli"
// type is handled (and short-circuited to false) by evaluateSetupCheck.
func isFreshnessCheck(checkType string) bool {
	switch strings.TrimSpace(checkType) {
	case "", "binaries", "ui-bundle":
		return true
	default:
		return false
	}
}

func (r *Runner) evaluateSetupChecks(item scenario.Scenario, force, freshnessOnly bool) (bool, []string, error) {
	reasons := []string{}
	if force {
		reasons = append(reasons, "Forced rebuild (restart)")
	}

	checks := item.Manifest.Lifecycle.Setup.Condition
	if checks == nil || len(checks.Checks) == 0 {
		return force, reasons, nil
	}

	setupNeeded := force
	for _, check := range checks.Checks {
		if freshnessOnly && !isFreshnessCheck(check.Type) {
			continue
		}
		needed, reason, err := r.evaluateSetupCheck(item, check)
		if err != nil {
			return false, nil, err
		}
		if needed {
			setupNeeded = true
			if reason != "" {
				reasons = append(reasons, reason)
			}
		}
	}
	return setupNeeded, reasons, nil
}

func (r *Runner) evaluateSetupCheck(item scenario.Scenario, check scenario.ConditionCheck) (bool, string, error) {
	switch strings.TrimSpace(check.Type) {
	case "", "binaries":
		return r.binariesFreshness(item, check)
	case "cli":
		// Runtime lifecycle freshness intentionally ignores installed CLI state.
		//
		// Scenario CLI freshness is owned by internal/cliinstall and enforced at
		// command boundaries (for example `vrooli scenario ...`), where the CLI
		// can be refreshed before execution. Treating stale installed CLIs as a
		// runtime setup input caused dependency restart loops because scenario
		// setup phases generally build API/UI artifacts, not scenario CLIs.
		return false, "", nil
	case "ui-bundle":
		return r.uiBundleFreshness(item, check)
	case "resources":
		return resourcesNeedSetup(r.Home, item.Path, check), "Resources not populated", nil
	case "dependencies":
		return dependenciesNeedSetup(item.Path, check), "Dependencies not installed", nil
	case "data":
		return dataNeedsSetup(item.Path, check), "Data directory missing", nil
	case "files":
		return filesNeedSetup(item.Path, check), "Required files missing", nil
	case "directories":
		return directoriesNeedSetup(item.Path, check), "Missing directories", nil
	default:
		return false, "", fmt.Errorf("unsupported setup condition type %q: only native lifecycle setup checks are supported", check.Type)
	}
}

func (r *Runner) ensureScenarioDatabase(item scenario.Scenario, env map[string]string, logWriter io.Writer) error {
	dbName := strings.TrimSpace(env["POSTGRES_DB"])
	if dbName == "" {
		return nil
	}

	r.infof(logWriter, "Ensuring database exists: %s", dbName)

	if err := r.ensurePostgresDatabaseExists(env, dbName, logWriter); err != nil {
		r.warnf(logWriter, "Database creation encountered errors: %v", err)
	}

	migrationsDir := filepath.Join(item.Path, "initialization", "postgres")
	if err := r.ensurePostgresBootstrapRegistry(env, dbName, logWriter); err != nil {
		r.warnf(logWriter, "Bootstrap registry setup encountered errors: %v", err)
		return nil
	}

	schemaFile := filepath.Join(migrationsDir, "schema.sql")
	if _, err := os.Stat(schemaFile); err == nil {
		if err := r.applyPostgresArtifact(item.Slug, env, dbName, bootstrapArtifactKindSchema, schemaFile, logWriter); err != nil {
			r.warnf(logWriter, "Schema bootstrap encountered errors: %v", err)
		}
	}

	pattern := filepath.Join(migrationsDir, "migration_*.sql")
	migrationFiles, err := filepath.Glob(pattern)
	if err != nil {
		r.warnf(logWriter, "Migration discovery encountered errors: %v", err)
		return nil
	}
	sort.Strings(migrationFiles)
	for _, migrationFile := range migrationFiles {
		if err := r.applyPostgresArtifact(item.Slug, env, dbName, bootstrapArtifactKindMigration, migrationFile, logWriter); err != nil {
			r.warnf(logWriter, "Migration %s encountered errors: %v", filepath.Base(migrationFile), err)
		}
	}
	return nil
}

type postgresBootstrapArtifactKind string

const (
	bootstrapArtifactKindSchema    postgresBootstrapArtifactKind = "schema"
	bootstrapArtifactKindMigration postgresBootstrapArtifactKind = "migration"
)

func (r *Runner) ensurePostgresBootstrapRegistry(env map[string]string, dbName string, logWriter io.Writer) error {
	const sql = `
CREATE TABLE IF NOT EXISTS vrooli_bootstrap_artifacts (
    scenario_slug TEXT NOT NULL,
    artifact_kind TEXT NOT NULL,
    artifact_name TEXT NOT NULL,
    checksum TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (scenario_slug, artifact_kind, artifact_name)
);`
	_, err := r.runPostgresCommand(env, dbName, []string{"-c", sql}, logWriter)
	return err
}

func (r *Runner) applyPostgresArtifact(
	scenarioSlug string,
	env map[string]string,
	dbName string,
	kind postgresBootstrapArtifactKind,
	filePath string,
	logWriter io.Writer,
) error {
	checksum, err := postgresArtifactChecksum(filePath)
	if err != nil {
		return err
	}

	artifactName := filepath.Base(filePath)
	appliedChecksum, err := r.lookupPostgresBootstrapArtifact(env, dbName, scenarioSlug, kind, artifactName, logWriter)
	if err != nil {
		return err
	}

	switch kind {
	case bootstrapArtifactKindSchema:
		if appliedChecksum == checksum {
			return nil
		}
	case bootstrapArtifactKindMigration:
		if appliedChecksum == checksum {
			return nil
		}
		if appliedChecksum != "" && appliedChecksum != checksum {
			return fmt.Errorf("migration %s was already applied with a different checksum; create a new migration instead of editing an existing one", artifactName)
		}
	}

	if err := r.executePostgresFile(env, dbName, filePath, logWriter); err != nil {
		return err
	}
	return r.recordPostgresBootstrapArtifact(env, dbName, scenarioSlug, kind, artifactName, checksum, logWriter)
}

func postgresArtifactChecksum(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (r *Runner) lookupPostgresBootstrapArtifact(
	env map[string]string,
	dbName string,
	scenarioSlug string,
	kind postgresBootstrapArtifactKind,
	artifactName string,
	logWriter io.Writer,
) (string, error) {
	sql := fmt.Sprintf(
		"SELECT checksum FROM vrooli_bootstrap_artifacts WHERE scenario_slug = %s AND artifact_kind = %s AND artifact_name = %s;",
		quotePostgresLiteral(scenarioSlug),
		quotePostgresLiteral(string(kind)),
		quotePostgresLiteral(artifactName),
	)
	output, err := r.runPostgresCommand(env, dbName, []string{"-tA", "-c", sql}, io.Discard)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (r *Runner) recordPostgresBootstrapArtifact(
	env map[string]string,
	dbName string,
	scenarioSlug string,
	kind postgresBootstrapArtifactKind,
	artifactName string,
	checksum string,
	logWriter io.Writer,
) error {
	sql := fmt.Sprintf(
		`INSERT INTO vrooli_bootstrap_artifacts (scenario_slug, artifact_kind, artifact_name, checksum, applied_at)
VALUES (%s, %s, %s, %s, NOW())
ON CONFLICT (scenario_slug, artifact_kind, artifact_name)
DO UPDATE SET checksum = EXCLUDED.checksum, applied_at = EXCLUDED.applied_at;`,
		quotePostgresLiteral(scenarioSlug),
		quotePostgresLiteral(string(kind)),
		quotePostgresLiteral(artifactName),
		quotePostgresLiteral(checksum),
	)
	_, err := r.runPostgresCommand(env, dbName, []string{"-c", sql}, logWriter)
	return err
}

func (r *Runner) ensurePostgresDatabaseExists(env map[string]string, dbName string, logWriter io.Writer) error {
	sql := fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname = %s;", quotePostgresLiteral(dbName))
	output, err := r.runPostgresCommand(env, "postgres", []string{"-tA", "-c", sql}, io.Discard)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(output)) == "1" {
		return nil
	}
	createSQL := fmt.Sprintf("CREATE DATABASE %s;", quotePostgresIdentifier(dbName))
	_, err = r.runPostgresCommand(env, "postgres", []string{"-c", createSQL}, logWriter)
	return err
}

func (r *Runner) executePostgresFile(env map[string]string, dbName, filePath string, logWriter io.Writer) error {
	_, err := r.runPostgresCommand(env, dbName, []string{"-f", filePath}, logWriter)
	return err
}

func (r *Runner) runPostgresCommand(env map[string]string, database string, args []string, logWriter io.Writer) ([]byte, error) {
	baseArgs := []string{
		"-q",
		"-v", "ON_ERROR_STOP=1",
		"-h", defaultEnv(env, "POSTGRES_HOST", "localhost"),
		"-p", defaultEnv(env, "POSTGRES_PORT", "5433"),
		"-U", defaultEnv(env, "POSTGRES_USER", "vrooli"),
		"-d", database,
	}
	baseArgs = append(baseArgs, args...)
	commandEnv := mergeEnv(os.Environ(), env)
	if password := strings.TrimSpace(env["POSTGRES_PASSWORD"]); password != "" {
		commandEnv = mergeEnv(commandEnv, map[string]string{"PGPASSWORD": password})
	}
	commandEnv = mergeEnv(commandEnv, map[string]string{"PGOPTIONS": appendPostgresOption(env["PGOPTIONS"], "--client-min-messages=warning")})

	containerName := postgresContainerName(r.Root)
	if output, ok, err := r.runPostgresInContainer(containerName, env, database, args, commandEnv, logWriter); ok {
		return output, err
	}

	if _, err := shell.LookPath("psql"); err == nil {
		cmd := shell.Command(shell.Spec{
			Name: "psql",
			Args: baseArgs,
			Dir:  r.Root,
			Env:  commandEnv,
		})
		output, runErr := cmd.CombinedOutput()
		if len(output) > 0 {
			_, _ = logWriter.Write(output)
		}
		if runErr == nil {
			return output, nil
		}
		return nil, runErr
	}

	if strings.TrimSpace(containerName) == "" {
		containerName = "vrooli-postgres-main"
	}
	output, _, err := r.runPostgresInContainer(containerName, env, database, args, commandEnv, logWriter)
	return output, err
}

func (r *Runner) runPostgresInContainer(containerName string, env map[string]string, database string, args []string, commandEnv []string, logWriter io.Writer) ([]byte, bool, error) {
	if strings.TrimSpace(containerName) == "" {
		return nil, false, nil
	}
	if _, err := shell.LookPath("docker"); err != nil {
		return nil, false, nil
	}

	stdin, filteredArgs, err := postgresFileInput(args)
	if err != nil {
		return nil, true, err
	}

	dockerArgs := []string{"exec"}
	if stdin != nil {
		dockerArgs = append(dockerArgs, "-i")
	}
	if password := strings.TrimSpace(env["POSTGRES_PASSWORD"]); password != "" {
		dockerArgs = append(dockerArgs, "-e", "PGPASSWORD="+password)
	}
	if pgOptions := postgresClientOptions(commandEnv); pgOptions != "" {
		dockerArgs = append(dockerArgs, "-e", "PGOPTIONS="+pgOptions)
	}
	dockerArgs = append(dockerArgs,
		containerName,
		"psql",
		"-q",
		"-v", "ON_ERROR_STOP=1",
		"-h", "localhost",
		"-p", "5432",
		"-U", defaultEnv(env, "POSTGRES_USER", "vrooli"),
		"-d", database,
	)
	dockerArgs = append(dockerArgs, filteredArgs...)

	cmd := shell.Command(shell.Spec{
		Name:  "docker",
		Args:  dockerArgs,
		Dir:   r.Root,
		Env:   commandEnv,
		Stdin: stdin,
	})
	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		_, _ = logWriter.Write(output)
	}
	if err != nil {
		return nil, true, err
	}
	return output, true, nil
}

func appendPostgresOption(existing, option string) string {
	existing = strings.TrimSpace(existing)
	if existing == "" {
		return option
	}
	if strings.Contains(existing, option) {
		return existing
	}
	return existing + " " + option
}

func postgresClientOptions(env []string) string {
	for _, entry := range env {
		if value, ok := strings.CutPrefix(entry, "PGOPTIONS="); ok {
			return value
		}
	}
	return ""
}

func postgresFileInput(args []string) (io.Reader, []string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] != "-f" {
			continue
		}
		if i+1 >= len(args) {
			return nil, nil, fmt.Errorf("postgres command missing file path after -f")
		}
		data, err := os.ReadFile(args[i+1])
		if err != nil {
			return nil, nil, err
		}
		filtered := append([]string{}, args[:i]...)
		filtered = append(filtered, args[i+2:]...)
		return bytes.NewReader(data), filtered, nil
	}
	return nil, append([]string{}, args...), nil
}

func postgresContainerName(root string) string {
	manifestPath := manifestpkg.DefaultPath(root, "postgres")
	resourceManifest, err := manifestpkg.Load(manifestPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(resourceManifest.Runtime.ContainerName)
}

func quotePostgresIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func quotePostgresLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}

func defaultEnv(env map[string]string, key, fallback string) string {
	if value := strings.TrimSpace(env[key]); value != "" {
		return value
	}
	return fallback
}
