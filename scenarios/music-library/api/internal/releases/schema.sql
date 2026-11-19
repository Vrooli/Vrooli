CREATE TABLE IF NOT EXISTS music_releases (
  release_id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL
);

-- A release version is immutable. Editing a draft appends a new revision and
-- carries its ordered occurrences, credits, assets, rights and lineage forward.
CREATE TABLE IF NOT EXISTS music_release_versions (
  release_id TEXT NOT NULL,
  revision INTEGER NOT NULL CHECK (revision > 0),
  kind TEXT NOT NULL CHECK (kind IN ('single','ep','album')),
  status TEXT NOT NULL CHECK (status IN ('draft','review_ready','archived')),
  title TEXT NOT NULL,
  default_locale TEXT NOT NULL,
  artist_owner TEXT NOT NULL DEFAULT '',
  artist_subject_id TEXT NOT NULL DEFAULT '',
  artist_subject_version TEXT NOT NULL DEFAULT '',
  localized_metadata_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (release_id, revision),
  FOREIGN KEY (release_id) REFERENCES music_releases(release_id),
  CHECK ((artist_owner = '' AND artist_subject_id = '' AND artist_subject_version = '') OR
         (artist_owner <> '' AND artist_subject_id <> '' AND artist_subject_version <> ''))
);

CREATE TABLE IF NOT EXISTS music_release_track_occurrences (
  release_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  track_order INTEGER NOT NULL CHECK (track_order > 0),
  track_sha256 TEXT NOT NULL,
  title_override TEXT NOT NULL DEFAULT '',
  language_tags_json TEXT NOT NULL,
  PRIMARY KEY (release_id, revision, track_order),
  FOREIGN KEY (release_id, revision)
    REFERENCES music_release_versions(release_id, revision),
  FOREIGN KEY (track_sha256) REFERENCES music_catalog_tracks(sha256),
  CHECK (length(track_sha256) = 64)
);
CREATE INDEX IF NOT EXISTS music_release_track_occurrences_hash
  ON music_release_track_occurrences(track_sha256, release_id, revision);

-- Catalog metadata is copied into each release revision at save time. The source
-- audio remains in its owner; the release binds its exact content hash to the
-- catalog take and receipt that were present when this revision was created.
CREATE TABLE IF NOT EXISTS music_release_recording_provenance (
  release_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  track_order INTEGER NOT NULL,
  track_sha256 TEXT NOT NULL,
  take_id TEXT NOT NULL,
  receipt_ref TEXT NOT NULL,
  operator_track_approval TEXT NOT NULL CHECK (operator_track_approval IN ('liked','unreviewed','rejected','unverified')),
  PRIMARY KEY (release_id, revision, track_order),
  UNIQUE (release_id, revision, track_sha256),
  FOREIGN KEY (release_id, revision, track_order)
    REFERENCES music_release_track_occurrences(release_id, revision, track_order),
  FOREIGN KEY (track_sha256) REFERENCES music_catalog_tracks(sha256),
  CHECK (length(track_sha256) = 64)
);

CREATE TABLE IF NOT EXISTS music_release_credits (
  credit_id TEXT NOT NULL,
  release_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  scope TEXT NOT NULL CHECK (scope IN ('release','track')),
  track_order INTEGER NOT NULL DEFAULT 0,
  subject_owner TEXT NOT NULL DEFAULT '',
  subject_id TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL,
  credited_as TEXT NOT NULL,
  provenance_ref TEXT NOT NULL DEFAULT '',
  FOREIGN KEY (release_id, revision)
    REFERENCES music_release_versions(release_id, revision),
  PRIMARY KEY (release_id, revision, credit_id),
  CHECK ((scope='release' AND track_order=0) OR (scope='track' AND track_order>0)),
  CHECK ((subject_owner='' AND subject_id='') OR (subject_owner<>'' AND subject_id<>''))
);
CREATE INDEX IF NOT EXISTS music_release_credits_version
  ON music_release_credits(release_id, revision, scope, track_order);
CREATE TABLE IF NOT EXISTS music_release_credit_identity_versions (
  release_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  credit_id TEXT NOT NULL,
  subject_version TEXT NOT NULL,
  PRIMARY KEY (release_id,revision,credit_id),
  FOREIGN KEY (release_id,revision,credit_id)
    REFERENCES music_release_credits(release_id,revision,credit_id)
);

