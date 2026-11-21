# Tech Tree Designer

Tech Tree Designer is Vrooli's ecosystem design environment. Its target combines source-derived observed software, a revisable intended capability horizon, and scoped proposal workspaces spanning scenarios, resources, shared packages and project artifacts.

Current implementation is narrower: a scenario interface graph, planned proto editor/materializer, and capability ontology. Repository-wide design bundles, revision-bound review and recoverable owner-directed application are documented development targets, not available guarantees. Start with [the development entry point](docs/START-HERE.md).

This scenario uses the current full-stack Vrooli layout. The old Gin/Postgres implementation was intentionally deleted; only product concepts carry forward.

## What You Get

The regenerated scenario contains the modern health surface, SQLite lifecycle wiring, generated graph/planning/ontology protos, Connect handlers, CLI bindings, and UI routes. The old Gin/Postgres implementation and the template notes example are gone.

Implemented product domains:

| Domain | Purpose |
|---|---|
| graph | Build scenario nodes and proto/Go import dependency edges from SDA behind `GraphSource`, then query and export the graph. |
| planning | Store planned scenarios as real `.proto` text, validate them, and materialize validated proto schemas. |
| ontology | Author top-down capability nodes, link live/planned scenarios through fulfillment, and compute cross-layer coverage. |

## Running

```bash
make start
make test
make logs
make stop
```

Use the scenario lifecycle commands above. Do not start binaries directly.

## CLI Surface

```bash
tech-tree-designer graph describe
tech-tree-designer graph neighbors <scenario>
tech-tree-designer graph path <from> <to>
tech-tree-designer graph ancestors <scenario>
tech-tree-designer graph export --format dot

tech-tree-designer plan create <slug>
tech-tree-designer plan add <slug> <path> --from-file <file>
tech-tree-designer plan validate <slug>
tech-tree-designer plan materialize <slug>

tech-tree-designer ontology capabilities
tech-tree-designer ontology coverage
tech-tree-designer ontology overlay
```

## Documentation Map

- [PRD.md](PRD.md)
- [docs/concepts/ARCHITECTURE.md](docs/concepts/ARCHITECTURE.md)
- [docs/concepts/DOMAINS.md](docs/concepts/DOMAINS.md)
- [docs/internal/SEAMS.md](docs/internal/SEAMS.md)
- [docs/internal/PROBLEMS.md](docs/internal/PROBLEMS.md)
- [requirements/index.json](requirements/index.json)
- [experience/README.md](experience/README.md) — draft page and journey contracts
- [docs/concepts/FLOWS.md](docs/concepts/FLOWS.md) — documentation-first review and apply
- [docs/internal/TESTING.md](docs/internal/TESTING.md) — planned qualification protocols
- [docs/internal/PERFORMANCE.md](docs/internal/PERFORMANCE.md) — scale and resource obligations

## Customize Safely

Use domain-owned folders for graph, planning, and ontology work. Keep generated proto output regenerated from schemas, and use scenario lifecycle commands for start/test/stop.
