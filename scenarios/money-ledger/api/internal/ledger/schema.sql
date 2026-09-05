CREATE TABLE IF NOT EXISTS books (id TEXT PRIMARY KEY, name TEXT NOT NULL, currency TEXT NOT NULL, created_at TEXT NOT NULL, archived INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS accounts (id TEXT PRIMARY KEY, book_id TEXT NOT NULL REFERENCES books(id), name TEXT NOT NULL, kind TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS postings (
  id TEXT PRIMARY KEY, external_id TEXT NOT NULL, adapter_id TEXT NOT NULL, account_id TEXT NOT NULL REFERENCES accounts(id), book_id TEXT NOT NULL REFERENCES books(id),
  amount_minor INTEGER NOT NULL, currency TEXT NOT NULL, occurred_at TEXT NOT NULL, fetched_at TEXT NOT NULL, basis INTEGER NOT NULL,
  description TEXT NOT NULL DEFAULT '', category TEXT NOT NULL DEFAULT '', reversal_of TEXT NOT NULL DEFAULT '', actor TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL,
  UNIQUE(adapter_id, external_id)
);
CREATE TABLE IF NOT EXISTS goals (id TEXT PRIMARY KEY, book_id TEXT NOT NULL REFERENCES books(id), name TEXT NOT NULL, metric TEXT NOT NULL, comparator TEXT NOT NULL, threshold_minor INTEGER NOT NULL, sustain_periods INTEGER NOT NULL, buffer_multiple REAL NOT NULL DEFAULT 1.0, threshold_ratio REAL NOT NULL DEFAULT 0, comparand_metric TEXT NOT NULL DEFAULT '', sustain_period_unit INTEGER NOT NULL DEFAULT 3, archived INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS adapter_status (adapter_id TEXT PRIMARY KEY, reason TEXT NOT NULL DEFAULT '', last_success_at TEXT NOT NULL DEFAULT '');
CREATE TABLE IF NOT EXISTS ledger_audit (id TEXT PRIMARY KEY, entity_type TEXT NOT NULL, entity_id TEXT NOT NULL, actor TEXT NOT NULL, reason TEXT NOT NULL, prior_value TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS adapters (id TEXT PRIMARY KEY, name TEXT NOT NULL, kind INTEGER NOT NULL, enabled INTEGER NOT NULL DEFAULT 1, last_success_at TEXT, availability_reason TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS ingest_receipts (id TEXT PRIMARY KEY, adapter_id TEXT NOT NULL REFERENCES adapters(id), from_at TEXT, to_at TEXT, read_count INTEGER NOT NULL DEFAULT 0, written_count INTEGER NOT NULL DEFAULT 0, skipped_duplicates INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL, error TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS idx_ingest_receipts_adapter_created ON ingest_receipts(adapter_id, created_at DESC);
CREATE INDEX IF NOT EXISTS postings_account_time ON postings(account_id, occurred_at);
CREATE INDEX IF NOT EXISTS postings_book_time ON postings(book_id, occurred_at);
CREATE TABLE IF NOT EXISTS operator_measures (
  id INTEGER PRIMARY KEY AUTOINCREMENT, path TEXT NOT NULL, value REAL NOT NULL, unit TEXT NOT NULL,
  window_days INTEGER NOT NULL, observed_at TEXT NOT NULL DEFAULT '', status TEXT NOT NULL, source_path TEXT NOT NULL,
  book_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS operator_measures_path_observed ON operator_measures(path, observed_at);
CREATE TABLE IF NOT EXISTS operator_input_findings (
  id INTEGER PRIMARY KEY AUTOINCREMENT, path TEXT NOT NULL, reason TEXT NOT NULL, created_at TEXT NOT NULL
);
