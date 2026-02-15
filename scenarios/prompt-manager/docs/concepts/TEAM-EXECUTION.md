# Team Execution Model

Team-level execution context that serializes heartbeat execution per team through a bounded FIFO queue.

## Overview

Previously, heartbeat triggers fired all members simultaneously with no coordination. The team execution model introduces serialized execution: only one member runs at a time per team, and additional triggers are queued.

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
│    Manual Trigger──▶│  running: *agentID  │                          │
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
    │  running: nil    │                        │
    │  queue: []       │                        │
    └────────┬─────────┘                        │
             │ Enqueue (first member)           │ queue empty after
             │                                  │ member completes
             ▼                                  │
    ┌──────────────────┐                        │
    │     ACTIVE       │────────────────────────┘
    │  running: agentX │
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
| active(agentA) | Enqueue(agentB) | active(agentA), queue=[agentB] | Append agentB to queue |
| active(agentA) | Enqueue(agentA) | — | Return 409 MemberAlreadyQueuedError |
| active(agentA) | agentA completes, queue=[agentB] | active(agentB) | Pop agentB, start it |
| active(agentA) | agentA completes, queue=[] | idle | Set running to nil |

## Bounded FIFO Queue

The queue enforces these constraints:

- **Max 1 entry per member**: A member cannot appear in the queue more than once. Attempting to enqueue an already-queued or currently-running member returns a `409 Conflict` (`MemberAlreadyQueuedError`).
- **Size bounded by team size**: Since each member can appear at most once, the queue is naturally bounded by the number of team members.
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
  "running": "agent-1",
  "queue": ["agent-2", "agent-3"],
  "profiles": {
    "agent-2": "prompt-manager-heartbeat",
    "agent-3": "custom-profile"
  }
}
```

On startup, `TeamExecutionStore.Recover()` reads persisted queue files and resumes execution. If the previously-running agent's run is still active (checked via agent-manager), it's kept as running. If terminal, the next queued member is started.

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
  "running": "agent-1",
  "queue": ["agent-2"]
}
```

### Trigger Endpoints (Updated)

`POST /teams/{teamId}/heartbeats/{agentId}/trigger` and `POST /teams/{teamId}/trigger` now route through the team execution queue. New possible response:

- **409 Conflict**: Member is already queued or running

## Implementation Reference

- [CODE: api/heartbeat/team_execution.go] - Core TeamExecutionContext and queue logic
- [CODE: api/heartbeat/team_execution_store.go] - Per-team context management
- [CODE: api/heartbeat/scheduler.go] - Scheduler integration with team execution
- [CODE: api/heartbeat/handlers.go] - HTTP handler integration

## Related Documentation

- [Heartbeats & Cron Execution](HEARTBEATS.md) - Base heartbeat system
- [Heartbeat API Reference](../reference/heartbeat-api.md) - API endpoint details
- [Testing Seams](../internal/SEAMS.md) - TeamExecutionManager interface seam
