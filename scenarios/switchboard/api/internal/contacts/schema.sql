CREATE TABLE IF NOT EXISTS switchboard_contacts (
 id TEXT PRIMARY KEY,
 channel_id TEXT NOT NULL,
 address TEXT NOT NULL,
 display_name TEXT NOT NULL DEFAULT '',
 tier TEXT NOT NULL DEFAULT 'stranger',
 first_seen TEXT NOT NULL,
 last_seen TEXT NOT NULL,
 message_count INTEGER NOT NULL DEFAULT 0,
 UNIQUE(channel_id, address)
);
CREATE TABLE IF NOT EXISTS switchboard_participants (
 thread_id TEXT NOT NULL,
 contact_id TEXT NOT NULL REFERENCES switchboard_contacts(id),
 joined_at TEXT NOT NULL,
 PRIMARY KEY(thread_id, contact_id)
);
CREATE INDEX IF NOT EXISTS switchboard_participants_contact ON switchboard_participants(contact_id);
