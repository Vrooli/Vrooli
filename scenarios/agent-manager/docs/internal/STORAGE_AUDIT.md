# Agent Manager Storage Architecture Audit

## Last Updated

2026-09-04

## Current Pattern

- [ ] Per-domain schema files (canonical)
- [x] Centralized schema

## Migration Strategy

- [x] Greenfield / personal local state — declarative schema plus a one-shot
  external migration script when local data must be preserved
- [ ] Brownfield versioned migrations

The application never mutates stored data at startup. On 2026-07-11, the local
development database was backed up and converted from legacy profile columns to
the role-only profile schema with a transactional script under
`/tmp/agent-manager/`. The script is intentionally not tracked.

## Architecture Status

- [ ] All domains own their SQL schema
- [x] Repository interfaces are present for primary data access
- [ ] System home contains only cross-cutting storage

## Issues Found

1. `api/internal/database/schema.sql` centrally defines domain tables; the
   storage-manager validator recommends splitting these into domain-owned schema
   providers.
2. The 2026-09-04 `storage-manager validate prove-isolation agent-manager`
   gate is still false and names `database.Open`, `database.EnsureSchemas`, and
   `RoutedRoots.Pick` as missing routed test-isolation seams.
3. The full storage validator reports 59 findings: approximately 255 MB outside
   declared roots, direct filesystem writers, one unproven creation mode, and
   two direct-SQL handler sites. These findings predate conversation search and
   remain authoritative debt rather than exceptions for new code.
4. `storage-manager declare inspect agent-manager` reports 5,589,508,864
   observed bytes covered by the current 6 GiB storage budgets.

## Conversation Search Decision

The `conversationsearch` domain will own only new regenerable projection tables
and indexes in a co-located `schema.sql`; it will not add columns to canonical
run/event tables. SQLite FTS remains the portable lexical floor. Qdrant is an
optional semantic companion whose collection name is resolved with
`storage.Collection("conversation-search")` so live and shadow variants remain
isolated. Semantic indexing is restricted to prose and quoted prose; explicit
tool-event recall remains a bounded SQLite text/regex operation and does not
amplify the vector corpus. Failed shadows are rolled back. Successful retired
vector generations are deliberately retained for reviewed rollback rather
than deleted automatically. Destructive search playbooks are not authorized
until the scenario's routed isolation gate passes.

The request/outcome telemetry table is content-free, automatically reclaims
rows older than 30 days, and caps retained rows at 100,000. Reclaim runs every
256 successful telemetry appends so it does not add a full retention query to
every search.

## Measured Conversation-Search Growth — 2026-09-04

The pre-feature live database was 1,241,026,560 bytes. During the first live
shadow exercise, indexing every tool call/result expanded the candidate to
about 554,974 SQLite documents; the candidate table occupied about 297 MB at
194,655 staged rows and repeated rolled-back writes grew the SQLite file to
about 2.37 GB. Qdrant had reached 5,249 points before the candidate was rolled
back. This was rejected as a vector-corpus policy, not accepted as a budget.

The correction keeps tool events available only in the SQLite projection and
limits semantic vectors to conversational prose/quoted prose. The live retry
must record final catalog/FTS/vector counts and physical bytes in
`docs/internal/PERFORMANCE.md`. Until that evidence exists, the existing 6 GiB
owned-data ceiling is an alarm ceiling, not proof that the projection is
efficient. Whole-file `VACUUM`, raw SQL deletion, and raw Qdrant collection
deletion remain prohibited recovery advice.

## Scope Decision

This maintenance pass removed committed startup migration/compatibility logic
and converted the local database safely. The broader schema/provider and
routed-isolation refactor is separate architecture work and was not changed in
this pass.

## Cross-References

- `storage-manager validate scenario agent-manager`
- `packages/api-core/database/schemas.go`
- `scenarios/storage-manager/docs/concepts/test-isolation-contract.md`
