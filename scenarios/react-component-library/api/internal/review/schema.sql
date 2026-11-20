-- Review domain schema — owned by internal/review/.
-- Applied at boot through database.EnsureSchemas via the modules registry.
--
-- findings is a server-owned projection over gate evidence and other evidence
-- owners. The projection may refresh every projected column except status and
-- owner: those are human triage decisions and must survive a refresh unless an
-- explicit assignment or disposition changes them.
CREATE TABLE IF NOT EXISTS findings (
  id            TEXT PRIMARY KEY,
  subject_kind  TEXT NOT NULL,
  subject_id    TEXT NOT NULL,
  gate          TEXT NOT NULL,
  severity      TEXT NOT NULL,
  status        TEXT NOT NULL,
  message       TEXT NOT NULL,
  remediation   TEXT NOT NULL DEFAULT '',
  evidence_json TEXT NOT NULL DEFAULT '{}',
  docs_ref      TEXT NOT NULL DEFAULT '',
  owner         TEXT NOT NULL DEFAULT '',
  created_at    TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_findings_status_severity
  ON findings(status, severity, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_findings_subject
  ON findings(subject_kind, subject_id, created_at DESC);

-- Dispositions are append-only triage history. A finding's current status is
-- the projection of its newest decision, but every decision remains readable.
CREATE TABLE IF NOT EXISTS finding_dispositions (
  id         TEXT PRIMARY KEY,
  finding_id TEXT NOT NULL,
  decision   TEXT NOT NULL,
  reason     TEXT NOT NULL,
  actor      TEXT NOT NULL,
  at         TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_finding_dispositions_finding
  ON finding_dispositions(finding_id, at DESC);
