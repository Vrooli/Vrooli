CREATE TABLE IF NOT EXISTS device_control_desktop_sessions (
    destination TEXT PRIMARY KEY,
    state TEXT NOT NULL
);

-- Historical cleanup evidence is separate from live input state. Epochs use text
-- to preserve the full uint64 domain without signed SQLite integer conversion.
CREATE TABLE IF NOT EXISTS device_control_desktop_cleanup_receipts (
    destination TEXT NOT NULL,
    epoch TEXT NOT NULL,
    receipt TEXT NOT NULL,
    PRIMARY KEY(destination, epoch)
);

-- Signed revocations permanently fence a grant, even if stale owner status
-- later lists it again. The original lease binding is immutable.
CREATE TABLE IF NOT EXISTS device_control_desktop_revocations (
 destination TEXT NOT NULL,
 grant_id TEXT NOT NULL,
 lease_metadata TEXT NOT NULL,
 PRIMARY KEY(destination,grant_id)
);
