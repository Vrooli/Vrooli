# API Endpoints

## Health
- `GET /health`

## Tasks
- `GET /api/tasks`
- `POST /api/tasks`
- `GET /api/tasks/{id}`
- `PUT /api/tasks/{id}`

## Queue
- `GET /api/queue/status`
- `POST /api/queue/start`
- `POST /api/queue/stop`

## Settings
- `GET /api/settings`
- `PUT /api/settings`
- `POST /api/settings/reset`

## Auto Steer
- `GET /api/auto-steer/execution/{taskId}`
- `POST /api/auto-steer/execution/seek`

[CODE: api/pkg/handlers/tasks.go]
[CODE: api/pkg/handlers/queue.go]
[CODE: api/pkg/handlers/settings.go]
[CODE: api/pkg/autosteer/handlers.go]
