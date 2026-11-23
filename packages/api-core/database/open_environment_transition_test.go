package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/vrooli/api-core/retry"
)

func TestSQLiteRecoveryAfterRepositoryRelocationRetainsSavedWork(t *testing.T) {
	storageRoot := t.TempDir()
	environment := map[string]string{
		"VROOLI_STORAGE_ROOT": storageRoot,
		"VROOLI_ROOT":         "/checkout/one",
	}
	config := Config{
		Driver:   DriverSQLite,
		Scenario: "vrooli-onboarding",
		EnvGetter: func(key string) string {
			return environment[key]
		},
	}

	first, err := Open(context.Background(), config)
	if err != nil {
		t.Fatalf("open initial environment: %v", err)
	}
	if _, err := first.ExecContext(context.Background(), `CREATE TABLE profile_work (id INTEGER PRIMARY KEY, answer TEXT NOT NULL)`); err != nil {
		_ = first.Close()
		t.Fatalf("create saved-work table: %v", err)
	}
	if _, err := first.ExecContext(context.Background(), `INSERT INTO profile_work (id, answer) VALUES (1, 'retain this answer')`); err != nil {
		_ = first.Close()
		t.Fatalf("save work: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close initial owner: %v", err)
	}

	// A restarted owner first encounters a provider-open failure. The failed
	// attempt must not redirect or erase the state owned by the pinned root.
	environment["VROOLI_ROOT"] = "/checkout/two"
	failed := config
	injected := errors.New("injected SQLite provider outage")
	failed.Opener = func(string, string) (*sql.DB, error) { return nil, injected }
	failed.Retry = &retry.Config{MaxAttempts: 1}
	if _, err := OpenWithClock(context.Background(), failed, nil); !errors.Is(err, injected) {
		t.Fatalf("unavailable provider error = %v, want injected outage", err)
	}

	recovered, err := OpenWithClock(context.Background(), config, nil)
	if err != nil {
		t.Fatalf("reopen after provider recovery: %v", err)
	}
	defer recovered.Close()
	var answer string
	if err := recovered.QueryRowContext(context.Background(), `SELECT answer FROM profile_work WHERE id = 1`).Scan(&answer); err != nil {
		t.Fatalf("read retained work: %v", err)
	}
	if answer != "retain this answer" {
		t.Fatalf("recovered answer = %q, want retained work", answer)
	}
}
