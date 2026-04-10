# Invariants

1. Task IDs are unique in queue storage.
2. Status transitions are represented by directory moves, not ad-hoc status drift.
3. Auto steer execution state is keyed by task identity.

[CODE: api/pkg/tasks/storage.go]
[CODE: api/pkg/tasks/lifecycle.go]
[CODE: api/pkg/autosteer/execution_state_manager.go]