CREATE TABLE IF NOT EXISTS music_release_assets (
  asset_id TEXT NOT NULL,
  release_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  role TEXT NOT NULL,
  owner_scenario TEXT NOT NULL,
  asset_ref TEXT NOT NULL,
  asset_sha256 TEXT NOT NULL,
  operation_id TEXT NOT NULL DEFAULT '',
  model_id TEXT NOT NULL DEFAULT '',
  model_revision TEXT NOT NULL DEFAULT '',
  license_lane TEXT NOT NULL DEFAULT '',
  rights_status TEXT NOT NULL CHECK (rights_status IN ('unknown','review_required','personal_use','commercial_cleared','blocked')),
  provenance_ref TEXT NOT NULL,
  FOREIGN KEY (release_id, revision)
    REFERENCES music_release_versions(release_id, revision),
  PRIMARY KEY (release_id, revision, asset_id),
  CHECK (length(asset_sha256) = 64)
);
CREATE INDEX IF NOT EXISTS music_release_assets_version
  ON music_release_assets(release_id, revision, role);

CREATE TABLE IF NOT EXISTS music_release_track_rights (
  release_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  track_sha256 TEXT NOT NULL,
  source_license TEXT NOT NULL DEFAULT '',
  usage_scope TEXT NOT NULL CHECK (usage_scope IN ('unknown','personal_only','commercial','blocked')),
  decision_status TEXT NOT NULL CHECK (decision_status IN ('unknown','review_required','operator_confirmed','blocked')),
  evidence_ref TEXT NOT NULL DEFAULT '',
  note TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (release_id, revision, track_sha256),
  FOREIGN KEY (release_id, revision)
    REFERENCES music_release_versions(release_id, revision),
  FOREIGN KEY (track_sha256) REFERENCES music_catalog_tracks(sha256),
  CHECK (length(track_sha256) = 64)
);

-- Rights are explicit operator attestations against one immutable revision and
-- its catalog receipt path. Draft-submitted fields alone never prove permission.
CREATE TABLE IF NOT EXISTS music_release_rights_decisions (
  decision_id TEXT PRIMARY KEY,
  release_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  track_sha256 TEXT NOT NULL,
  source_license TEXT NOT NULL,
  usage_scope TEXT NOT NULL CHECK (usage_scope IN ('personal_only','commercial','blocked')),
  decision TEXT NOT NULL CHECK (decision IN ('confirmed','denied')),
  evidence_ref TEXT NOT NULL,
  recording_provenance_sha256 TEXT NOT NULL CHECK (length(recording_provenance_sha256)=64),
  reviewer_source TEXT NOT NULL CHECK (reviewer_source='operator'),
  note TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (release_id,revision,track_sha256)
    REFERENCES music_release_recording_provenance(release_id,revision,track_sha256),
  CHECK (length(track_sha256)=64)
);
CREATE INDEX IF NOT EXISTS music_release_rights_decisions_latest
  ON music_release_rights_decisions(release_id,revision,track_sha256,created_at DESC,decision_id DESC);

CREATE TABLE IF NOT EXISTS music_release_derivations (
  derivation_id TEXT NOT NULL,
  release_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('remix','sample')),
  source_track_sha256 TEXT NOT NULL,
  derived_track_sha256 TEXT NOT NULL,
  operation_owner TEXT NOT NULL,
  operation_id TEXT NOT NULL,
  source_start_seconds REAL NOT NULL CHECK (source_start_seconds >= 0),
  source_end_seconds REAL NOT NULL CHECK (source_end_seconds > source_start_seconds),
  derived_start_seconds REAL NOT NULL CHECK (derived_start_seconds >= 0),
  derived_end_seconds REAL NOT NULL CHECK (derived_end_seconds > derived_start_seconds),
  inherited_rights_scope TEXT NOT NULL CHECK (inherited_rights_scope IN ('unknown','personal_only','commercial','blocked')),
  evidence_ref TEXT NOT NULL,
  FOREIGN KEY (release_id, revision)
    REFERENCES music_release_versions(release_id, revision),
  PRIMARY KEY (release_id, revision, derivation_id),
  FOREIGN KEY (source_track_sha256) REFERENCES music_catalog_tracks(sha256),
  FOREIGN KEY (derived_track_sha256) REFERENCES music_catalog_tracks(sha256),
  CHECK (length(source_track_sha256) = 64 AND length(derived_track_sha256) = 64),
  CHECK (source_track_sha256 <> derived_track_sha256)
);
CREATE INDEX IF NOT EXISTS music_release_derivations_version
  ON music_release_derivations(release_id, revision, kind);

CREATE TABLE IF NOT EXISTS music_release_reviews (
  review_id TEXT PRIMARY KEY,
  release_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  decision TEXT NOT NULL CHECK (decision IN ('approved','changes_requested')),
  reviewer_source TEXT NOT NULL CHECK (reviewer_source='operator'),
  note TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (release_id, revision)
    REFERENCES music_release_versions(release_id, revision)
);
CREATE INDEX IF NOT EXISTS music_release_reviews_release
  ON music_release_reviews(release_id, revision, created_at);
