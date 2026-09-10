# Quick Start

## Purpose Of This Document

Get the regenerated Tech Tree Designer running through the Vrooli lifecycle.

## Prerequisites

- Vrooli repo setup has completed.
- `scenario-dependency-analyzer` is available when testing the live graph domain.
- UI dependencies are installed by the scenario lifecycle or generator hook.

## Start

```bash
cd scenarios/tech-tree-designer
make start
```

Health (the CLI resolves the managed API location):

```bash
tech-tree-designer status
```

## Test

From the same scenario directory, select a focused phase:

```bash
vrooli scenario test tech-tree-designer --phases unit
```

Use the server-owned wait command returned by the runner. See
[TESTING](internal/TESTING.md) for documentation checks and broader validation scope.

For focused local checks during implementation, from the scenario directory:

```bash
(cd api && GOWORK=off go test ./internal/planning)
(cd cli && GOWORK=off go test ./domains/planning)
```

## Current Limitations

The regenerated scenario exposes graph, planning, and ontology surfaces through the Connect API, CLI, and UI. Use `tech-tree-designer graph --help`, `tech-tree-designer plan --help`, and `tech-tree-designer ontology --help` for command details.

SDA is the current observed-interface source. General repository-wide proposals, immutable review, owner-directed application/recovery and scale qualification remain targets; the existing proto materializer does not provide them. AI strategy and qualified experiments have later priority. Read [START-HERE](START-HERE.md) and [TESTING](internal/TESTING.md) before expanding scope. For documentation-only changes, use the listed owner validators rather than treating a full runtime suite as necessary certification.

## Cross-References

- [`../PRD.md`](../PRD.md)
- [`concepts/ARCHITECTURE.md`](concepts/ARCHITECTURE.md)
- [`internal/PROBLEMS.md`](internal/PROBLEMS.md)
