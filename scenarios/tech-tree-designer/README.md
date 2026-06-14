# Tech Tree Designer

Tech Tree Designer is Vrooli's planning surface for scenario interfaces. It renders the actual cross-scenario interface graph and will let agents design future scenarios as proto contracts before implementation.

This scenario was greenfield-regenerated from the `react-vite` template. The old Gin/Postgres implementation was intentionally deleted; only product concepts carry forward.

## What You Get

Phase 1 contains the modern scaffold, health surface, SQLite lifecycle wiring, generated health/error protos, and TTD-specific planning docs. The template notes example has been removed.

Planned product domains:

| Domain | Purpose |
|---|---|
| graph | Build scenario nodes and proto-import dependency edges from `proto-health` behind `GraphSource`. |
| planning | Store planned scenarios as real `.proto` text, validate them, and materialize validated proto schemas. |
| roadmap | Attach sectors, tiers, and milestones as graph metadata overlays. |

## Running

```bash
make start
make test
make logs
make stop
```

Use the scenario lifecycle commands above. Do not start binaries directly.

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
