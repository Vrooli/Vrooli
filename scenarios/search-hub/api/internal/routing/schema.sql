-- Durable zero-yield demotion evidence belongs to the routing policy domain.
-- It is state about provider eligibility, never corpus content or vectors.

CREATE TABLE IF NOT EXISTS provider_demotion_state (
  provider_id TEXT PRIMARY KEY,
  routed INTEGER NOT NULL DEFAULT 0,
  hits INTEGER NOT NULL DEFAULT 0,
  empty_streak INTEGER NOT NULL DEFAULT 0,
  demoted INTEGER NOT NULL DEFAULT 0,
  probation INTEGER NOT NULL DEFAULT 0,
  decay_deadline TEXT NOT NULL DEFAULT '',
  trigger TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);
