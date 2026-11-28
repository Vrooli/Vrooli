CREATE TABLE IF NOT EXISTS music_catalog_tracks (
 sha256 TEXT PRIMARY KEY, title TEXT NOT NULL, source_path TEXT NOT NULL,
 take_id TEXT NOT NULL, receipt_path TEXT NOT NULL, tags_json TEXT NOT NULL,
 approval TEXT NOT NULL, feedback TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS music_catalog_uses (
 sha256 TEXT NOT NULL, project TEXT NOT NULL, revision TEXT NOT NULL,
 status TEXT NOT NULL CHECK (status IN ('draft','accepted')),
 PRIMARY KEY (sha256, project, revision),
 FOREIGN KEY (sha256) REFERENCES music_catalog_tracks(sha256)
);
CREATE INDEX IF NOT EXISTS music_catalog_uses_project ON music_catalog_uses(project, status);
CREATE TABLE IF NOT EXISTS music_catalog_use_purposes (
 sha256 TEXT NOT NULL,
 project TEXT NOT NULL,
 revision TEXT NOT NULL,
 purpose TEXT NOT NULL,
 PRIMARY KEY (sha256, project, revision),
 FOREIGN KEY (sha256, project, revision)
   REFERENCES music_catalog_uses(sha256, project, revision)
);
CREATE TABLE IF NOT EXISTS music_catalog_track_purposes (
 sha256 TEXT NOT NULL, purpose TEXT NOT NULL,
 PRIMARY KEY (sha256, purpose),
 FOREIGN KEY (sha256) REFERENCES music_catalog_tracks(sha256)
);
CREATE TABLE IF NOT EXISTS music_catalog_ranks (
 sha256 TEXT PRIMARY KEY,
 priority INTEGER NOT NULL DEFAULT 100,
 FOREIGN KEY (sha256) REFERENCES music_catalog_tracks(sha256)
);
CREATE TABLE IF NOT EXISTS music_catalog_reservations (
 reservation_id TEXT PRIMARY KEY,
 sha256 TEXT NOT NULL,
 consumer TEXT NOT NULL,
 purpose TEXT NOT NULL,
 project TEXT NOT NULL,
 revision TEXT NOT NULL,
 state TEXT NOT NULL CHECK (state IN ('active','released','expired','committed','rolled_back')),
 created_at TEXT NOT NULL,
 expires_at TEXT NOT NULL,
 completed_at TEXT NOT NULL DEFAULT '',
 FOREIGN KEY (sha256) REFERENCES music_catalog_tracks(sha256)
);
CREATE UNIQUE INDEX IF NOT EXISTS music_catalog_one_active_reservation
 ON music_catalog_reservations(sha256) WHERE state='active';
CREATE INDEX IF NOT EXISTS music_catalog_reservations_expiry
 ON music_catalog_reservations(state, expires_at);

CREATE TABLE IF NOT EXISTS music_catalog_replenish_outbox (
 event_id TEXT PRIMARY KEY,
 reservation_id TEXT NOT NULL UNIQUE,
 sha256 TEXT NOT NULL,
 purpose TEXT NOT NULL,
 project TEXT NOT NULL,
 revision TEXT NOT NULL,
 style_id TEXT NOT NULL DEFAULT '',
 state TEXT NOT NULL CHECK (state IN ('pending','delivered','cancel_pending','cancelled','blocked')),
 attempts INTEGER NOT NULL DEFAULT 0,
 available_at TEXT NOT NULL,
 lease_token TEXT NOT NULL DEFAULT '',
 lease_until TEXT NOT NULL DEFAULT '',
 last_error TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL,
 delivered_at TEXT NOT NULL DEFAULT '',
 FOREIGN KEY (reservation_id) REFERENCES music_catalog_reservations(reservation_id)
);
CREATE INDEX IF NOT EXISTS music_catalog_replenish_due
 ON music_catalog_replenish_outbox(state, available_at, lease_until, created_at);

-- Existing records already carried launch-video as a descriptive tag. Keep
-- that established intent when the purpose-specific bank is introduced.
INSERT INTO music_catalog_track_purposes (sha256,purpose)
 SELECT t.sha256,'launch-video'
 FROM music_catalog_tracks t
 WHERE t.tags_json LIKE '%"launch-video"%'
 AND NOT EXISTS (SELECT 1 FROM music_catalog_track_purposes p
                 WHERE p.sha256=t.sha256 AND p.purpose='launch-video');
