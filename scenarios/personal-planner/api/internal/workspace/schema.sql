CREATE TABLE IF NOT EXISTS planning_profiles (
  id TEXT PRIMARY KEY,
  timezone TEXT NOT NULL,
  week_start TEXT NOT NULL,
  daily_capacity_minutes INTEGER NOT NULL,
  reserve_minutes INTEGER NOT NULL,
  focus_session_minutes INTEGER NOT NULL,
  revision INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS availability_rules (
  id TEXT PRIMARY KEY,
  weekday INTEGER NOT NULL,
  start_minute INTEGER NOT NULL,
  end_minute INTEGER NOT NULL,
  timezone TEXT NOT NULL,
  effective_start_date TEXT NOT NULL DEFAULT '',
  effective_end_date TEXT NOT NULL DEFAULT '',
  priority INTEGER NOT NULL DEFAULT 0,
  revision INTEGER NOT NULL,
  UNIQUE (weekday, start_minute, end_minute, effective_start_date, effective_end_date, priority)
);

CREATE TABLE IF NOT EXISTS availability_exceptions (
  id TEXT PRIMARY KEY,
  date TEXT NOT NULL,
  start_minute INTEGER NOT NULL,
  end_minute INTEGER NOT NULL,
  kind TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  revision INTEGER NOT NULL
);

-- Authenticated workspace state is stored separately from the historical,
-- ownerless singleton tables above. Legacy rows are deliberately not mapped.
CREATE TABLE IF NOT EXISTS actor_planning_profiles (
  owner_subject TEXT NOT NULL,
  id TEXT NOT NULL,
  timezone TEXT NOT NULL,
  week_start TEXT NOT NULL,
  daily_capacity_minutes INTEGER NOT NULL,
  reserve_minutes INTEGER NOT NULL,
  focus_session_minutes INTEGER NOT NULL,
  revision INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (owner_subject, id)
);

CREATE TABLE IF NOT EXISTS actor_availability_rules (
  owner_subject TEXT NOT NULL,
  id TEXT NOT NULL,
  weekday INTEGER NOT NULL,
  start_minute INTEGER NOT NULL,
  end_minute INTEGER NOT NULL,
  timezone TEXT NOT NULL,
  effective_start_date TEXT NOT NULL DEFAULT '',
  effective_end_date TEXT NOT NULL DEFAULT '',
  priority INTEGER NOT NULL DEFAULT 0,
  revision INTEGER NOT NULL,
  PRIMARY KEY (owner_subject, id),
  UNIQUE (owner_subject, weekday, start_minute, end_minute, effective_start_date, effective_end_date, priority)
);

CREATE TABLE IF NOT EXISTS actor_availability_exceptions (
  owner_subject TEXT NOT NULL,
  id TEXT NOT NULL,
  date TEXT NOT NULL,
  start_minute INTEGER NOT NULL,
  end_minute INTEGER NOT NULL,
  kind TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  revision INTEGER NOT NULL,
  PRIMARY KEY (owner_subject, id)
);
