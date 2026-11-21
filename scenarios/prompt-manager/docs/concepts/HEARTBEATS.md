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
create configuration revisions. Manual triggers bypass this gate. Delivery
leaders use the same heartbeat admission and execution path as other members;
their effort association is recorded on the resulting run.

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
`controlState`, `scheduled`, `schedulerRegistered`, and `schedulerState`.

Treat `effectiveState` as the execution authority:

| State | Meaning |
|---|---|
| `scheduled` | The team and heartbeat are enabled, heartbeat control allows starts, cron is running, and a scheduler entry exists. |
| `team-archived` | The team is retained for historical recovery, but all execution is refused until it is restored. |
| `team-disabled` | The heartbeat configuration is retained, but the team is turned off. |
| `paused` | Heartbeat control is blocking new starts. |
| `disabled` | The member heartbeat configuration is off. |
| `not-scheduled` | Configuration is enabled, but no live scheduler entry exists. |
| `scheduler-stopped` | Enabled configuration has no running cron, even if an entry was registered. |
| `scheduler-failed` | Startup failed; the response retains the startup reason. |
| `unavailable` | Prompt Manager could not establish a trustworthy state; do not infer that execution is active. |

All scheduled, manual, and retry dispatch paths re-check the team archive/enable
state and heartbeat-control gates before starting work.

The team dashboard labels local-log coverage separately and leaves total run
counts and success rate unavailable; empty logs are not no execution.

## Team-native effort commissioning

Delivery teams use the existing shared team contract and ordinary heartbeat route. The team declares its effort reference and leader; its operating contract carries member policy and plan-of-record/source paths; and the leader member heartbeat selects the profile. Prompt Manager does not implement a separate finite-leader binding, amendment, retirement, completion, or restart API.

Keep a prepared team and heartbeat disabled while the receiver follows the reusable [large-effort-orchestration commissioning checklist](../../store/skills/packs/core/large-effort-orchestration/SKILL.md#61-commission-an-accurate-accessible-handoff) for the actual goal home, selected sources, and owner route. On a configured finite delivery-lead dispatch, Prompt Manager requires a valid team contract, declared leader, active member, matching profile, and exact current Agent Manager enrollment before creating a task or run. It attaches the enrollment's non-withdrawn target revision as a public, verified `orchestrator` WorkReference in the frozen CreateRun request. This is observational work identity and grants no authority. Missing, unavailable, or mismatched identity fails admission. When a prior run ID exists, PM retains that exact owner obligation as uncertain and blocks competing work until identity can be confirmed. It watches a verified active/parked owner through the existing owner path; a verified terminal owner permits a fresh dispatch under current configuration. Finite non-lead members and ordinary standing/non-delivery heartbeats remain supported without orchestrator association.

Source acceptance stays in the effort's canonical source manifest and admitted task snapshot. A fresh receiver must open the repository-relative selected materials and record coverage and limitations in the effort's goal-home receipt before enablement. Agent Manager owns enrollment and exact run identity; Prompt Manager does not copy accepted revisions or source references into another config. The ordinary parked orchestrator and child-watch route remains the continuation model. Event check-ins are proposed and unavailable; the signal source in Agent Manager alone does not enable them. See the [heartbeat CLI commissioning route](../reference/heartbeat-cli.md#native-delivery-team-commissioning).

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

1. AM's existing declaration installer applies the role-only files under
   `.vrooli/agent-manager/`; installation owns updates and override policy.
2. Ordinary scheduler startup resolves the installed profiles using one current
   unpaginated `GET /api/v1/profiles` per exact key. Startup requires matching
   owner, source path/hash, last-applied hash, and declared execution controls,
   including budgets and sandbox defaults. Local overrides or drift hold startup
   for installer disposition. It never invokes ensure/reconciliation or retries
   anonymously to install a profile.
3. Historical profile-filtered run reads require exact key/UUID identity only;
   retired roles, drift or local overrides do not invalidate historical reads.
   Missing/ambiguous/malformed or unsupported owner responses return unavailable
   evidence; a conflicting UUID/key pair is refused rather than weakening a filter.
4. AM starts the run using the declared profile reference. Task creation submits
   once with local redirect refusal. Definitive 4xx/pre-send failures are typed
   refusals; lost responses, 5xx, malformed success and invalid returned UUIDs
   retain uncertain effects for owner recovery. CreateRun proof remains specific
   to CreateRun; it is never copied to the task or other endpoints.

Scheduler registration, running cron, future next fire and an observed callback
are separate observations. `Scheduler.State` retains startup failure and the
last same-instance cron callback; health or a registered entry does not prove
cron is running. A failed startup stops initial schedule loading.

A bounded reviewer may have an explicitly disabled configuration (`enabled:false`)
with an empty schedule. The native creation command is
`prompt-manager team heartbeat-disable <team> <member> --on-demand --profile=<key>`.
It requires an explicit profile and verifies the disabled, schedule-free owner
readback. Create/update accepts that configuration without a dummy
timer, and enabling it requires a real schedule before persistence. The finite
orchestrator assigns independent review as a bounded child operation through the
existing lawful owner route; roster membership or disabled configuration supplies
no task/run authority and starts no reviewer loop.

Matching terminal finite health can repair a stale finite lifecycle projection
under the existing store lock. The repair revalidates the retained state and the
config captured after the owner read, changes only status, and never replays
completion, registry, queue, dispatch or attempt effects. Concurrent predecessor
changes and local admission/backoff holds retain their owning meaning. Failure
of the existing save returns its error and restores the in-memory status.

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
