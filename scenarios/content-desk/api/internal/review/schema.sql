CREATE TABLE IF NOT EXISTS review_runs (id TEXT PRIMARY KEY, draft_id TEXT NOT NULL, outcome TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS review_verdicts (run_id TEXT NOT NULL REFERENCES review_runs(id), mode TEXT NOT NULL, passed INTEGER NOT NULL, evidence TEXT NOT NULL DEFAULT '', finding TEXT NOT NULL DEFAULT '', PRIMARY KEY(run_id, mode));
CREATE TABLE IF NOT EXISTS review_supersessions (superseded_run_id TEXT PRIMARY KEY REFERENCES review_runs(id), superseding_run_id TEXT NOT NULL REFERENCES review_runs(id));
CREATE TABLE IF NOT EXISTS review_policy_failure_modes (mode TEXT PRIMARY KEY);

INSERT OR IGNORE INTO review_policy_failure_modes (mode) VALUES
  ('credential_claim_by_persona'),
  ('real_person_impersonation'),
  ('fabricated_customer_testimonial'),
  ('missing_platform_disclosure');
