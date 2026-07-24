# Agent Manager Storage Architecture Audit

## Last Updated

2026-07-11

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
   storage-health validator recommends splitting these into domain-owned schema
   providers.
2. `api/main.go` has outstanding routed-database/test-isolation seams and a
   captured raw database handle.
3. `api/internal/handlers/pricing.go` contains raw SQL in a transport handler.

## Scope Decision

This maintenance pass removed committed startup migration/compatibility logic
and converted the local database safely. The broader schema/provider and
routed-isolation refactor is separate architecture work and was not changed in
this pass.

## Cross-References

- `storage-health validate scenario agent-manager`
- `packages/api-core/database/schemas.go`
- `scenarios/storage-health/docs/concepts/test-isolation-contract.md`
