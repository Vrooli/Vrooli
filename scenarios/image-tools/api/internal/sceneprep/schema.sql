-- Immutable photographic preparation metadata; pixels/geometry stay in blobs.
CREATE TABLE IF NOT EXISTS prepared_photograph (
  id TEXT PRIMARY KEY,
  json TEXT NOT NULL,
  created_at_unix_ms INTEGER NOT NULL
);

-- Stable review request IDs make retries return the same immutable child.
CREATE TABLE IF NOT EXISTS photograph_review_operation (
  operation_id TEXT PRIMARY KEY,
  request_sha256 TEXT NOT NULL,
  photograph_id TEXT NOT NULL
);
