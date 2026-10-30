-- Operator-intervention classifications (P4-W4). One row per operator message
-- event and classifier version. status 'unclassified' means the decision
-- capability did not answer: it is retried after next_attempt_at and never
-- read as "no friction". 'initial' is a run's opening message (no agent work
-- before it); 'duplicate' is a forked rollout replaying a recorded message.
CREATE TABLE IF NOT EXISTS operator_interventions (
    event_id TEXT NOT NULL,
    classifier_version TEXT NOT NULL,
    run_id TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    occurred_at TEXT NOT NULL,
    message_key TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('classified', 'unclassified', 'initial', 'duplicate')),
    category TEXT NOT NULL DEFAULT '',
    -- probability of category; NULL when the answering method reports no distribution
    probability REAL,
    probabilities_json TEXT NOT NULL DEFAULT '{}',
    method TEXT NOT NULL DEFAULT '',
    provider TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    resource_role TEXT NOT NULL DEFAULT '',
    unclassified_reason TEXT NOT NULL DEFAULT '',
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TEXT NOT NULL DEFAULT '',
    -- agent work since the previous operator message in the run
    work_requests INTEGER NOT NULL DEFAULT 0,
    work_tool_calls INTEGER NOT NULL DEFAULT 0,
    work_tokens INTEGER NOT NULL DEFAULT 0,
    -- an earlier quality_miss report of the same symptom this one repeats
    repeat_of_event_id TEXT NOT NULL DEFAULT '',
    recorded_at TEXT NOT NULL,
    PRIMARY KEY (event_id, classifier_version)
);

CREATE INDEX IF NOT EXISTS idx_operator_interventions_window
    ON operator_interventions(classifier_version, occurred_at);
CREATE INDEX IF NOT EXISTS idx_operator_interventions_key
    ON operator_interventions(classifier_version, message_key);
