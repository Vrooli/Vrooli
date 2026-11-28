-- Label observations. One row per (scope, label, definition hash, model, model
-- version). Existing observations are never invalidated by a pack change: a
-- compatible predecessor is copied to a new hash with state 'compatible', and a
-- breaking change is evaluated fresh. The same shape backs two tables:
-- run_annotations (production answers) and run_annotation_trials (answers from a
-- candidate pinned for a bake-off, which must never mix with production ones).
--
-- state semantics:
--   observed      the evaluator answered for this hash
--   compatible    a declared compatible predecessor's value was reused
--   not_observed  the evaluator ran and could not determine a value
--   unavailable   the evaluator could not run (no model, no signal, deferred)
--
-- probability is NULL when the method reports no calibrated distribution
-- (deterministic rules and LLM_ENUM). Absence is never stored as zero.
--
-- model is the answering model and model_version is the resource role that
-- served it; together they are who a gate measurement proves. requested_role is
-- the decision role the engine asked. run_id is the run a message-scope row
-- belongs to (equal to scope_id for a run) and occurred_at is when that message
-- was written, so a per-message time series can be read.
CREATE TABLE IF NOT EXISTS {{table}} (
    scope_kind TEXT NOT NULL CHECK (scope_kind IN ('message', 'episode', 'run')),
    scope_id TEXT NOT NULL,
    run_id TEXT NOT NULL DEFAULT '',
    label_id TEXT NOT NULL,
    definition_hash TEXT NOT NULL,
    -- Fingerprint of the bounded evidence this answer read. A changed run
    -- invalidates the cached answer even when the label definition did not.
    evidence_hash TEXT NOT NULL DEFAULT '',
    requested_role TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    model_version TEXT NOT NULL DEFAULT '',
    value_json TEXT NOT NULL DEFAULT '{}',
    probability REAL,
    confidence REAL,
    method TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT '',
    evidence_event_ids_json TEXT NOT NULL DEFAULT '[]',
    occurred_at TEXT NOT NULL DEFAULT '',
    observed_at TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('observed', 'compatible', 'not_observed', 'unavailable')),
    reason TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (scope_kind, scope_id, label_id, definition_hash, model, model_version)
);

CREATE INDEX IF NOT EXISTS idx_{{table}}_scope
    ON {{table}}(scope_kind, scope_id, label_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_{{table}}_label_window
    ON {{table}}(label_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_{{table}}_hash
    ON {{table}}(label_id, definition_hash, observed_at DESC);
