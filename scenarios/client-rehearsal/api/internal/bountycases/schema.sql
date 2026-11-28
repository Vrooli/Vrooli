CREATE TABLE IF NOT EXISTS bounty_replay_cases (
  case_id TEXT NOT NULL,
  public_url TEXT NOT NULL,
  pull_request_url TEXT NOT NULL,
  bundle_version INTEGER NOT NULL,
  bundle_digest TEXT NOT NULL,
  bundle_json TEXT NOT NULL,
  registered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY(case_id, bundle_version),
  UNIQUE(bundle_digest)
);
CREATE INDEX IF NOT EXISTS idx_bounty_replay_cases_public_url ON bounty_replay_cases(public_url, bundle_version DESC);
CREATE TRIGGER IF NOT EXISTS bounty_replay_cases_no_update
BEFORE UPDATE ON bounty_replay_cases
BEGIN
  SELECT RAISE(ABORT, 'registered bounty case bundles are immutable');
END;
CREATE TRIGGER IF NOT EXISTS bounty_replay_cases_no_delete
BEFORE DELETE ON bounty_replay_cases
BEGIN
  SELECT RAISE(ABORT, 'registered bounty case bundles are immutable');
END;
