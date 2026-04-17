package phases

import (
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/orchestrator/workspace"
)

func TestDetectResourceNeedsSupportsSQLite(t *testing.T) {
	scenarioDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir service dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioDir, "initialization", "storage", "sqlite"), 0o755); err != nil {
		t.Fatalf("mkdir sqlite dir: %v", err)
	}
	serviceJSON := `{
  "dependencies": {
    "resources": {
      "sqlite": { "type": "sqlite", "required": true, "enabled": true }
    }
  },
  "environment": {
    "MY_SQLITE_PATH": "${SCENARIO_DATA_DIR}/my.db"
  }
}`
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatalf("write service manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "initialization", "storage", "sqlite", "schema.sql"), []byte("CREATE TABLE demo(id INTEGER);"), 0o644); err != nil {
		t.Fatalf("write sqlite schema: %v", err)
	}

	needs := detectResourceNeeds(workspace.Environment{ScenarioDir: scenarioDir}, nil)
	if !needs.RequireSQLite {
		t.Fatalf("expected sqlite requirement, got %#v", needs)
	}
	if needs.RequirePostgres || needs.RequireRedis {
		t.Fatalf("did not expect postgres/redis requirements, got %#v", needs)
	}
	if len(needs.SQLiteEnvVars) != 1 || needs.SQLiteEnvVars[0] != "MY_SQLITE_PATH" {
		t.Fatalf("expected sqlite env var passthrough, got %#v", needs.SQLiteEnvVars)
	}
}

func TestDetectResourceNeedsSupportsSQLiteWithoutResourceDeclaration(t *testing.T) {
	scenarioDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir service dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioDir, "initialization", "storage", "sqlite"), 0o755); err != nil {
		t.Fatalf("mkdir sqlite dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755); err != nil {
		t.Fatalf("mkdir api dir: %v", err)
	}
	serviceJSON := `{
  "dependencies": {
    "resources": {
      "openrouter": { "type": "openrouter", "required": true, "enabled": true }
    }
  }
}`
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatalf("write service manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "initialization", "storage", "sqlite", "schema.sql"), []byte("CREATE TABLE demo(id INTEGER);"), 0o644); err != nil {
		t.Fatalf("write sqlite schema: %v", err)
	}
	goMod := `module bas-fixture

go 1.24

require modernc.org/sqlite v1.40.1
`
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	needs := detectResourceNeeds(workspace.Environment{ScenarioDir: scenarioDir}, nil)
	if !needs.RequireSQLite {
		t.Fatalf("expected sqlite requirement without resource declaration, got %#v", needs)
	}
	if needs.RequirePostgres || needs.RequireRedis {
		t.Fatalf("did not expect postgres/redis requirements, got %#v", needs)
	}
}

func TestDetectResourceNeedsDefaultsWhenManifestMissing(t *testing.T) {
	needs := detectResourceNeeds(workspace.Environment{ScenarioDir: t.TempDir()}, nil)
	if !needs.RequirePostgres || !needs.RequireRedis || needs.RequireSQLite {
		t.Fatalf("expected postgres+redis fallback, got %#v", needs)
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
