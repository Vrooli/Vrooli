# Error Semantics

## Categories

1. Validation errors for malformed task/config payloads.
2. Integration errors for discovery or dependency failures.
3. Execution errors for agent runtime failures.

## Recovery

1. Validation: return actionable error to caller.
2. Integration: preserve task state and retry when healthy.
3. Execution: mark failure and capture logs/output for operator review.

[CODE: api/pkg/handlers/types.go]
[CODE: api/pkg/queue/execution_manager.go]
[CODE: api/pkg/systemlog/log.go]
