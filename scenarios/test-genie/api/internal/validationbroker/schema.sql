CREATE TABLE IF NOT EXISTS validation_receipts (
    receipt_id TEXT PRIMARY KEY,
    lineage_id TEXT NOT NULL,
    intent_id TEXT NOT NULL UNIQUE,
    caller_scenario TEXT NOT NULL,
    caller_execution_id TEXT NOT NULL DEFAULT '',
    plan_id TEXT NOT NULL DEFAULT '',
    idempotency_key TEXT NOT NULL,
    intent_fingerprint TEXT NOT NULL,
    execution_key TEXT NOT NULL,
    state INTEGER NOT NULL,
    is_producer INTEGER NOT NULL DEFAULT 0 CHECK (is_producer IN (0, 1)),
    intent_proto BLOB NOT NULL,
    receipt_proto BLOB NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    revision INTEGER NOT NULL DEFAULT 1,
    UNIQUE (caller_scenario, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_validation_receipts_execution
    ON validation_receipts (execution_key, state, is_producer);
CREATE INDEX IF NOT EXISTS idx_validation_receipts_caller
    ON validation_receipts (caller_scenario, caller_execution_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_validation_receipts_plan
    ON validation_receipts (plan_id, created_at DESC);

-- At most one receipt owns producer work for a compatible execution identity.
-- Attached observers have their own durable receipts in the same lineage.
CREATE UNIQUE INDEX IF NOT EXISTS idx_validation_receipts_active_producer
    ON validation_receipts (execution_key)
    WHERE is_producer = 1 AND state IN (1, 2, 3, 4, 5);

-- Immutable lifecycle evidence. receipt_proto is the complete post-transition
-- projection, so historical decisions remain explainable after later updates.
CREATE TABLE IF NOT EXISTS validation_receipt_transitions (
    receipt_id TEXT NOT NULL,
    revision INTEGER NOT NULL,
    from_state INTEGER NOT NULL,
    to_state INTEGER NOT NULL,
    reason_code INTEGER NOT NULL,
    receipt_proto BLOB NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (receipt_id, revision),
    FOREIGN KEY (receipt_id) REFERENCES validation_receipts(receipt_id)
);

CREATE INDEX IF NOT EXISTS idx_validation_receipt_transitions_created
    ON validation_receipt_transitions (created_at, receipt_id);

-- Shadow comparisons are observation-only migration evidence. They never
-- choose or rewrite the production receipt outcome.
CREATE TABLE IF NOT EXISTS validation_shadow_comparisons (
    comparison_id TEXT PRIMARY KEY,
    source_kind TEXT NOT NULL,
    source_id TEXT NOT NULL,
    receipt_id TEXT NOT NULL,
    receipt_revision INTEGER NOT NULL,
    legacy_state TEXT NOT NULL,
    receipt_state INTEGER NOT NULL,
    matched INTEGER NOT NULL CHECK (matched IN (0, 1)),
    reason_code TEXT NOT NULL,
    legacy_evidence_count INTEGER NOT NULL,
    receipt_evidence_count INTEGER NOT NULL,
    observed_at TEXT NOT NULL,
    UNIQUE (source_kind, source_id, receipt_id, receipt_revision)
);

CREATE INDEX IF NOT EXISTS idx_validation_shadow_mismatches
    ON validation_shadow_comparisons (matched, reason_code, observed_at);
