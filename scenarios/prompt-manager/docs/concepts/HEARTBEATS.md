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
│ 1. SOUL.md (from store/agents/{agent})         │
│    Agent's personality                          │
├─────────────────────────────────────────────────┤
│ 2. RESPONSIBILITIES.md (from team members/)    │
│    Role-specific instructions for this team    │
├─────────────────────────────────────────────────┤
│ 3. Effective Skills (computed)                 │
│    - Agent skillPins                           │
│    - Agent-skill relations                     │
│    - Team role grants                          │
├─────────────────────────────────────────────────┤
│ 4. HEARTBEAT.md (from team members/)           │
│    The specific task to execute now            │
└─────────────────────────────────────────────────┘
```

This layered approach means:
- **SOUL.md**: "Who I am" (global, persists across teams)
- **RESPONSIBILITIES.md**: "What I do in this team" (team-specific)
- **Skills**: "Tools I have available" (computed from all sources)
- **HEARTBEAT.md**: "What I need to do right now" (cron task)

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
2. **Profile Controls**: Agent-manager profiles control permissions and resources
3. **Logging**: All executions are logged for audit
4. **Manual Override**: Heartbeats can be manually triggered for testing

## Related Documentation

- [API Reference: Heartbeat Endpoints](../reference/heartbeat-api.md)
- [CLI Reference: Heartbeat Commands](../reference/heartbeat-cli.md)
- [Effective Skills](./EFFECTIVE-SKILLS.md)
