CREATE TABLE IF NOT EXISTS development_snapshots (
    digest TEXT PRIMARY KEY,
    body BLOB NOT NULL
);
CREATE TABLE IF NOT EXISTS development_engagements (
    work_item TEXT PRIMARY KEY,
    version INTEGER NOT NULL,
    body BLOB NOT NULL
);
