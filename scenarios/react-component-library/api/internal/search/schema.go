// Package search owns the one lexical retrieval index for the workbench.
//
// The index is deliberately a projection, not a source of truth: workspaces,
// pages, displays, states, catalog assets, commands and findings are all owned
// by other packages, and this package only rebuilds search_docs/search_fts from
// them. That is why Reindex reads the owned tables and files directly instead of
// duplicating their write paths here.
package search

// Schema is the forward-only declarative schema for the search projection.
// EnsureSchemas applies it at boot; Reindex never creates tables because a
// missing schema is a wiring defect, not a search-time concern.
//
// search_fts is an external-content-free FTS5 table: every row is written
// explicitly beside its search_docs row in the same transaction. Keeping the two
// independent (rather than using content=search_docs) means a failed rebuild can
// be rolled back without leaving FTS internal tables out of step, and it keeps
// the column boost order in bm25() explicit.
func Schema() string {
	return `
CREATE TABLE IF NOT EXISTS search_docs (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  title TEXT NOT NULL,
  subtitle TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  href TEXT NOT NULL DEFAULT '',
  availability TEXT NOT NULL DEFAULT 'available',
  badges_json TEXT NOT NULL DEFAULT '[]',
  relations_json TEXT NOT NULL DEFAULT '[]',
  updated_at TEXT NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS search_fts USING fts5(
  id UNINDEXED,
  kind UNINDEXED,
  title,
  subtitle,
  body
);

CREATE TABLE IF NOT EXISTS search_state (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
`
}

// Schema keys written into search_state. They live here so the status reader and
// the reindex writer cannot drift apart.
const (
	stateIndexedAt     = "indexed_at"
	stateDocumentCount = "document_count"
	stateLastError     = "last_error"
	// stateWarnings records non-fatal degradations (for example an availability
	// resolver error for one asset). It is separate from last_error because a
	// warning does not mean the index is stale: the rebuild committed.
	stateWarnings = "warnings"
	// stateProjectionVersion records which document projection produced the
	// index. A change to hrefs, fields or ranking inputs bumps it so a startup
	// reindex rebuilds an index that would otherwise look current forever.
	stateProjectionVersion = "projection_version"
)

// projectionVersion is the current document projection. Bump it whenever the
// shape of an indexed document changes (not when source content changes).
const projectionVersion = "4"
