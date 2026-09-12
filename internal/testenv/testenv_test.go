package testenv

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	_ "modernc.org/sqlite" // Registers the driver exercised by the SQLite fixture tests.
)

func TestCredentialStoreSupportsValuesAndFailureInjection(t *testing.T) {
	notFound := errors.New("not found")
	store := NewCredentialStore(notFound)
	if _, err := store.Get("service", "missing"); !errors.Is(err, notFound) {
		t.Fatalf("Get(missing) error = %v, want injected sentinel", err)
	}
	if err := store.Put("service", "key", "value"); err != nil {
		t.Fatal(err)
	}
	if got, err := store.Get("service", "key"); err != nil || got != "value" {
		t.Fatalf("Get() = %q, %v", got, err)
	}
	store.GetError = errors.New("backend unavailable")
	if _, err := store.Get("service", "key"); err == nil || err.Error() != "backend unavailable" {
		t.Fatalf("Get() error = %v, want injected backend failure", err)
	}
}

func TestClockAdvancesDeterministically(t *testing.T) {
	start := time.Date(2026, 8, 29, 12, 0, 0, 0, time.FixedZone("fixture", -4*60*60))
	clock := NewClock(start)
	clock.Advance(90 * time.Second)
	if got, want := clock.Now(), start.UTC().Add(90*time.Second); !got.Equal(want) {
		t.Fatalf("Now() = %s, want %s", got, want)
	}
}

func TestDecodeJSONReturnsTypedFixture(t *testing.T) {
	got := DecodeJSON[struct {
		Name string `json:"name"`
	}](t, []byte(`{"name":"fixture"}`))
	if got.Name != "fixture" {
		t.Fatalf("Name = %q", got.Name)
	}
}

func TestSQLiteHarnessOwnsPathCleanupAndSchemaStamp(t *testing.T) {
	var dbPath string
	db := NewSQLiteStore(t, "fixture.db", func(path string) (*sql.DB, error) {
		dbPath = path
		opened, err := sql.Open("sqlite", path)
		if err != nil {
			return nil, err
		}
		if _, err := opened.Exec(`CREATE TABLE fixture (id INTEGER PRIMARY KEY)`); err != nil {
			_ = opened.Close()
			return nil, err
		}
		return opened, nil
	})
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	StampSQLiteUserVersion(t, dbPath, 7)
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 7 {
		t.Fatalf("user_version = %d, want 7", version)
	}
}

func TestAsSudoUserSetsCompleteIdentity(t *testing.T) {
	AsSudoUser(t, "alice")
	for key, want := range map[string]string{
		"SUDO_USER": "alice",
		"SUDO_UID":  "1000",
		"SUDO_GID":  "1000",
		"USER":      "root",
	} {
		if got := os.Getenv(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestAsCurrentUserClearsInheritedSudoIdentity(t *testing.T) {
	AsSudoUser(t, "stale")
	AsCurrentUser(t, "alice")
	for key, want := range map[string]string{
		"USER": "alice", "SUDO_USER": "", "SUDO_UID": "", "SUDO_GID": "",
	} {
		if got := os.Getenv(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestSetIdentityEnvSetsExactResolverValues(t *testing.T) {
	SetIdentityEnv(t, map[string]string{"HOME": "/fixture/home", "XDG_CACHE_HOME": "relative/cache"})
	if got := os.Getenv("HOME"); got != "/fixture/home" {
		t.Fatalf("HOME = %q", got)
	}
	if got := os.Getenv("XDG_CACHE_HOME"); got != "relative/cache" {
		t.Fatalf("XDG_CACHE_HOME = %q", got)
	}
}

func TestRuntimeHomeSetsXDGLocations(t *testing.T) {
	home := RuntimeHome(t)
	if os.Getenv("HOME") != home {
		t.Fatalf("HOME = %q, want %q", os.Getenv("HOME"), home)
	}
	for key, suffix := range map[string]string{
		"XDG_CACHE_HOME":  filepath.Join(".cache"),
		"XDG_DATA_HOME":   filepath.Join(".local", "share"),
		"XDG_RUNTIME_DIR": filepath.Join(".runtime"),
	} {
		path := os.Getenv(key)
		if path != filepath.Join(home, suffix) {
			t.Errorf("%s = %q, want under %q", key, path, home)
		}
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Errorf("%s directory %q is not present: %v", key, path, err)
		}
	}
}

func TestSudoRuntimeHomeSetsCompleteEnvironment(t *testing.T) {
	home := SudoRuntimeHome(t, "alice")
	for key, want := range map[string]string{
		"HOME":      home,
		"SUDO_USER": "alice",
		"SUDO_UID":  "1000",
		"SUDO_GID":  "1000",
		"USER":      "root",
	} {
		if got := os.Getenv(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	for _, key := range []string{"XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_RUNTIME_DIR"} {
		got := os.Getenv(key)
		rel, err := filepath.Rel(home, got)
		if got == "" || err != nil || rel == ".." || len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator) {
			t.Errorf("%s = %q, want a path under %q", key, got, home)
		}
	}
}

func TestNewRepositoryTreeUsesContractPaths(t *testing.T) {
	tree := NewRepositoryTree(t, "fixture")
	wantScenarioRoot := repocontract.ScenarioRoot(tree.Root, tree.Scenario)
	if tree.ScenarioRoot != wantScenarioRoot {
		t.Fatalf("ScenarioRoot = %q, want %q", tree.ScenarioRoot, wantScenarioRoot)
	}
	wantScenarioService, err := repocontract.ScenarioServiceManifestPath(tree.Root, tree.Scenario)
	if err != nil {
		t.Fatal(err)
	}
	if tree.ScenarioServicePath != wantScenarioService {
		t.Fatalf("ScenarioServicePath = %q, want %q", tree.ScenarioServicePath, wantScenarioService)
	}
	for _, path := range []string{tree.ProjectServicePath, tree.ScenarioServicePath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("fixture path %q: %v", path, err)
		}
	}
}

func TestNewRepositoryTreeAcceptsServiceOverrides(t *testing.T) {
	tree := NewRepositoryTree(t, "fixture", WithScenarioServiceJSON([]byte(`{"service":"scenario"}`)))
	data, err := os.ReadFile(tree.ScenarioServicePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"service":"scenario"}` {
		t.Fatalf("scenario service = %q", data)
	}
}
