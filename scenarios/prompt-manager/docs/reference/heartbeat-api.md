# Heartbeat API Reference

API endpoints for managing heartbeat configurations and member documents.

## Base URL

All endpoints are prefixed with `/api/v1`.

**Membership requirement:** Endpoints scoped to a specific team member require that
the agent is currently a member of the team. Requests for non-members return `404`.

---

## Heartbeat Configuration

### List Heartbeats

List all heartbeat configurations for a team.

```
GET /teams/{teamId}/heartbeats
```

**Response:**
```json
[
  {
    "teamId": "my-team",
    "agentId": "agent-1",
    "enabled": true,
    "schedule": "0 */6 * * *",
    "profileKey": "prompt-manager-heartbeat",
    "lastExecution": {
      "startedAt": "2026-02-01T10:00:00Z",
      "endedAt": "2026-02-01T10:05:32Z",
      "status": "completed",
      "runId": "abc123",
      "logPath": "2026-02-01T10-00-00Z.log"
    },
    "nextExecution": "2026-02-01T16:00:00Z",
    "createdAt": "2026-01-15T00:00:00Z",
    "updatedAt": "2026-02-01T10:05:32Z"
  }
]
```

---

### Get Heartbeat

Get heartbeat configuration for a specific member.

```
GET /teams/{teamId}/heartbeats/{agentId}
```

**Response:**
```json
{
  "teamId": "my-team",
  "agentId": "agent-1",
  "enabled": true,
  "schedule": "0 */6 * * *",
  "profileKey": "prompt-manager-heartbeat",
  "lastExecution": null,
  "nextExecution": "2026-02-01T16:00:00Z",
  "createdAt": "2026-01-15T00:00:00Z",
  "updatedAt": "2026-01-15T00:00:00Z"
}
```

**Errors:**
- `404 Not Found` - Team or heartbeat config not found

---

### Create Heartbeat

Create a new heartbeat configuration for a member.

```
POST /teams/{teamId}/heartbeats/{agentId}
```

