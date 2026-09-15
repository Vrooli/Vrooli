CREATE TABLE IF NOT EXISTS capabilities (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  medium TEXT NOT NULL CHECK (medium IN ('text','image','video','audio','distribution','measurement','governance')),
  aliases TEXT NOT NULL DEFAULT '[]',
  channels TEXT NOT NULL DEFAULT '[]',
  audience_applicability TEXT NOT NULL DEFAULT '',
  delivery_applicability TEXT NOT NULL DEFAULT '',
  producing_operation TEXT NOT NULL DEFAULT '',
  prerequisites TEXT NOT NULL DEFAULT '[]',
  priority INTEGER NOT NULL DEFAULT 0,
  priority_reason TEXT NOT NULL DEFAULT '',
  priority_scope TEXT NOT NULL DEFAULT '',
  definition_status TEXT NOT NULL CHECK (definition_status IN ('documented','absent')),
  implementation_status TEXT NOT NULL CHECK (implementation_status IN ('not-started','skeleton','implemented')),
  operational_readiness TEXT NOT NULL CHECK (operational_readiness IN ('qualified-in-environment','unverified','unavailable')),
  output_quality TEXT NOT NULL CHECK (output_quality IN ('accepted-by-review','unassessed')),
  distribution_connectivity TEXT NOT NULL CHECK (distribution_connectivity IN ('connected','disconnected','not-applicable')),
  owner TEXT NOT NULL DEFAULT '',
  source_refs TEXT NOT NULL DEFAULT '[]',
  next_action TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS capabilities_medium_idx ON capabilities(medium);
CREATE INDEX IF NOT EXISTS capabilities_name_idx ON capabilities(name);

-- Aliases are persisted as JSON on capabilities.aliases. This derived table is
-- the indexed alias lookup only, rewritten with its capability in one
-- transaction; it is never a second source of truth.
CREATE TABLE IF NOT EXISTS capability_aliases (
  alias TEXT PRIMARY KEY,
  capability_id TEXT NOT NULL REFERENCES capabilities(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS capability_aliases_capability_idx ON capability_aliases(capability_id);

CREATE TABLE IF NOT EXISTS capability_qualifications (
  id TEXT PRIMARY KEY,
  capability_id TEXT NOT NULL REFERENCES capabilities(id) ON DELETE CASCADE,
  latest_artifact_id TEXT NOT NULL DEFAULT '',
  latest_run_id TEXT NOT NULL DEFAULT '',
  environment TEXT NOT NULL DEFAULT '',
  validated_at TEXT NOT NULL DEFAULT '',
  observed_at TEXT NOT NULL DEFAULT '',
  freshness_basis TEXT NOT NULL CHECK (freshness_basis IN ('candidate_identity','max_age')),
  candidate_identity TEXT NOT NULL DEFAULT '',
  max_age_seconds INTEGER NOT NULL DEFAULT -1,
  limitation TEXT NOT NULL DEFAULT '',
  next_action TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS capability_qualifications_capability_idx ON capability_qualifications(capability_id, created_at);

CREATE TABLE IF NOT EXISTS capability_links (
  id TEXT PRIMARY KEY,
  capability_id TEXT NOT NULL REFERENCES capabilities(id) ON DELETE CASCADE,
  relation TEXT NOT NULL CHECK (relation IN ('produces','prerequisite','evidence','distribution','owner','related')),
  target_id TEXT NOT NULL CHECK (target_id <> ''),
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS capability_links_capability_idx ON capability_links(capability_id);
CREATE INDEX IF NOT EXISTS capability_links_target_idx ON capability_links(target_id);
