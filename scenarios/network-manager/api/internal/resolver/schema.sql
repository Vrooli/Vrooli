CREATE TABLE IF NOT EXISTS resolver_backends (
    backend TEXT PRIMARY KEY,
    base_url TEXT NOT NULL,
    username TEXT NOT NULL,
    token_ref TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS resolver_upstreams (
    id TEXT PRIMARY KEY,
    backend TEXT NOT NULL,
    upstream TEXT NOT NULL,
    sort_order INTEGER NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_resolver_upstreams_backend
ON resolver_upstreams(backend, sort_order ASC);
