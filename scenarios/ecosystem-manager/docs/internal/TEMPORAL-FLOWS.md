# Temporal Flows

## Execution Flow

1. Task transitions to active processing.
2. Prompt assembly and optional auto steer initialization run.
3. Agent execution streams output.
4. Result persisted and queue state updated.

## Timing-sensitive Areas

1. Queue cooldown and concurrent slot controls.
2. Auto steer iteration counters and phase transitions.

[CODE: api/pkg/queue/concurrency_manager.go]
[CODE: api/pkg/tasks/cooldown.go]
[CODE: api/pkg/autosteer/phase_coordinator.go]
