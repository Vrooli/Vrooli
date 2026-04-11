package lifecycle

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/vrooli/vrooli/internal/scenario"
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
		return cliNeedsSetup(item.Path, check)
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
		needed, err := runExternalSetupChecker(r.Root, item.Path, check)
		if err != nil {
			return false, "", err
		}
		if needed {
			return true, "Check failed: " + check.Type, nil
		}
		return false, "", nil
	}
}

func (r *Runner) ensureScenarioDatabase(item scenario.Scenario, env map[string]string, logWriter io.Writer) error {
	dbName := strings.TrimSpace(env["POSTGRES_DB"])
	if dbName == "" {
		return nil
	}

	r.infof(logWriter, "Ensuring database exists: %s", dbName)

	script := fmt.Sprintf(`
set -e
export APP_ROOT=%s
export VROOLI_ROOT=%s
scenario_path=%s
db_name=%s
postgres_db_lib="$APP_ROOT/resources/postgres/lib/database.sh"
postgres_common_lib="$APP_ROOT/resources/postgres/lib/common.sh"
postgres_defaults="$APP_ROOT/resources/postgres/config/defaults.sh"
if [[ ! -f "$postgres_db_lib" ]]; then
  exit 0
fi
source "$postgres_defaults" >/dev/null 2>&1 || true
source "$postgres_common_lib" >/dev/null 2>&1 || true
source "$postgres_db_lib" >/dev/null 2>&1 || true
if ! postgres::common::is_running "main" >/dev/null 2>&1; then
  echo "Postgres not running, skipping database creation for: $db_name"
  exit 0
fi
postgres::database::create "main" "$db_name" >/dev/null 2>&1 || true
schema_file="$scenario_path/initialization/postgres/schema.sql"
if [[ -f "$schema_file" ]]; then
  postgres::database::execute_file "main" "$schema_file" "$db_name" >/dev/null 2>&1 || true
fi
migrations_dir="$scenario_path/initialization/postgres"
if [[ -d "$migrations_dir" ]] && ls "$migrations_dir"/migration_*.sql >/dev/null 2>&1; then
  postgres::database::migrate "main" "$migrations_dir" "$db_name" >/dev/null 2>&1 || true
fi
`, shellQuote(r.Root), shellQuote(r.Root), shellQuote(item.Path), shellQuote(dbName))

	cmd := exec.Command("bash", "-lc", script)
	cmd.Dir = item.Path
	cmd.Env = mergeEnv(os.Environ(), env)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	if err := cmd.Run(); err != nil {
		r.warnf(logWriter, "Database bootstrap encountered errors: %v", err)
	}
	return nil
}
