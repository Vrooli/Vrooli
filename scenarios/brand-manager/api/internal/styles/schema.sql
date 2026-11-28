-- Styles tables — owned by internal/styles/. Container styles and product lines
-- are records render, apply and validation read. Use CREATE TABLE IF NOT EXISTS
-- so re-runs are no-ops; times are RFC3339Nano strings.
CREATE TABLE IF NOT EXISTS container_styles (
  id                    TEXT PRIMARY KEY,
  name                  TEXT NOT NULL UNIQUE,
  shape                 TEXT NOT NULL DEFAULT 'rounded_square',
  corner_ratio          REAL NOT NULL DEFAULT 0,
  background_kind       TEXT NOT NULL DEFAULT 'gradient',
  background_top        TEXT NOT NULL DEFAULT '',
  background_bottom     TEXT NOT NULL DEFAULT '',
  mark_scale            REAL NOT NULL DEFAULT 0.86,
  maskable_scale        REAL NOT NULL DEFAULT 0.40,
  accent_color          TEXT NOT NULL DEFAULT '',
  glow                  TEXT NOT NULL DEFAULT '[]',
  small_mark_threshold_px INTEGER NOT NULL DEFAULT 32,
  created_at            TEXT NOT NULL,
  updated_at            TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS product_lines (
  id                TEXT PRIMARY KEY,
  name              TEXT NOT NULL UNIQUE,
  container_style_id TEXT NOT NULL DEFAULT '',
  products          TEXT NOT NULL DEFAULT '[]',
  created_at        TEXT NOT NULL
);
