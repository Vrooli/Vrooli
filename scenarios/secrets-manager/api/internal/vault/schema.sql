CREATE TABLE IF NOT EXISTS pm_vaults (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'locked',
    key_version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workspace_id, name)
);

CREATE TABLE IF NOT EXISTS pm_workspace_enrollments (
    workspace_id TEXT PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'enrolled',
    enrolled_by TEXT NOT NULL,
    enrolled_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pm_workspace_members (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    principal_id TEXT NOT NULL,
    role TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP,
    UNIQUE(workspace_id, principal_id)
);

CREATE TABLE IF NOT EXISTS pm_owner_tokens (
    token_digest TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    principal_id TEXT NOT NULL,
    role TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pm_machine_principals (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    machine_id TEXT NOT NULL,
    label TEXT NOT NULL,
    token_digest TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'active',
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP,
    UNIQUE(workspace_id, machine_id)
);

CREATE INDEX IF NOT EXISTS pm_machine_principals_workspace_idx
    ON pm_machine_principals(workspace_id, status);

CREATE TABLE IF NOT EXISTS pm_native_host_enrollments (
    workspace_id TEXT NOT NULL,
    extension_id TEXT NOT NULL,
    origin TEXT NOT NULL,
    enrolled_by TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    enrolled_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP,
    PRIMARY KEY (workspace_id, extension_id)
);

CREATE INDEX IF NOT EXISTS pm_native_host_enrollments_origin_idx
    ON pm_native_host_enrollments(workspace_id, origin, status);

CREATE TABLE IF NOT EXISTS pm_items (
    id TEXT PRIMARY KEY,
    vault_id TEXT NOT NULL REFERENCES pm_vaults(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL,
    item_type TEXT NOT NULL,
    name TEXT NOT NULL,
    username TEXT,
    uri TEXT,
    folder TEXT,
    tags TEXT NOT NULL DEFAULT '[]',
    favorite BOOLEAN NOT NULL DEFAULT FALSE,
    trashed BOOLEAN NOT NULL DEFAULT FALSE,
    version INTEGER NOT NULL DEFAULT 1,
    encrypted_payload TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS pm_items_vault_idx ON pm_items(vault_id, trashed, updated_at);
CREATE INDEX IF NOT EXISTS pm_items_workspace_idx ON pm_items(workspace_id, name);

CREATE TABLE IF NOT EXISTS pm_idempotency (
    workspace_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    item_id TEXT NOT NULL REFERENCES pm_items(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS pm_item_history (
    id TEXT PRIMARY KEY,
    item_id TEXT NOT NULL REFERENCES pm_items(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    encrypted_payload TEXT NOT NULL,
    changed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    changed_by TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pm_grants (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    vault_id TEXT NOT NULL,
    item_id TEXT,
    parent_grant_id TEXT,
    principal_type TEXT NOT NULL,
    principal_id TEXT NOT NULL,
    selector_mode TEXT NOT NULL DEFAULT 'current_snapshot',
    members TEXT NOT NULL DEFAULT '[]',
    operations TEXT NOT NULL DEFAULT '[]',
    target TEXT,
    expires_at TIMESTAMP,
    status TEXT NOT NULL DEFAULT 'active',
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pm_access_requests (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    grant_id TEXT NOT NULL REFERENCES pm_grants(id) ON DELETE CASCADE,
    item_id TEXT NOT NULL,
    operation TEXT NOT NULL,
    request_digest TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    requested_by TEXT NOT NULL,
    decided_by TEXT,
    decision_reason TEXT,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    decided_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pm_access_request_idempotency (
    workspace_id TEXT NOT NULL,
    requester_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_id TEXT NOT NULL REFERENCES pm_access_requests(id) ON DELETE CASCADE,
    request_digest TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, requester_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS pm_access_request_idempotency_request_idx
    ON pm_access_request_idempotency (request_id);

CREATE TABLE IF NOT EXISTS pm_assurance_tokens (
    digest TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    operation TEXT NOT NULL,
    request_digest TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    consumed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pm_audit_events (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    action TEXT NOT NULL,
	item_id TEXT,
	request_id TEXT,
	correlation_id TEXT NOT NULL DEFAULT '',
	event_key TEXT NOT NULL DEFAULT '',
	outcome TEXT NOT NULL,
	decision TEXT NOT NULL DEFAULT '',
	destination TEXT NOT NULL DEFAULT '',
	detail TEXT,
	integrity_hash TEXT NOT NULL DEFAULT '',
	integrity_status TEXT NOT NULL DEFAULT 'unknown',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pm_broker_sessions (
    id TEXT PRIMARY KEY,
    token_digest TEXT NOT NULL UNIQUE,
    workspace_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    run_id TEXT NOT NULL DEFAULT '',
    grant_id TEXT NOT NULL,
    item_id TEXT NOT NULL,
    target_origin TEXT NOT NULL,
    resolved_ips TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMP NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    one_use BOOLEAN NOT NULL DEFAULT FALSE,
    use_count INTEGER NOT NULL DEFAULT 0,
    recovery_epoch INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pm_recovery_state (
    workspace_id TEXT PRIMARY KEY,
    epoch INTEGER NOT NULL DEFAULT 1,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pm_export_handles (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    vault_id TEXT NOT NULL,
    format TEXT NOT NULL,
    capability_digest TEXT NOT NULL,
    encrypted_payload TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pm_sources (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    label TEXT NOT NULL,
    endpoint TEXT,
    bootstrap_ref TEXT,
    capabilities TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'unverified',
    last_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pm_source_bindings (
    item_id TEXT PRIMARY KEY REFERENCES pm_items(id) ON DELETE CASCADE,
    source_id TEXT NOT NULL REFERENCES pm_sources(id) ON DELETE RESTRICT,
    external_ref TEXT NOT NULL,
    source_revision TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_id, external_ref)
);
