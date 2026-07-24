package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/database"
	"secrets-manager-api/internal/testutil"
)

func TestDesktopDatabaseProvidesPrivatePortableSecretMetadata(t *testing.T) {
	t.Setenv("VROOLI_DESKTOP_MODE", "true")
	t.Setenv("APP_DATA_DIR", t.TempDir())
	db, err := openDesktopDatabase(context.Background())
	testutil.RequireNoError(t, err, "open desktop database")
	defer db.Close()

	_, err = db.ExecContext(context.Background(), `INSERT INTO resource_secrets (id, resource_name, secret_key, secret_type, required) VALUES ('secret-1', 'vault', 'VAULT_TOKEN', 'token', 1)`)
	testutil.RequireNoError(t, err, "insert resource secret")
	_, err = db.ExecContext(context.Background(), `INSERT INTO secret_deployment_strategies (id, resource_secret_id, tier, handling_strategy) VALUES ('strategy-1', 'secret-1', 'tier-2-desktop', 'prompt')`)
	testutil.RequireNoError(t, err, "insert deployment strategy")

	detail, err := fetchResourceDetail(context.Background(), db, "vault")
	testutil.RequireNoError(t, err, "fetch resource detail")
	if len(detail.Secrets) != 1 || detail.Secrets[0].TierStrategies["tier-2-desktop"] != "prompt" {
		t.Fatalf("unexpected desktop secret detail: %#v", detail)
	}

	databasePath, err := desktopDatabasePath(context.Background())
	if err != nil {
		t.Fatalf("resolve desktop database path: %v", err)
	}
	if _, err := os.Stat(databasePath); err != nil {
		t.Fatalf("desktop database missing from private app data: %v", err)
	}
	info, err := os.Stat(databasePath)
	if err != nil {
		t.Fatalf("stat desktop database: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("desktop database permissions = %o, want 600", got)
	}
}

func TestDesktopDatabaseMigratesLegacyPrivateState(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("VROOLI_DESKTOP_MODE", "true")
	t.Setenv("APP_DATA_DIR", appData)
	legacyPath := filepath.Join(appData, "runtime", "api", "secrets-manager.sqlite")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o700); err != nil {
		t.Fatalf("create legacy directory: %v", err)
	}
	legacyDB, err := database.Connect(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: "file:" + legacyPath, MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	if _, err := legacyDB.Exec(`CREATE TABLE resource_secrets (id TEXT PRIMARY KEY, resource_name TEXT NOT NULL, secret_key TEXT NOT NULL, secret_type TEXT NOT NULL, required BOOLEAN NOT NULL DEFAULT 1, UNIQUE(resource_name, secret_key))`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	if _, err := legacyDB.Exec(`INSERT INTO resource_secrets (id, resource_name, secret_key, secret_type, required) VALUES ('legacy-secret', 'vault', 'VAULT_TOKEN', 'token', 1)`); err != nil {
		t.Fatalf("seed legacy data: %v", err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("close legacy database: %v", err)
	}

	db, err := openDesktopDatabase(context.Background())
	if err != nil {
		t.Fatalf("open migrated desktop database: %v", err)
	}
	defer db.Close()
	var secretKey string
	if err := db.QueryRowContext(context.Background(), `SELECT secret_key FROM resource_secrets WHERE id = 'legacy-secret'`).Scan(&secretKey); err != nil {
		t.Fatalf("read migrated data: %v", err)
	}
	if secretKey != "VAULT_TOKEN" {
		t.Fatalf("migrated secret key = %q, want VAULT_TOKEN", secretKey)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy database should be moved, stat error: %v", err)
	}
	newPath, err := desktopDatabasePath(context.Background())
	if err != nil {
		t.Fatalf("resolve migrated desktop database path: %v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("migrated database missing: %v", err)
	}
}
