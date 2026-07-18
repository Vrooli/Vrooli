# System Invariants

## Workflow boundary

- Code-created programmatic agent work starts through a declared Agent Manager
  workflow. The raw Run substrate is used only by Agent Manager workflow nodes.
- Human-authored conversations use Agent Sessions and their Runs; they are not
  a substitute for programmatic workflows.
- A transition registry entry is exactly one of `session`, `workflow`, or
  `deterministic`. It contains no prompt text, branch conditions, retry
  policy, loop, or scheduler.
- Every workflow transition references a declared workflow file owned by
  `swarm-manager`.

## Domain authority

- Swarm Manager builds immutable inputs and applies typed terminal outcomes;
  Agent Manager never mutates Swarm domain records directly.
- Application is exactly once per correlation and rejects stale entity, plan,
  or evidence frontiers.
- Plan references become authoritative only after Plan Manager validation.
- Test and regression decisions consume typed Test Genie and Git Control Tower
  evidence, never agent prose.

## Historical compatibility

- Retired operating-mode and agent-operation data is readable for provenance.
- Historical compatibility values are never accepted as new starts, new
  sessions, or new programmatic executions.
