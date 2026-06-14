# Quick Start

## Purpose Of This Document

Get the regenerated Tech Tree Designer running through the Vrooli lifecycle.

## Prerequisites

- Vrooli repo setup has completed.
- `proto-health` is available when testing the live graph domain.
- UI dependencies are installed by the scenario lifecycle or generator hook.

## Start

```bash
cd scenarios/tech-tree-designer
make start
```

Health:

```bash
curl -s "http://localhost:${API_PORT}/health"
tech-tree-designer status
```

## Test

```bash
cd scenarios/tech-tree-designer
make test
```

For focused local checks during implementation:

```bash
cd scenarios/tech-tree-designer/api && GOWORK=off go test ./...
cd scenarios/tech-tree-designer/cli && GOWORK=off go test ./...
cd scenarios/tech-tree-designer/ui && corepack pnpm test
```

## Current Limitations

The regenerated scenario exposes graph, planning, and roadmap surfaces through the Connect API, CLI, and UI. Use `tech-tree-designer graph --help`, `tech-tree-designer plan --help`, and `tech-tree-designer roadmap --help` for command details.

Deferred integrations are tracked in [`internal/PROBLEMS.md`](internal/PROBLEMS.md): AI strategic analysis, the future SDA graph source, scenario scaffold generation from planned nodes, and proto-health import governance.

## Cross-References

- [`../PRD.md`](../PRD.md)
- [`concepts/ARCHITECTURE.md`](concepts/ARCHITECTURE.md)
- [`internal/PROBLEMS.md`](internal/PROBLEMS.md)
