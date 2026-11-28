CREATE TABLE IF NOT EXISTS journal_events (
    id TEXT PRIMARY KEY,
    token_type_id TEXT NOT NULL REFERENCES token_types(id),
    holder_id TEXT NOT NULL CHECK (length(trim(holder_id)) > 0),
    amount INTEGER NOT NULL CHECK (amount > 0),
    kind TEXT NOT NULL CHECK (kind IN ('mint', 'credit', 'debit', 'reversal', 'expiry')),
    cause_reference TEXT NOT NULL CHECK (length(trim(cause_reference)) > 0),
    reason TEXT NOT NULL DEFAULT '',
    actor_identity TEXT NOT NULL DEFAULT '',
    actor_kind TEXT NOT NULL DEFAULT 'operator' CHECK (actor_kind IN ('operator', 'agent')),
    actor_verification_status TEXT NOT NULL DEFAULT 'absent' CHECK (actor_verification_status IN ('verified', 'unavailable', 'invalid', 'absent')),
    actor_run_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_journal_events_projection
    ON journal_events(holder_id, token_type_id, created_at, id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_journal_events_one_reversal
    ON journal_events(cause_reference)
    WHERE kind = 'reversal';

CREATE TABLE IF NOT EXISTS journal_reversal_receipts (
    idempotency_key TEXT PRIMARY KEY CHECK (length(trim(idempotency_key)) > 0),
    original_event_id TEXT NOT NULL REFERENCES journal_events(id),
    reversal_event_id TEXT NOT NULL UNIQUE REFERENCES journal_events(id),
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS balance_projections (
    holder_id TEXT NOT NULL,
    token_type_id TEXT NOT NULL REFERENCES token_types(id),
    amount INTEGER NOT NULL,
    rebuilt_at TEXT NOT NULL,
    PRIMARY KEY (holder_id, token_type_id)
);
