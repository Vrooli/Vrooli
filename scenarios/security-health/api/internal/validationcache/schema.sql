-- Bounded normalized scanner evidence. One current row exists per
-- (scenario, scanner); a correctness-key change replaces it atomically.
-- Scanner-native output and secret match material are never stored here.
CREATE TABLE IF NOT EXISTS validation_evidence_cache (
    scenario        TEXT NOT NULL,
    scanner         TEXT NOT NULL,
    fingerprint     TEXT NOT NULL,
    payload_version INTEGER NOT NULL,
    findings_json   TEXT NOT NULL,
    created_at      TEXT NOT NULL,
    expires_at      TEXT NOT NULL,
    PRIMARY KEY (scenario, scanner)
);

CREATE INDEX IF NOT EXISTS idx_validation_evidence_expiry
    ON validation_evidence_cache(expires_at);
