-- Responsible-Use deployment gate (IMG-P1-015): the consent audit log and the
-- labelled reviewer-feedback corpus.
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

-- Operator feedback on reviewer verdicts: the labelled refusal/flag corpus.
-- One row per (job, output); re-labelling replaces it. Evidence and finding are
-- the reviewer's JSON snapshotted at record time, so the corpus survives job
-- retention and never needs to read another domain's tables.
CREATE TABLE IF NOT EXISTS safety_feedback (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id           TEXT    NOT NULL,
    output_ref       TEXT    NOT NULL DEFAULT '',
    label            TEXT    NOT NULL,
    action           TEXT    NOT NULL DEFAULT '',
    reason_code      TEXT    NOT NULL DEFAULT '',
    safety_review    TEXT    NOT NULL DEFAULT '',
    score            REAL    NOT NULL DEFAULT 0,
    operation        TEXT    NOT NULL DEFAULT '',
    model_id         TEXT    NOT NULL DEFAULT '',
    profile_name     TEXT    NOT NULL DEFAULT '',
    reviewer_version TEXT    NOT NULL DEFAULT '',
    note             TEXT    NOT NULL DEFAULT '',
    actor            TEXT    NOT NULL DEFAULT '',
    evidence         TEXT    NOT NULL DEFAULT '',
    finding          TEXT    NOT NULL DEFAULT '',
    recorded_at      TEXT    NOT NULL,
    UNIQUE (job_id, output_ref)
);

CREATE INDEX IF NOT EXISTS idx_safety_feedback_label ON safety_feedback (label);
CREATE INDEX IF NOT EXISTS idx_safety_feedback_reason ON safety_feedback (reason_code);
CREATE INDEX IF NOT EXISTS idx_safety_feedback_recorded ON safety_feedback (recorded_at DESC);