**Request Body:**
```json
{
  "schedule": "0 */6 * * *",
  "profileKey": "my-custom-profile",
  "enabled": false
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `schedule` | string | Yes | Cron expression |
| `profileKey` | string | No | Agent-manager profile key |
| `enabled` | boolean | No | Defaults to `false` |

**Response:** `201 Created` with heartbeat config

**Errors:**
- `400 Bad Request` - Invalid schedule or missing required fields
- `404 Not Found` - Team not found
- `409 Conflict` - Heartbeat config already exists

---

### Update Heartbeat

Update an existing heartbeat configuration.

```
PUT /teams/{teamId}/heartbeats/{agentId}
```

**Request Body:**
```json
{
  "schedule": "0 0 * * *",
  "enabled": true
}
```

All fields are optional. Only provided fields are updated.

**Response:** Updated heartbeat config

**Errors:**
- `400 Bad Request` - Invalid schedule
- `404 Not Found` - Team or config not found

---

### Delete Heartbeat

Delete a heartbeat configuration.

```
DELETE /teams/{teamId}/heartbeats/{agentId}
```

**Response:** `204 No Content`

---

### Trigger Heartbeat

Manually trigger a heartbeat execution.

```
POST /teams/{teamId}/heartbeats/{agentId}/trigger
```

**Response:** `202 Accepted`
```json
{
  "teamId": "my-team",
  "agentId": "agent-1",
  "runId": "run-xyz789",
  "status": "running",
  "logPath": "2026-02-01T15-30-00Z.log"
}
```

**Errors:**
- `404 Not Found` - Team, member, or heartbeat config not found
- `409 Conflict` - Team is disabled, or member is already queued/running

---

### Trigger Team

Trigger heartbeats for an entire team. Behavior depends on the team's `spawnMode`.

```
POST /teams/{teamId}/trigger
```

- **`single-process`**: Triggers only the team lead's heartbeat (identified from the org chart).
- **`multi-process`** (default): Triggers all members that have heartbeat configs.

**Response:** `202 Accepted`
```json
{
  "teamId": "my-team",
  "spawnMode": "multi-process",
  "triggers": [
    {
      "teamId": "my-team",
      "agentId": "agent-1",
      "runId": "run-xyz",
      "status": "running",
      "logPath": "2026-02-01T10-00-00Z.log"
    }
  ]
}
```

**Errors:**
- `400 Bad Request` - No team lead found (single-process mode)
- `404 Not Found` - Team not found
- `409 Conflict` - Team is disabled
- `503 Service Unavailable` - Executor not configured

---

### Get Team Execution Status

Get the current execution queue status for a team.

```
GET /teams/{teamId}/execution-status
```

**Response:**
```json
{
  "teamId": "my-team",
  "state": "active",
  "running": "agent-1",
  "queue": ["agent-2"]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `state` | string | `"idle"` or `"active"` |
| `running` | string? | Agent ID currently executing, null if idle |
| `queue` | string[] | Agent IDs waiting to execute (FIFO order) |

---

### Get Member Context

Get the full context prompt for a team member (excludes HEARTBEAT.md task instructions). Used by single-process spawn mode for teammate bootstrapping.

```
GET /teams/{teamId}/members/{agentId}/context
```

**Response:**
```json
{
  "teamId": "my-team",
  "agentId": "agent-1",
  "prompt": "# Agent Files (Markdown)\n\n## SOUL.md\n\n..."
}
```

**Errors:**
- `404 Not Found` - Team or member not found
- `503 Service Unavailable` - Executor not configured

---

## Execution Logs

## Run Operations

### List Runs

List runs proxied from agent-manager, with optional filtering.

```
GET /runs
```

Supported query parameters:

- `status`
- `tag_prefix`
- `task_id`
- `limit`
- `offset`
- `investigates_run_id` - list investigation runs linked to a specific source run
- `applies_investigation_run_id` - list apply runs linked to a specific investigation run

### Create Investigation Run

Create a new investigation run for one or more source runs.

```
POST /runs/investigate
```

Request body:

```json
{
  "run_ids": ["<source-run-id>"],
  "depth": "standard",
  "custom_context": "Optional extra context"
}
```

### Create Investigation Apply Run

Create a run that applies recommendations from an investigation run.

```
POST /runs/investigation-apply
```

Request body:

```json
{
  "investigation_run_id": "<investigation-run-id>",
  "custom_context": "Optional extra context"
}
```

### List Logs

List execution logs for a member.

```
GET /teams/{teamId}/heartbeats/{agentId}/logs
```

**Response:**
```json
{
  "teamId": "my-team",
  "agentId": "agent-1",
  "logs": [
    {
      "filename": "2026-02-01T10-00-00Z.log",
      "timestamp": "2026-02-01T10-00-00Z"
    },
    {
      "filename": "2026-02-01T04-00-00Z.log",
      "timestamp": "2026-02-01T04-00-00Z"
    }
  ]
}
```

---

### Get Log Content

Get the content of a specific log file.

```
GET /teams/{teamId}/heartbeats/{agentId}/logs/{logId}
```

**Response:**
```json
{
  "teamId": "my-team",
  "agentId": "agent-1",
  "filename": "2026-02-01T10-00-00Z.log",
  "content": "Heartbeat execution for my-team/agent-1\nStarted: 2026-02-01T10:00:00Z\n..."
}
```

---

## Member Documents

### Get Responsibilities

Get RESPONSIBILITIES.md content for a team member.

```
GET /teams/{teamId}/members/{agentId}/responsibilities
```

**Response:**
```json
{
  "teamId": "my-team",
  "agentId": "agent-1",
  "content": "# Responsibilities\n\nThis agent is responsible for..."
}
```

---

### Set Responsibilities

Set RESPONSIBILITIES.md content for a team member.

```
PUT /teams/{teamId}/members/{agentId}/responsibilities
```

**Request Body:**
```json
{
  "content": "# Responsibilities\n\nUpdated content..."
}
```

**Response:** Updated document response

---

### Get Heartbeat Instructions

Get HEARTBEAT.md content for a team member.

```
GET /teams/{teamId}/members/{agentId}/heartbeat-instructions
```

**Response:**
```json
{
  "teamId": "my-team",
  "agentId": "agent-1",
  "content": "# Heartbeat Task\n\nOn each heartbeat, review..."
}
```

---

### Set Heartbeat Instructions

Set HEARTBEAT.md content for a team member.

```
PUT /teams/{teamId}/members/{agentId}/heartbeat-instructions
```

**Request Body:**
```json
{
  "content": "# Heartbeat Task\n\nUpdated instructions..."
}
```

**Response:** Updated document response

---

## Agent Soul API

### Get Soul

Get SOUL.md content for an agent.

```
GET /agents/{agentId}/soul
```

**Response:**
```json
{
  "agentId": "agent-1",
  "content": "# Agent Personality\n\nI am a helpful assistant..."
}
```

---

### Set Soul

Set SOUL.md content for an agent.

```
PUT /agents/{agentId}/soul
```

**Request Body:**
```json
{
  "content": "# Agent Personality\n\nUpdated personality..."
}
```

**Response:** Updated soul response

---

## Handoff

### Get Latest Handoff
`GET /teams/{teamId}/members/{agentId}/handoff`

Returns the most recent handoff for a team member.

**Response**: `200 OK`
```json
{
  "teamId": "string",
  "agentId": "string",
  "content": "string (markdown)"
}
```

**Errors**: `404 Not Found` if no handoff exists.

### Get Handoff History
`GET /teams/{teamId}/handoff-history`

Returns handoff history entries for the team.

**Query Parameters**:
- `agent` (optional) — filter by agent ID
- `last` (optional, default: 20) — number of entries to return

**Response**: `200 OK`
```json
{
  "teamId": "string",
  "entries": [
    {
      "agentId": "string",
      "runId": "string",
      "timestamp": "string (RFC3339)",
      "content": "string"
    }
  ]
}
```

## Task Board

### Get Task Board
`GET /teams/{teamId}/tasks`

Returns all tasks on the team's task board.

**Response**: `200 OK`
```json
{
  "teamId": "string",
  "tasks": [
    {
      "id": "string",
      "title": "string",
      "status": "todo|in-progress|blocked|done",
      "assignee": "string",
      "priority": "P1-P5",
      "createdBy": "string",
      "createdAt": "string (RFC3339)",
      "updatedAt": "string (RFC3339)",
      "notes": [{"at": "string", "by": "string", "text": "string"}]
    }
  ]
}
```

### Add Task
`POST /teams/{teamId}/tasks`

Creates a new task on the board.

**Request Body**:
```json
{
  "title": "string (required)",
  "assignee": "string (optional)",
  "priority": "string (optional, default: P3)",
  "from": "string (creator agent ID)"
}
```

**Response**: `201 Created` — returns the created task.

### Update Task
`PATCH /teams/{teamId}/tasks/{taskId}`

Updates a task (partial update).

**Request Body**:
```json
{
  "status": "string (optional)",
  "assignee": "string (optional)",
  "priority": "string (optional)",
  "note": "string (optional — appends a note)"
}
```

**Response**: `200 OK` — returns the updated task.

### Delete Task
`DELETE /teams/{teamId}/tasks/{taskId}`

Deletes a task from the board.

**Response**: `204 No Content`

## Decision Log

### Add Decision
`POST /teams/{teamId}/decisions`

Records a decision in the team's decision log.

**Request Body**:
```json
{
  "by": "string (agent ID, required)",
  "decision": "string (required)",
  "rationale": "string (required)",
  "context": "string (optional — tag for grouping)",
  "supersedes": "string (optional — ID of decision this replaces)"
}
```

**Response**: `201 Created` — returns the created decision entry.

### Get Decisions
`GET /teams/{teamId}/decisions`

Returns decision log entries.

**Query Parameters**:
- `context` (optional) — filter by context tag
- `last` (optional, default: 20) — number of entries to return

**Response**: `200 OK`
```json
{
  "teamId": "string",
  "entries": [
    {
      "id": "string",
      "at": "string (RFC3339)",
      "by": "string",
      "decision": "string",
      "rationale": "string",
      "context": "string",
      "supersedes": "string"
    }
  ]
}
```

---

## Implementation Reference

- [CODE: api/heartbeat/handlers.go] - HTTP handlers
- [CODE: api/heartbeat/scheduler.go] - Cron scheduler
- [CODE: api/heartbeat/executor.go] - Execution logic
- [CODE: api/heartbeat/client.go] - Agent-manager client
