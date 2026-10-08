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

CREATE TABLE IF NOT EXISTS calendar_schedule_state (
  id TEXT PRIMARY KEY,
  revision INTEGER NOT NULL DEFAULT 1
);
INSERT OR IGNORE INTO calendar_schedule_state (id, revision) VALUES ('default', 1);

-- Native events preserve civil all-day ranges separately from timed instants.
-- range_start/range_end are derived local civil bounds used only for overlap
-- queries; end is exclusive for both event kinds.
CREATE TABLE IF NOT EXISTS calendar_events (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  availability TEXT NOT NULL,
  timezone TEXT NOT NULL,
  all_day INTEGER NOT NULL,
  start_date TEXT NOT NULL DEFAULT '',
  end_date_exclusive TEXT NOT NULL DEFAULT '',
  start_at TEXT NOT NULL DEFAULT '',
  end_at TEXT NOT NULL DEFAULT '',
  range_start TEXT NOT NULL,
  range_end TEXT NOT NULL,
  provider TEXT NOT NULL DEFAULT '',
  provider_calendar_id TEXT NOT NULL DEFAULT '',
  provider_event_id TEXT NOT NULL DEFAULT '',
  occurrence_id TEXT NOT NULL DEFAULT '',
  revision INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_calendar_events_range ON calendar_events(range_start, range_end);
CREATE UNIQUE INDEX IF NOT EXISTS idx_calendar_events_provider_identity
  ON calendar_events(provider, provider_calendar_id, provider_event_id, occurrence_id)
  WHERE provider_event_id <> '';
CREATE TABLE IF NOT EXISTS calendar_event_commands (
  idempotency_key TEXT PRIMARY KEY,
  fingerprint TEXT NOT NULL,
  event_id TEXT NOT NULL REFERENCES calendar_events(id),
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS placement_proposals (
  id TEXT PRIMARY KEY,
  work_item_id TEXT NOT NULL,
  local_date TEXT NOT NULL,
  start_minutes INTEGER NOT NULL,
  duration_minutes INTEGER NOT NULL,
  state TEXT NOT NULL,
  reason TEXT NOT NULL,
  base_revision INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  applied_allocation_id TEXT NOT NULL DEFAULT '',
  applied_idempotency_key TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_placement_proposals_created ON placement_proposals(created_at);

CREATE TABLE IF NOT EXISTS schedule_proposals (
  id TEXT PRIMARY KEY,
  local_date TEXT NOT NULL,
  start_minutes INTEGER NOT NULL,
  state TEXT NOT NULL,
  reason TEXT NOT NULL,
  base_revision INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  applied_idempotency_key TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS schedule_proposal_placements (
  id TEXT PRIMARY KEY,
  proposal_id TEXT NOT NULL REFERENCES schedule_proposals(id),
  work_item_id TEXT NOT NULL,
  title TEXT NOT NULL,
  local_date TEXT NOT NULL,
  start_minutes INTEGER NOT NULL,
  duration_minutes INTEGER NOT NULL,
  state TEXT NOT NULL,
  reason TEXT NOT NULL,
  applied_allocation_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_schedule_proposal_placements_proposal ON schedule_proposal_placements(proposal_id);

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

CREATE TABLE IF NOT EXISTS reschedule_history (
  id TEXT PRIMARY KEY,
  allocation_id TEXT NOT NULL,
  from_date TEXT NOT NULL,
  to_date TEXT NOT NULL,
  reason_code TEXT NOT NULL,
  rescheduled_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_reschedule_history_allocation ON reschedule_history(allocation_id, rescheduled_at DESC);

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
