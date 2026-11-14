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
  -- The quality bar this style's source must meet. 'procedural' is the default
  -- because every style seeded before the column existed was drawn in-process,
  -- so the default states what those rows already were rather than guessing.
  quality_tier TEXT NOT NULL DEFAULT 'procedural',
  plate_spec TEXT NOT NULL DEFAULT '[]',
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

-- Model-authored vector generators, stored as catalog data.
--
-- A generator is data rather than code because a model cannot add a Go function
-- to a running binary. It carries its own admission record: the prompt that
-- produced it, the model that wrote it, and the validation verdict that let it
-- in. A row here is a generator some style may bind to, and the render path
-- refuses a binding to an id that is absent or unvalidated.
CREATE TABLE IF NOT EXISTS backdrop_authored_generators (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  template    TEXT NOT NULL,
  params      TEXT NOT NULL DEFAULT '[]',
  inks        TEXT NOT NULL DEFAULT '[]',
  prompt      TEXT NOT NULL DEFAULT '',
  model_id    TEXT NOT NULL DEFAULT '',
  validation  TEXT NOT NULL DEFAULT '{}',
  -- validated is the column the render path checks. It is stored rather than
  -- recomputed from the validation JSON so the check is an index lookup and
  -- cannot disagree with itself.
  validated   INTEGER NOT NULL DEFAULT 0,
  created_at  TEXT NOT NULL
);
