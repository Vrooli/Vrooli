# Heartbeats & Cron Execution

Heartbeats enable team members (agents) to execute autonomous tasks on a schedule. This system allows agents to periodically perform work without human initiation.

## Overview

Each team member can have at most one heartbeat configuration. Heartbeats are **disabled by default** to prevent accidental expensive LLM usage - they must be explicitly enabled.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Heartbeat System                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────────┐ │
│  │  Scheduler  │───▶│  Executor   │───▶│  Agent Manager Client   │ │
│  │  (cron)     │    │  (prompt)   │    │  (gRPC/HTTP)            │ │
│  └─────────────┘    └─────────────┘    └─────────────────────────┘ │
│         │                  │                       │                │
│         ▼                  ▼                       ▼                │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────────┐ │
│  │  Config     │    │  Prompt     │    │  agent-manager scenario │ │
│  │  (JSON)     │    │  Builder    │    │  (runs agents)          │ │
│  └─────────────┘    └─────────────┘    └─────────────────────────┘ │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Storage Structure

```
store/teams/{team-id}/
├── team.json
├── roles.json
├── org.json
├── shared/
└── members/
    └── {agent-id}/
        ├── heartbeat.json       # Schedule, enabled state, execution params
        ├── HEARTBEAT.md         # Cron task instructions (what to do each heartbeat)
        ├── RESPONSIBILITIES.md  # Role-specific instructions for assigned roles
        └── logs/
            └── {timestamp}.log  # Execution logs
```

### File Purposes

| File | Purpose |
|------|---------|
| `heartbeat.json` | Configuration: schedule, enabled state, profile key |
| `HEARTBEAT.md` | Task instructions for what to do on each heartbeat |
| `RESPONSIBILITIES.md` | General role responsibilities in this team |
| `logs/*.log` | Execution history and output |

## HeartbeatConfig Schema

```json
{
  "kind": "heartbeat-config",
  "schemaVersion": 1,
  "teamId": "example",
  "agentId": "agent-1",
  "enabled": false,
  "schedule": "0 */6 * * *",
  "profileKey": "prompt-manager-heartbeat",
  "lastExecution": {
    "startedAt": "2026-02-01T10:00:00Z",
    "endedAt": "2026-02-01T10:05:32Z",
    "status": "completed",
    "runId": "run-abc123",
    "logPath": "logs/2026-02-01T10-00-00Z.log"
  },
  "createdAt": "...",
  "updatedAt": "..."
}
```

## Prompt Building

When a heartbeat executes, the prompt is built from multiple sources in order:

```
┌─────────────────────────────────────────────────┐
│ 1. Agent .md files (from store/agents/{agent})  │
│    Personality + operating notes                │
├─────────────────────────────────────────────────┤
│ 2. RESPONSIBILITIES.md (from team members/)     │
│    Role-specific instructions for this team     │
├─────────────────────────────────────────────────┤
│ 3. Team Relationships                           │
│    Reporting lines + coordination commands       │
├─────────────────────────────────────────────────┤
│ 4. Coordination Skill Reference                 │
│    Spawn-mode-specific guidance (multi/single)  │
├─────────────────────────────────────────────────┤
│ 5. Team Inbox                                   │
│    Pending messages from other team members      │
├─────────────────────────────────────────────────┤
│ 6. HEARTBEAT.md (from team members/)            │
│    The specific task to execute now             │
└─────────────────────────────────────────────────┘
```

