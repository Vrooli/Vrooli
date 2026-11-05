-- Reusable message text owned by the sender. A snippet names no receiver,
-- group, role, or skill and every row is freely deletable.
CREATE TABLE IF NOT EXISTS message_snippets (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    color TEXT NOT NULL DEFAULT '',
    pinned INTEGER NOT NULL DEFAULT 0,
    use_count INTEGER NOT NULL DEFAULT 0,
    -- Written only by Go's RFC3339Nano formatter. Mixing SQLite's fixed
    -- millisecond form into this text column corrupts recency ordering.
    last_used_at TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_message_snippets_recent
    ON message_snippets(pinned DESC, last_used_at DESC, use_count DESC, id);
