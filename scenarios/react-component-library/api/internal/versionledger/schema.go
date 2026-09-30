package versionledger

func Schema() string {
	return `
CREATE TABLE IF NOT EXISTS version_ledger (
  library_id TEXT NOT NULL,
  version TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT '',
  released_at TEXT NOT NULL DEFAULT '',
  retired_at TEXT NOT NULL DEFAULT '',
  lifecycle_state TEXT NOT NULL DEFAULT '',
  gate_pass_count INTEGER NOT NULL DEFAULT 0,
  gate_fail_count INTEGER NOT NULL DEFAULT 0,
  test_runs INTEGER NOT NULL DEFAULT 0,
  test_pass_rate REAL NOT NULL DEFAULT 0,
  adoption_current INTEGER NOT NULL DEFAULT 0,
  adoption_peak INTEGER NOT NULL DEFAULT 0,
  file_count INTEGER NOT NULL DEFAULT 0,
  lines_of_code INTEGER NOT NULL DEFAULT 0,
  dependency_count INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (library_id, version)
);
CREATE INDEX IF NOT EXISTS idx_version_ledger_library ON version_ledger(library_id, version);
CREATE INDEX IF NOT EXISTS idx_version_ledger_state ON version_ledger(lifecycle_state);
`
}
