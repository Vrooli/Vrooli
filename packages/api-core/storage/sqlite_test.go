package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// legacyFleetDSN is the pragma string that 60+ scenarios each carried a private
// copy of before this package owned the concern. It is reproduced here verbatim
// so a change to the canonical DSN cannot pass unnoticed: consolidation must not
// alter how any already-created database is opened.
const legacyFleetDSN = "file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)"

func TestSQLiteDSNAtMatchesLegacyFleetString(t *testing.T) {
	path := filepath.Join(t.TempDir(), "example.db")

	got, err := SQLiteDSNAt(path, SQLiteTuning{})
	if err != nil {
		t.Fatalf("SQLiteDSNAt: %v", err)
	}
	want := strings.Replace(legacyFleetDSN, "%s", path, 1)
	if got != want {
		t.Fatalf("canonical DSN drifted from the fleet string\n got: %s\nwant: %s", got, want)
	}
}

func TestSQLiteDSNAtCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "deeper", "example.db")

	if _, err := SQLiteDSNAt(path, SQLiteTuning{}); err != nil {
		t.Fatalf("SQLiteDSNAt: %v", err)
	}
	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("parent directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("parent path is not a directory")
	}
}

func TestSQLiteDSNAtRejectsAssembledDSN(t *testing.T) {
	// Accepting a "file:" input would silently discard the tuning argument.
	if _, err := SQLiteDSNAt("file:/tmp/example.db?_pragma=journal_mode(WAL)", SQLiteTuning{}); err == nil {
		t.Fatal("expected an assembled DSN to be rejected")
	}
	if _, err := SQLiteDSNAt("   ", SQLiteTuning{}); err == nil {
		t.Fatal("expected an empty path to be rejected")
	}
}

func TestSQLiteDSNTuningEmissionOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tuned.db")

	got, err := SQLiteDSNAt(path, SQLiteTuning{
		PageSizeBytes: 4096,
		MMapSizeBytes: 268435456,
		TimeFormat:    "sqlite",
	})
	if err != nil {
		t.Fatalf("SQLiteDSNAt: %v", err)
	}
	// Reproduces, byte for byte, the tuned string browser-automation-studio and
	// git-control-tower had converged on independently.
	want := "file:" + path + "?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)" +
		"&_pragma=cache_size(-2000)&_pragma=page_size(4096)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)" +
		"&_pragma=mmap_size(268435456)&_time_format=sqlite"
	if got != want {
		t.Fatalf("tuned DSN mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestSQLiteDSNTuningOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "over.db")

	got, err := SQLiteDSNAt(path, SQLiteTuning{
		BusyTimeout:  5 * time.Second,
		CacheSizeKiB: 8000,
		Synchronous:  "full",
	})
	if err != nil {
		t.Fatalf("SQLiteDSNAt: %v", err)
	}
	for _, want := range []string{"busy_timeout(5000)", "cache_size(-8000)", "synchronous(FULL)"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in %s", want, got)
		}
	}
}

func TestSQLiteTuningRejectsUnsafeValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.db")

	cases := map[string]SQLiteTuning{
		// Truncates to 0ms, which SQLite reads as "never wait for the lock".
		"sub-millisecond busy timeout": {BusyTimeout: 500 * time.Microsecond},
		"negative busy timeout":        {BusyTimeout: -time.Second},
		"negative cache size":          {CacheSizeKiB: -1},
		"negative page size":           {PageSizeBytes: -1},
		"negative mmap size":           {MMapSizeBytes: -1},
		"unknown synchronous mode":     {Synchronous: "SOMETIMES"},
	}
	for name, tuning := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := SQLiteDSNAt(path, tuning); err == nil {
				t.Fatalf("expected %s to be rejected", name)
			}
		})
	}
}

func TestSQLitePathRequiresScenario(t *testing.T) {
	if _, err := SQLitePath(SQLiteConfig{}); err == nil {
		t.Fatal("expected a missing scenario to be rejected")
	}
	if _, err := SQLitePath(SQLiteConfig{Scenario: "  "}); err == nil {
		t.Fatal("expected a blank scenario to be rejected")
	}
}

func TestSQLitePathRejectsEscapingFilename(t *testing.T) {
	for _, name := range []string{"../sibling.db", "nested/db.sqlite", "..", "."} {
		if _, err := SQLitePath(SQLiteConfig{Scenario: "example", Filename: name}); err == nil {
			t.Fatalf("expected filename %q to be rejected", name)
		}
	}
}

