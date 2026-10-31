# Heartbeat API Reference

API endpoints for managing heartbeat configurations and member documents.

## Base URL

All endpoints are prefixed with `/api/v1`.

**Membership requirement:** Endpoints scoped to a specific team member require that
the agent is currently a member of the team. Requests for non-members return `404`.

---

## Heartbeat Control

Heartbeat control manages global and per-team auto-pause state. Pause state is independent from member heartbeat config `enabled` flags.

### Get Global Control Status

```
GET /heartbeats/control
```

Returns global policy/state plus per-team summaries.

```json
{
  "scope": "global",
  "status": "active",
  "effectivePolicy": {
    "enabled": true,
    "pauseAfterDaysWithoutHumanEngagement": 14,
    "warningAfterDaysWithoutHumanEngagement": 10,
    "resumeMode": "manual"
  },
  "lastHumanEngagementAt": "2026-06-19T12:00:00Z",
  "lastHumanEngagementReason": "work-dispositioned",
  "warningAt": "2026-06-29T12:00:00Z",
  "autoPauseAt": "2026-07-03T12:00:00Z",
  "teams": []
}
```

### Update Global Policy

```
PUT /heartbeats/control/policy
```

Requires `X-Vrooli-Attribution` with `kind=operator-direct`.

```json
{
  "enabled": true,
  "pauseAfterDaysWithoutHumanEngagement": 14,
  "warningAfterDaysWithoutHumanEngagement": 10,
  "resumeMode": "manual"
}
```

### Pause or Resume Global Scheduling

```
POST /heartbeats/control/pause
POST /heartbeats/control/resume
```

Pause accepts an optional reason:

```json
{ "reason": "operator requested quiet period" }
```

### Team Control

```
GET /teams/{teamId}/heartbeats/control
PUT /teams/{teamId}/heartbeats/control/policy
POST /teams/{teamId}/heartbeats/control/pause
POST /teams/{teamId}/heartbeats/control/resume
```

Team pause/resume requires canonical authority for this exact team: a capable verified
human, its current verified creator, or another verified agent with exact
`team.manage` coverage. The same team-owner decision governs membership and
heartbeat configuration. Attribution headers alone grant no authority. Global control and policy writes keep their human-only paths.
Agent calls retain their caller/owner in audit logs and do not count as human engagement.
An explicit team resume after a global manual pause releases only that team; a later
global pause holds it again. Automatic inactivity holds remain effective for agent calls.
The global pause and other teams' states are preserved.

Team policy accepts an override:

```json
{
  "mode": "custom",
  "pauseAfterDaysWithoutHumanEngagement": 21,
  "warningAfterDaysWithoutHumanEngagement": 14,
  "resumeMode": "manual"
}
```

`mode=inherit` follows the global policy. `mode=disabled` disables team-specific auto-pause, but a global pause still blocks the team.

### Paused Error Contract

Manual trigger endpoints return `423 Locked` when blocked by heartbeat control:

```json
{
  "error": "heartbeat_paused",
  "scope": "global",
  "status": "paused-auto-idle",
  "message": "Heartbeats are auto-paused because no human engagement was recorded since 2026-06-01T00:00:00Z.",
  "resumeHint": "Run prompt-manager heartbeat-control resume or use the UI Resume button."
}
```

---

## Heartbeat Configuration

### Ordinary wake admission

Create and update requests may include `wakeAdmission` for ordinary (non-finite
leader) heartbeats:

```json
{
  "wakeAdmission": {
    "mode": "on-change",
    "changeSources": ["team", "member", "inbox", "corpus"]
  }
}
```

`mode=on-change` requires at least one source. Supported sources are `team`,
`member`, `inbox`, and `corpus`; their bounded identities are compared between
scheduled ticks. The first tick admits, unchanged identities are skipped, and a
changed identity admits. Omitted `wakeAdmission` and `mode=always` preserve
legacy cadence. Runtime admission state is not configuration state and is not
returned as a configuration revision. Manual triggers bypass scheduled wake
admission. Evidence and baseline read failures admit work rather than silently
suppressing it.

### Team-native effort identity

