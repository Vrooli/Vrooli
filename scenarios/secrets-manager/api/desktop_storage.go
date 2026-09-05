package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"secrets-manager-api/internal/secrets"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
)

// openDesktopDatabase creates the scenario-private metadata store used by a
// bundled Secrets Manager. APP_DATA_DIR is owned by the desktop runtime and is
// resolved through api-core/storage, so a desktop bundle never aliases either
// the control-plane PostgreSQL database or a sibling scenario's private data.
func openDesktopDatabase(ctx context.Context) (*database.RoutedDB, error) {
	path, err := desktopDatabasePath(ctx)
	if err != nil {
		return nil, err
	}
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		return nil, err
	}
	db, err := database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, storage.SecretFilePerm); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("restrict desktop database permissions: %w", err)
	}
	if err := initializeDesktopSchema(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func desktopDatabasePath(ctx context.Context) (string, error) {
	appData := strings.TrimSpace(os.Getenv("APP_DATA_DIR"))
	if appData == "" {
		return "", fmt.Errorf("APP_DATA_DIR is required for desktop storage")
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileDesktop})
	if err != nil {
		return "", fmt.Errorf("create desktop storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("secrets-manager")
	if err != nil {
		return "", fmt.Errorf("resolve desktop storage namespace: %w", err)
	}
	opts := storage.Options{ScenarioID: scenarioID, RootOverride: appData}
	if _, err := storage.EnsureClassDir(resolver, opts, storage.ClassData, 0o700); err != nil {
		return "", fmt.Errorf("create desktop database directory: %w", err)
	}
	path, err := resolver.Path(opts, storage.ClassData, "secrets-manager.sqlite")
	if err != nil {
		return "", fmt.Errorf("resolve desktop database path: %w", err)
	}
	if err := migrateLegacyDesktopDatabase(ctx, filepath.Join(appData, "runtime", "api", "secrets-manager.sqlite"), path); err != nil {
		return "", err
	}
	return path, nil
}

// migrateLegacyDesktopDatabase preserves metadata written by the first desktop
// bundle implementation, which used APP_DATA_DIR/runtime/api directly. The
// checkpoint makes a SQLite WAL database self-contained before its main file is
// moved into the shared storage namespace.
func migrateLegacyDesktopDatabase(ctx context.Context, legacyPath, destinationPath string) error {
	if _, err := os.Stat(destinationPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect desktop database destination: %w", err)
	}
	if _, err := os.Stat(legacyPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect legacy desktop database: %w", err)
	}

	legacyDB, err := database.Connect(ctx, database.Config{Driver: database.DriverSQLite, DSN: "file:" + legacyPath, MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		return fmt.Errorf("open legacy desktop database: %w", err)
	}
	if _, err := legacyDB.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		_ = legacyDB.Close()
		return fmt.Errorf("checkpoint legacy desktop database: %w", err)
	}
	if err := legacyDB.Close(); err != nil {
		return fmt.Errorf("close legacy desktop database: %w", err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := os.Remove(legacyPath + suffix); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove checkpointed legacy database sidecar: %w", err)
		}
	}
	if err := os.Rename(legacyPath, destinationPath); err != nil {
		return fmt.Errorf("migrate legacy desktop database: %w", err)
	}
	return nil
}

func initializeDesktopSchema(ctx context.Context, db database.SchemaExecer) error {
	if _, err := db.ExecContext(ctx, secrets.DesktopSchema()); err != nil {
		return fmt.Errorf("initialize desktop schema: %w", err)
	}
	return nil
}
