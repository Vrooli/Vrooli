# System Invariants

## Workflow boundary

- Code-created programmatic agent work starts through a declared transition and
  the shared transition runner. The raw Agent Manager workflow transport is
  owned only by `internal/transitionrunner` and `internal/agentmanager`.
- Human-authored conversations use Agent Sessions and their Runs; they are not
  a substitute for programmatic workflows.
- A transition registry entry is exactly one of `session`, `workflow`, or
  `deterministic`. It contains no prompt text, branch conditions, retry
  policy, loop, or scheduler.
- Every workflow transition references a declared workflow file owned by
  `swarm-manager`.
- Every declared workflow and deterministic apply action is registered at boot;
  an incomplete dispatch table prevents startup.

## Domain authority

- Swarm Manager builds immutable inputs and applies typed terminal outcomes;
  Agent Manager never mutates Swarm domain records directly.
- Application is exactly once per correlation and rejects stale entity, plan,
  or evidence frontiers.
- Deterministic transitions have no workflow correlation; session transitions
  remain on the Agent Session surface.
- Plan references become authoritative only after Plan Manager validation.
- Test and regression decisions consume typed Test Genie and Git Control Tower
  evidence, never agent prose.

## Historical compatibility

- Retired operating-mode and agent-operation data is readable for provenance.
- Historical compatibility values are never accepted as new starts, new
  sessions, or new programmatic executions.
