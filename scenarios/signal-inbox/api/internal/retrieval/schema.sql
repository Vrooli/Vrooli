CREATE VIRTUAL TABLE IF NOT EXISTS signal_fts USING fts5(
  signal_id UNINDEXED,
  content
);
