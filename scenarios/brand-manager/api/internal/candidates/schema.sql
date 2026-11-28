-- Logo candidates — owned by internal/candidates/. One row per proposed mark;
-- exactly one PICKED per brand is enforced by a partial unique index.
CREATE TABLE IF NOT EXISTS logo_candidates (
  id             TEXT PRIMARY KEY,
  brand_id       TEXT NOT NULL,
  asset_id       TEXT NOT NULL,
  media_type     TEXT NOT NULL DEFAULT '',
  concept        TEXT NOT NULL DEFAULT '',
  prompt         TEXT NOT NULL DEFAULT '',
  role           TEXT NOT NULL DEFAULT '',
  model          TEXT NOT NULL DEFAULT '',
  seed           INTEGER NOT NULL DEFAULT 0,
  origin         TEXT NOT NULL DEFAULT 'IMPORTED',
  parent_id      TEXT NOT NULL DEFAULT '',
  status         TEXT NOT NULL DEFAULT 'PROPOSED',
  note           TEXT NOT NULL DEFAULT '',
  generation_ref TEXT NOT NULL DEFAULT '',
  created_at     TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_logo_candidates_brand ON logo_candidates(brand_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_logo_candidates_one_picked
  ON logo_candidates(brand_id) WHERE status = 'PICKED';
