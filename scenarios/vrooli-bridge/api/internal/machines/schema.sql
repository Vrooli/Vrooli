-- Machine intent is deliberately separate from paired registry Nodes and
-- Presence. It can be created before the first network contact.
CREATE TABLE IF NOT EXISTS machines (
  id TEXT PRIMARY KEY,
  lifecycle TEXT NOT NULL DEFAULT 'active',
  version INTEGER NOT NULL DEFAULT 1,
  desired_profile_id TEXT NOT NULL DEFAULT '',
  desired_profile_version TEXT NOT NULL DEFAULT '',
  desired_selection_json TEXT NOT NULL DEFAULT '',
  applied_profile_id TEXT NOT NULL DEFAULT '',
  applied_profile_version TEXT NOT NULL DEFAULT '',
  applied_selection_json TEXT NOT NULL DEFAULT '',
  applied_at TEXT NOT NULL DEFAULT '',
  trust_ref TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  archived_at TEXT NOT NULL DEFAULT '',
  removed_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS machine_locators (
  machine_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  kind TEXT NOT NULL,
  value TEXT NOT NULL,
  normalized_value TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY(machine_id, ordinal),
  UNIQUE(machine_id, kind, normalized_value)
);
CREATE TABLE IF NOT EXISTS machine_node_lineage (
  id TEXT PRIMARY KEY,
  machine_id TEXT NOT NULL,
  node_id TEXT NOT NULL,
  is_current INTEGER NOT NULL DEFAULT 1,
  linked_at TEXT NOT NULL,
  superseded_at TEXT NOT NULL DEFAULT '',
  source_correlation_id TEXT NOT NULL DEFAULT '',
  UNIQUE(machine_id, node_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_machine_one_current_node
  ON machine_node_lineage(machine_id) WHERE is_current = 1;
-- A physical Node can be current for only one durable Machine. The migration
-- reconciles legacy duplicates before creating this index.
CREATE UNIQUE INDEX IF NOT EXISTS idx_machine_one_current_node_global
  ON machine_node_lineage(node_id) WHERE is_current = 1;
CREATE INDEX IF NOT EXISTS idx_machine_locators_normalized
  ON machine_locators(kind, normalized_value);

-- Active locator claims are the database-level uniqueness boundary for Machine
-- identity.  machine_locators remains historical and may contain the same
-- locator on archived Machines after a merge; claims contain only the active
-- owner, so concurrent creates cannot mint sibling Machines.
CREATE TABLE IF NOT EXISTS machine_locator_claims (
  kind TEXT NOT NULL,
  normalized_value TEXT NOT NULL,
  machine_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY(kind, normalized_value),
  UNIQUE(machine_id, kind, normalized_value)
);
CREATE INDEX IF NOT EXISTS idx_machine_locator_claims_machine
  ON machine_locator_claims(machine_id);

-- Legacy Nodes and one-shot onboarding operations lack the durable pairing
-- correlation required to infer operator intent. They are retained as typed,
-- reviewable evidence rather than silently becoming Machines.
CREATE TABLE IF NOT EXISTS machine_migration_reviews (
  id TEXT PRIMARY KEY,
  legacy_source TEXT NOT NULL,
  legacy_id TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'needs_review',
  confidence TEXT NOT NULL DEFAULT 'ambiguous',
  reason TEXT NOT NULL,
  created_at TEXT NOT NULL,
  reviewed_at TEXT NOT NULL DEFAULT '',
  UNIQUE(legacy_source, legacy_id)
);
CREATE INDEX IF NOT EXISTS idx_machine_migration_reviews_status
  ON machine_migration_reviews(status, created_at);

-- Local revocation is durable before remote SSH cleanup is attempted. A failed
-- remote cleanup remains an actionable tombstone rather than reviving access.
CREATE TABLE IF NOT EXISTS machine_cleanup_tombstones (
  id TEXT PRIMARY KEY,
  machine_id TEXT NOT NULL,
  action TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  detail TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  acknowledged_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_machine_cleanup_tombstones_machine
  ON machine_cleanup_tombstones(machine_id, status, created_at);
-- Retry/concurrent requests for the same outstanding effect must converge on
-- one actionable tombstone. Terminal history remains repeatable.
CREATE UNIQUE INDEX IF NOT EXISTS idx_machine_cleanup_pending_effect
  ON machine_cleanup_tombstones(machine_id, action) WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS machine_audit_events (
  id TEXT PRIMARY KEY,
  machine_id TEXT NOT NULL,
  action TEXT NOT NULL,
  actor TEXT NOT NULL DEFAULT '',
  detail TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_machine_audit_events_machine
  ON machine_audit_events(machine_id, created_at DESC);

-- Only opaque private-key references and public fingerprints are durable here.
-- Private SSH material remains in the Bridge-owned trust store filesystem.
CREATE TABLE IF NOT EXISTS machine_trust (
  machine_id TEXT PRIMARY KEY,
  client_key_ref TEXT NOT NULL DEFAULT '',
  client_key_fingerprint TEXT NOT NULL DEFAULT '',
  host_key_fingerprint TEXT NOT NULL DEFAULT '',
  host_key_state TEXT NOT NULL DEFAULT 'unverified',
  ssh_user TEXT NOT NULL DEFAULT '',
  ssh_port INTEGER NOT NULL DEFAULT 22,
  connection_state TEXT NOT NULL DEFAULT 'untrusted',
  updated_at TEXT NOT NULL
);

-- Resolved policy is persisted independently of observed Node capabilities and
-- Registry-approved scopes. Re-resolving a profile never rewrites history.
CREATE TABLE IF NOT EXISTS machine_policy_snapshots (
  machine_id TEXT PRIMARY KEY,
  profile_id TEXT NOT NULL,
  profile_version TEXT NOT NULL,
  snapshot_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);

-- Historical profile decisions are append-only. The original single-snapshot
-- table remains readable for early installations; all new policy changes use
-- this history so upgrade, downgrade, and repair remain auditable.
CREATE TABLE IF NOT EXISTS machine_policy_snapshot_history (
  id TEXT PRIMARY KEY,
  machine_id TEXT NOT NULL,
  profile_id TEXT NOT NULL,
  profile_version TEXT NOT NULL,
  overrides_json TEXT NOT NULL DEFAULT '{}',
  snapshot_json TEXT NOT NULL,
  actor TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_machine_policy_history_machine
  ON machine_policy_snapshot_history(machine_id, created_at);
