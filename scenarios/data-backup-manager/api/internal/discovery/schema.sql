-- Discovery dismissals — owned by internal/discovery/. Embedded by schema.go
-- and applied via database.EnsureSchemas at boot through the modules.AllSchemas
-- registry. The ONLY durable state the discovery domain keeps: suggestions
-- themselves are derived each call and never stored (Contract Decision D2).
-- Keyed by the stable suggestion id (sha256(kind|locator), truncated) so a
-- dismissal survives rescans and restarts. Time is an RFC3339Nano string
-- matching the round-trip in sqlite.go. CREATE ... IF NOT EXISTS so re-runs are
-- no-ops.
CREATE TABLE IF NOT EXISTS discovery_dismissals (
  id           TEXT PRIMARY KEY,
  kind         TEXT NOT NULL,
  dismissed_at TEXT NOT NULL
);
