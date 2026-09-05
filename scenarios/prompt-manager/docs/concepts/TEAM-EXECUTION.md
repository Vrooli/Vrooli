# Team Execution Model

Team-level execution contexts enforce each team's runtime and execution policy through a bounded FIFO queue.

## Overview

Previously, heartbeat triggers fired all members simultaneously with no coordination. The team execution model now applies the configured execution policy for each team:

- `serialized`: exactly one member runs at a time
- `bounded-parallel`: up to `maxConcurrentRuns` members run at a time

Additional triggers are queued and deduplicated per member.

For `single-process` + `leader-led` teams, `POST /teams/{teamId}/trigger` targets only the configured lead member. That trigger now validates that the lead is an active team member and has a heartbeat config before work is accepted into the queue.

## Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                     Team Execution Pipeline                          │
│                                                                      │
│  ┌─────────────┐    ┌─────────────────────┐    ┌──────────────────┐ │
│  │  Scheduler  │───▶│ TeamExecutionStore   │───▶│    Executor      │ │
│  │  (cron)     │    │  (per-team contexts) │    │  (agent-manager) │ │
│  └─────────────┘    └─────────────────────┘    └──────────────────┘ │
│         │                     │                         │            │
│         │                     ▼                         │            │
│         │           ┌─────────────────────┐             │            │
│         │           │ TeamExecutionContext │             │            │
│         │           │  ┌───────────────┐  │             │            │
│         │           │  │ Bounded FIFO  │  │             │            │
│         └──────────▶│  │    Queue      │──┼─────────────┘            │
│                     │  └───────────────┘  │                          │
│    Manual Trigger──▶│  running: *agentIDs │                          │
│    (HTTP handler)   │  state: idle|active │                          │
│                     └─────────────────────┘                          │
│                              │                                       │
│                              ▼                                       │
│                     ┌─────────────────────┐                          │
│                     │  RunRegistry        │                          │
│                     │  (active run track) │                          │
│                     └─────────────────────┘                          │
└──────────────────────────────────────────────────────────────────────┘
```

## Lifecycle States

```
         heartbeat trigger
              │
              ▼
    ┌──────────────────┐
    │      IDLE        │◄───────────────────────┐
    │  running: []     │                        │
    │  queue: []       │                        │
    └────────┬─────────┘                        │
             │ Enqueue (first member)           │ queue empty after
             │                                  │ member completes
             ▼                                  │
    ┌──────────────────┐                        │
    │     ACTIVE       │────────────────────────┘
    │  running: [A...] │
    │  queue: [...]    │◄──┐
    └────────┬─────────┘   │
             │             │ more members enqueue
             │             │ while active
             └─────────────┘
```

### State Transitions

| Current State | Event | New State | Action |
|--------------|-------|-----------|--------|
| idle | Enqueue(agentA) | active | Start agentA immediately |
| active | Enqueue(agentB) and capacity available | active | Start agentB immediately |
| active | Enqueue(agentC) and capacity full | active, queue=[agentC] | Append agentC to queue |
| active | Enqueue(agentA) while queued/running | — | Return 409 MemberAlreadyQueuedError |
| active | agent completes, queued work remains and capacity available | active | Pop queued member(s), start next work |
| active | last running agent completes and queue=[] | idle | Clear running set |

## Bounded FIFO Queue

The queue enforces these constraints:

- **Max 1 entry per member**: A member cannot appear in the queue more than once. Attempting to enqueue an already-queued or currently-running member returns a `409 Conflict` (`MemberAlreadyQueuedError`).
- **Size bounded by team size**: Since each member can appear at most once across the running set and queue, the structure is naturally bounded by the number of team members.
- **FIFO ordering**: Members are dequeued in the order they were enqueued.

## 409 Conflict Behavior

When a trigger (manual or cron) attempts to enqueue a member that is already running or queued:

```json
HTTP/1.1 409 Conflict

"member team-1/agent-1 is already queued or running"
```

The scheduler handles this gracefully by logging and skipping (cron schedules continue normally).

## Persistence and Recovery

Queue state is persisted to disk for crash recovery:

```
store/team-queue-{teamID}.json
```

**Persisted data:**
```json
{
  "teamId": "my-team",
  "queuePolicy": "bounded-parallel",
  "maxConcurrentRuns": 2,
  "running": [
    { "agentId": "agent-1", "profileKey": "prompt-manager/heartbeat" }
  ],
  "queue": [
    { "agentId": "agent-2", "profileKey": "prompt-manager/heartbeat" },
    { "agentId": "agent-3", "profileKey": "custom-profile" }
  ]
}
```

On startup, `TeamExecutionStore.Recover()` reads persisted queue files and resumes execution. If previously-running agents are still active (checked via agent-manager), they remain in the running set. When capacity becomes available, queued members are started according to FIFO order and the configured queue policy.

## API Endpoints

### Get Execution Status

```
GET /api/v1/teams/{teamId}/execution-status
```

Returns the current execution state for a team:

```json
{
  "teamId": "my-team",
  "state": "active",
  "runningAgentIds": ["agent-1", "agent-2"],
  "queue": ["agent-3"],
  "queuePolicy": "bounded-parallel",
  "maxConcurrentRuns": 2
}
```

### Trigger Endpoints (Updated)

`POST /teams/{teamId}/heartbeats/{agentId}/trigger` and `POST /teams/{teamId}/trigger` now route through the team execution queue. New possible responses:

- **409 Conflict**: Member is already queued or running
- **400 Bad Request**: A leader-led single-process team does not currently have a valid active lead member or lead heartbeat config

## Implementation Reference

- [CODE: api/heartbeat/team_execution.go] - Core TeamExecutionContext and queue logic
- [CODE: api/heartbeat/team_execution_store.go] - Per-team context management
- [CODE: api/heartbeat/scheduler.go] - Scheduler integration with team execution
- [CODE: api/heartbeat/handlers.go] - HTTP handler integration

## Related Documentation

- [Heartbeats & Cron Execution](HEARTBEATS.md) - Base heartbeat system
- [Heartbeat API Reference](../reference/heartbeat-api.md) - API endpoint details
- [Testing Seams](../internal/SEAMS.md) - TeamExecutionManager interface seam
