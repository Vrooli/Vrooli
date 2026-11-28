CREATE TABLE IF NOT EXISTS library_bundle_revisions (
  bundle_id TEXT NOT NULL,
  revision INTEGER NOT NULL CHECK (revision > 0),
  manifest_sha256 TEXT NOT NULL CHECK (length(manifest_sha256) = 64),
  manifest_json TEXT NOT NULL,
  review_sha256 TEXT NOT NULL CHECK (length(review_sha256) = 64),
  review_json TEXT NOT NULL,
  published_at TEXT NOT NULL,
  PRIMARY KEY (bundle_id, revision)
);

CREATE TABLE IF NOT EXISTS library_bundle_artifacts (
  bundle_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  artifact_id TEXT NOT NULL,
  blob_key TEXT NOT NULL,
  PRIMARY KEY (bundle_id, revision, artifact_id),
  FOREIGN KEY (bundle_id, revision)
    REFERENCES library_bundle_revisions(bundle_id, revision) ON DELETE RESTRICT
);

CREATE TRIGGER IF NOT EXISTS library_bundle_revisions_no_update
BEFORE UPDATE ON library_bundle_revisions
BEGIN
  SELECT RAISE(ABORT, 'library bundle revisions are immutable');
END;

CREATE TRIGGER IF NOT EXISTS library_bundle_revisions_no_delete
BEFORE DELETE ON library_bundle_revisions
BEGIN
  SELECT RAISE(ABORT, 'library bundle revisions are immutable');
END;

CREATE TRIGGER IF NOT EXISTS library_bundle_artifacts_no_update
BEFORE UPDATE ON library_bundle_artifacts
BEGIN
  SELECT RAISE(ABORT, 'library bundle artifacts are immutable');
END;

CREATE TRIGGER IF NOT EXISTS library_bundle_artifacts_no_delete
BEFORE DELETE ON library_bundle_artifacts
BEGIN
  SELECT RAISE(ABORT, 'library bundle artifacts are immutable');
END;

CREATE TABLE IF NOT EXISTS library_intake_candidate_revisions (
  candidate_id TEXT NOT NULL,
  revision INTEGER NOT NULL CHECK (revision > 0),
  sha256 TEXT NOT NULL CHECK (length(sha256) = 64),
  candidate_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (candidate_id, revision)
);

CREATE TRIGGER IF NOT EXISTS library_intake_candidate_revisions_no_update
BEFORE UPDATE ON library_intake_candidate_revisions
BEGIN
  SELECT RAISE(ABORT, 'library intake candidate revisions are immutable');
END;

CREATE TRIGGER IF NOT EXISTS library_intake_candidate_revisions_no_delete
BEFORE DELETE ON library_intake_candidate_revisions
BEGIN
  SELECT RAISE(ABORT, 'library intake candidate revisions are immutable');
END;
