CREATE UNIQUE INDEX IF NOT EXISTS idx_releases_idempotency
    ON releases(profile_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
