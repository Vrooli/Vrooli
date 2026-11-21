package runannotations

import (
	_ "embed"
	"strings"
)

//go:embed schema.sql
var observationSchema string

// provenanceSchema is one row per annotated run: how much of its operator text
// a person wrote. It is the population denominator for affect labels, written as
// a by-product of annotating (the facts are already read), never by a scan.
const provenanceSchema = `
CREATE TABLE IF NOT EXISTS run_annotation_provenance (
    run_id TEXT PRIMARY KEY,
    workload_kind TEXT NOT NULL DEFAULT '',
    operator_messages INTEGER NOT NULL DEFAULT 0,
    human_messages INTEGER NOT NULL DEFAULT 0,
    brief_messages INTEGER NOT NULL DEFAULT 0,
    last_human_at TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    policy_hash TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_run_annotation_provenance_updated
    ON run_annotation_provenance(updated_at);
`

// Schema is the DDL this package owns: the production and trial observation
// tables and the run provenance table. It is applied idempotently by
// Store.EnsureSchema; the shared migration registry is not edited so this
// package does not serialize on a shared generated artifact.
func Schema() string {
	var builder strings.Builder
	for _, table := range []string{TableObservations, TableTrials} {
		builder.WriteString(strings.ReplaceAll(observationSchema, "{{table}}", table))
		builder.WriteString("\n")
	}
	builder.WriteString(provenanceSchema)
	return builder.String()
}
