-- The P0 spine deliberately keeps declarations, production, and judgement
-- separate. Foreign keys are soft IDs: a released provenance chain must remain
-- readable even when a future import source is retired.
CREATE TABLE IF NOT EXISTS studio_identities (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL CHECK (kind IN ('character','scene','product')),
  head_version INTEGER NOT NULL,
  referenced INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS studio_identity_versions (
  id TEXT PRIMARY KEY,
  identity_id TEXT NOT NULL,
  version INTEGER NOT NULL,
  name TEXT NOT NULL,
  traits_json TEXT NOT NULL,
  reference_images_json TEXT NOT NULL,
  conditioning_references_json TEXT NOT NULL,
  credential_claims TEXT NOT NULL,
  actor_id TEXT NOT NULL,
  actor_kind TEXT NOT NULL CHECK (actor_kind IN ('operator','agent')),
  created_at TEXT NOT NULL,
  UNIQUE(identity_id, version)
);
CREATE TABLE IF NOT EXISTS studio_specs (
  id TEXT PRIMARY KEY,
  template TEXT NOT NULL,
  fields_json TEXT NOT NULL,
  campaign_ref TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS studio_renders (
  id TEXT PRIMARY KEY,
  spec_id TEXT NOT NULL,
  status TEXT NOT NULL,
  estimated_cost REAL NOT NULL,
  actual_cost REAL,
  provenance_json TEXT,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS studio_assets (
  id TEXT PRIMARY KEY,
  render_id TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('produced','in_review','discarded','released')),
  alt_text TEXT NOT NULL,
  disclosure TEXT NOT NULL,
  ai_generated INTEGER NOT NULL,
  credential_claims TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS studio_conformance_verdicts (
  id TEXT PRIMARY KEY,
  asset_id TEXT NOT NULL,
  identity_version_id TEXT NOT NULL,
  actor_id TEXT NOT NULL,
  actor_kind TEXT NOT NULL CHECK (actor_kind = 'operator'),
  passed INTEGER NOT NULL,
  basis TEXT NOT NULL CHECK (basis IN ('reference-sheet','reference-image-set','conditioning-artifact','prose-only')),
  supersedes_id TEXT,
  created_at TEXT NOT NULL
);
-- The initial P0 adapter persists the aggregate snapshot transactionally while
-- the per-record tables above remain the ownership/analytics projection.
-- Keeping the snapshot in this domain avoids a process-local state machine
-- during the greenfield vertical slice and makes restarts safe.
CREATE TABLE IF NOT EXISTS studio_runtime_state (
  singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
  state_json TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
