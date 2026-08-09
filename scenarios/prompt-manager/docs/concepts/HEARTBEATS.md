# Heartbeats & Cron Execution

Heartbeats enable team members (agents) to execute autonomous tasks on a schedule. This system allows agents to periodically perform work without human initiation.

## Overview

Each team member can have at most one heartbeat configuration. Heartbeats are **disabled by default** to prevent accidental expensive LLM usage - they must be explicitly enabled.

## Engagement Auto-Pause

Prompt-manager also has a global heartbeat control layer that can pause future scheduled/manual heartbeat starts when operator engagement goes idle. This is separate from `heartbeat.json.enabled`: auto-pause never disables or deletes member heartbeat configs.

Default policy:
- Auto-pause enabled.
- Warning after 10 days without operator engagement.
- Pause after 14 days without operator engagement.
- Resume mode is manual.
- A new/missing control store initializes `lastHumanEngagementAt` to the current time so upgrades do not immediately pause all teams.

Operator engagement signals:
- `operator-direct` Swarm Manager work dispositions such as accepted, rejected, deferred, or pending.
- `operator-direct` manual heartbeat or team trigger.
- `operator-direct` heartbeat control/policy changes.
- `operator-direct` heartbeat config changes.

Non-signals:
- Agent-member work-item transitions.
- Writer-skill or agent knowledge writes.
- Scheduled heartbeat starts/completions.
- Read-only UI polling.

Visible states:
- `active`: scheduling and manual triggers are allowed.
- `warning-idle-soon`: scheduling is allowed, but the warning threshold has elapsed.
- `paused-auto-idle`: new scheduled/manual starts are blocked because the idle threshold elapsed.
- `paused-manual`: new scheduled/manual starts are blocked by an explicit operator pause.

Manual resume clears pause state and reschedules enabled heartbeats for enabled teams. Already-running agent-manager runs are not cancelled by auto-pause.

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
  "profileKey": "prompt-manager/heartbeat",
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

When a heartbeat executes, the prompt is built from a volatility-ordered
context followed by the task:

```
<context>
  <standing-rules>universal guidance</standing-rules>
  <operating-policy-team>team-stable policy</operating-policy-team>
  <operating-policy-member>member contract</operating-policy-member>
  <topic-contract>member topic declarations</topic-contract>
  <responsibilities>standing member duty</responsibilities>
  <agent-files>agent identity</agent-files>
  <team-inbox>live inbox, when enabled</team-inbox>
  <team-context-wake>live Source Ledger wake</team-context-wake>
  <contract-findings>live validation findings</contract-findings>
</context>
<heartbeat-task>the job to do now, in prose</heartbeat-task>
```

This layered approach means:
- **Standing Rules**: universal routing, authority, and safety guidance.
- **Operating Policy (Team)**: team charter, runtime, coordination, and governance.
- **Operating Policy (Member)**: the member's declared lane and constraints.
- **Storage Map**: declared storage surfaces and available commands.
- **Team Org Context**: "Who I report to + who I direct" (when enabled by team policy).
- **RESPONSIBILITIES.md**: "What I do in this team" (team-specific)
- **Agent .md files**: "Who I am + how I operate" (global, persists across teams)
- **HEARTBEAT.md**: "What I need to do right now" (cron task)
- **Volatile sections**: current inbox, Source Ledger wake, fallback, and validation findings.

The sections are emitted inside one `<context>` element with named child
elements. The task remains outside that element because it is an instruction to
execute, not reference material. Universal, team, and member sections precede
run-volatile sections so a provider can reuse the stable prefix. Volatility
outranks nominal scope when the two conflict.

The Source Ledger is the normal continuity surface. A healthy ledger produces
no handoff instruction. If the ledger is unavailable or its wake read fails,
the builder emits a bounded `<continuity-fallback>` section asking the agent to
record concise continuity in the final response. Attribution receipts, not a
run's own final response, confirm declared-topic writes.

