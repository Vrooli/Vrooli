CREATE TABLE IF NOT EXISTS calendar_allocations (
  id TEXT PRIMARY KEY,
  work_item_id TEXT NOT NULL,
  local_date TEXT NOT NULL,
  start_minutes INTEGER NOT NULL,
  duration_minutes INTEGER NOT NULL,
  state TEXT NOT NULL DEFAULT 'accepted',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_calendar_allocations_date ON calendar_allocations(local_date, start_minutes);

-- A carry-forward preserves the original accepted promise as history while
-- making exactly one new accepted placement. The source key makes retries
-- idempotent and keeps one planning demand from becoming two active blocks.
CREATE TABLE IF NOT EXISTS allocation_carry_forwards (
  source_allocation_id TEXT PRIMARY KEY,
  carried_allocation_id TEXT NOT NULL UNIQUE,
  target_local_date TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_allocation_carry_forwards_target ON allocation_carry_forwards(target_local_date);

CREATE TABLE IF NOT EXISTS routines (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  kind TEXT NOT NULL,
  timezone TEXT NOT NULL,
  start_date TEXT NOT NULL,
  end_date TEXT NOT NULL DEFAULT '',
  weekdays TEXT NOT NULL DEFAULT '',
  start_minute INTEGER NOT NULL,
  duration_minutes INTEGER NOT NULL,
  frequency_per_week INTEGER NOT NULL DEFAULT 1,
  revision INTEGER NOT NULL DEFAULT 1,
  active INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_routines_dates ON routines(start_date, end_date, active);
CREATE TABLE IF NOT EXISTS routine_occurrence_overrides (
  routine_id TEXT NOT NULL,
  local_date TEXT NOT NULL,
  status TEXT NOT NULL,
  revision INTEGER NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (routine_id, local_date)
);
CREATE INDEX IF NOT EXISTS idx_routine_overrides_date ON routine_occurrence_overrides(local_date, status);
CREATE TABLE IF NOT EXISTS routine_occurrence_reschedules (
  routine_id TEXT NOT NULL,
  local_date TEXT NOT NULL,
  start_minute INTEGER NOT NULL,
  revision INTEGER NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (routine_id, local_date)
);
