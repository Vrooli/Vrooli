# Heartbeats & Cron Execution

Heartbeats enable team members (agents) to execute autonomous tasks on a schedule. This system allows agents to periodically perform work without human initiation.

## Overview

Each team member can have at most one heartbeat configuration. Heartbeats are **disabled by default** to prevent accidental expensive LLM usage - they must be explicitly enabled.

### Ordinary wake admission

Ordinary heartbeats retain the historical run-on-every-schedule behavior unless
their configuration explicitly sets `wakeAdmission` to `on-change`:

```json
{
  "wakeAdmission": {
    "mode": "on-change",
    "changeSources": ["team", "member", "inbox", "corpus"]
  }
}
```

The first observation admits a run. Later schedule ticks are quiet when the
selected bounded source identities are unchanged, and admit again when any
selected identity changes. Admission evidence is runtime-only, stored beside
the member heartbeat state rather than in `heartbeat.json`, so polling does not
create configuration revisions. Manual triggers bypass this gate. Finite-leader
heartbeats retain their own admission protocol.

The gate is deliberately fail-open: if source evidence or its runtime baseline
cannot be read, the scheduled run is admitted. A failed queue or execution does
not consume the observed change, allowing a later tick to retry it. Use
`mode: "always"` (or omit the block) for proactive members whose cadence is
itself the work, such as recurring scans.

Runtime admission state also retains `admittedCount`, `quietCount` and
`failOpenCount`. These counters are diagnostic evidence for tuning the gate;
they are not a billing or token budget. A fail-open caused by an unreadable
state store cannot be persisted, so a missing or stale state file is unknown,
not evidence of zero failures.

## Effort supervisor

The `effort-supervision` team's `effort-supervisor` member is an ordinary
member heartbeat: a daily schedule plus inbox wakes. An orchestrator that
records a step-back, a repeated workaround or a spend spike sends the member one
inbox message and triggers its heartbeat. `wakeAdmission: on-change` with the
`inbox` source skips a scheduled wake whose inbox and team state are unchanged.
The supervisor reads goal home files, steers through `FEEDBACK.md`, and records
one knowledge entry per wake. The method is `large-effort-supervision`; the
contract is [EFFORT_SUPERVISION.md](../../../../docs/agent-system/EFFORT_SUPERVISION.md).
Prompt Manager has no supervisor-specific admission, dispatch or recovery path,
and enabling a delivery team does not enable the supervision team.

### Effective execution state

Heartbeat configuration is retained when a team is turned off so that the
operator can restore the prior schedule. Therefore `enabled: true` on a
heartbeat configuration is not an execution claim. The API and CLI also expose
the derived fields `teamEnabled`, `effectiveState`, `effectiveReason`,
`controlState`, and `scheduled`.

Treat `effectiveState` as the execution authority:

| State | Meaning |
|---|---|
| `scheduled` | The team and heartbeat are enabled, heartbeat control allows starts, and a scheduler entry exists. |
| `team-archived` | The team is retained for historical recovery, but all execution is refused until it is restored. |
| `team-disabled` | The heartbeat configuration is retained, but the team is turned off. |
| `paused` | Heartbeat control is blocking new starts. |
| `disabled` | The member heartbeat configuration is off. |
| `not-scheduled` | Configuration is enabled, but no live scheduler entry exists. |
| `unavailable` | Prompt Manager could not establish a trustworthy state; do not infer that execution is active. |

All scheduled, manual, retry, and finite-leader dispatch paths re-check the
team archive/enable state and heartbeat-control gates before starting work.

The team dashboard labels local-log coverage separately and leaves total run
counts and success rate unavailable; empty logs are not no execution.

## Finite effort leader binding

The optional `finiteLeader` heartbeat configuration binds `effortRef`,
`acceptedRevision`, `coordinatorPromptRef` and 1–8 `sourceRefs` to the heartbeat's
explicit `profileKey`. The selected member must be the active leader of a
serialized team. Its normal assembled heartbeat prompt remains the coordinator
prompt; references identify the accepted context and do not grant authority.

PM retains one admission identity per effort in its existing RuntimeData store.
The identity survives queue loss and restart. Before enabling a migration, the
operator must settle the legacy driver and its queued, parked or uncertain owner
operations. An unused disabled heartbeat can acquire a binding; an ordinary
heartbeat with execution history cannot silently change into a finite leader.

The scheduler reserves before queueing and dispatch rechecks current controls.
Unknown dispatch remains reserved. An owner run ending does not accept the
effort; only an orchestrator leader is relaunched (see Orchestrator liveness).
Setting `finiteLeader.retired=true` permanently fences future dispatch; disable
and global/team pause retain the reservation. Retirement does not cancel a run.
Identity and profile changes and binding deletion are refused.

### Orchestrator liveness

A finite leader whose binding declares `"keepAlive": true` is a delivery
orchestrator (`large-effort-orchestration`). Its heartbeat (about every 30
minutes) only keeps it alive:

- While the orchestrator run is queued, running or parked, the tick observes it
  and does nothing else.
- When the run is terminal and the effort has no completion receipt, the tick
  records the run in `restartHistory` (evidence `liveness-heartbeat:<status>`,
  newest 20 kept), clears the reservation and queues one fresh leader. The new
  run resumes from the goal home. One tick relaunches at most once.
- A run that failed or ended within 30 minutes is short. Consecutive short
  relaunches back off (30 minutes, then 1 hour after the previous relaunch);
  after 3 the heartbeat stops relaunching and reports status
  `relaunch-capped` with the reason in `error`. A healthy long run resets the
  count; so does the explicit `Restart` operation, which is the way out of the
  cap once the cause is repaired.
- A completed, retired or disabled effort is never relaunched.

`keepAlive` is policy, not binding identity: it can be set on an existing
binding with `heartbeat-bind-effort --update`.

Every other finite leader keeps one dispatch per effort: a terminal owner run
neither accepts the effort nor authorizes a replacement, and recovery uses the
explicit `Restart` operation on the exact retained run.

Leaders park instead of waiting in-turn. `agent-manager run park <run-id>
--producer children --key <run-id> --timeout 15m` suspends the run with no token
use. Agent Manager wakes the same run when any child run ends after the park, when
the timer expires, or on `agent-manager run wake --key <run-id>`. The waiter is
server-owned and survives an Agent Manager restart. A leader with no pending
child ends its turn with a durable handoff; it does not claim to be parked.

Focused tests: `go test ./internal/heartbeat ./internal/store ./internal/teamconfig
-run '^TestFiniteLeader|^TestOrchestratorHeartbeat'`.

### Team-native qualification boundary

The finite-leader binding is part of one governed team-native effort start. A
real effort needs a finite delivery team, a validated operating contract, a
registered Source Ledger scope and an exact accepted effort revision. Keep the
team disabled until the operator starts the goal. The effort supervisor is not
the finite leader.

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

The generated Operating Policy embeds the lean `shared/TEAM.md` charter before the generated runtime and contract policy. It also includes top-level runtime, coordination, and execution fields from `team.json`. Repository-owned paths render relative to the repository root; read-only `external` document references retain their absolute authority path and cannot be declared as writes. For example, a stored `team-shared` path such as `RUN_LESSONS.md` renders as `scenarios/prompt-manager/store/teams/meta-optimization/shared/RUN_LESSONS.md`.

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
