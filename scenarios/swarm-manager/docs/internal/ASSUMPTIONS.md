# Assumptions

## Runtime Assumptions

1. Swarm Manager runs in a single-operator environment by default.
2. Backlog folders remain writable and git-trackable.
3. Agent execution is available through `agent-manager` when autonomous runs are requested.
4. Scenario lifecycle operations are available through Vrooli CLI/runtime services.

## Product Assumptions

1. Prompt-manager teams are responsible for producing recommendations/findings.
2. Swarm-manager remains the control plane for execution governance.
3. Backlog artifacts are the authoritative planning objects for scenario change.

## Failure-Mode Assumptions

1. Missing optional integrations should degrade gracefully.
2. Execution failures should preserve run history and failure reason.
3. Archived scenario context should remain recoverable from backlog files.

## Validation Pointers

- [CODE: api/internal/backlog/handler.go]
- [CODE: api/internal/execution/handler.go]
- [CODE: api/internal/execution/service.go]
- [CODE: api/internal/scenarios/handler.go]
