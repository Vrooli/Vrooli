# Assumptions

1. `agent-manager` remains available for task execution delegation.
2. Queue status continues to map one-to-one with task storage directories.
3. UI and CLI consumers both rely on stable API contracts.

[CODE: api/pkg/agentmanager/client.go]
[CODE: api/pkg/tasks/storage.go]
