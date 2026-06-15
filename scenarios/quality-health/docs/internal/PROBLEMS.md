# Problems — Quality Health

## Open Problems

### P1 — Autofix coverage is intentionally narrow

Phase 2 implements safe preview/apply for `TS_CONFIG_STRICT`. ESLint, golangci, and Makefile autofix remain planned because those writers need stronger structure preservation before mutation is safe.

### P1 — Run history storage decision is deferred

Quality Health v1 can be stateless. If `explain` needs latest-run lookup or the UI needs run history, add SQLite with a retention policy and tests.

### P2 — Python quality contracts are deferred

The plan mentions Python support if Code Facts or current lint heuristics justify it. Keep Python contract work deferred until surface metadata is reliable.

## Resolved Problems

### Template sample domain removed

The generated `notes` reference stack was removed from proto, API, CLI, UI, routes, selectors, locale strings, endpoint metadata, and active tests during Phase 1.

### UI product domain implemented

Phase 3 replaced the generated placeholder dashboard with `ui/src/features/audit/ScenarioAuditWorkbench.tsx`, wired to the Phase 2 Connect `AuditService` model. The UI covers the audit overview, surface breakdown, findings workbench, contract detail, command results, and explicit autofix preview/apply controls.
