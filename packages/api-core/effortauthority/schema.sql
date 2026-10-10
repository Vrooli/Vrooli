-- The authority domain owns immutable grants and conservative reservations.
CREATE TABLE IF NOT EXISTS finite_effort_authority (
 id TEXT PRIMARY KEY,
 record TEXT NOT NULL,
 version BIGINT NOT NULL DEFAULT 1
);