func TestSQLitePathDefaultsToScenarioNamedFile(t *testing.T) {
	root := t.TempDir()
	path, err := SQLitePath(SQLiteConfig{
		Scenario:     "example-scenario",
		RootOverride: root,
		EnvGet:       func(string) string { return "" },
	})
	if err != nil {
		t.Fatalf("SQLitePath: %v", err)
	}
	if filepath.Base(path) != "example-scenario.db" {
		t.Fatalf("expected the default filename, got %s", path)
	}
	if !strings.Contains(path, "example-scenario") {
		t.Fatalf("expected the path to be scoped to the scenario, got %s", path)
	}
}

// TestSQLitePathIgnoresGenericEnvironmentOverrides is the regression test for
// the defect this package was built to close. A sibling scenario's inherited
// environment must not be able to redirect this scenario's database.
func TestSQLitePathIgnoresGenericEnvironmentOverrides(t *testing.T) {
	root := t.TempDir()
	hijack := map[string]string{
		"SQLITE_PATH":  "/hijacked/autoheal.sqlite",
		"SQLITE_DB":    "/hijacked/autoheal.sqlite",
		"DATABASE_URL": "file:/hijacked/autoheal.sqlite",
	}

	path, err := SQLitePath(SQLiteConfig{
		Scenario:     "victim-scenario",
		RootOverride: root,
		EnvGet:       func(key string) string { return hijack[key] },
	})
	if err != nil {
		t.Fatalf("SQLitePath: %v", err)
	}
	if strings.Contains(path, "hijacked") || strings.Contains(path, "autoheal") {
		t.Fatalf("a generic environment variable redirected the database: %s", path)
	}
	if filepath.Base(path) != "victim-scenario.db" {
		t.Fatalf("expected the scenario's own database, got %s", path)
	}
}

// TestSQLitePathHonoursShadowNamespace proves the one environment input that IS
// read stays safe under inheritance: it is scenario-agnostic, so a child that
// inherits it still resolves to its own identity under its own variant.
func TestSQLitePathHonoursShadowNamespace(t *testing.T) {
	root := t.TempDir()
	env := map[string]string{
		EnvStorageNamespace: "example-scenario_shadow",
		EnvVariant:          "shadow",
	}

	shadow, err := SQLitePath(SQLiteConfig{
		Scenario:     "example-scenario",
		RootOverride: root,
		EnvGet:       func(key string) string { return env[key] },
	})
	if err != nil {
		t.Fatalf("SQLitePath (shadow): %v", err)
	}
	live, err := SQLitePath(SQLiteConfig{
		Scenario:     "example-scenario",
		RootOverride: root,
		EnvGet:       func(string) string { return "" },
	})
	if err != nil {
		t.Fatalf("SQLitePath (live): %v", err)
	}
	if shadow == live {
		t.Fatalf("shadow and live resolved to the same file: %s", live)
	}
	if !strings.Contains(shadow, "example-scenario_shadow") {
		t.Fatalf("shadow path is not variant-scoped: %s", shadow)
	}
}

// TestSQLitePathFailsLoudOnInconsistentVariant guards the one case where a
// silent fallback would alias a shadow's writes onto live.
func TestSQLitePathFailsLoudOnInconsistentVariant(t *testing.T) {
	env := map[string]string{EnvVariant: "shadow"} // non-live, but no namespace root
	if _, err := SQLitePath(SQLiteConfig{
		Scenario:     "example-scenario",
		RootOverride: t.TempDir(),
		EnvGet:       func(key string) string { return env[key] },
	}); err == nil {
		t.Fatal("expected a non-live variant with no injected namespace root to fail loudly")
	}
}

func TestSQLiteDSNResolvesAndCreatesDirectory(t *testing.T) {
	root := t.TempDir()
	dsn, err := SQLiteDSN(SQLiteConfig{
		Scenario:     "example-scenario",
		RootOverride: root,
		EnvGet:       func(string) string { return "" },
	})
	if err != nil {
		t.Fatalf("SQLiteDSN: %v", err)
	}
	if !strings.HasPrefix(dsn, "file:") || !strings.Contains(dsn, "journal_mode(WAL)") {
		t.Fatalf("unexpected DSN: %s", dsn)
	}
	path := strings.TrimPrefix(strings.SplitN(dsn, "?", 2)[0], "file:")
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("parent directory was not created: %v", err)
	}
}

