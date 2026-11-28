package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"decentralized-deploy/internal/modules"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/api-core/storage/storagetest"
)

func TestMain(m *testing.M) { os.Exit(storagetest.RunWithIsolatedStorage(m)) }

func TestSelectedKuboBinaryUsesOwnerSelection(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "ipfs")
	if err := os.WriteFile(binary, []byte("managed test executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	selection, err := json.Marshal(hostToolSelection{Command: binary, Installed: true, Version: "0.43.1", OK: true})
	if err != nil {
		t.Fatal(err)
	}
	got, err := selectedKuboBinary(selection, "")
	if err != nil || got != binary {
		t.Fatalf("selected Kubo binary=%q err=%v", got, err)
	}
	if _, err := selectedKuboBinary(selection, "/other/ipfs"); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("mismatched override error=%v", err)
	}
	selection, err = json.Marshal(hostToolSelection{Command: binary, Installed: false, Version: "0.43.1", OK: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := selectedKuboBinary(selection, ""); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("uninstalled owner selection error=%v", err)
	}
}

func testRuntimeDeps(t *testing.T) (runtimeDeps, *[]string, **database.RoutedDB, **sql.DB) {
	t.Helper()
	var order []string
	var scenario string
	var opened *database.RoutedDB
	var primary *sql.DB
	deps := productionRuntimeDeps()
	deps.preflightRun = func(cfg preflight.Config) bool {
		order = append(order, "preflight")
		scenario = cfg.ScenarioName
		if scenario == "" {
			t.Error("preflight scenario name is empty")
		}
		return false
	}
	deps.openDatabase = func(ctx context.Context, cfg database.Config) (*database.RoutedDB, error) {
		order = append(order, "database")
		if cfg.Scenario == "" || cfg.Scenario != scenario || cfg.Driver != database.DriverSQLite {
			t.Errorf("database config = %+v, preflight scenario = %q", cfg, scenario)
		}
		cfg.DSN = filepath.Join(t.TempDir(), "career-desk-test.sqlite")
		var err error
		opened, err = database.Open(ctx, cfg)
		if opened != nil {
			primary = opened.Primary()
		}
		return opened, err
	}
	deps.storageRoots = func() (storage.Paths, error) {
		order = append(order, "storage")
		root := t.TempDir()
		return storage.Paths{ConfigDir: root, DataDir: root, CacheDir: root, LogsDir: root, StateDir: root, TestRunsDir: root}, nil
	}
	deps.authConfig = func(func(string) string) (authn.Config, error) {
		order = append(order, "auth")
		return authn.Config{}, nil
	}
	return deps, &order, &opened, &primary
}

func TestRunStartupSuccessWithoutBindingPort(t *testing.T) {
	deps, order, opened, primary := testRuntimeDeps(t)
	deps.runServer = func(cfg apiserver.Config) error {
		*order = append(*order, "server")
		if cfg.Handler == nil {
			t.Fatal("server handler is nil")
		}
		if cfg.Authentication != nil {
			t.Fatal("disabled test authentication should not be passed to server")
		}
		if cfg.Cleanup == nil {
			t.Fatal("server cleanup callback is nil")
		}
		return cfg.Cleanup(context.Background())
	}

	if err := run(deps); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if want := []string{"preflight", "database", "storage", "auth", "server"}; !equalStrings(*order, want) {
		t.Fatalf("startup order = %v, want %v", *order, want)
	}
	if *opened == nil {
		t.Fatal("database was not opened")
	}
	assertDatabaseClosed(t, *opened, *primary)
}

func TestRunConfiguredAuthenticationWithoutBindingPort(t *testing.T) {
	deps, _, _, _ := testRuntimeDeps(t)
	deps.authConfig = func(func(string) string) (authn.Config, error) {
		return authn.Config{Providers: []authn.Provider{authn.TokenProvider{}}}, nil
	}
	deps.runServer = func(cfg apiserver.Config) error {
		if cfg.Authentication == nil || !cfg.Authentication.Enabled() {
			t.Fatal("configured authentication was not passed to server")
		}
		return cfg.Cleanup(context.Background())
	}
	if err := run(deps); err != nil {
		t.Fatalf("run() error = %v", err)
	}
}

func TestRunPreflightStopsBeforeDatabase(t *testing.T) {
	deps, order, _, _ := testRuntimeDeps(t)
	deps.preflightRun = func(preflight.Config) bool {
		*order = append(*order, "preflight")
		return true
	}
	deps.openDatabase = func(context.Context, database.Config) (*database.RoutedDB, error) {
		t.Fatal("database opened after preflight requested re-exec")
		return nil, nil
	}
	if err := run(deps); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if want := []string{"preflight"}; !equalStrings(*order, want) {
		t.Fatalf("startup order = %v, want %v", *order, want)
	}
}

func TestRunDatabaseOpenError(t *testing.T) {
	deps, order, _, _ := testRuntimeDeps(t)
	wantErr := errors.New("open refused")
	deps.openDatabase = func(context.Context, database.Config) (*database.RoutedDB, error) {
		*order = append(*order, "database")
		return nil, wantErr
	}
	if err := run(deps); !errors.Is(err, wantErr) {
		t.Fatalf("run() error = %v, want wrapped %v", err, wantErr)
	}
	if want := []string{"preflight", "database"}; !equalStrings(*order, want) {
		t.Fatalf("startup order = %v, want %v", *order, want)
	}
}

func TestRunSchemaErrorClosesDatabase(t *testing.T) {
	wantErr := errors.New("schema refused")
	assertFailureClosesDatabase(t, "schema", wantErr)
}

func TestRunStorageErrorClosesDatabase(t *testing.T) {
	wantErr := errors.New("storage refused")
	assertFailureClosesDatabase(t, "storage", wantErr)
}

func TestRunMeasuresErrorClosesDatabase(t *testing.T) {
	wantErr := errors.New("measures refused")
	assertFailureClosesDatabase(t, "measures", wantErr)
}

func TestRunAuthErrorClosesDatabase(t *testing.T) {
	wantErr := errors.New("auth refused")
	assertFailureClosesDatabase(t, "auth", wantErr)
}

func TestRunServerErrorClosesDatabase(t *testing.T) {
	wantErr := errors.New("server refused")
	deps, order, opened, primary := testRuntimeDeps(t)
	deps.runServer = func(apiserver.Config) error {
		*order = append(*order, "server")
		return wantErr
	}
	if err := run(deps); !errors.Is(err, wantErr) {
		t.Fatalf("run() error = %v, want wrapped %v", err, wantErr)
	}
	assertDatabaseClosed(t, *opened, *primary)
}

func assertFailureClosesDatabase(t *testing.T, failure string, wantErr error) {
	t.Helper()
	deps, _, opened, primary := testRuntimeDeps(t)
	switch failure {
	case "schema":
		deps.ensureSchemas = func(context.Context, *database.RoutedDB) error { return wantErr }
	case "storage":
		deps.storageRoots = func() (storage.Paths, error) { return storage.Paths{}, wantErr }
	case "measures":
		deps.measuresHandler = func(*database.RoutedDB) (http.Handler, error) { return nil, wantErr }
	case "auth":
		deps.authConfig = func(func(string) string) (authn.Config, error) { return authn.Config{}, wantErr }
	default:
		t.Fatalf("unknown failure branch %q", failure)
	}
	if err := run(deps); !errors.Is(err, wantErr) {
		t.Fatalf("run() error = %v, want wrapped %v", err, wantErr)
	}
	assertDatabaseClosed(t, *opened, *primary)
}

func assertDatabaseClosed(t *testing.T, db *database.RoutedDB, primary *sql.DB) {
	t.Helper()
	if db == nil {
		t.Fatal("database was not opened")
	}
	if primary == nil {
		t.Fatal("primary database pool was not captured")
	}
	if err := primary.PingContext(context.Background()); err == nil {
		t.Fatal("database remained open after bootstrap failure")
	}
}

func TestScenarioStorageRoots(t *testing.T) {
	paths, err := scenarioStorageRoots()
	if err != nil {
		t.Fatalf("scenarioStorageRoots() error = %v", err)
	}
	for _, class := range []storage.Class{storage.ClassConfig, storage.ClassData, storage.ClassCache, storage.ClassLogs, storage.ClassState, storage.ClassTestRuns} {
		path, err := paths.ForClass(class)
		if err != nil {
			t.Fatalf("ForClass(%q) error = %v", class, err)
		}
		if path == "" {
			t.Errorf("ForClass(%q) returned an empty path", class)
		}
	}
}

func TestEnsureScenarioSchemasAndMeasuresHandler(t *testing.T) {
	tempDSN := filepath.Join(t.TempDir(), "schemas.sqlite")
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: tempDSN})
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer db.Close()
	if err := ensureScenarioSchemas(context.Background(), db); err != nil {
		t.Fatalf("ensureScenarioSchemas() error = %v", err)
	}
	if err := ensureScenarioSchemas(context.Background(), db); err != nil {
		t.Fatalf("ensureScenarioSchemas() second call error = %v", err)
	}
	if _, err := notesMeasuresHandler(db); err != nil {
		t.Fatalf("notesMeasuresHandler() error = %v", err)
	}
	if len(modules.AllSchemas()) == 0 {
		t.Fatal("scenario schema registry is empty")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestRunErrorMessagesNameStartupStage(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{name: "schema", want: "schema initialization failed"},
		{name: "storage", want: "file storage configuration failed"},
		{name: "measures", want: "measures registry"},
		{name: "auth", want: "authentication configuration failed"},
		{name: "server", want: "server error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deps, _, _, _ := testRuntimeDeps(t)
			failing := errors.New(tc.name + " failed")
			switch tc.name {
			case "schema":
				deps.ensureSchemas = func(context.Context, *database.RoutedDB) error { return failing }
			case "storage":
				deps.storageRoots = func() (storage.Paths, error) { return storage.Paths{}, failing }
			case "measures":
				deps.measuresHandler = func(*database.RoutedDB) (http.Handler, error) { return nil, failing }
			case "auth":
				deps.authConfig = func(func(string) string) (authn.Config, error) { return authn.Config{}, failing }
			case "server":
				deps.runServer = func(apiserver.Config) error { return failing }
			}
			err := run(deps)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("run() error = %v, want context %q", err, tc.want)
			}
		})
	}
}
