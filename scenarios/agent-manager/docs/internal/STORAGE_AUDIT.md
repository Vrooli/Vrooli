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
isolated. Destructive search playbooks are not authorized until the scenario's
routed isolation gate passes.

## Scope Decision

This maintenance pass removed committed startup migration/compatibility logic
and converted the local database safely. The broader schema/provider and
routed-isolation refactor is separate architecture work and was not changed in
this pass.

## Cross-References

- `storage-manager validate scenario agent-manager`
- `packages/api-core/database/schemas.go`
- `scenarios/storage-manager/docs/concepts/test-isolation-contract.md`
