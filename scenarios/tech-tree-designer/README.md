# Tech Tree Designer

Tech Tree Designer is Vrooli's planning surface for scenario interfaces. It renders the actual cross-scenario interface graph and lets agents design future scenarios as proto contracts before implementation.

This scenario was greenfield-regenerated from the `react-vite` template. The old Gin/Postgres implementation was intentionally deleted; only product concepts carry forward.

## What You Get

The regenerated scenario contains the modern health surface, SQLite lifecycle wiring, generated graph/planning/roadmap protos, Connect handlers, CLI bindings, and UI routes. The old Gin/Postgres implementation and the template notes example are gone.

Implemented product domains:

| Domain | Purpose |
|---|---|
| graph | Build scenario nodes and proto-import dependency edges from `proto-health` behind `GraphSource`, then query and export the graph. |
| planning | Store planned scenarios as real `.proto` text, validate them, and materialize validated proto schemas. |
| roadmap | Attach sectors, tiers, and milestones as graph metadata overlays and roll up progress. |

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

tech-tree-designer roadmap sectors
tech-tree-designer roadmap progress
```

## Key Documents

## Documentation Map

- [PRD.md](PRD.md)
- [docs/concepts/ARCHITECTURE.md](docs/concepts/ARCHITECTURE.md)
- [docs/concepts/DOMAINS.md](docs/concepts/DOMAINS.md)
- [docs/internal/SEAMS.md](docs/internal/SEAMS.md)
- [docs/internal/PROBLEMS.md](docs/internal/PROBLEMS.md)
- [requirements/index.json](requirements/index.json)

## Customize Safely

Use domain-owned folders for graph, planning, and roadmap work. Keep generated proto output regenerated from schemas, and use scenario lifecycle commands for start/test/stop.
