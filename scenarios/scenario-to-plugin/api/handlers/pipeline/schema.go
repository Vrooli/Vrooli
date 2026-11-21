package pipeline

// Schema owns durable package references. Artifact bytes stay in the
// file-store root; SQLite contains only the identity, digest, and references
// needed to resume gate decisions after an API restart.
func Schema() string {
	return `
CREATE TABLE IF NOT EXISTS plugin_packages (
  id TEXT PRIMARY KEY,
  scenario TEXT NOT NULL,
  source_revision TEXT NOT NULL,
  digest TEXT NOT NULL,
  artifact_root TEXT NOT NULL,
  state TEXT NOT NULL,
  mcp_authentication TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS plugin_publications (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  package_id TEXT NOT NULL,
  channel TEXT NOT NULL,
  digest TEXT NOT NULL,
  coordinate TEXT NOT NULL,
  withdrawn INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(package_id, channel)
);`
}
