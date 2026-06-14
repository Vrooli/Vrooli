# Testing - Tech Tree Designer

## Purpose Of This Document

Record the test strategy for the regenerated scenario.

## Current Scaffold

Phase 1 validates:

- API health handler, module registry, endpoint generation, and server wiring.
- CLI app shell and empty domain aggregator.
- UI health card, routing, accessibility, selectors, i18n, theme provider, and test utilities.
- Proto generation for health and error contracts.

## Commands

```bash
vrooli scenario test tech-tree-designer
cd scenarios/tech-tree-designer/api && GOWORK=off go test ./...
cd scenarios/tech-tree-designer/cli && GOWORK=off go test ./...
cd scenarios/tech-tree-designer/ui && corepack pnpm test
```

## Future Coverage

| Domain | Required Coverage |
|---|---|
| graph | fake GraphSource mapping, proto-health client mapping, graph queries, export formats, Connect handler, CLI commands |
| planning | SQLite file-tree CRUD, protocompile validation, findings, materialize guard, CLI filesystem-feel commands |
| roadmap | sector/tier/milestone storage, overlay attachment, progress rollup |
| UI | D3 graph rendering, planned/live styling, editor CRUD, validation findings, loading/error/empty states |

## Requirement Tags

Tag tests with `[REQ:<id>]` as implementation lands so requirements sync can earn statuses from live evidence.

## Cross-References

- [`../../requirements/index.json`](../../requirements/index.json)
- [`SEAMS.md`](SEAMS.md)
- [`PROBLEMS.md`](PROBLEMS.md)
