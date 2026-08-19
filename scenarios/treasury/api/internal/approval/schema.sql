CREATE TABLE IF NOT EXISTS approval_requests (
    id TEXT PRIMARY KEY,
    authorization_id TEXT NOT NULL UNIQUE REFERENCES authorizations(id),
    mandate_id TEXT NOT NULL,
    requesting_agent TEXT NOT NULL,
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    currency TEXT NOT NULL,
    counterparty TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued','approved','declined','expired')),
    resolver_identity TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    resolved_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS approval_book_bindings (
    approval_id TEXT PRIMARY KEY REFERENCES approval_requests(id),
    book_id TEXT NOT NULL CHECK (length(trim(book_id)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_approval_book_bindings_book
ON approval_book_bindings(book_id, approval_id);

CREATE TRIGGER IF NOT EXISTS approval_book_binding_insert_guard
BEFORE INSERT ON approval_book_bindings
FOR EACH ROW
WHEN NOT EXISTS (
    SELECT 1
    FROM approval_requests p
    JOIN authorization_book_bindings a ON a.authorization_id = p.authorization_id
    WHERE p.id = NEW.approval_id AND a.book_id = NEW.book_id
)
BEGIN
    SELECT RAISE(ABORT, 'approval book must match authorization book');
END;

CREATE TRIGGER IF NOT EXISTS approval_book_binding_update_guard
BEFORE UPDATE OF book_id, approval_id ON approval_book_bindings
FOR EACH ROW
WHEN NOT EXISTS (
    SELECT 1
    FROM approval_requests p
    JOIN authorization_book_bindings a ON a.authorization_id = p.authorization_id
    WHERE p.id = NEW.approval_id AND a.book_id = NEW.book_id
)
BEGIN
    SELECT RAISE(ABORT, 'approval book must match authorization book');
END;

CREATE TABLE IF NOT EXISTS approval_relay_attempts (
    id TEXT PRIMARY KEY,
    approval_id TEXT NOT NULL REFERENCES approval_requests(id),
    outcome TEXT NOT NULL CHECK (outcome IN ('skipped','sent','failed')),
    error TEXT NOT NULL DEFAULT '',
    attempted_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_approval_queue ON approval_requests(status, created_at);
CREATE INDEX IF NOT EXISTS idx_approval_relay_attempts ON approval_relay_attempts(approval_id, attempted_at);
