-- Durable cohort supervision. Cursor internals stay server-side; clients only
-- receive the random cursor_token that identifies the committed checkpoint.
CREATE TABLE IF NOT EXISTS cohort_watches (
    watch_id TEXT PRIMARY KEY,
    idempotency_key TEXT NOT NULL UNIQUE,
    revision INTEGER NOT NULL,
    status INTEGER NOT NULL,
    family_execution_id TEXT NOT NULL,
    parent_run_id TEXT NOT NULL DEFAULT '',
    spec_json TEXT NOT NULL,
    cursor_token TEXT NOT NULL UNIQUE,
    cursor_version INTEGER NOT NULL,
    cursor_rowid INTEGER NOT NULL,
    retention_generation INTEGER NOT NULL,
    filter_digest TEXT NOT NULL,
    next_wake_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    terminal_at TEXT
);

CREATE TABLE IF NOT EXISTS cohort_watch_subjects (
    watch_id TEXT NOT NULL,
    family_execution_id TEXT NOT NULL,
    plan_id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    PRIMARY KEY (watch_id, run_id),
    FOREIGN KEY (watch_id) REFERENCES cohort_watches(watch_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS cohort_watch_decisions (
    decision_id TEXT PRIMARY KEY,
    watch_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    disposition INTEGER NOT NULL,
    decision_json TEXT NOT NULL,
    cursor_before TEXT NOT NULL,
    cursor_after TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY (watch_id) REFERENCES cohort_watches(watch_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS cohort_watch_actions (
    action_id TEXT PRIMARY KEY,
    watch_id TEXT NOT NULL,
    decision_id TEXT,
    idempotency_key TEXT NOT NULL UNIQUE,
    kind INTEGER NOT NULL,
    target_run_id TEXT NOT NULL DEFAULT '',
    state INTEGER NOT NULL,
    action_json TEXT NOT NULL,
    cooldown_until TEXT,
    created_at TEXT NOT NULL,
    acknowledged_at TEXT,
    FOREIGN KEY (watch_id) REFERENCES cohort_watches(watch_id) ON DELETE CASCADE,
    FOREIGN KEY (decision_id) REFERENCES cohort_watch_decisions(decision_id)
);

CREATE TABLE IF NOT EXISTS cohort_watch_action_transitions (
    action_id TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    state INTEGER NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    occurred_at TEXT NOT NULL,
    PRIMARY KEY (action_id, sequence),
    FOREIGN KEY (action_id) REFERENCES cohort_watch_actions(action_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cohort_watches_due
    ON cohort_watches(status, next_wake_at);
CREATE INDEX IF NOT EXISTS idx_cohort_watches_family
    ON cohort_watches(family_execution_id, created_at);
CREATE INDEX IF NOT EXISTS idx_cohort_watch_subjects_family
    ON cohort_watch_subjects(family_execution_id, watch_id);
CREATE INDEX IF NOT EXISTS idx_cohort_watch_subjects_plan
    ON cohort_watch_subjects(plan_id, watch_id);
CREATE INDEX IF NOT EXISTS idx_cohort_watch_subjects_run
    ON cohort_watch_subjects(run_id, watch_id);
CREATE INDEX IF NOT EXISTS idx_cohort_watch_decisions_watch
    ON cohort_watch_decisions(watch_id, created_at);
CREATE INDEX IF NOT EXISTS idx_cohort_watch_actions_watch
    ON cohort_watch_actions(watch_id, created_at);
CREATE INDEX IF NOT EXISTS idx_cohort_watch_actions_pending
    ON cohort_watch_actions(state, created_at);

-- Supervision learning is append-first and policy versions are immutable.
-- Promotion only changes lifecycle state; watches retain the version embedded
-- in their immutable WatchSpec for the lifetime of a family execution.
CREATE TABLE IF NOT EXISTS supervision_policies (
    version TEXT PRIMARY KEY,
    state TEXT NOT NULL CHECK (state IN ('candidate','active','retired','rejected','rolled_back')),
    policy_json TEXT NOT NULL,
    policy_digest TEXT NOT NULL,
    supersedes TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL,
    created_at TEXT NOT NULL,
    reviewed_by TEXT NOT NULL DEFAULT '',
    reviewed_at TEXT,
    rejection_reason TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_supervision_policy_one_active
    ON supervision_policies(state) WHERE state = 'active';

CREATE TABLE IF NOT EXISTS supervision_policy_gates (
    version TEXT PRIMARY KEY,
    sample_count INTEGER NOT NULL,
    false_positives INTEGER NOT NULL,
    false_negatives INTEGER NOT NULL,
    safety_violations INTEGER NOT NULL,
    completion_impact REAL NOT NULL,
    rollout_samples INTEGER NOT NULL,
    replay_passed INTEGER NOT NULL,
    rollout_passed INTEGER NOT NULL,
    evaluated_at TEXT NOT NULL,
    FOREIGN KEY (version) REFERENCES supervision_policies(version)
);

CREATE TABLE IF NOT EXISTS supervision_outcomes (
    outcome_id TEXT PRIMARY KEY,
    idempotency_key TEXT NOT NULL UNIQUE,
    policy_version TEXT NOT NULL,
    family_execution_id TEXT NOT NULL,
    watch_id TEXT NOT NULL,
    decision_id TEXT NOT NULL,
    action_id TEXT NOT NULL DEFAULT '',
    child_run_id TEXT NOT NULL DEFAULT '',
    evidence_json TEXT NOT NULL,
    predicted_class TEXT NOT NULL,
    observed_class TEXT NOT NULL,
    overridden INTEGER NOT NULL,
    counterexample INTEGER NOT NULL,
    safety_violation INTEGER NOT NULL,
    completion_impact REAL NOT NULL,
    supersedes_outcome_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    FOREIGN KEY (policy_version) REFERENCES supervision_policies(version)
);

CREATE INDEX IF NOT EXISTS idx_supervision_outcomes_policy
    ON supervision_outcomes(policy_version, created_at);

CREATE TABLE IF NOT EXISTS supervision_policy_control (
    singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
    disabled INTEGER NOT NULL,
    reason TEXT NOT NULL,
    updated_by TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Exact bounded inputs are committed with the decision and cursor. Raw event bodies are excluded.
CREATE TABLE IF NOT EXISTS supervision_evaluation_inputs (
 decision_id TEXT PRIMARY KEY REFERENCES cohort_watch_decisions(decision_id),
 input_json TEXT NOT NULL,
 created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS supervision_policy_artifacts (
 version TEXT PRIMARY KEY REFERENCES supervision_policies(version),
 evaluator_digest TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS supervision_replay_evidence (
 version TEXT NOT NULL REFERENCES supervision_policies(version),
 decision_id TEXT NOT NULL REFERENCES cohort_watch_decisions(decision_id),
 outcome_id TEXT NOT NULL REFERENCES supervision_outcomes(outcome_id),
 evaluator_digest TEXT NOT NULL,
 predicted_class TEXT NOT NULL,
 observed_class TEXT NOT NULL,
 disposition INTEGER NOT NULL,
 evaluated_at TEXT NOT NULL,
 PRIMARY KEY(version, decision_id)
);

-- New assessments invalidate cached promotion verdicts. Quiet observations do not.
CREATE TRIGGER IF NOT EXISTS supervision_assessment_invalidates_gates
AFTER INSERT ON supervision_outcomes WHEN NEW.observed_class <> ''
BEGIN DELETE FROM supervision_policy_gates; END;


-- A monotonic corpus revision detects inserts and retention deletions while a
-- bounded evaluator runs without holding a database transaction open.
CREATE TABLE IF NOT EXISTS supervision_corpus_revision (
 singleton INTEGER PRIMARY KEY CHECK(singleton=1), revision INTEGER NOT NULL
);
INSERT INTO supervision_corpus_revision(singleton,revision) VALUES(1,0) ON CONFLICT(singleton) DO NOTHING;
CREATE TRIGGER IF NOT EXISTS supervision_corpus_insert
AFTER INSERT ON supervision_outcomes WHEN NEW.observed_class <> ''
BEGIN UPDATE supervision_corpus_revision SET revision=revision+1 WHERE singleton=1; END;
CREATE TRIGGER IF NOT EXISTS supervision_corpus_delete
AFTER DELETE ON supervision_outcomes WHEN OLD.observed_class <> ''
BEGIN
 UPDATE supervision_corpus_revision SET revision=revision+1 WHERE singleton=1;
 DELETE FROM supervision_policy_gates;
END;

-- Effective provider/model/sampling identity is pinned independently of program
-- content. A changed gateway route cannot silently reuse experiment evidence.
CREATE TABLE IF NOT EXISTS supervision_inference_identity (
 version TEXT PRIMARY KEY REFERENCES supervision_policies(version),
 identity_json TEXT NOT NULL,
 identity_digest TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS supervision_outcome_measurements (
 outcome_id TEXT PRIMARY KEY REFERENCES supervision_outcomes(outcome_id) ON DELETE CASCADE,
 completion_impact_observed INTEGER NOT NULL
);
