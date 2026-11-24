-- Workspace domain schema — owned by internal/workspaces/.
-- Applied via database.EnsureSchemas at boot through modules.AllSchemas().
-- Idempotent: CREATE TABLE/INDEX IF NOT EXISTS so re-runs are no-ops and
-- internal/workspaces.Migrate can safely re-apply the same text before it
-- imports legacy studio_designs rows.
-- Migrate installs the one-owner binding index after diagnosing duplicate
-- legacy bindings, so schema initialization never picks a winner.
--
-- A workspace is the durable composition aggregate: pages own displays and
-- displays own named states. spec_json/args_json/scenario_data_json are
-- opaque, client-owned JSON; SQL never re-models the composition. The columns
-- SQL does read — name, provenance, binding, revision, sort_order,
-- updated_at — are materialised so list ordering, filtering and the
-- optimistic-concurrency guard never parse a document.
--
-- Bindings are stored as empty text rather than NULL because the proto wire
-- shape uses "" for "unbound"; keeping one representation avoids a second
-- null-handling branch in every read.
--
-- theme_json/options_json default to {} so a pre-chrome client and a freshly
-- imported design both read back an object without a rewrite.
CREATE TABLE IF NOT EXISTS workspaces (
  id                  TEXT PRIMARY KEY,
  name                TEXT NOT NULL,
  summary             TEXT NOT NULL DEFAULT '',
  provenance          TEXT NOT NULL DEFAULT 'authored',
  tags_json           TEXT NOT NULL DEFAULT '[]',
  binding_scenario    TEXT NOT NULL DEFAULT '',
  binding_page        TEXT NOT NULL DEFAULT '',
  theme_json          TEXT NOT NULL DEFAULT '{}',
  options_json        TEXT NOT NULL DEFAULT '{}',
  origin_workspace_id TEXT NOT NULL DEFAULT '',
  revision            INTEGER NOT NULL DEFAULT 1,
  created_at          TEXT NOT NULL,
  updated_at          TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS workspaces_updated_at
  ON workspaces (updated_at DESC);

CREATE INDEX IF NOT EXISTS workspaces_provenance_updated_at
  ON workspaces (provenance, updated_at DESC);

-- Retain exact legacy source and conversion evidence after adoption. These
-- receipts intentionally have no workspace foreign key so recovery survives
-- deletion of the converted workspace.
CREATE TABLE IF NOT EXISTS workspace_conversion_receipts (
  id TEXT PRIMARY KEY,
  scenario TEXT NOT NULL,
  page TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  source_hash TEXT NOT NULL,
  target_hash TEXT NOT NULL,
  receipt_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE (scenario, page)
);
CREATE INDEX IF NOT EXISTS idx_workspace_conversion_receipts_workspace
  ON workspace_conversion_receipts (workspace_id, created_at DESC);
CREATE TRIGGER IF NOT EXISTS workspace_conversion_receipts_no_update
  BEFORE UPDATE ON workspace_conversion_receipts
  BEGIN
    SELECT RAISE(ABORT, 'workspace conversion receipts are immutable');
  END;

-- Pages, displays and states are owned rows of one aggregate. ON DELETE
-- CASCADE makes delete a single statement and guarantees no nested row can
-- outlive its workspace.
CREATE TABLE IF NOT EXISTS workspace_pages (
  id           TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  name         TEXT NOT NULL,
  route        TEXT NOT NULL DEFAULT '',
  page_type_id TEXT NOT NULL DEFAULT '',
  sort_order   INTEGER NOT NULL DEFAULT 0,
  spec_json    TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS workspace_pages_workspace
  ON workspace_pages (workspace_id, sort_order, id);

CREATE TABLE IF NOT EXISTS workspace_displays (
  id         TEXT PRIMARY KEY,
  page_id    TEXT NOT NULL REFERENCES workspace_pages (id) ON DELETE CASCADE,
  device     TEXT NOT NULL,
  width      INTEGER NOT NULL DEFAULT 0,
  height     INTEGER NOT NULL DEFAULT 0,
  sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS workspace_displays_page
  ON workspace_displays (page_id, sort_order, id);

CREATE TABLE IF NOT EXISTS workspace_states (
  id                  TEXT PRIMARY KEY,
  display_id          TEXT NOT NULL REFERENCES workspace_displays (id) ON DELETE CASCADE,
  name                TEXT NOT NULL,
  args_json           TEXT NOT NULL DEFAULT '{}',
  scenario_data_json  TEXT NOT NULL DEFAULT '',
  status              TEXT NOT NULL DEFAULT 'draft',
  capture_id          TEXT NOT NULL DEFAULT '',
  sort_order          INTEGER NOT NULL DEFAULT 0
);

-- BAS artifact references are kept out of scenario_data_json, which remains
-- scenario-owned payload. One row records the latest completed capture for a
-- state; saves that change rendered composition invalidate these rows.
CREATE TABLE IF NOT EXISTS workspace_state_capture_evidence (
  workspace_id    TEXT NOT NULL,
  state_id        TEXT NOT NULL,
  request_token   TEXT NOT NULL,
  source_revision INTEGER NOT NULL,
  capture_id      TEXT NOT NULL,
  evidence_json   TEXT NOT NULL,
  captured_at     TEXT NOT NULL,
  PRIMARY KEY (workspace_id, state_id)
);
CREATE INDEX IF NOT EXISTS idx_workspace_capture_request_token
  ON workspace_state_capture_evidence (request_token);

-- Successful capture operations have immutable receipts even though the table
-- above remains the fast, current-evidence projection used by older clients.
CREATE TABLE IF NOT EXISTS workspace_capture_receipts (
  id              TEXT PRIMARY KEY,
  workspace_id    TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  state_id        TEXT NOT NULL,
  source_revision INTEGER NOT NULL,
  capture_id      TEXT NOT NULL,
  attempt_id      TEXT NOT NULL,
  request_token   TEXT NOT NULL,
  evidence_json   TEXT NOT NULL,
  captured_at     TEXT NOT NULL,
  UNIQUE (workspace_id, state_id, source_revision, attempt_id)
);
CREATE INDEX IF NOT EXISTS idx_workspace_capture_receipts_state
  ON workspace_capture_receipts (workspace_id, state_id, captured_at DESC, id DESC);
CREATE TRIGGER IF NOT EXISTS workspace_capture_receipts_no_update
  BEFORE UPDATE ON workspace_capture_receipts
  BEGIN
    SELECT RAISE(ABORT, 'workspace capture receipts are immutable');
  END;

CREATE TABLE IF NOT EXISTS workspace_visual_comparisons (
  id                  TEXT PRIMARY KEY,
  workspace_id        TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  state_id            TEXT NOT NULL,
  baseline_receipt_id TEXT NOT NULL REFERENCES workspace_capture_receipts (id) ON DELETE CASCADE,
  current_receipt_id  TEXT NOT NULL REFERENCES workspace_capture_receipts (id) ON DELETE CASCADE,
  result_json         TEXT NOT NULL,
  heatmap_ref         TEXT NOT NULL DEFAULT '',
  created_at          TEXT NOT NULL,
  CHECK (baseline_receipt_id <> current_receipt_id)
);
CREATE INDEX IF NOT EXISTS idx_workspace_visual_comparisons_state
  ON workspace_visual_comparisons (workspace_id, state_id, created_at DESC, id DESC);
CREATE TRIGGER IF NOT EXISTS workspace_visual_comparisons_no_update
  BEFORE UPDATE ON workspace_visual_comparisons
  BEGIN
    SELECT RAISE(ABORT, 'workspace visual comparisons are immutable');
  END;

-- Source screenshots are candidates until a separate explicit human action
-- inserts an acceptance receipt. Baseline rows and acceptance records are
-- immutable, keeping source identity and approval provenance auditable.
CREATE TABLE IF NOT EXISTS workspace_state_source_baselines (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  state_id TEXT NOT NULL,
  asset_id TEXT NOT NULL REFERENCES workspace_assets (id) ON DELETE RESTRICT,
  sha256 TEXT NOT NULL CHECK (length(sha256) = 64 AND sha256 NOT GLOB '*[^0-9a-f]*'),
  source_locator TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  UNIQUE (workspace_id, state_id, id)
);
CREATE INDEX IF NOT EXISTS idx_workspace_source_baselines_state
  ON workspace_state_source_baselines (workspace_id, state_id, created_at DESC, id DESC);
CREATE TRIGGER IF NOT EXISTS workspace_state_source_baselines_no_update
  BEFORE UPDATE ON workspace_state_source_baselines
  BEGIN SELECT RAISE(ABORT, 'workspace source baselines are immutable'); END;
CREATE TABLE IF NOT EXISTS workspace_state_source_baseline_acceptances (
  baseline_id TEXT PRIMARY KEY REFERENCES workspace_state_source_baselines (id) ON DELETE CASCADE,
  workspace_id TEXT NOT NULL,
  state_id TEXT NOT NULL,
  accepted_by TEXT NOT NULL,
  accepted_at TEXT NOT NULL,
  acceptance_method TEXT NOT NULL DEFAULT 'explicit_human_action',
  FOREIGN KEY (workspace_id, state_id, baseline_id)
    REFERENCES workspace_state_source_baselines (workspace_id, state_id, id) ON DELETE CASCADE
);
CREATE TRIGGER IF NOT EXISTS workspace_source_baseline_acceptances_no_update
  BEFORE UPDATE ON workspace_state_source_baseline_acceptances
  BEGIN SELECT RAISE(ABORT, 'workspace source baseline acceptance is immutable'); END;
CREATE TABLE IF NOT EXISTS workspace_source_visual_comparisons (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  state_id TEXT NOT NULL,
  source_baseline_id TEXT NOT NULL REFERENCES workspace_state_source_baselines (id) ON DELETE CASCADE,
  current_receipt_id TEXT NOT NULL REFERENCES workspace_capture_receipts (id) ON DELETE CASCADE,
  result_json TEXT NOT NULL,
  heatmap_ref TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_workspace_source_visual_comparisons_state
  ON workspace_source_visual_comparisons (workspace_id, state_id, created_at DESC, id DESC);
CREATE TRIGGER IF NOT EXISTS workspace_source_visual_comparisons_no_update
  BEFORE UPDATE ON workspace_source_visual_comparisons
  BEGIN SELECT RAISE(ABORT, 'workspace source visual comparisons are immutable'); END;

CREATE TABLE IF NOT EXISTS workspace_state_capture_attempts (
  workspace_id    TEXT NOT NULL,
  state_id        TEXT NOT NULL,
  source_revision INTEGER NOT NULL,
  attempt_id      TEXT NOT NULL,
  request_token   TEXT NOT NULL UNIQUE,
  state           TEXT NOT NULL CHECK (state IN ('in_progress', 'completed', 'failed')),
  error           TEXT NOT NULL DEFAULT '',
  created_at      TEXT NOT NULL,
  updated_at      TEXT NOT NULL,
  PRIMARY KEY (workspace_id, state_id, source_revision, attempt_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_workspace_capture_one_active_attempt
  ON workspace_state_capture_attempts (workspace_id, state_id, source_revision)
  WHERE state = 'in_progress';

CREATE INDEX IF NOT EXISTS workspace_states_display
  ON workspace_states (display_id, sort_order, id);

-- Every save appends a snapshot at the revision it wrote. (workspace_id,
-- revision) is unique because a revision is an immutable point in the
-- workspace's history, not an append-only log of retries.
CREATE TABLE IF NOT EXISTS workspace_revisions (
  id            TEXT PRIMARY KEY,
  workspace_id  TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  revision      INTEGER NOT NULL,
  snapshot_json TEXT NOT NULL,
  actor         TEXT NOT NULL DEFAULT '',
  created_at    TEXT NOT NULL,
  UNIQUE (workspace_id, revision)
);

CREATE INDEX IF NOT EXISTS workspace_revisions_workspace
  ON workspace_revisions (workspace_id, revision DESC);

-- Reference images uploaded beside a workspace: screenshots to compose against
-- or regions captured from one. They are references, never components, and are
-- never converted silently.
--
-- Bytes live in this table because they are the object being stored, not a
-- cache: a 1-hour browser cache header is a transport detail. The 8 MiB upload
-- limit is enforced before the insert, so one row cannot grow without bound.
--
-- The id is the asset's durable identity (legacy design assets keep theirs on
-- import), which is also the ETag the byte route returns.
CREATE TABLE IF NOT EXISTS workspace_assets (
  id           TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  name         TEXT NOT NULL,
  mime         TEXT NOT NULL,
  bytes        INTEGER NOT NULL DEFAULT 0,
  width        INTEGER NOT NULL DEFAULT 0,
  height       INTEGER NOT NULL DEFAULT 0,
  fidelity     TEXT NOT NULL DEFAULT 'reference',
  data         BLOB NOT NULL,
  created_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS workspace_assets_workspace
  ON workspace_assets (workspace_id, created_at, id);

-- The reviewed template shelf's applied version. A boot applies a seed file only
-- when its version is newer, so reading the file cannot rewrite a workspace
-- that a person is already looking at on every restart.
CREATE TABLE IF NOT EXISTS workspace_seed_state (
  id         INTEGER PRIMARY KEY CHECK (id = 1),
  version    INTEGER NOT NULL,
  applied_at TEXT NOT NULL
);

-- One row per legacy row already imported. The primary key is the idempotency
-- guard: a second boot must not create a duplicate workspace.
--
-- This table deliberately has no foreign key to workspaces. A receipt records
-- that the source row was imported, not that the imported workspace still
-- exists; if a user deletes an imported workspace, the next boot must not
-- resurrect it from the still-present legacy table. Deleting the legacy table
-- after parity removes the receipt's reason to exist.
CREATE TABLE IF NOT EXISTS workspace_migrations (
  source       TEXT NOT NULL,
  source_id    TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  migrated_at  TEXT NOT NULL,
  PRIMARY KEY (source, source_id)
);
