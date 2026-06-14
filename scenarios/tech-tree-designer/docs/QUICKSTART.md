# Quick Start

## Purpose Of This Document

Get the regenerated Tech Tree Designer scaffold running through the Vrooli lifecycle.

## Prerequisites

- Vrooli repo setup has completed.
- `proto-health` is available when testing the future live graph domain.
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

Phase 1 is health-only. Graph, planning, and roadmap commands/API/UI are added in later phases.

## Cross-References

- [`../PRD.md`](../PRD.md)
- [`concepts/ARCHITECTURE.md`](concepts/ARCHITECTURE.md)
- [`internal/PROBLEMS.md`](internal/PROBLEMS.md)
