# Troubleshooting

## API Not Reachable

1. Check service state: `make logs`.
2. Confirm port binding in startup logs.
3. Re-run `make stop && make start`.

## Queue Not Processing

1. Validate queue settings in UI Settings modal.
2. Inspect processor status endpoint and logs.
3. Confirm agent-manager discovery resolves.

## Auto Steer State Missing

1. Verify task has a profile attached.
2. Check `/api/auto-steer/execution/{taskId}` response.
3. Trigger manual seek initialization from UI.

[CODE: api/pkg/handlers/queue.go]
[CODE: api/pkg/handlers/settings.go]
[CODE: api/pkg/autosteer/handlers.go]