CREATE TABLE IF NOT EXISTS music_release_review_pins (
  review_id TEXT PRIMARY KEY,
  release_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  edition_sha256 TEXT NOT NULL CHECK (length(edition_sha256)=64),
  FOREIGN KEY (review_id) REFERENCES music_release_reviews(review_id),
  FOREIGN KEY (release_id,revision) REFERENCES music_release_versions(release_id,revision)
);
CREATE TABLE IF NOT EXISTS music_release_review_attestations (
  review_id TEXT PRIMARY KEY,
  heard_all_tracks INTEGER NOT NULL CHECK (heard_all_tracks IN (0,1)),
  reviewed_track_hashes_json TEXT NOT NULL,
  FOREIGN KEY (review_id) REFERENCES music_release_reviews(review_id)
);

-- Review and approval evidence is append-only. Draft edits create a new
-- revision; no writer can rewrite the historical edition behind an approval.
CREATE TRIGGER IF NOT EXISTS music_release_versions_no_update BEFORE UPDATE ON music_release_versions BEGIN SELECT RAISE(ABORT,'release versions are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_versions_no_delete BEFORE DELETE ON music_release_versions BEGIN SELECT RAISE(ABORT,'release versions are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_occurrences_no_update BEFORE UPDATE ON music_release_track_occurrences BEGIN SELECT RAISE(ABORT,'release occurrences are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_occurrences_no_delete BEFORE DELETE ON music_release_track_occurrences BEGIN SELECT RAISE(ABORT,'release occurrences are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_recording_no_update BEFORE UPDATE ON music_release_recording_provenance BEGIN SELECT RAISE(ABORT,'recording provenance is immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_recording_no_delete BEFORE DELETE ON music_release_recording_provenance BEGIN SELECT RAISE(ABORT,'recording provenance is immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_credits_no_update BEFORE UPDATE ON music_release_credits BEGIN SELECT RAISE(ABORT,'release credits are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_credits_no_delete BEFORE DELETE ON music_release_credits BEGIN SELECT RAISE(ABORT,'release credits are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_credit_identity_no_update BEFORE UPDATE ON music_release_credit_identity_versions BEGIN SELECT RAISE(ABORT,'credit identity versions are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_credit_identity_no_delete BEFORE DELETE ON music_release_credit_identity_versions BEGIN SELECT RAISE(ABORT,'credit identity versions are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_assets_no_update BEFORE UPDATE ON music_release_assets BEGIN SELECT RAISE(ABORT,'release assets are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_assets_no_delete BEFORE DELETE ON music_release_assets BEGIN SELECT RAISE(ABORT,'release assets are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_rights_no_update BEFORE UPDATE ON music_release_track_rights BEGIN SELECT RAISE(ABORT,'release rights snapshots are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_rights_no_delete BEFORE DELETE ON music_release_track_rights BEGIN SELECT RAISE(ABORT,'release rights snapshots are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_rights_decisions_no_update BEFORE UPDATE ON music_release_rights_decisions BEGIN SELECT RAISE(ABORT,'release rights decisions are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_rights_decisions_no_delete BEFORE DELETE ON music_release_rights_decisions BEGIN SELECT RAISE(ABORT,'release rights decisions are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_derivations_no_update BEFORE UPDATE ON music_release_derivations BEGIN SELECT RAISE(ABORT,'release lineage is immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_derivations_no_delete BEFORE DELETE ON music_release_derivations BEGIN SELECT RAISE(ABORT,'release lineage is immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_reviews_no_update BEFORE UPDATE ON music_release_reviews BEGIN SELECT RAISE(ABORT,'release reviews are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_reviews_no_delete BEFORE DELETE ON music_release_reviews BEGIN SELECT RAISE(ABORT,'release reviews are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_review_pins_no_update BEFORE UPDATE ON music_release_review_pins BEGIN SELECT RAISE(ABORT,'release review pins are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_review_pins_no_delete BEFORE DELETE ON music_release_review_pins BEGIN SELECT RAISE(ABORT,'release review pins are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_review_attestations_no_update BEFORE UPDATE ON music_release_review_attestations BEGIN SELECT RAISE(ABORT,'release review attestations are immutable'); END;
CREATE TRIGGER IF NOT EXISTS music_release_review_attestations_no_delete BEFORE DELETE ON music_release_review_attestations BEGIN SELECT RAISE(ABORT,'release review attestations are immutable'); END;
