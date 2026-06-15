# Problems — Quality Health

## Open Problems

### P0 — API/CLI/UI product domains are not implemented

The generated sample domain has been removed, so the scenario currently exposes only the lifecycle health surface plus foundation docs. Phase 2 owns the API/CLI Quality Health domains; Phase 3 owns the product UI.

### P1 — Run history storage decision is deferred

Quality Health v1 can be stateless. If `explain` needs latest-run lookup or the UI needs run history, add SQLite with a retention policy and tests.

### P2 — Python quality contracts are deferred

The plan mentions Python support if Code Facts or current lint heuristics justify it. Keep Python contract work deferred until surface metadata is reliable.

## Resolved Problems

### Template sample domain removed

The generated `notes` reference stack was removed from proto, API, CLI, UI, routes, selectors, locale strings, endpoint metadata, and active tests during Phase 1.
