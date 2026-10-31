-- Will-exhaust notification dedupe state (SPEC A7). One row per window: the
-- projected exhaustion that was last notified. A window is notified again only
-- when its projection moves earlier than this by more than 12 hours.
CREATE TABLE IF NOT EXISTS usage_forecast_alerts (
    window_key TEXT PRIMARY KEY,
    projected_exhaustion_at TEXT NOT NULL,
    notified_at TEXT NOT NULL
);
