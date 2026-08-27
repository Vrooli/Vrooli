# Storage audit — React Component Library

## Current architecture

The components domain owns the version schema and repository. `component_versions`
stores durable identity and entry metadata; `component_version_files` stores the
complete file set and SHA-256 for every version. `version_ledger` is a separate
analytical projection and does not own source bytes. Adoption, catalog evidence,
and component-test reports remain in their own domain tables.

The library working tree is a warm projection, not the sole source of version
history. `presence` distinguishes materialized files from evicted identity and
mirror rows. Components owns the materializer and validates the full mirror
before an atomic restore. Version-ledger owns graph-based reconciliation and
archive/doctor operations; it reaches the components repository through narrow
interfaces.

## Schema and migration policy

Fresh schema declarations live beside the interpreting code in
`api/internal/components/schema.sql`. Startup calls the additive
`EnsureMigrations` check for the `presence` column and index so an existing
development database survives the rollout. This is a compatibility bridge for
the live host, not a second schema owner. Future production shape changes earn
versioned migrations; local greenfield-with-data repairs use explicit stopped
one-shot SQL.

## Storage budgets and retention

The measured report workload is closure-heavy JSON, so row count alone is not a
safe budget. Component-test retention keeps five recent payloads and pinned
first-pass/first-fail evidence, then enforces a 256 MiB aggregate payload
ceiling in the same transaction. Rollup counters remain durable when payloads
are trimmed. The declared service data budget and the repository policy use the
same workload-derived ceiling.

## Risks and recovery

Eviction refuses missing or mismatched mirrors and never interprets an absent
source directory as an absent version. `versions doctor` detects missing and
mismatched evicted mirrors. `versions export-archive` and `import-archive`
provide the portable backup/recovery boundary for the host-local SQLite
identity and file mirror. Archive import is checksum-verified and refuses to
overwrite non-empty tables without explicit confirmation.
