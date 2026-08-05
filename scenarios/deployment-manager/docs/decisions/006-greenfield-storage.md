# ADR-006: Greenfield Per-Domain Storage

## Status

Accepted

## Context

deployment-manager has three competing sources of schema truth:

- `initialization/postgres/schema.sql` declares six tables
- `api/migrations/002` through `005` declare four more tables plus column changes, and no code applies them
- `EnsureSchema` methods on four repositories execute `CREATE TABLE IF NOT EXISTS` at boot

The runtime methods are what actually built the live database, because the lifecycle populate step that would have applied `schema.sql` failed and every schema error is downgraded to a warning. One table, `visual_validations`, is declared only in the unapplied files and has no runtime provider, so it does not exist and the endpoints that query it fail.

The scenario has never been deployed. No user data exists anywhere.

The `storage-steer` skill defines exactly two strategies, and the dividing line is whether real users exist — not whether the dev database has data. deployment-manager is unambiguously on the greenfield side of that line.

The chosen engine is SQLite, not PostgreSQL: it is embedded, CGO-free through
`modernc.org/sqlite`, portable across local and shadow test environments, and
removes the scenario's mandatory database resource dependency.

## Decision

Adopt the greenfield per-domain storage architecture.

**Each domain owns its schema.** A domain ships `schema.sql` next to the code that interprets it, embedded through `//go:embed` and exposed as `func Schema() string`. Adding a column is one file edit. Deleting a domain is deleting its folder.

**One application path.** The API binary calls `database.EnsureSchemas(ctx, db, modules.AllSchemas()...)` at boot from a modules registry. No repository executes DDL of its own.

**No migrations folder.** `api/migrations/` is removed. `initialization/postgres/schema.sql` is removed. The per-domain `schema.sql` files describe the desired clean state at all times.

**Cross-cutting definitions go to a system home.** `internal/database/system.sql` holds extensions, custom types, and cross-domain views, and stays empty by default. A `CREATE TABLE` there is a signal that a domain is missing.

`maturity: "greenfield"` is declared in `.vrooli/service.json` so `storage-manager` derives the correct stage rather than defaulting to it.

## Consequences

- Schema errors become fatal at boot instead of warnings, so a missing table cannot reach a request handler
- `visual_validations` either gains a domain and a provider or is removed with the capability it serves
- Repositories hold the RoutedDB-compatible persistence seam rather than a captured production pool, which is what the test-isolation seam requires
- Any change to an existing table's columns needs a one-shot script under `/tmp/deployment-manager/`, run once and discarded, because `EnsureSchemas` only creates missing tables and indexes
- When real users exist, the scenario crosses to brownfield and the per-domain `schema.sql` becomes `migrations/001_initial.sql`. That transition is one-way and is not this decision

## References

- `prompt-manager skill read storage-steer`
- `storage-manager validate scenario deployment-manager`
- [ADR-005](005-governance-plane-boundary.md)

## Executed amendment — 2026-08-04

The selected engine is SQLite through `modernc.org/sqlite`, not PostgreSQL.
The implementation now registers embedded per-domain schemas once at boot and
fails startup on schema errors. Repositories retain the RoutedDB-compatible
seam so isolated tests can substitute an in-memory database. This amendment
records the implementation choice made during the greenfield re-platform.