Prompt Manager has no separate finite-leader heartbeat API. A team declares its
effort reference and leader through the existing team contract, while its member
heartbeat selects the profile. Before dispatching that configured finite
delivery lead, PM requires a valid contract, active membership, matching
profile, and exact current Agent Manager enrollment. The matching non-withdrawn
target revision is included in the frozen CreateRun request's WorkReferences;
this descriptive reference does not grant authority. Missing, unavailable, or
mismatched identity fails admission before task/run creation. When a prior run
ID exists, PM retains that exact owner obligation as uncertain rather than
creating competing work. Verified active/parked owners remain on the existing
watch path, while a verified terminal owner allows a fresh dispatch under the
current contract. Finite non-lead members and ordinary standing/non-delivery
heartbeats remain supported without effort association. For receiver
qualification and source handoff, follow §6.1 of the canonical
[`large-effort-orchestration` skill](../../store/skills/packs/core/large-effort-orchestration/SKILL.md#61-commission-an-accurate-accessible-handoff)
for the specific effort's goal home and source manifest.

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
    "profileKey": "prompt-manager/heartbeat",
    "lastExecution": {
      "startedAt": "2026-02-01T10:00:00Z",
      "endedAt": "2026-02-01T10:05:32Z",
      "status": "completed",
      "runId": "abc123",
      "logPath": "2026-02-01T10-00-00Z.log"
    },
    "lastSuccessfulExecution": {
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

`lastExecution` is the current or most recent attempt. `lastSuccessfulExecution` remains unchanged while a newer attempt is running or failed, so rolling fleet-health calculations do not lose valid success evidence.

### List Heartbeat Attempts

Read newest-first logical attempt projections from the shared event stream.

```http
GET /heartbeat-attempts?team_id=<team-id>&agent_id=<member-id>&limit=20&offset=0
```

The response includes `attempts`, `total`, `totalExact`, `hasMore`, and
`scannedEvents`. The endpoint scans at most the newest 1,000 heartbeat-attempt
event records across members, then applies team/member/status/profile filters.
When `totalExact` is false, `total` is a lower bound observed in that
bounded recent window and `hasMore` indicates either an incomplete scan or
additional known output. `offset` pages the currently observed window; it does
not provide a stable cursor through all retained history. The window combines
durable event projections with eligible pre-run configuration records before
applying `limit` and `offset`. Counts are not global lifetime totals.

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
  "profileKey": "prompt-manager/heartbeat",
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
- `423 Locked` - Heartbeat control is paused (`heartbeat_paused`)

---

### Trigger Team

Trigger heartbeats for an entire team. Behavior depends on the team's resolved runtime and coordination policy.

```
POST /teams/{teamId}/trigger
```

- **`single-process` + `leader-led`**: Triggers only the configured lead agent.
- **All other team policies**: Triggers all members that have heartbeat configs.

**Response:** `202 Accepted`
```json
{
  "teamId": "my-team",
  "runtimeMode": "multi-process",
  "coordinationPattern": "independent",
  "queuePolicy": "bounded-parallel",
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
- `400 Bad Request` - Invalid leader-led single-process configuration, inactive/missing lead member, or missing lead heartbeat config
- `404 Not Found` - Team not found
- `409 Conflict` - Team is disabled, or the requested member is already queued/running
- `423 Locked` - Heartbeat control is paused (`heartbeat_paused`)
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
  "runningAgentIds": ["agent-1", "agent-2"],
  "queue": ["agent-3"],
  "queuePolicy": "bounded-parallel",
  "maxConcurrentRuns": 2,
  "preparationInterruptedAgentIds": ["agent-3"],
  "admissionDeferredAgentIds": ["agent-4"],
  "owners": [
    {"agentId": "agent-1", "phase": "retained", "state": "dispatch_uncertain", "dispatchStage": "task_intent", "taskId": "task-123"},
    {"agentId": "agent-3", "phase": "queued", "dispatchStage": "preparing"}
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `state` | string | `"idle"` or `"active"` |
| `runningAgentIds` | string[] | Agent IDs currently executing |
| `queue` | string[] | Agent IDs waiting to execute (FIFO order) |
| `queuePolicy` | string | `serialized` or `bounded-parallel` |
| `maxConcurrentRuns` | integer | Concurrency cap for the team execution context |
| `preparationInterruptedAgentIds` | string[] | Accepted pre-dispatch work retained after a restart; requires a fresh, normally gated admission and is not currently executing |
| `admissionDeferredAgentIds` | string[] | Exact heartbeat task/run intent retained after Agent Manager definitively refuses only because the current grant capacity is full; it is not a running model and is replayed on an otherwise eligible scheduled callback |
| `owners` | object[] | Current in-memory owner entries for retained slots (`phase: retained`) and queued members (`phase: queued`); includes only member ID, phase/state, dispatch stage, task/run IDs, admission origin, owner start time, and obligation ID. It omits task/request snapshots and caller bindings. |

`owners` is the diagnostic detail behind the category arrays. A retained slot does not prove that an external process is active. A missing run ID is meaningful for pre-dispatch or uncertain entries; it does not prove that no external run exists.

An otherwise eligible scheduled callback retries only an intent whose persisted
admission origin is scheduled. An explicit manual trigger can retry a retained
manual intent before completed-trigger deduplication. Both paths preserve the
original task/run request bytes and idempotency key. Manual retry still passes
the existing request authorization, team/member and pause gates. Caller-backed
intents require the same freshly verified caller binding. Service-backed intents
mint fresh Prompt Manager service proof through the existing owner client and
retain service authority even when a newly authorized agent triggers the retry. Service proof qualification preserves initiating attribution, cancellation and deadlines while removing forwardable triggering-agent credentials from the outbound service context. The task, token exchange and run requests carry service proof without an unrelated agent baseline or operation token.
An unknown origin or authority mode is never inferred from a new request. A retry
does not replace a retained owner, bypass a service limit or create a second run.

---

### Clear One Retained Queue Owner

`prompt-manager team queue-clear <team-id> <agent-id>` uses
`DELETE /teams/{teamId}/queue/running/{agentId}` under exact-team management.
The default path requires Agent Manager to confirm the recorded run is terminal.
For a task-only dispatch intent, Agent Manager must instead confirm the exact task
is cancelled, has no current run and has a complete empty task-filtered attempt
inventory. The retained slot is checked again after these owner reads and removed
durably before any queued work is admitted. A failed persistence write restores it.
Missing, mismatched, active, attempted, partial or unreachable owners remain retained.

When an obsolete task prompt should never execute, approve cancellation of that
exact task through Agent Manager first, cancel it natively, then clear this queue
owner without `--force`. Task cancellation requires its separate exact
`agent.manage` authority; team management does not confer it. These operations can
be performed by an agent after browser consent; the operator need not run commands.
The existing explicit force recovery remains an operator escape hatch, not the
normal activation procedure.

---

### Get Member Context

Get standing context for a team member without the active `HEARTBEAT.md` task instructions. Used by leader-led single-process teams or external workflows that need taskless teammate bootstrapping. Use `/prompt-preview` or `/prompt-preview-structured` to audit the full runtime heartbeat prompt.

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

## Swarm Manager work and disposition

Prompt Manager does not expose a local approval stream. A member records its
evidence in the team's Source Ledger scope and files one work item or capture
through Swarm Manager when an outcome or operator judgment is needed. The
operator dispositions that item in Swarm Manager's next-action feed. On a later
heartbeat the member reads `swarm-manager backlog list --actor-id=<verified-profile-key>`
and `swarm-manager backlog get <item-id> --json` to continue, follow up, or
archive. This return path is a member-side poll; Swarm Manager does not call
back into Prompt Manager.

---

## Implementation Reference

- [CODE: api/heartbeat/handlers.go] - HTTP handlers
- [CODE: api/heartbeat/prompt_builder.go] - Generated team-scope and work guidance
- [CODE: api/heartbeat/scheduler.go] - Cron scheduler
- [CODE: api/heartbeat/executor.go] - Execution logic
- [CODE: api/heartbeat/client.go] - Agent-manager client
