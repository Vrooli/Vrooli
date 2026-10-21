CREATE TABLE IF NOT EXISTS shots (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  head_revision INTEGER NOT NULL CHECK (head_revision > 0)
);
CREATE INDEX IF NOT EXISTS idx_shots_project_created_at_id
  ON shots(project_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS shot_revisions (
  shot_id TEXT NOT NULL REFERENCES shots(id) ON DELETE CASCADE,
  revision INTEGER NOT NULL CHECK (revision > 0),
  created_at TEXT NOT NULL,
  name TEXT NOT NULL,
  snapshot BLOB NOT NULL,
  recipe_sha256 TEXT NOT NULL CHECK (length(recipe_sha256) = 64),
  scene_sha256 TEXT NOT NULL CHECK (length(scene_sha256) = 64),
  presentation_sha256 TEXT NOT NULL CHECK (length(presentation_sha256) = 64),
  scene_version TEXT NOT NULL,
  presentation_version TEXT NOT NULL,
  compositor_profile TEXT NOT NULL,
  plate BLOB,
  active_mask BLOB,
  hardware_mask BLOB,
  screen BLOB,
  render_eligible INTEGER NOT NULL CHECK (render_eligible IN (0, 1)),
  issues BLOB NOT NULL,
  PRIMARY KEY (shot_id, revision)
);
CREATE INDEX IF NOT EXISTS idx_shot_revisions_shot_revision
  ON shot_revisions(shot_id, revision DESC);

CREATE TABLE IF NOT EXISTS output_readability_reviews (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  shot_id TEXT NOT NULL,
  shot_revision INTEGER NOT NULL,
  render_id TEXT NOT NULL,
  output_sha256 TEXT NOT NULL CHECK (length(output_sha256) = 64),
  proof_sha256 TEXT NOT NULL CHECK (length(proof_sha256) = 64),
  verdict INTEGER NOT NULL CHECK (verdict IN (1, 2)),
  note TEXT NOT NULL,
  output_width INTEGER NOT NULL CHECK (output_width > 0),
  output_height INTEGER NOT NULL CHECK (output_height > 0),
  region_width INTEGER NOT NULL CHECK (region_width > 0),
  region_height INTEGER NOT NULL CHECK (region_height > 0),
  reviewed_at TEXT NOT NULL,
  FOREIGN KEY (shot_id, shot_revision) REFERENCES shot_revisions(shot_id, revision)
);
CREATE INDEX IF NOT EXISTS idx_output_readability_binding
  ON output_readability_reviews(shot_id, shot_revision, render_id, reviewed_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_output_readability_one_verdict_per_output
  ON output_readability_reviews(shot_id, shot_revision, render_id, output_sha256);

CREATE TABLE IF NOT EXISTS physical_shot_preset_revisions (
  preset_id TEXT NOT NULL,
  revision INTEGER NOT NULL CHECK (revision > 0),
  sha256 TEXT NOT NULL CHECK (length(sha256) = 64),
  candidate_json BLOB NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (preset_id, revision)
);
CREATE INDEX IF NOT EXISTS idx_physical_shot_preset_revisions_head
  ON physical_shot_preset_revisions(preset_id, revision DESC);
