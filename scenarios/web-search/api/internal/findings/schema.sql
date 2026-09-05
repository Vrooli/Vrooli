-- findings is the self-curating knowledge store: one citation-backed claim per
-- row. Findings are never hard-deleted in the normal flow — an outdated claim
-- is SUPERSEDED (status + superseded_by) and a contested one is DISPUTED.
-- Timestamps are stored as RFC3339Nano TEXT so a lexical range comparison is a
-- correct time filter for a fixed (UTC) zone.
CREATE TABLE IF NOT EXISTS findings (
  id             TEXT PRIMARY KEY,
  claim          TEXT NOT NULL,
  brief_id       TEXT NOT NULL DEFAULT '',
  confidence     REAL NOT NULL DEFAULT 0,
  status         TEXT NOT NULL DEFAULT 'active',
  retrieval_date TEXT NOT NULL,
  query          TEXT NOT NULL DEFAULT '',
  superseded_by  TEXT NOT NULL DEFAULT '',
  dispute_note   TEXT NOT NULL DEFAULT '',
  source         TEXT NOT NULL DEFAULT 'manual',
  created_at     TEXT NOT NULL,
  updated_at     TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_findings_status_created
  ON findings(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_findings_created_at
  ON findings(created_at DESC);

-- briefs are the research artifacts (L2/L3) findings are distilled from.
CREATE TABLE IF NOT EXISTS briefs (
  id            TEXT PRIMARY KEY,
  query         TEXT NOT NULL DEFAULT '',
  level         TEXT NOT NULL DEFAULT '',
  summary       TEXT NOT NULL DEFAULT '',
  agent_run_id  TEXT NOT NULL DEFAULT '',
  run_timestamp TEXT NOT NULL,
  metadata      TEXT NOT NULL DEFAULT ''
);

-- finding_citations are the cited sources backing a finding's claim.
CREATE TABLE IF NOT EXISTS finding_citations (
  id          TEXT PRIMARY KEY,
  finding_id  TEXT NOT NULL,
  url         TEXT NOT NULL DEFAULT '',
  title       TEXT NOT NULL DEFAULT '',
  retrieved_at TEXT NOT NULL,
  FOREIGN KEY (finding_id) REFERENCES findings(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_finding_citations_finding
  ON finding_citations(finding_id);

-- finding_audit is the immutable trail: every finding mutation appends one row
-- describing what changed and why. Never updated, never deleted.
CREATE TABLE IF NOT EXISTS finding_audit (
  id              TEXT PRIMARY KEY,
  finding_id      TEXT NOT NULL,
  mutation_type   TEXT NOT NULL,
  reason          TEXT NOT NULL DEFAULT '',
  source_brief_id TEXT NOT NULL DEFAULT '',
  actor           TEXT NOT NULL DEFAULT '',
  created_at      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_finding_audit_finding
  ON finding_audit(finding_id, created_at DESC);

-- finding_usage is the usage-telemetry side table (OT-P2-001). It is kept
-- SEPARATE from the findings row so surfacing events never mutate the
-- provenance-bearing finding (claim/confidence/audit stay immutable). A row is
-- a running counter of how often a finding was surfaced (returned by search)
-- and explicitly marked used, plus when it was last surfaced. A never-surfaced
-- finding simply has no row (treated as zero). The row cascades on delete so a
-- pruned finding's counters go with it.
CREATE TABLE IF NOT EXISTS finding_usage (
  finding_id       TEXT PRIMARY KEY,
  surfaced_count   INTEGER NOT NULL DEFAULT 0,
  used_count       INTEGER NOT NULL DEFAULT 0,
  last_surfaced_at TEXT NOT NULL DEFAULT '',
  FOREIGN KEY (finding_id) REFERENCES findings(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_finding_usage_surfaced
  ON finding_usage(surfaced_count, last_surfaced_at);
