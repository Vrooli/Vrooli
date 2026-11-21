-- Workspace domain schema — owned by internal/workspaces/.
-- Applied via database.EnsureSchemas at boot through modules.AllSchemas().
-- Idempotent: CREATE TABLE/INDEX IF NOT EXISTS so re-runs are no-ops and
-- internal/workspaces.Migrate can safely re-apply the same text before it
-- imports legacy studio_designs rows.
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
