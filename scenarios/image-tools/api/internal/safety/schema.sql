-- Responsible-Use deployment gate (IMG-P1-015): the consent audit log.
-- On a public deployment tier, and for any route that leaves this machine, every
-- affirmed high-consent-weight op (identity/body/clothing/pose-altering) records
-- one row here, so an operator can audit who affirmed consent for which
-- identity-altering edits and where the work was sent (`route`). A local run on
-- the local tier writes nothing. Forward-only declarative; an install that
-- predates `route` gains it through ConsentLog.EnsureSchema at boot.
CREATE TABLE IF NOT EXISTS consent_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    operation   TEXT    NOT NULL,
    weight      TEXT    NOT NULL,
    tier        TEXT    NOT NULL,
    route       TEXT    NOT NULL DEFAULT '',
    affirmed_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_consent_log_affirmed_at ON consent_log (affirmed_at);
