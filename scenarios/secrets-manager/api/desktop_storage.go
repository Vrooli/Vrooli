package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	db, err := database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)",
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
	for _, statement := range desktopSchemaStatements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("initialize desktop schema: %w", err)
		}
	}
	return nil
}

var desktopSchemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS resource_secrets (
		id TEXT PRIMARY KEY, resource_name TEXT NOT NULL, secret_key TEXT NOT NULL,
		secret_type TEXT NOT NULL, required BOOLEAN NOT NULL DEFAULT 1, description TEXT,
		validation_pattern TEXT, documentation_url TEXT, default_value TEXT,
		classification TEXT NOT NULL DEFAULT 'service', owner_team TEXT, owner_contact TEXT,
		rotation_period_days INTEGER DEFAULT 0, last_rotated_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(resource_name, secret_key)
	)`,
	`CREATE TABLE IF NOT EXISTS secret_validations (
		id TEXT PRIMARY KEY, resource_secret_id TEXT NOT NULL REFERENCES resource_secrets(id) ON DELETE CASCADE,
		validation_status TEXT NOT NULL, validation_method TEXT NOT NULL,
		validation_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, error_message TEXT, validation_details TEXT
	)`,
	`CREATE TABLE IF NOT EXISTS secret_scans (
		id TEXT PRIMARY KEY, scan_type TEXT NOT NULL DEFAULT 'full', resources_scanned TEXT,
		secrets_discovered INTEGER DEFAULT 0, scan_duration_ms INTEGER,
		scan_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, scan_status TEXT NOT NULL DEFAULT 'completed',
		error_message TEXT, scan_metadata TEXT
	)`,
	`CREATE TABLE IF NOT EXISTS secret_provisions (
		id TEXT PRIMARY KEY, resource_secret_id TEXT NOT NULL REFERENCES resource_secrets(id) ON DELETE CASCADE,
		storage_method TEXT NOT NULL, storage_location TEXT NOT NULL,
		provisioned_at DATETIME DEFAULT CURRENT_TIMESTAMP, provisioned_by TEXT,
		provision_status TEXT NOT NULL DEFAULT 'active', expiration_date DATETIME
	)`,
	`CREATE TABLE IF NOT EXISTS secret_deployment_strategies (
		id TEXT PRIMARY KEY, resource_secret_id TEXT NOT NULL REFERENCES resource_secrets(id) ON DELETE CASCADE,
		tier TEXT NOT NULL, handling_strategy TEXT NOT NULL, fallback_strategy TEXT,
		requires_user_input BOOLEAN NOT NULL DEFAULT 0, prompt_label TEXT, prompt_description TEXT,
		generator_template TEXT, bundle_hints TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP, UNIQUE(resource_secret_id, tier)
	)`,
	`CREATE TABLE IF NOT EXISTS scenario_secret_strategy_overrides (
		id TEXT PRIMARY KEY, scenario_name TEXT NOT NULL,
		resource_secret_id TEXT NOT NULL REFERENCES resource_secrets(id) ON DELETE CASCADE,
		tier TEXT NOT NULL, handling_strategy TEXT, fallback_strategy TEXT, requires_user_input BOOLEAN,
		prompt_label TEXT, prompt_description TEXT, generator_template TEXT, bundle_hints TEXT,
		override_reason TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP, UNIQUE(scenario_name, resource_secret_id, tier)
	)`,
	`CREATE TABLE IF NOT EXISTS deployment_manifests (
		id TEXT PRIMARY KEY, scenario_name TEXT NOT NULL, tier TEXT NOT NULL, manifest TEXT NOT NULL,
		generated_by TEXT, generated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS security_scan_runs (
		id TEXT PRIMARY KEY, scan_id TEXT NOT NULL, component_filter TEXT, component_type TEXT,
		severity_filter TEXT, files_scanned INTEGER DEFAULT 0, files_skipped INTEGER DEFAULT 0,
		vulnerabilities_found INTEGER DEFAULT 0, risk_score INTEGER DEFAULT 0, duration_ms INTEGER DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'completed', error_message TEXT, metadata TEXT,
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP, completed_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS security_vulnerabilities (
		id TEXT PRIMARY KEY, scan_run_id TEXT, fingerprint TEXT NOT NULL UNIQUE,
		component_type TEXT NOT NULL, component_name TEXT NOT NULL, file_path TEXT NOT NULL,
		line_number INTEGER, severity TEXT NOT NULL, vulnerability_type TEXT NOT NULL, title TEXT NOT NULL,
		description TEXT, recommendation TEXT, code_snippet TEXT, can_auto_fix BOOLEAN DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'open', assigned_to TEXT, fix_request_id TEXT,
		first_observed_at DATETIME DEFAULT CURRENT_TIMESTAMP, last_observed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		resolved_at DATETIME, metadata TEXT
	)`,
	`CREATE TABLE IF NOT EXISTS deployment_campaigns (
		id TEXT PRIMARY KEY, scenario TEXT NOT NULL, tier TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'unknown',
		progress INTEGER NOT NULL DEFAULT 0, blockers INTEGER NOT NULL DEFAULT 0, next_action TEXT,
		last_step TEXT, summary TEXT, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, UNIQUE(scenario, tier)
	)`,
	`CREATE VIEW IF NOT EXISTS secret_health_summary AS
	SELECT rs.resource_name,
		COUNT(rs.id) AS total_secrets,
		SUM(CASE WHEN rs.required THEN 1 ELSE 0 END) AS required_secrets,
		SUM(CASE WHEN (SELECT validation_status FROM secret_validations sv WHERE sv.resource_secret_id = rs.id ORDER BY sv.validation_timestamp DESC LIMIT 1) = 'valid' THEN 1 ELSE 0 END) AS valid_secrets,
		SUM(CASE WHEN rs.required AND COALESCE((SELECT validation_status FROM secret_validations sv WHERE sv.resource_secret_id = rs.id ORDER BY sv.validation_timestamp DESC LIMIT 1), 'missing') = 'missing' THEN 1 ELSE 0 END) AS missing_required_secrets,
		SUM(CASE WHEN (SELECT validation_status FROM secret_validations sv WHERE sv.resource_secret_id = rs.id ORDER BY sv.validation_timestamp DESC LIMIT 1) = 'invalid' THEN 1 ELSE 0 END) AS invalid_secrets,
		MAX((SELECT validation_timestamp FROM secret_validations sv WHERE sv.resource_secret_id = rs.id ORDER BY sv.validation_timestamp DESC LIMIT 1)) AS last_validation
	FROM resource_secrets rs GROUP BY rs.resource_name`,
	`CREATE TRIGGER IF NOT EXISTS update_resource_secrets_updated_at
	AFTER UPDATE ON resource_secrets FOR EACH ROW BEGIN
		UPDATE resource_secrets SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END`,
	`CREATE TRIGGER IF NOT EXISTS update_scenario_overrides_updated_at
	AFTER UPDATE ON scenario_secret_strategy_overrides FOR EACH ROW BEGIN
		UPDATE scenario_secret_strategy_overrides SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END`,
}
