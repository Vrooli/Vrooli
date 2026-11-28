CREATE TABLE IF NOT EXISTS review_runs (id TEXT PRIMARY KEY, draft_id TEXT NOT NULL, outcome TEXT NOT NULL, created_at TEXT NOT NULL, invalidated_at TEXT NOT NULL DEFAULT '');
-- Exact revision/body basis required for reviews that can authorize approval.
-- Older runs without a basis remain visible as history but are not approval evidence.
CREATE TABLE IF NOT EXISTS review_run_basis (run_id TEXT PRIMARY KEY REFERENCES review_runs(id), source_revision_id TEXT NOT NULL DEFAULT '', body_sha256 TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS review_verdicts (run_id TEXT NOT NULL REFERENCES review_runs(id), mode TEXT NOT NULL, passed INTEGER NOT NULL, evidence TEXT NOT NULL DEFAULT '', finding TEXT NOT NULL DEFAULT '', PRIMARY KEY(run_id, mode));
CREATE TABLE IF NOT EXISTS review_supersessions (superseded_run_id TEXT PRIMARY KEY REFERENCES review_runs(id), superseding_run_id TEXT NOT NULL REFERENCES review_runs(id));
CREATE TABLE IF NOT EXISTS review_policy_failure_modes (mode TEXT PRIMARY KEY);
CREATE TABLE IF NOT EXISTS editorial_review_packages (
  id TEXT PRIMARY KEY,
  draft_id TEXT NOT NULL,
  source_revision_id TEXT NOT NULL DEFAULT '',
  scope TEXT NOT NULL CHECK (scope = 'destination_unbound_wording'),
  body TEXT NOT NULL,
  body_sha256 TEXT NOT NULL,
  package_sha256 TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS editorial_review_packages_draft_idx ON editorial_review_packages(draft_id, created_at DESC);

INSERT OR IGNORE INTO review_policy_failure_modes (mode) VALUES
  ('credential_claim_by_persona'),
  ('real_person_impersonation'),
  ('fabricated_customer_testimonial'),
  ('missing_platform_disclosure');

CREATE TABLE IF NOT EXISTS editorial_review_package_operations (
  idempotency_key TEXT PRIMARY KEY,
  request_fingerprint TEXT NOT NULL,
  package_id TEXT NOT NULL REFERENCES editorial_review_packages(id),
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS editorial_review_package_manifests (
  package_id TEXT PRIMARY KEY REFERENCES editorial_review_packages(id),
  manifest_json TEXT NOT NULL
);
