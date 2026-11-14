CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    color TEXT,
    description TEXT
);

CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);
