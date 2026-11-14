CREATE TABLE IF NOT EXISTS releases (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL,
    deployment_id TEXT,
    profile_version INTEGER,
    git_commit_hash TEXT NOT NULL,
    artifact_digest TEXT,
    candidate_id TEXT,
    destination_revision_id TEXT,
    authorization_epoch INTEGER NOT NULL DEFAULT 1,
    idempotency_key TEXT,
    readiness_review_key TEXT,
    release_version TEXT NOT NULL,
    channel TEXT NOT NULL DEFAULT 'stable',
    status TEXT NOT NULL DEFAULT 'pending',
    release_notes TEXT,
    released_by TEXT,
    promoted_from_release_id TEXT,
    readiness_goal_ref TEXT,
    approved_at_commit TEXT,
    verification_evidence TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS release_platforms (
    release_id TEXT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    platform TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    approval_id TEXT,
    lpbs_artifact_id INTEGER,
    published_at TIMESTAMP,
    verified_at TIMESTAMP,
    error TEXT,
    PRIMARY KEY(release_id, platform)
);
CREATE INDEX IF NOT EXISTS idx_releases_profile_channel ON releases(profile_id, channel, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_releases_status ON releases(status);
CREATE INDEX IF NOT EXISTS idx_releases_commit ON releases(profile_id, git_commit_hash);
CREATE INDEX IF NOT EXISTS idx_releases_deployment ON releases(deployment_id);
CREATE TABLE IF NOT EXISTS release_profile_locks (
    profile_id TEXT PRIMARY KEY,
    acquired_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS release_operations (
    id TEXT PRIMARY KEY,
    release_id TEXT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    profile_id TEXT NOT NULL,
    idempotency_key TEXT,
    request_snapshot TEXT,
    status TEXT NOT NULL DEFAULT 'queued',
    active_stage TEXT,
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    UNIQUE(release_id),
    UNIQUE(profile_id, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_release_operations_status ON release_operations(status);
CREATE TABLE IF NOT EXISTS release_operation_leases (
    operation_id TEXT PRIMARY KEY REFERENCES release_operations(id) ON DELETE CASCADE,
    owner TEXT NOT NULL,
    fence INTEGER NOT NULL DEFAULT 1,
    lease_until TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS release_operation_events (
    event_id TEXT PRIMARY KEY,
    operation_id TEXT NOT NULL REFERENCES release_operations(id) ON DELETE CASCADE,
    fence INTEGER NOT NULL,
    status TEXT NOT NULL,
    active_stage TEXT,
    message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_release_operation_events_operation
    ON release_operation_events(operation_id, created_at);

-- Immutable release identities are stored beside the release ledger. The
-- canonical JSON is the authoritative value; indexed columns support lookup
-- and do not make request-provided labels trustworthy.
CREATE TABLE IF NOT EXISTS release_candidates (
    candidate_id TEXT PRIMARY KEY,
    canonical_json TEXT NOT NULL,
    artifact_manifest_digest TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS release_destination_revisions (
    destination_revision_id TEXT PRIMARY KEY,
    canonical_json TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS release_review_bindings (
    review_id TEXT PRIMARY KEY,
    canonical_json TEXT NOT NULL,
    candidate_id TEXT NOT NULL REFERENCES release_candidates(candidate_id),
    destination_revision_id TEXT NOT NULL REFERENCES release_destination_revisions(destination_revision_id),
    status TEXT NOT NULL,
    approved_at TIMESTAMP,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_release_review_bindings_identity
    ON release_review_bindings(candidate_id, destination_revision_id, status);
CREATE TABLE IF NOT EXISTS release_publication_receipts (
    receipt_id TEXT PRIMARY KEY,
    release_id TEXT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    candidate_id TEXT NOT NULL,
    destination_revision_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    artifact_digest TEXT NOT NULL,
    destination_object TEXT NOT NULL,
    producer TEXT NOT NULL,
    external_receipt TEXT NOT NULL,
    outcome TEXT NOT NULL,
    observed_at TIMESTAMP NOT NULL,
    review_key TEXT,
    predecessor_artifact_digest TEXT,
    evidence_ref TEXT,
    UNIQUE(release_id, target_id, external_receipt)
);
CREATE INDEX IF NOT EXISTS idx_release_publication_receipts_release
    ON release_publication_receipts(release_id, target_id);

CREATE TABLE IF NOT EXISTS release_client_update_receipts (
    receipt_id TEXT PRIMARY KEY,
    release_id TEXT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    candidate_id TEXT NOT NULL,
    predecessor_ref TEXT NOT NULL,
    successor_digest TEXT NOT NULL,
    target_id TEXT NOT NULL,
    verified_version TEXT NOT NULL,
    outcome TEXT NOT NULL,
    producer TEXT NOT NULL,
    external_receipt TEXT NOT NULL,
    observed_at TIMESTAMP NOT NULL,
    UNIQUE(release_id, target_id, external_receipt)
);
CREATE INDEX IF NOT EXISTS idx_release_client_update_receipts_release
    ON release_client_update_receipts(release_id, target_id, observed_at);

CREATE TABLE IF NOT EXISTS release_recovery_receipts (
    receipt_id TEXT PRIMARY KEY,
    release_id TEXT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    candidate_id TEXT NOT NULL,
    destination_revision_id TEXT NOT NULL,
    deployment_id TEXT NOT NULL,
    action TEXT NOT NULL,
    outcome TEXT NOT NULL,
    health TEXT NOT NULL,
    bundle_sha256 TEXT,
    external_receipt TEXT NOT NULL,
    observed_at TIMESTAMP NOT NULL,
    dry_run INTEGER NOT NULL DEFAULT 0,
    UNIQUE(release_id, external_receipt)
);
CREATE INDEX IF NOT EXISTS idx_release_recovery_receipts_release
    ON release_recovery_receipts(release_id, observed_at);
