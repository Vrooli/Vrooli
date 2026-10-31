-- Responsible-Use deployment gate (IMG-P1-015): the consent audit log.
-- On a public deployment tier, every affirmed high-consent-weight op
-- (identity/body/clothing/pose-altering) records one row here, so an operator
-- can audit who affirmed consent for which identity-altering edits. The local
-- tier writes nothing (personal use is unrestricted). Forward-only declarative.
CREATE TABLE IF NOT EXISTS consent_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    operation   TEXT    NOT NULL,
    weight      TEXT    NOT NULL,
    tier        TEXT    NOT NULL,
    affirmed_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_consent_log_affirmed_at ON consent_log (affirmed_at);