func TestSQLiteDSNHonoursClassAndFilename(t *testing.T) {
	root := t.TempDir()
	dsn, err := SQLiteDSN(SQLiteConfig{
		Scenario:     "example-scenario",
		Filename:     "index.sqlite",
		Class:        ClassCache,
		RootOverride: root,
		EnvGet:       func(string) string { return "" },
	})
	if err != nil {
		t.Fatalf("SQLiteDSN: %v", err)
	}
	if !strings.Contains(dsn, "index.sqlite") {
		t.Fatalf("expected the requested filename in %s", dsn)
	}
	if !strings.Contains(dsn, string(ClassCache)) {
		t.Fatalf("expected the cache class root in %s", dsn)
	}
}

// --- lifecycle data directory -------------------------------------------
//
// These cover the guard that lets consolidation keep every existing database
// exactly where it already is, without reopening the inheritance hole.

func TestSQLitePathUsesLifecycleDataDirForLiveInstance(t *testing.T) {
	dataDir := t.TempDir()
	env := map[string]string{
		"SCENARIO_NAME":     "example-scenario",
		EnvScenario:         "example-scenario",
		"SCENARIO_DATA_DIR": dataDir,
	}

	path, err := SQLitePath(SQLiteConfig{
		Scenario: "example-scenario",
		EnvGet:   func(key string) string { return env[key] },
	})
	if err != nil {
		t.Fatalf("SQLitePath: %v", err)
	}
	want := filepath.Join(dataDir, "example-scenario.db")
	if path != want {
		t.Fatalf("expected the lifecycle data directory\n got: %s\nwant: %s", path, want)
	}
}

// TestSQLitePathRejectsInheritedLifecycleDataDir is the second half of the
// hijack regression. A supervisor leaks its own SCENARIO_DATA_DIR into the
// children it restarts; the identity guard must reject it.
func TestSQLitePathRejectsInheritedLifecycleDataDir(t *testing.T) {
	supervisorData := t.TempDir()
	inherited := map[string]string{
		// Both values arrive together, and both name the SUPERVISOR.
		"SCENARIO_NAME":     "vrooli-autoheal",
		EnvScenario:         "vrooli-autoheal",
		"SCENARIO_DATA_DIR": supervisorData,
	}

	path, err := SQLitePath(SQLiteConfig{
		Scenario: "victim-scenario",
		EnvGet:   func(key string) string { return inherited[key] },
	})
	if err != nil {
		t.Fatalf("SQLitePath: %v", err)
	}
	if strings.HasPrefix(path, supervisorData) {
		t.Fatalf("an inherited data directory captured this scenario: %s", path)
	}
	if filepath.Base(path) != "victim-scenario.db" {
		t.Fatalf("expected this scenario's own database, got %s", path)
	}
}