This layered approach means:
- **Agent .md files**: "Who I am + how I operate" (global, persists across teams)
- **RESPONSIBILITIES.md**: "What I do in this team" (team-specific)
- **Team Relationships**: "Who I report to + who I direct" (coordination rules + CLI commands for messaging)
- **Coordination Skill**: Mode-specific guidance (see [Coordination Skills](SWARM-MODEL.md#coordination-skills))
- **Team Inbox**: "Pending messages to act on or reply to"
- **HEARTBEAT.md**: "What I need to do right now" (cron task)

### Action Discovery Guidance

Actions are typed executable wrappers over Vrooli-controlled CLI commands. Heartbeat prompts can include a compact runtime rule:

```text
Before manual operational work, use `prompt-manager discover "<what you need>" --type all`; prefer an exact Action contract over prose instructions when the task is deterministic. Until Action execution governance lands, inspect matching Actions with `prompt-manager action show <id>` rather than running them through prompt-manager.
```

This keeps judgment in skills and execution in Actions without bloating every heartbeat prompt. See [Actions](ACTIONS.md) and [Memory Promotion](MEMORY-PROMOTION.md).

## Prompt Pipeline UI

The Team Members heartbeat UI exposes a **Prompt Pipeline** view that mirrors the exact prompt assembly order (Agent Files → Responsibilities → Relationships → Inbox → Heartbeat Task). The pipeline lives in the member detail panel's **Overview** tab and is shared between the graph and list layouts. The UI loads the assembled prompt through `/prompt-preview` and renders each section so operators can see precisely what will run on the next heartbeat.

- [CODE: ui/src/components/editor/MemberDetailPanel.tsx] - Shared member detail panel pipeline
- [CODE: ui/src/components/editor/TeamEditorPanel.tsx] - Members layout wiring (graph + list)
- [CODE: api/heartbeat/handlers.go] - `POST /prompt-preview` endpoint used by the pipeline view

The pipeline preview uses **saved** agent + team files. Save `RESPONSIBILITIES.md` or `HEARTBEAT.md` updates before refreshing the preview.

## Cron Schedule Format

The schedule uses standard cron expression format with optional seconds:

```
┌───────────── second (optional)
│ ┌───────────── minute (0 - 59)
│ │ ┌───────────── hour (0 - 23)
│ │ │ ┌───────────── day of month (1 - 31)
│ │ │ │ ┌───────────── month (1 - 12)
│ │ │ │ │ ┌───────────── day of week (0 - 6) (Sun-Sat)
│ │ │ │ │ │
* * * * * *
```

### Common Schedule Examples

| Schedule | Description |
|----------|-------------|
| `0 * * * *` | Every hour |
| `0 */6 * * *` | Every 6 hours |
| `0 0 * * *` | Daily at midnight |
| `0 9 * * *` | Daily at 9am |
| `0 0 * * 1` | Weekly on Monday |

## Integration with agent-manager

Heartbeats execute via the agent-manager scenario using profiles:

1. **Profile Resolution**: Uses `EnsureProfile` to create/retrieve the heartbeat profile
2. **Task Creation**: Creates a task with the built prompt
3. **Run Execution**: Starts a run with `RUN_MODE_IN_PLACE`
4. **Completion Tracking**: Polls for completion and updates config

See [CODE: api/heartbeat/client.go] for the client implementation.

## Safety Considerations

1. **Off by Default**: Heartbeats must be explicitly enabled
2. **Team Gating**: Heartbeats (scheduled or manual) only run when the team is enabled
3. **Profile Controls**: Agent-manager profiles control permissions and resources
4. **Logging**: All executions are logged for audit
5. **Manual Override**: Heartbeats can be manually triggered for testing

## Team Execution Model

Heartbeat execution is serialized at the team level. Rather than firing all member heartbeats simultaneously, the system uses a bounded FIFO queue per team:

- **One at a time**: Only one member executes per team at any given moment
- **Queued execution**: Additional triggers are queued and executed in order
- **Dedup protection**: A member cannot be queued twice; duplicate triggers return 409
- **Crash recovery**: Queue state is persisted to disk and recovered on restart

This prevents resource contention and ensures predictable execution ordering. For full details on the queue lifecycle, state transitions, and persistence model, see [Team Execution Model](TEAM-EXECUTION.md).

## Related Documentation

- [Team Execution Model](TEAM-EXECUTION.md) - Serialized execution and bounded queue
- [API Reference: Heartbeat Endpoints](../reference/heartbeat-api.md)
- [CLI Reference: Heartbeat Commands](../reference/heartbeat-cli.md)
