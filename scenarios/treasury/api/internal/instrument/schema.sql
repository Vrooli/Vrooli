CREATE TABLE IF NOT EXISTS instruments (
    id TEXT PRIMARY KEY,
    book_id TEXT NOT NULL REFERENCES books(id),
    mandate_id TEXT NOT NULL REFERENCES mandates(id),
    rail TEXT NOT NULL CHECK (length(trim(rail)) > 0),
    credential_reference TEXT NOT NULL CHECK (length(trim(credential_reference)) > 0),
    cap_minor INTEGER NOT NULL CHECK (cap_minor > 0),
    currency TEXT NOT NULL CHECK (length(trim(currency)) > 0),
    counterparty TEXT NOT NULL CHECK (length(trim(counterparty)) > 0),
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- INVARIANT: onlyOperatorBeneficiaryCanBeRepresented
-- An instrument and its mandate must remain in the same operator-owned book.
CREATE TRIGGER IF NOT EXISTS instruments_book_mandate_match_insert
BEFORE INSERT ON instruments
FOR EACH ROW
WHEN NOT EXISTS (
    SELECT 1 FROM mandates
    WHERE mandates.id = NEW.mandate_id AND mandates.book_id = NEW.book_id
)
BEGIN
    SELECT RAISE(ABORT, 'instrument mandate must belong to instrument book');
END;

CREATE TRIGGER IF NOT EXISTS instruments_book_mandate_match_update
BEFORE UPDATE OF book_id, mandate_id ON instruments
FOR EACH ROW
WHEN NOT EXISTS (
    SELECT 1 FROM mandates
    WHERE mandates.id = NEW.mandate_id AND mandates.book_id = NEW.book_id
)
BEGIN
    SELECT RAISE(ABORT, 'instrument mandate must belong to instrument book');
END;
