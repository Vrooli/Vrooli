package lifecycle

import (
	"bytes"
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

func (r *Runner) SetupNeeded(item scenario.Scenario, force bool) (bool, []string, error) {
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
		return binariesNeedSetup(item.Path, check)
	case "cli":
		return cliNeedsSetup(item, check)
	case "ui-bundle":
		return uiBundleNeedsSetup(item.Path, check)
	case "resources":
		return resourcesNeedSetup(item.Path, check), "Resources not populated", nil
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

	schemaFile := filepath.Join(item.Path, "initialization", "postgres", "schema.sql")
	if _, err := os.Stat(schemaFile); err == nil {
		if err := r.executePostgresFile(env, dbName, schemaFile, logWriter); err != nil {
			r.warnf(logWriter, "Schema bootstrap encountered errors: %v", err)
		}
	}

	migrationsDir := filepath.Join(item.Path, "initialization", "postgres")
	pattern := filepath.Join(migrationsDir, "migration_*.sql")
	migrationFiles, err := filepath.Glob(pattern)
	if err != nil {
		r.warnf(logWriter, "Migration discovery encountered errors: %v", err)
		return nil
	}
	sort.Strings(migrationFiles)
	for _, migrationFile := range migrationFiles {
		if err := r.executePostgresFile(env, dbName, migrationFile, logWriter); err != nil {
			r.warnf(logWriter, "Migration %s encountered errors: %v", filepath.Base(migrationFile), err)
		}
	}
	return nil
}

func (r *Runner) ensurePostgresDatabaseExists(env map[string]string, dbName string, logWriter io.Writer) error {
	sql := fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname = %s;", quotePostgresLiteral(dbName))
	output, err := r.runPostgresCommand(env, "postgres", []string{"-tA", "-c", sql}, logWriter)
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

	containerName := postgresContainerName(r.Root)
	if output, err, ok := r.runPostgresInContainer(containerName, env, database, args, commandEnv, logWriter); ok {
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
	output, err, _ := r.runPostgresInContainer(containerName, env, database, args, commandEnv, logWriter)
	return output, err
}

func (r *Runner) runPostgresInContainer(containerName string, env map[string]string, database string, args []string, commandEnv []string, logWriter io.Writer) ([]byte, error, bool) {
	if strings.TrimSpace(containerName) == "" {
		return nil, nil, false
	}
	if _, err := shell.LookPath("docker"); err != nil {
		return nil, nil, false
	}

	stdin, filteredArgs, err := postgresFileInput(args)
	if err != nil {
		return nil, err, true
	}

	dockerArgs := []string{"exec"}
	if stdin != nil {
		dockerArgs = append(dockerArgs, "-i")
	}
	if password := strings.TrimSpace(env["POSTGRES_PASSWORD"]); password != "" {
		dockerArgs = append(dockerArgs, "-e", "PGPASSWORD="+password)
	}
	dockerArgs = append(dockerArgs,
		containerName,
		"psql",
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
		return nil, err, true
	}
	return output, nil, true
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