Every team must define `operatingContract` in `team.json`. The prompt builder fails rather than inferring missing contract policy from `TEAM.md`, `RESPONSIBILITIES.md`, `HEARTBEAT.md`, or agent files. Contract-owned policy includes work types, numeric caps, read-only behavior, supersession rules, knowledge topics, source documents, and write surfaces.

The generated Operating Policy embeds the lean `shared/TEAM.md` charter before the generated runtime and contract policy. It also includes top-level runtime, coordination, and execution fields from `team.json`. The rendered policy uses repo-root-relative paths only. For example, a stored `team-shared` path such as `RUN_LESSONS.md` renders as `scenarios/prompt-manager/store/teams/meta-optimization/shared/RUN_LESSONS.md`.

Source ownership:
- `team.runtime`, `team.coordination`, and `team.execution`: runtime mechanics.
- `team.operatingContract`: enforceable member/team policy.
- `shared/TEAM.md`: mission, scope, and team-specific principles.
- `RESPONSIBILITIES.md`: role-specific application of the policy.
- `HEARTBEAT.md`: recurring task loop.
- Agent markdown files: global agent identity and behavior.

### Action Discovery Guidance

Actions are typed executable wrappers over Vrooli-controlled CLI commands. Heartbeat prompts can include a compact runtime rule:

```text
Before manual deterministic operational work, use `prompt-manager discover "<what you need>" --type all`; prefer an exact Action contract over prose instructions when the task is deterministic. Inspect matching Actions with `prompt-manager action show <id>`, validate with `prompt-manager action validate <id>`, and use `prompt-manager action run <id> --dry-run` before execution when running is appropriate.
```

This keeps judgment in skills and execution in Actions without bloating every heartbeat prompt. See [Actions](ACTIONS.md) and [Memory Promotion](MEMORY-PROMOTION.md).

## Prompt Pipeline UI

The Team Members heartbeat UI exposes a **Prompt Pipeline** view that renders
the backend-provided structured prompt order (universal → team → member →
volatile → task, omitting sections that are not present for a member). The
pipeline lives in the member detail panel's **Overview** tab and is shared
between the graph and list layouts.

The UI loads `/prompt-preview-structured` and renders the returned `sections[]` directly. Backend prompt assembly is the source of truth for section order; the UI does not parse flat markdown to infer pipeline order. `/prompt-preview` remains the exact flat runtime prompt used to audit what a heartbeat receives.

- [CODE: ui/src/components/editor/MemberDetailPanel.tsx] - Shared member detail panel pipeline
- [CODE: ui/src/components/editor/TeamEditorPanel.tsx] - Members layout wiring (graph + list)
- [CODE: api/heartbeat/handlers.go] - `POST /prompt-preview`, `POST /prompt-preview-structured`, and `GET /teams/{id}/prompt-matrix`

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

Heartbeats execute via Agent Manager using Prompt Manager's declared profiles:

1. **Profile Reconciliation**: Scheduler startup reconciles the registered
   role-only files under `.vrooli/agent-profiles/`.
2. **Task Creation**: Creates a task with the built prompt.
3. **Run Execution**: Starts a run with the reconciled profile key; Agent
   Manager owns role resolution and concrete runner/model selection.
4. **Completion Tracking**: Polls for completion and updates config.

See [CODE: api/heartbeat/client.go] for the client implementation.

## Safety Considerations

1. **Off by Default**: Heartbeats must be explicitly enabled
2. **Team Gating**: Heartbeats (scheduled or manual) only run when the team is enabled
3. **Engagement Gate**: Auto-pause blocks new starts when operator engagement has gone idle
4. **Profile Controls**: Agent-manager profiles control permissions and resources
5. **Logging**: All executions are logged for audit
6. **Manual Trigger**: Heartbeats can be manually triggered for testing while the engagement gate is `active` or `warning-idle-soon`

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
