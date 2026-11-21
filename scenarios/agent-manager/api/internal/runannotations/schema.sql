-- Label observations (WS-D). One row per (scope, label, definition hash,
-- model, model version). Existing observations are never invalidated by a pack
-- change: a compatible predecessor is copied to a new hash with state
-- 'compatible', and a breaking change is evaluated fresh.
--
-- state semantics:
--   observed      the evaluator answered for this hash
--   compatible    a declared compatible predecessor's value was reused
--   not_observed  the evaluator ran and could not determine a value
--   unavailable   the evaluator could not run (no model, no signal, deferred)
--
-- probability is NULL when the method reports no calibrated distribution
-- (deterministic rules and LLM_ENUM). Absence is never stored as zero.
CREATE TABLE IF NOT EXISTS run_annotations (
    scope_kind TEXT NOT NULL CHECK (scope_kind IN ('message', 'episode', 'run')),
    scope_id TEXT NOT NULL,
    label_id TEXT NOT NULL,
    definition_hash TEXT NOT NULL,
    -- Fingerprint of the bounded evidence this answer read. A changed run
    -- invalidates the cached answer even when the label definition did not.
    evidence_hash TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    model_version TEXT NOT NULL DEFAULT '',
    value_json TEXT NOT NULL DEFAULT '{}',
    probability REAL,
    confidence REAL,
    method TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT '',
    evidence_event_ids_json TEXT NOT NULL DEFAULT '[]',
    observed_at TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('observed', 'compatible', 'not_observed', 'unavailable')),
    reason TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (scope_kind, scope_id, label_id, definition_hash, model, model_version)
);

CREATE INDEX IF NOT EXISTS idx_run_annotations_scope
    ON run_annotations(scope_kind, scope_id, label_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_run_annotations_label_window
    ON run_annotations(label_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_run_annotations_hash
    ON run_annotations(label_id, definition_hash, observed_at DESC);
