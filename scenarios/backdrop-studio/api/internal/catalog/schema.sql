CREATE TABLE IF NOT EXISTS backdrop_surfaces (
  id TEXT PRIMARY KEY, name TEXT NOT NULL, kind TEXT NOT NULL,
  width INTEGER NOT NULL, height INTEGER NOT NULL, placements TEXT NOT NULL,
  authority TEXT NOT NULL, confirmed_on TEXT NOT NULL,
  origin TEXT NOT NULL DEFAULT 'seed', seed_version INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS backdrop_styles (
  id TEXT PRIMARY KEY, name TEXT NOT NULL, version INTEGER NOT NULL,
  role TEXT NOT NULL, subject TEXT NOT NULL, lineage TEXT NOT NULL,
  strategy TEXT NOT NULL, treatments TEXT NOT NULL, placements TEXT NOT NULL,
  regions TEXT NOT NULL, contrast_threshold REAL NOT NULL,
  scaffold TEXT NOT NULL DEFAULT 'null', generation TEXT NOT NULL DEFAULT 'null',
  parent_id TEXT NOT NULL DEFAULT '', treatment_params TEXT NOT NULL DEFAULT '{}',
  inks TEXT NOT NULL DEFAULT '{}',
  quality TEXT NOT NULL DEFAULT 'null',
  origin TEXT NOT NULL DEFAULT 'seed', seed_version INTEGER NOT NULL DEFAULT 0,
  released INTEGER NOT NULL
);
-- Seed content is versioned data, so an install records which versions it has
-- applied. `Seed` applies every version it has not seen, which is what makes
-- the catalog upgradeable in place instead of frozen at whatever it
-- bootstrapped with.
CREATE TABLE IF NOT EXISTS backdrop_seed_versions (
  version INTEGER PRIMARY KEY, applied_on TEXT NOT NULL
);
