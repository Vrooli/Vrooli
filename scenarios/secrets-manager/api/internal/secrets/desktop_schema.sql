CREATE TABLE IF NOT EXISTS resource_secrets (
		id TEXT PRIMARY KEY, resource_name TEXT NOT NULL, secret_key TEXT NOT NULL,
		secret_type TEXT NOT NULL, required BOOLEAN NOT NULL DEFAULT 1, description TEXT,
		validation_pattern TEXT, documentation_url TEXT, default_value TEXT,
		classification TEXT NOT NULL DEFAULT 'service', owner_team TEXT, owner_contact TEXT,
		rotation_period_days INTEGER DEFAULT 0, last_rotated_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(resource_name, secret_key)
	);

CREATE TABLE IF NOT EXISTS secret_validations (
		id TEXT PRIMARY KEY, resource_secret_id TEXT NOT NULL REFERENCES resource_secrets(id) ON DELETE CASCADE,
		validation_status TEXT NOT NULL, validation_method TEXT NOT NULL,
		validation_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, error_message TEXT, validation_details TEXT
	);

CREATE TABLE IF NOT EXISTS secret_scans (
		id TEXT PRIMARY KEY, scan_type TEXT NOT NULL DEFAULT 'full', resources_scanned TEXT,
		secrets_discovered INTEGER DEFAULT 0, scan_duration_ms INTEGER,
		scan_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, scan_status TEXT NOT NULL DEFAULT 'completed',
		error_message TEXT, scan_metadata TEXT
	);

CREATE TABLE IF NOT EXISTS secret_provisions (
		id TEXT PRIMARY KEY, resource_secret_id TEXT NOT NULL REFERENCES resource_secrets(id) ON DELETE CASCADE,
		storage_method TEXT NOT NULL, storage_location TEXT NOT NULL,
		provisioned_at DATETIME DEFAULT CURRENT_TIMESTAMP, provisioned_by TEXT,
		provision_status TEXT NOT NULL DEFAULT 'active', expiration_date DATETIME
	);

CREATE TABLE IF NOT EXISTS secret_deployment_strategies (
		id TEXT PRIMARY KEY, resource_secret_id TEXT NOT NULL REFERENCES resource_secrets(id) ON DELETE CASCADE,
		tier TEXT NOT NULL, handling_strategy TEXT NOT NULL, fallback_strategy TEXT,
		requires_user_input BOOLEAN NOT NULL DEFAULT 0, prompt_label TEXT, prompt_description TEXT,
		generator_template TEXT, bundle_hints TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP, UNIQUE(resource_secret_id, tier)
	);

CREATE TABLE IF NOT EXISTS scenario_secret_strategy_overrides (
		id TEXT PRIMARY KEY, scenario_name TEXT NOT NULL,
		resource_secret_id TEXT NOT NULL REFERENCES resource_secrets(id) ON DELETE CASCADE,
		tier TEXT NOT NULL, handling_strategy TEXT, fallback_strategy TEXT, requires_user_input BOOLEAN,
		prompt_label TEXT, prompt_description TEXT, generator_template TEXT, bundle_hints TEXT,
		override_reason TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP, UNIQUE(scenario_name, resource_secret_id, tier)
	);

CREATE TABLE IF NOT EXISTS deployment_manifests (
		id TEXT PRIMARY KEY, scenario_name TEXT NOT NULL, tier TEXT NOT NULL, manifest TEXT NOT NULL,
		generated_by TEXT, generated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

CREATE TABLE IF NOT EXISTS security_scan_runs (
		id TEXT PRIMARY KEY, scan_id TEXT NOT NULL, component_filter TEXT, component_type TEXT,
		severity_filter TEXT, files_scanned INTEGER DEFAULT 0, files_skipped INTEGER DEFAULT 0,
		vulnerabilities_found INTEGER DEFAULT 0, risk_score INTEGER DEFAULT 0, duration_ms INTEGER DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'completed', error_message TEXT, metadata TEXT,
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP, completed_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

CREATE TABLE IF NOT EXISTS security_vulnerabilities (
		id TEXT PRIMARY KEY, scan_run_id TEXT, fingerprint TEXT NOT NULL UNIQUE,
		component_type TEXT NOT NULL, component_name TEXT NOT NULL, file_path TEXT NOT NULL,
		line_number INTEGER, severity TEXT NOT NULL, vulnerability_type TEXT NOT NULL, title TEXT NOT NULL,
		description TEXT, recommendation TEXT, code_snippet TEXT, can_auto_fix BOOLEAN DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'open', assigned_to TEXT, fix_request_id TEXT,
		first_observed_at DATETIME DEFAULT CURRENT_TIMESTAMP, last_observed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		resolved_at DATETIME, metadata TEXT
	);

CREATE TABLE IF NOT EXISTS deployment_campaigns (
		id TEXT PRIMARY KEY, scenario TEXT NOT NULL, tier TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'unknown',
		progress INTEGER NOT NULL DEFAULT 0, blockers INTEGER NOT NULL DEFAULT 0, next_action TEXT,
		last_step TEXT, summary TEXT, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, UNIQUE(scenario, tier)
	);

CREATE VIEW IF NOT EXISTS secret_health_summary AS
	SELECT rs.resource_name,
		COUNT(rs.id) AS total_secrets,
		SUM(CASE WHEN rs.required THEN 1 ELSE 0 END) AS required_secrets,
		SUM(CASE WHEN (SELECT validation_status FROM secret_validations sv WHERE sv.resource_secret_id = rs.id ORDER BY sv.validation_timestamp DESC LIMIT 1) = 'valid' THEN 1 ELSE 0 END) AS valid_secrets,
		SUM(CASE WHEN rs.required AND COALESCE((SELECT validation_status FROM secret_validations sv WHERE sv.resource_secret_id = rs.id ORDER BY sv.validation_timestamp DESC LIMIT 1), 'missing') = 'missing' THEN 1 ELSE 0 END) AS missing_required_secrets,
		SUM(CASE WHEN (SELECT validation_status FROM secret_validations sv WHERE sv.resource_secret_id = rs.id ORDER BY sv.validation_timestamp DESC LIMIT 1) = 'invalid' THEN 1 ELSE 0 END) AS invalid_secrets,
		MAX((SELECT validation_timestamp FROM secret_validations sv WHERE sv.resource_secret_id = rs.id ORDER BY sv.validation_timestamp DESC LIMIT 1)) AS last_validation
	FROM resource_secrets rs GROUP BY rs.resource_name;

CREATE TRIGGER IF NOT EXISTS update_resource_secrets_updated_at
	AFTER UPDATE ON resource_secrets FOR EACH ROW BEGIN
		UPDATE resource_secrets SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;

CREATE TRIGGER IF NOT EXISTS update_scenario_overrides_updated_at
	AFTER UPDATE ON scenario_secret_strategy_overrides FOR EACH ROW BEGIN
		UPDATE scenario_secret_strategy_overrides SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;
