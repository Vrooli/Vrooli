CREATE TABLE IF NOT EXISTS music_decompositions (
  track_sha256 TEXT NOT NULL,
  analysis_job_id TEXT NOT NULL,
  source_sha256 TEXT NOT NULL,
  result_json TEXT NOT NULL,
  received_at TEXT NOT NULL,
  PRIMARY KEY (track_sha256, analysis_job_id),
  CHECK (length(track_sha256) = 64),
  CHECK (length(source_sha256) = 64),
  CHECK (track_sha256 = source_sha256)
);
CREATE INDEX IF NOT EXISTS music_decompositions_latest
  ON music_decompositions(track_sha256, received_at DESC);

CREATE TABLE IF NOT EXISTS music_decomposition_sources (
  track_sha256 TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  artist TEXT NOT NULL DEFAULT '',
  uploader TEXT NOT NULL DEFAULT '',
  file_name TEXT NOT NULL,
  media_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  acquired_at TEXT NOT NULL DEFAULT '',
  CHECK (length(track_sha256) = 64),
  CHECK (size_bytes > 0 AND size_bytes <= 134217728)
);

CREATE TABLE IF NOT EXISTS music_decomposition_reviews (
  review_id TEXT PRIMARY KEY,
  track_sha256 TEXT NOT NULL,
  analysis_job_id TEXT NOT NULL,
  verdict TEXT NOT NULL CHECK (verdict IN ('confirmed','corrected','needs_work')),
  note TEXT NOT NULL,
  corrections_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (track_sha256, analysis_job_id)
    REFERENCES music_decompositions(track_sha256, analysis_job_id)
);
CREATE INDEX IF NOT EXISTS music_decomposition_reviews_track
  ON music_decomposition_reviews(track_sha256, created_at DESC);

CREATE TABLE IF NOT EXISTS music_decomposition_taste_features (
  track_sha256 TEXT NOT NULL,
  analysis_job_id TEXT NOT NULL,
  review_id TEXT NOT NULL,
  attribute TEXT NOT NULL,
  value REAL NOT NULL,
  confidence REAL NOT NULL CHECK (confidence > 0 AND confidence <= 1),
  evidence_source TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (track_sha256, analysis_job_id, review_id, attribute),
  FOREIGN KEY (review_id) REFERENCES music_decomposition_reviews(review_id),
  CHECK (length(track_sha256) = 64),
  CHECK (value >= -1 AND value <= 1)
);
CREATE INDEX IF NOT EXISTS music_decomposition_taste_features_track
  ON music_decomposition_taste_features(track_sha256, attribute);
