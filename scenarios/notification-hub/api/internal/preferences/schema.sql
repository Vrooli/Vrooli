CREATE TABLE IF NOT EXISTS notification_sources (
  source_scenario TEXT PRIMARY KEY,
  display_name TEXT NOT NULL DEFAULT '',
  icon TEXT NOT NULL DEFAULT '',
  deep_link_base TEXT NOT NULL DEFAULT '',
  declared INTEGER NOT NULL DEFAULT 0,
  first_seen TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS notification_categories (
  source_scenario TEXT NOT NULL,
  category TEXT NOT NULL,
  display_name TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  declared INTEGER NOT NULL DEFAULT 0,
  default_enabled INTEGER NOT NULL DEFAULT 1,
  default_min_urgency TEXT NOT NULL DEFAULT 'low',
  first_seen TEXT NOT NULL,
  PRIMARY KEY(source_scenario, category)
);
CREATE TABLE IF NOT EXISTS notification_preferences (
  recipient_key TEXT NOT NULL,
  source_scenario TEXT NOT NULL,
  category TEXT NOT NULL DEFAULT '',
  enabled INTEGER NOT NULL DEFAULT 1,
  min_urgency TEXT NOT NULL DEFAULT '',
  channel_override TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL,
  PRIMARY KEY(recipient_key, source_scenario, category)
);
-- Per-instance overrides: a recipient can mute or force-on a SINGLE entity
-- (e.g. one web-console session) without touching their global category
-- preference. scope_key is the producer-supplied entity id carried on the
-- notification's neutral facts. This is the most-specific preference tier.
CREATE TABLE IF NOT EXISTS notification_scope_preferences (
  recipient_key TEXT NOT NULL,
  source_scenario TEXT NOT NULL,
  category TEXT NOT NULL DEFAULT '',
  scope_key TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  min_urgency TEXT NOT NULL DEFAULT '',
  channel_override TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL,
  PRIMARY KEY(recipient_key, source_scenario, category, scope_key)
);
