package phases

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/workspace"
)

func TestDetectResourceNeedsManifestPostgresOnly(t *testing.T) {
	scenarioDir := t.TempDir()
	writeServiceJSON(t, scenarioDir, `{
  "dependencies": {
    "resources": {
      "pg": { "type": "postgres", "required": true, "enabled": true }
    }
  }
}`)

	var log bytes.Buffer
	needs := resolveDBNeeds(context.Background(), workspace.Environment{ScenarioDir: scenarioDir}, &log)
	if !needs.RequirePostgres {
		t.Fatalf("expected postgres required, got %#v", needs)
	}
	if needs.RequireRedis || needs.RequireSQLite {
		t.Fatalf("expected only postgres, got %#v", needs)
	}
	if !strings.Contains(log.String(), "db-detect:") {
		t.Fatalf("expected evidence chain in log, got: %s", log.String())
	}
	if !strings.Contains(log.String(), "postgres:") {
		t.Fatalf("expected postgres line, got: %s", log.String())
	}
}

func TestDetectResourceNeedsSQLiteFromGoMod(t *testing.T) {
	scenarioDir := t.TempDir()
	writeServiceJSON(t, scenarioDir, `{
  "dependencies": {
    "resources": {
      "openrouter": { "type": "openrouter", "required": true, "enabled": true }
    }
  },
  "environment": {
    "MY_SQLITE_PATH": "${SCENARIO_DATA_DIR}/my.db"
  }
}`)
	if err := os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	goMod := "module bas-fixture\n\ngo 1.24\n\nrequire modernc.org/sqlite v1.40.1\n"
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	var log bytes.Buffer
	needs := resolveDBNeeds(context.Background(), workspace.Environment{ScenarioDir: scenarioDir}, &log)
	if !needs.RequireSQLite {
		t.Fatalf("expected sqlite required, got %#v (log: %s)", needs, log.String())
	}
	if needs.RequirePostgres || needs.RequireRedis {
		t.Fatalf("did not expect postgres/redis, got %#v", needs)
	}
	if len(needs.SQLiteEnvVars) != 1 || needs.SQLiteEnvVars[0] != "MY_SQLITE_PATH" {
		t.Fatalf("expected sqlite env var passthrough, got %#v", needs.SQLiteEnvVars)
	}
}

func TestDetectResourceNeedsNoEvidenceProvisionsNothing(t *testing.T) {
	var log bytes.Buffer
	needs := resolveDBNeeds(context.Background(), workspace.Environment{ScenarioDir: t.TempDir()}, &log)
	if needs.RequirePostgres || needs.RequireRedis || needs.RequireSQLite {
		t.Fatalf("expected no provisioning when no evidence, got %#v", needs)
	}
	if !strings.Contains(log.String(), "db-detect:") {
		t.Fatalf("expected evidence chain in log, got: %s", log.String())
	}
}

func TestDetectResourceNeedsMultiDB(t *testing.T) {
	scenarioDir := t.TempDir()
	writeServiceJSON(t, scenarioDir, `{
  "dependencies": {
    "resources": {
      "pg": { "type": "postgres", "required": true, "enabled": true }
    }
  }
}`)
	if err := os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	goMod := "module x\n\ngo 1.24\n\nrequire modernc.org/sqlite v1.40.1\n"
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	var log bytes.Buffer
	needs := resolveDBNeeds(context.Background(), workspace.Environment{ScenarioDir: scenarioDir}, &log)
	if !needs.RequirePostgres || !needs.RequireSQLite {
		t.Fatalf("expected postgres+sqlite, got %#v (log: %s)", needs, log.String())
	}
	if needs.RequireRedis {
		t.Fatalf("did not expect redis, got %#v", needs)
	}
}

func TestCollectMigrationFilesMergesLegacyAndScopedDirectories(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "001-legacy.sql"), []byte("SELECT 1;"), 0o644); err != nil {
		t.Fatalf("write legacy migration: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "common"), 0o755); err != nil {
		t.Fatalf("mkdir common migrations: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "common", "002-common.sql"), []byte("SELECT 2;"), 0o644); err != nil {
		t.Fatalf("write common migration: %v", err)
	}

	files, err := collectMigrationFiles(root, "common")
	if err != nil {
		t.Fatalf("collectMigrationFiles: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected merged migration files, got %#v", files)
	}
}

func writeServiceJSON(t *testing.T, scenarioDir, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
}
