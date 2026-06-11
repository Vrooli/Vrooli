package storagepaths

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForTestPlacesEveryClassUnderRoot(t *testing.T) {
	root := t.TempDir()
	p := ForTest(root)

	cases := map[string]string{
		"QueueDir":      p.QueueDir,
		"DBPath":        p.DBPath,
		"SystemLogDir":  p.SystemLogDir,
		"TaskRunLogDir": p.TaskRunLogDir,
		"SettingsPath":  p.SettingsPath,
	}
	for name, got := range cases {
		if !strings.HasPrefix(got, root+string(filepath.Separator)) {
			t.Errorf("%s = %q, want under root %q", name, got, root)
		}
	}

	if want := filepath.Join(root, "data", "queue"); p.QueueDir != want {
		t.Errorf("QueueDir = %q, want %q", p.QueueDir, want)
	}
	if want := filepath.Join(root, "data", Scenario+".db"); p.DBPath != want {
		t.Errorf("DBPath = %q, want %q", p.DBPath, want)
	}
	if want := filepath.Join(root, "logs", "task-runs"); p.TaskRunLogDir != want {
		t.Errorf("TaskRunLogDir = %q, want %q", p.TaskRunLogDir, want)
	}
	if want := filepath.Join(root, "config", "settings.json"); p.SettingsPath != want {
		t.Errorf("SettingsPath = %q, want %q", p.SettingsPath, want)
	}
}

// TaskRunLogDir must be resolved from the logs class, never derived as a sibling
// of the queue dir — that coupling is exactly what the cut-over removes.
func TestTaskRunLogDirIsNotUnderQueue(t *testing.T) {
	p := ForTest(t.TempDir())
	if strings.HasPrefix(p.TaskRunLogDir, p.QueueDir) {
		t.Fatalf("TaskRunLogDir %q must not live under QueueDir %q", p.TaskRunLogDir, p.QueueDir)
	}
}

func TestSQLiteDSNHonorsEnvOverride(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "nested", "custom.db")
	t.Setenv("SQLITE_PATH", custom)

	dsn, err := ForTest(t.TempDir()).SQLiteDSN()
	if err != nil {
		t.Fatalf("SQLiteDSN: %v", err)
	}
	if !strings.HasPrefix(dsn, "file:"+custom+"?") {
		t.Errorf("dsn = %q, want SQLITE_PATH override %q", dsn, custom)
	}
	for _, pragma := range []string{"foreign_keys(ON)", "journal_mode(WAL)", "busy_timeout(10000)"} {
		if !strings.Contains(dsn, pragma) {
			t.Errorf("dsn = %q missing pragma %q", dsn, pragma)
		}
	}
	// Parent directory is created as a side effect so the driver can open it.
	if _, err := os.Stat(filepath.Dir(custom)); err != nil {
		t.Errorf("expected parent dir created: %v", err)
	}
}

func TestSQLiteDSNDefaultsToDBPath(t *testing.T) {
	t.Setenv("SQLITE_PATH", "")
	t.Setenv("SQLITE_DB", "")
	p := ForTest(t.TempDir())
	dsn, err := p.SQLiteDSN()
	if err != nil {
		t.Fatalf("SQLiteDSN: %v", err)
	}
	if !strings.HasPrefix(dsn, "file:"+p.DBPath+"?") {
		t.Errorf("dsn = %q, want default DBPath %q", dsn, p.DBPath)
	}
}

// Resolve must route through the variant-aware namespace so a shadow engagement
// is physically isolated from live.
func TestResolveHonorsNamespaceOverride(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_NAMESPACE", Scenario+"_shadow")
	p, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !strings.Contains(p.DBPath, Scenario+"_shadow") {
		t.Errorf("DBPath = %q, want shadow namespace segment", p.DBPath)
	}
	if !strings.HasSuffix(p.DBPath, Scenario+".db") {
		t.Errorf("DBPath = %q, want %s.db filename", p.DBPath, Scenario)
	}
	if !strings.Contains(p.QueueDir, Scenario+"_shadow") {
		t.Errorf("QueueDir = %q, want shadow namespace segment", p.QueueDir)
	}
}