func TestSQLitePathIgnoresRelativeLifecycleDataDir(t *testing.T) {
	env := map[string]string{
		"SCENARIO_NAME":     "example-scenario",
		EnvScenario:         "example-scenario",
		"SCENARIO_DATA_DIR": "relative/data",
	}
	path, err := SQLitePath(SQLiteConfig{
		Scenario: "example-scenario",
		EnvGet:   func(key string) string { return env[key] },
	})
	if err != nil {
		t.Fatalf("SQLitePath: %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("expected an absolute path, got %s", path)
	}
}

// TestSQLitePathShadowIgnoresLifecycleDataDir proves the variant carve-out: the
// assigned data directory is not variant-scoped, so honouring it for a shadow
// would strand the shadow's writes in live's directory.
func TestSQLitePathShadowIgnoresLifecycleDataDir(t *testing.T) {
	liveData := t.TempDir()
	env := map[string]string{
		"SCENARIO_NAME":     "example-scenario",
		EnvScenario:         "example-scenario",
		"SCENARIO_DATA_DIR": liveData,
		EnvStorageNamespace: "example-scenario_shadow",
		EnvVariant:          "shadow",
	}

	path, err := SQLitePath(SQLiteConfig{
		Scenario: "example-scenario",
		EnvGet:   func(key string) string { return env[key] },
	})
	if err != nil {
		t.Fatalf("SQLitePath: %v", err)
	}
	if strings.HasPrefix(path, liveData) {
		t.Fatalf("shadow resolved into live's data directory: %s", path)
	}
	if !strings.Contains(path, "example-scenario_shadow") {
		t.Fatalf("shadow path is not variant-scoped: %s", path)
	}
}

// TestSQLitePathRootOverrideBeatsLifecycleDataDir keeps the test seam
// authoritative: an explicitly passed root must win over ambient state.
func TestSQLitePathRootOverrideBeatsLifecycleDataDir(t *testing.T) {
	lifecycleData := t.TempDir()
	override := t.TempDir()
	env := map[string]string{
		"SCENARIO_NAME":     "example-scenario",
		EnvScenario:         "example-scenario",
		"SCENARIO_DATA_DIR": lifecycleData,
	}

	path, err := SQLitePath(SQLiteConfig{
		Scenario:     "example-scenario",
		RootOverride: override,
		EnvGet:       func(key string) string { return env[key] },
	})
	if err != nil {
		t.Fatalf("SQLitePath: %v", err)
	}
	if !strings.HasPrefix(path, override) {
		t.Fatalf("expected the explicit root override to win, got %s", path)
	}
}

// TestSQLitePathNonDataClassIgnoresLifecycleDataDir keeps the carve-out narrow:
// the assigned directory is the DATA class root, and nothing else.
func TestSQLitePathNonDataClassIgnoresLifecycleDataDir(t *testing.T) {
	lifecycleData := t.TempDir()
	env := map[string]string{
		"SCENARIO_NAME":     "example-scenario",
		EnvScenario:         "example-scenario",
		"SCENARIO_DATA_DIR": lifecycleData,
	}
	path, err := SQLitePath(SQLiteConfig{
		Scenario: "example-scenario",
		Class:    ClassCache,
		EnvGet:   func(key string) string { return env[key] },
	})
	if err != nil {
		t.Fatalf("SQLitePath: %v", err)
	}
	if strings.HasPrefix(path, lifecycleData) {
		t.Fatalf("a cache-class database landed in the data directory: %s", path)
	}
}

// --- inherited namespace ------------------------------------------------

// TestSQLitePathRejectsForeignNamespace closes the third inheritance channel.
// VROOLI_STORAGE_NAMESPACE cannot alias two scenarios onto one FILE, but an
// inherited one still names the wrong scenario and would place this scenario's
// database inside the supervisor's directory.
func TestSQLitePathRejectsForeignNamespace(t *testing.T) {
	env := map[string]string{
		EnvStorageNamespace: "vrooli-autoheal",
		EnvScenario:         "vrooli-autoheal",
		"SCENARIO_NAME":     "vrooli-autoheal",
	}
	path, err := SQLitePath(SQLiteConfig{
		Scenario: "victim-scenario",
		EnvGet:   func(key string) string { return env[key] },
	})
	if err != nil {
		t.Fatalf("SQLitePath: %v", err)
	}
	if strings.Contains(path, "vrooli-autoheal") {
		t.Fatalf("an inherited namespace placed this scenario under a sibling: %s", path)
	}
	if !strings.Contains(path, "victim-scenario") {
		t.Fatalf("expected this scenario's own namespace, got %s", path)
	}
}

// TestSQLitePathRejectsForeignShadowNamespace covers the same leak arriving
// with a variant suffix, and confirms it does not become a hard startup error:
// a supervisor restarting a sick scenario is routine, and refusing to start
// would turn a misplaced directory into an outage.
func TestSQLitePathRejectsForeignShadowNamespace(t *testing.T) {
	env := map[string]string{
		EnvStorageNamespace: "vrooli-autoheal_shadow",
		EnvVariant:          "shadow",
		EnvScenario:         "vrooli-autoheal",
		"SCENARIO_NAME":     "vrooli-autoheal",
	}
	path, err := SQLitePath(SQLiteConfig{
		Scenario: "victim-scenario",
		EnvGet:   func(key string) string { return env[key] },
	})
	if err != nil {
		t.Fatalf("expected a routine fallback, not a startup failure: %v", err)
	}
	if strings.Contains(path, "autoheal") {
		t.Fatalf("an inherited shadow namespace captured this scenario: %s", path)
	}
}

// TestSQLitePathNamespacePrefixIsNotSubstring guards the separator check.
// Without it, scenario "web" would accept "web-console"'s namespace root.
func TestSQLitePathNamespacePrefixIsNotSubstring(t *testing.T) {
	env := map[string]string{
		EnvStorageNamespace: "web-console",
		EnvScenario:         "web-console",
		"SCENARIO_NAME":     "web-console",
	}
	path, err := SQLitePath(SQLiteConfig{
		Scenario: "web",
		EnvGet:   func(key string) string { return env[key] },
	})
	if err != nil {
		t.Fatalf("SQLitePath: %v", err)
	}
	if strings.Contains(path, "web-console") {
		t.Fatalf("scenario \"web\" claimed \"web-console\"'s namespace: %s", path)
	}
}

// TestSQLitePathKeepsOwnShadowNamespace confirms the guard did not break the
// mechanism it protects: this scenario's OWN shadow namespace is still used.
func TestSQLitePathKeepsOwnShadowNamespace(t *testing.T) {
	env := map[string]string{
		EnvStorageNamespace: "example-scenario_shadow",
		EnvVariant:          "shadow",
		EnvScenario:         "example-scenario",
		"SCENARIO_NAME":     "example-scenario",
	}
	path, err := SQLitePath(SQLiteConfig{
		Scenario: "example-scenario",
		EnvGet:   func(key string) string { return env[key] },
	})
	if err != nil {
		t.Fatalf("SQLitePath: %v", err)
	}
	if !strings.Contains(path, "example-scenario_shadow") {
		t.Fatalf("this scenario's own shadow namespace was discarded: %s", path)
	}
}

// --- read-only inspection ------------------------------------------------

func TestSQLiteReadOnlyDSNAtOnAQuiescentDatabase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subject.db")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	dsn, err := SQLiteReadOnlyDSNAt(path)
	if err != nil {
		t.Fatalf("SQLiteReadOnlyDSNAt: %v", err)
	}
	for _, want := range []string{"mode=ro", "immutable=1", "query_only(1)"} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("expected %q in %s", want, dsn)
		}
	}
	if strings.Contains(dsn, "nolock") {
		t.Fatalf("a quiescent database should not need nolock: %s", dsn)
	}
}

