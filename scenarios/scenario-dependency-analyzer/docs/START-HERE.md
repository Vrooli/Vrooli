# Start Here

For dependency changes, use the [install gateway](../../../docs/package-governance.md#installing-through-the-gateway).
For other commands, use the [CLI reference](reference/cli-commands.md).
For changes to analyzer behavior, start with the affected domain.
Consult the [PRD](../PRD.md) for requirement changes and the
[architecture](concepts/ARCHITECTURE.md) for ownership changes.

## Architecture Rules

SDA owns fleet-level interpretation, graph assembly, drift semantics, API/CLI/UI presentation, and SQLite-backed reports. Proto extraction belongs to `proto-health`; source import extraction belongs to `code-facts`.

## Cross-References

- `concepts/ARCHITECTURE.md`
- `concepts/DOMAINS.md`
- `internal/SEAMS.md`
- `internal/TESTING.md`
