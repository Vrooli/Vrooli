CREATE TABLE IF NOT EXISTS redemptions (
    id TEXT PRIMARY KEY,
    holder_id TEXT NOT NULL REFERENCES holders(id),
    catalog_entry_id TEXT NOT NULL REFERENCES catalog_entries(id),
    token_type_id TEXT NOT NULL REFERENCES token_types(id),
    grant_id TEXT NOT NULL REFERENCES grants(id),
    amount INTEGER NOT NULL CHECK (amount > 0),
    idempotency_key TEXT NOT NULL UNIQUE CHECK (length(trim(idempotency_key)) > 0),
    state TEXT NOT NULL CHECK (state IN ('pending_approval', 'settled', 'denied')),
    approval_posture TEXT NOT NULL CHECK (approval_posture IN ('immediate', 'requires_approval')),
    decider_subject TEXT NOT NULL DEFAULT '',
    decision_reason TEXT NOT NULL DEFAULT '',
    requested_at TEXT NOT NULL,
    decided_at TEXT,
    settled_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_redemptions_pending
    ON redemptions(state, requested_at, id);

CREATE TABLE IF NOT EXISTS reservations (
    id TEXT PRIMARY KEY,
    redemption_id TEXT NOT NULL UNIQUE REFERENCES redemptions(id),
    holder_id TEXT NOT NULL REFERENCES holders(id),
    token_type_id TEXT NOT NULL REFERENCES token_types(id),
    amount INTEGER NOT NULL CHECK (amount > 0),
    state TEXT NOT NULL CHECK (state IN ('active', 'settled', 'released')),
    created_at TEXT NOT NULL,
    released_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_reservations_available_balance
    ON reservations(holder_id, token_type_id, state);

CREATE TABLE IF NOT EXISTS redemption_events (
    id TEXT PRIMARY KEY,
    redemption_id TEXT NOT NULL REFERENCES redemptions(id),
    kind TEXT NOT NULL CHECK (kind IN ('requested', 'approved', 'denied', 'settled')),
    actor_subject TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_redemption_events_history
    ON redemption_events(redemption_id, created_at, id);

CREATE TABLE IF NOT EXISTS redemption_mutations (
    idempotency_key TEXT PRIMARY KEY CHECK (length(trim(idempotency_key)) > 0),
    redemption_id TEXT NOT NULL REFERENCES redemptions(id),
    operation TEXT NOT NULL CHECK (operation IN ('approved', 'denied')),
    created_at TEXT NOT NULL
);