// TestSQLiteReadOnlyDSNAtOnALiveDatabase is the correctness case that makes the
// locking mode conditional. immutable=1 tells SQLite the file cannot change,
// which lets it skip the WAL — so a reader would silently miss every committed
// transaction still in the WAL and report a stale view as authoritative.
func TestSQLiteReadOnlyDSNAtOnALiveDatabase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subject.db")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed db: %v", err)
	}
	if err := os.WriteFile(path+"-wal", []byte("pending"), 0o600); err != nil {
		t.Fatalf("seed wal: %v", err)
	}

	dsn, err := SQLiteReadOnlyDSNAt(path)
	if err != nil {
		t.Fatalf("SQLiteReadOnlyDSNAt: %v", err)
	}
	if strings.Contains(dsn, "immutable=1") {
		t.Fatalf("immutable=1 on a database with a WAL would skip committed data: %s", dsn)
	}
	for _, want := range []string{"mode=ro", "nolock=1", "query_only(1)"} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("expected %q in %s", want, dsn)
		}
	}
}

func TestSQLiteReadOnlyDSNAtCreatesNothing(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "absent", "subject.db")
	if _, err := SQLiteReadOnlyDSNAt(nested); err != nil {
		t.Fatalf("SQLiteReadOnlyDSNAt: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(nested)); err == nil {
		t.Fatal("read-only inspection created a directory")
	}
}

func TestSQLiteReadOnlyDSNAtRejectsBadInput(t *testing.T) {
	if _, err := SQLiteReadOnlyDSNAt("  "); err == nil {
		t.Fatal("expected an empty path to be rejected")
	}
	if _, err := SQLiteReadOnlyDSNAt("file:/tmp/x.db?mode=ro"); err == nil {
		t.Fatal("expected an assembled DSN to be rejected")
	}
}

// TestSQLitePathSeparatorsAreRejected covers the characters that terminate a
// path inside a DSN. Accepting one would silently address a different file
// rather than fail.
func TestSQLiteDSNRejectsDSNTerminatingCharacters(t *testing.T) {
	for _, path := range []string{"/tmp/we?ird.db", "/tmp/we#ird.db"} {
		if _, err := SQLiteDSNAt(path, SQLiteTuning{}); err == nil {
			t.Fatalf("expected %q to be rejected", path)
		}
		if _, err := SQLiteReadOnlyDSNAt(path); err == nil {
			t.Fatalf("expected %q to be rejected for read-only too", path)
		}
	}
}
