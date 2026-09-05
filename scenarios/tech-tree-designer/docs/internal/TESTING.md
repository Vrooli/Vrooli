# Testing - Tech Tree Designer

## Purpose Of This Document

Record the test strategy for the regenerated scenario.

## Current Scaffold

The regenerated scaffold has been extended into the real graph, planning, and ontology product surface. Current validation covers:

- API health handler, module registry, endpoint generation, and server wiring.
- Graph `GraphSource` mapping, graph service queries/export, Connect handler mounting, and CLI commands.
- Planning SQLite file-tree CRUD, protocompile validation, materialization, Connect handlers, CLI commands, and UI editor behavior.
- Ontology SQLite storage, import mapping, fulfillment links, coverage/focus math, overlay projection, CLI commands, and UI route behavior.
- UI health card, graph/planning/ontology routing, accessibility, selectors, i18n, theme provider, and test utilities.
- Proto generation for health, graph, planning, ontology, and error contracts.

## Commands

```bash
vrooli scenario test tech-tree-designer
cd scenarios/tech-tree-designer/api && GOWORK=off go test ./...
cd scenarios/tech-tree-designer/cli && GOWORK=off go test ./...
cd scenarios/tech-tree-designer/ui && corepack pnpm test
```

## Future Coverage

No required product domain is intentionally uncovered in the shipped scope. Future coverage should expand when deferred integrations from `PROBLEMS.md` are implemented.

| Domain | Current Coverage |
|---|---|
| graph | fake GraphSource mapping, proto-health client mapping, graph queries, export formats, Connect handler, CLI commands, UI graph rendering |
| planning | SQLite file-tree CRUD, protocompile validation, findings, materialize guard, CLI filesystem-feel commands, UI editor CRUD |
| ontology | capability storage, topology import, fulfillment, coverage/focus analytics, overlay projection, UI coverage display |
| UI | live/planned styling, editor CRUD, validation findings, loading/error/empty states |

## Requirement Tags

Tag tests with `[REQ:<id>]` as implementation lands so requirements sync can earn statuses from live evidence.

## Cross-References

- [`../../requirements/index.json`](../../requirements/index.json)
- [`SEAMS.md`](SEAMS.md)
- [`PROBLEMS.md`](PROBLEMS.md)
