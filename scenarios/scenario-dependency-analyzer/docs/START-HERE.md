# Start Here

## Initialization Protocol

Read `../PRD.md`, this file, `concepts/ARCHITECTURE.md`, and `internal/SEAMS.md` before changing behavior. Run `vrooli help` and use lifecycle commands for scenario start/test flows.

## Architecture Rules

SDA owns fleet-level interpretation, graph assembly, drift semantics, API/CLI/UI presentation, and SQLite-backed reports. Proto extraction belongs to `proto-health`; source import extraction belongs to `code-facts`.

## Replacing The Example Domain

This is not a template demo. Preserve the actual interface graph and import drift domain unless a new plan explicitly replaces it.

## Cross-References

- `concepts/ARCHITECTURE.md`
- `concepts/DOMAINS.md`
- `internal/SEAMS.md`
- `internal/TESTING.md`
