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
and standing-supervision heartbeats retain their own admission protocols.

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

## Standing effort supervision

For the organizational model, read
[departments, committees, and supervision](SWARM-MODEL.md#teams-departments-committees-and-supervision).
This is a standing service supervising potentially finite efforts; it is not
one long-lived agent or a replacement leader for each effort.

`heartbeat.json.supervision` selects the standing admission policy described in
[EFFORT_SUPERVISION.md](../../../../docs/agent-system/EFFORT_SUPERVISION.md),
ES-09, ES-10 and ES-13. The existing scheduler supplies recovery ticks. Each
tick first reconciles Agent Manager's bounded discovery projection and the exact
pending supervisor run. Empty or unchanged evidence requires no prompt building
or inference. Discovery continues after the last effort retires and after restart.

Configuration specifies `discoveryLimit`, `maxEffortsPerWake` and
`minWakeIntervalSeconds`; the heartbeat schedule sets discovery cadence. It does
not name efforts, paths, providers or models. Agent Manager resolves its own
protected discovery roots. The existing heartbeat profile resolves resource
policy. Global/team engagement gates and member enabled state still apply.

The consumer-owned `EffortSupervisionOwner` port returns a bounded joined
discovery cut:
stable effort ID, target revision, meaningful evidence revision, eligibility,
owner wait reference, retirement standing, board reference, coverage and errors.
Each selected row also carries its prior assessment, changed evidence references,
usage, named waits and detail references. The heartbeat prompt uses this one
owner read and follows detail references only when they can change the decision;
it does not repeat full board or transcript reads. A server-owned dispatch
authorization anchor is excluded by its typed authorization metadata and empty
subject set, including when a supervisor run is attributed.
Timestamps alone must not change evidence identity. Unknown, missing, conflicting
and removed sources remain unavailable, never accepted completion. AM owns
enrollment and directive fencing. PM stores only wake admission state.

PM persists a wake identity before queue admission and the task identity before
run dispatch. Queued, running, parked, missing and uncertain owner runs retain
the reservation. Lost dispatch responses reconcile the original identity; they
cannot authorize a replacement. Only exact terminal owner evidence releases it.
Disable pauses new work while retaining the reservation. Deleting a standing
heartbeat with unresolved wake state is refused; recreating it in ordinary mode
must not bypass the original run's overlap fence.
Terminal status (including success) is not an assessment receipt. PM reads AM's
`last_assessment` for each selected effort and requires the exact wake idempotency
key, supervisor run and target revision before marking evidence assessed. Only a
matching `sample` receipt completes a healthy sample. Failed, cancelled and
receipt-less attempts remain unassessed; their reservation is released with an
explicit reopen disposition, cooldown and the same charged diagnostic allowance.
Attempt ordering provides fairness without pretending that failed work was served.
Recheck eligibility before dispatch so effort retirement fences queued work.
Retirement of one effort leaves all other efforts eligible. Bounded selection
serves the least recently attempted effort first, with stable-ID tie breaking.

The standing policy extends the heartbeat admission seam. It does not change
finite leader execution semantics. The owner scheduler and durable state are
implementation; a disabled configuration or a prompt preview alone is not
operational qualification. The main implementation owner performs scoped pilot
activation after focused fake-owner/time tests; this work does not enable global
heartbeat policy or mutate existing teams.

When an enabled finite delivery team with at least one `effortRef` is admitted,
Prompt Manager automatically arms the canonical `effort-supervision` team
through the same scheduler. This transition is idempotent and schedules only
the configured standing supervisor member; it never creates one supervisor per
effort. An empty effort board leaves the standing team in cheap discovery-only
idle. Disabling the last effort team does not silently disable the standing
service; explicit lifecycle withdrawal remains the stop operation.

The AM adapter consumes generated `AgentManagerService.GetEffortBoard` pages
(at most 100 rows). AM's existing supervision scheduler owns discovery scans.
PM admits non-withdrawn rows with valid effort and evidence identities
for bounded observation, including stale/unavailable source cuts. Those cuts
retain explicit freshness and observation-only standing: uncertainty can be
assessed, but stale evidence cannot justify steering. An unknown accepted target
is retained as an exact empty target revision in the assessment map and forces
observation-only standing; it is not fabricated or excluded. A receipt must
explicitly contain that map key even when its value is empty. Missing effort
or evidence identity excludes the row with a diagnostic; it does not block a
valid sibling. Unchanged stale evidence coalesces like any other assessed cut;
healthy sampling does not turn an unavailable cut into a healthy sample. Subject
runtime activity and subject waits do not suppress a supervisor wake. Target
revision plus row `change_identity` identify a changed cut; enrollment CAS revision
is bookkeeping, not a trigger. A separately persisted
UUID identifies each wake. A new UUID is never minted to recover a lost dispatch.
The row identity excludes the supervisor's own assessment/accounting changes;
the full display cut may change independently. A receipt only covers the selected
prior cut, never unrelated subject changes that arrive while a wake is running.

Observation identity is provisioned by ordinary AM CreateRun, not by prompt text
or a manually supplied owner credential. PM attaches one typed public, active,
verified `WorkReference` per owner-selected effort, with relationship `supervisor`
and the exact target revision. On a recurring wake, PM uses the supplied compact
owner cut and exact target revision; AM atomically validates membership and
revision, and PM rereads only a bounded compact detail when a material decision
needs freshness. This grants no steering authority. AM persists these references and its
discovery scheduler joins them as observed-supervisor membership. The run uses
the ordinary AM-issued signed identity token to submit an assessment. Before the
periodic join, AM can verify the exact persisted run references directly; pending
board membership is not a reason to skip the receipt. A refused identity or target
revision remains an explicit refusal, never a request for operator credentials. PM does
not mint tokens, read signing keys, exchange a local human principal, maintain a
private token store or forward broad owner credentials. Missing optional steering
delegation never blocks observation.

Stable `supervisorOwnerSubject`/`supervisorScope` delegation and actual permitted
actions remain separate AM owner grants. PM does not provision these grants.
Autonomous steering remains unavailable until its credential-authority route is
qualified. The CreateRun work-reference transport and persistence must be present
in the adopted AM build; Run/RunReport read fields alone do not qualify this path.

The wake prompt supplies a typed assessment request skeleton and calls
`agent-manager effort assess --request-file <request.json> --json` under the
run's signed identity (`WATCH_AUTHORITY_FAMILY_PARENT`, the existing run-token
enum, not a fabricated family). AM's receipt is authoritative; the team knowledge
entry links it and cannot replace it. Evidence/rationale and unknown usage must
be recorded honestly before submission.
For this typed link, use the existing `prompt-manager team knowledge-add` facade
with the runtime-selected team and `supervision-assessment/<wake-id>` topic. The
facade encodes topic and runtime attribution into the team's existing Source
Ledger corpus. A plain journal note has no such typed-topic envelope. Submit one
concise link after AM acceptance, then finish. If linking fails, retain the AM
receipt and the link failure; do not repeat assessment, scan repository-wide topic
definitions or improvise another ledger. Linking failure does not erase AM's receipt.
If AM refuses the exact run membership or target revision, retain that refusal and
finish the wake without claiming a receipt. Discovery reconciliation is an
operator-only owner operation; the supervisor must not invoke it or escalate
credentials. Exact-run membership verification is an AM concern, not a private
PM repair or a polling loop in the prompt.

Optional `healthySampleIntervalSeconds` and `maxHealthySamplesPerWake` enable
bounded independent sampling of unchanged eligible efforts. Zero disables
sampling. Samples share normal cooldown and capacity gates, are identified in the
wake prompt, and do not grant steering authority. With no eligible efforts,
sampling cannot buy inference. The state response reports coverage, waits,
sampling selections and any unresolved dispatch identity without reading AM.
An admitted policy-selected sample remains required for a stopped or
observation-only effort; perform it as diagnostic evidence and keep it separate
from steering authority.

`diagnosticAllowance` persists `maxWakesPerWindow`, `windowSeconds` and
`accountingRef`. Its unit is attempted supervisor inference wakes, including
healthy samples and uncertain dispatches; it does not claim a token/dollar cap.
AM's qualified resource profile still limits each run. PM records discovery and
owner-run read counts under the shared accounting reference even when idle.
Each wake retains that reference, selected efforts, sampling reason and AM run ID.
An exhausted window remains in `allowance-wait` with `allowanceResumesAt`; it
reopens on the scheduler's configured window. Clock rollback cannot reset it.
Pending uncertain runs retain the overlap fence across allowance windows.

### Read status without confusing activity with success

Use the owner views together:

```bash
prompt-manager team heartbeat effort-supervision effort-supervisor --json
agent-manager effort board
```

The team/member selectors above identify the installed standing service, not
an allowlist of supervised efforts. New subjects come from Agent Manager discovery.

| Observation | Meaning and next read |
|---|---|
| `enabled: true` and a future `nextExecution` | Scheduling is configured. Also inspect engagement gates, lifecycle errors and the pending owner run; this alone proves neither execution nor success. |
| `supervisionState.pending` | One wake is reserved. Its run/task IDs are the recovery identity, not an invitation to launch another supervisor. |
| Reconciled assessment / increasing `sequence` | A selected evidence cut obtained an owner receipt. This measures supervision activity, not business outcomes or causal benefit. |
| Idle/no-change or allowance wait | Intentional bounded scheduling. Discovery can continue without another inference run; retain the stated reopening condition. |
| Active enrollment | The watch has not been withdrawn. It does not mean the orchestrator is running. |
| Board `finished`, `unknown`, or legacy `loop_state=stopped` | Inspect orchestrator/worker coverage, outcome standing and stop reasons separately. Observer runs no longer imply business runtime; terminal executors still do not prove acceptance. |

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
counts and success rate unavailable. Standing supervisor runs can have owner
receipts but no such files. Exact team-wide AM accounting still needs public,
owner-verified team attribution; a generic `supervision-` tag is insufficient.
Use retained run IDs and assessment receipts; empty logs are not no execution.

Read one consequential stopped effort's current owner operation and acceptance
checkpoint. A live producer wait, an exhausted allowance, missing authority, a
failed runtime prerequisite, and an interrupted driver require different actions.
An empty tmux listing does not prove absence of the driver or its children.
Do not treat a repeatedly read old stop reason as freshly verified blocker evidence.

### Recovery capability and current limits

The supervisor's target includes detecting premature stops, questioning stale
blockers, and selecting an authorized recovery. Current discovery and signed
assessments qualify observation, not universal autonomous recovery.

Agent Manager's effort-directive delivery continues an exact target at
`needs_review`, or a failed run with an explicit CONTINUE grant, retained session
and recovery hypothesis/comparison. Current source evidence, expiry and cumulative
limits remain required. Active/parked targets retain their existing operation;
cancelled/completed targets are not restarted. A CONTINUE grant does not make
a missing session or failed legacy driver resumable. The route does not execute tmux
commands or edit another effort's control files. See
[Agent Manager's owner reference](../../../agent-manager/docs/reference/effort-supervision.md).

Closing this gap requires an owner-backed leader/run binding, an attenuated
grant, and a qualified recovery operation that reconciles unresolved effects
before resuming. The assessment must name the recovery/repair owner and a
testable reopening condition. Lack of steering authority restricts action; it
does not make an observed failure healthy or prevent reporting that a claimed
blocker may be stale. Source code containing a repair is not proof that the
served runtime adopted it or that the original session can resume.

Declared resolution sources and the generic legacy operator-answer source now
contribute to AM's changed-evidence cut. The skill requires checking these before
repeating a stopped checkpoint's wait. A fresh source without a current steering
grant remains observation-only. Text claims are not credentials, permission or
proof of a successful repair.

Keep this distinction visible during adoption: "supervisor running" means the
bounded observation loop is operating. It does not yet mean every stopped
orchestrator will be diagnosed, repaired and restarted without intervention.

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
Unknown dispatch remains reserved. An owner run ending does not accept the effort
or authorize a replacement. Recovery uses AM controls on the exact retained run.
Setting `finiteLeader.retired=true` permanently fences future dispatch; disable
and global/team pause retain the reservation. Retirement does not cancel a run.
Identity and profile changes and binding deletion are refused. This minimum
interface does not implement Aquila's unbuilt finite-team runtime or autonomous
fresh-run recovery, and does not supply missing human deployment inputs.

### Child cohorts, parking, and watchdogs

Finite coordinators that delegate child runs use Agent Manager's durable cohort
watch rather than a five-minute polling loop:

1. Resolve the verified parent identity with `agent-manager run identity --json`.
2. Create children with `agent-manager run create --parent-run-id <parent-run-id>`.
3. Create one cohort watch with the parent run and the exact child run subjects.
4. Park the parent with `agent-manager run park <parent-run-id> --producer supervision --key <watch-id>`.
5. Let the watch wake the same parent when all children are terminal. The parent
   reconciles the child results and cancels the settled watch.

The park deadline is a watchdog, not a replacement-run timer. A timeout wakes the
same parent so it can classify active, terminal, missing, or uncertain children.
The parent must not retry an uncertain child or create a fresh coordinator run.
The supervision waiter and watch action path are server-owned and survive an
Agent Manager restart. A finite coordinator that has no child cohort should end
normally with a durable handoff; it should not claim to be parked.

Implementation checkpoint (2026-09-12, scoped W3 repair): twelve finite-leader
test functions pass with the race detector across heartbeat, store and teamconfig,
alongside the existing atomic config-publication and standing-supervisor wiring
regressions. They exercise concurrent admission, duplicate effort binding,
restart, parked/queued/uncertain and terminal owners, lost task/run responses,
retirement, member/team disable, global/team pause, exact work references,
ordinary heartbeat history, AM-owned continuation observation and queue wiring.
The initial expanded test runs exposed two fixture defects (missing `EnabledSet`
and relation-store wiring); the corrected focused run passed. Command:

```bash
go test -race ./internal/heartbeat ./internal/store ./internal/teamconfig -run '^TestFiniteLeader|^TestHeartbeatConfigPublicationPreservesInFlightReaderSnapshot$|^TestStandingSupervisorWiringUsesExistingHeartbeatHistory$' -count=1 -timeout=90s
```

This is shared-worktree fixture evidence, not live migration qualification.
The existing Connect `HeartbeatService.CreateHeartbeat` and `UpdateHeartbeat`
accept the binding in their JSON `body`. The canonical `team heartbeat-bind-effort`
command now provisions it disabled from a bounded request file; see the
[CLI reference](../reference/heartbeat-cli.md#finite-effort-leader-provisioning).
Remaining adoption work includes attenuated recurring authority and fresh
grant/source validation, disposable live qualification, exact predecessor
settlement, and useful-progress verification. The minimum runtime observes AM
continuation; it does not schedule it. No driver, business run, heartbeat
activation or scenario restart was performed for this checkpoint.

Recurrence proposal pending owner integration: use the AM workflow's pinned
definition digest, exact execution and node-attempt identities, and validated
structured-result/wait/signal journal records. An explicitly selected continuation
edge with a satisfied owner wait can dispatch under the exact effort grant;
run termination alone cannot select an edge. An accepted retirement branch must
retain its owner evidence. PM retains the workflow/admission reference in its
existing heartbeat runtime state and provides a recovery tick; AM retains the
workflow transitions, attempt idempotency and cumulative allowance.

The currently implemented `agent-manager:supervisor-dispatch:v1` purpose grants
`SupervisorScope` through `CreateSupervisorRun`. It does not establish finite
coordinator authority or a purpose-bound workflow continuation operation. The
finite integration must use an explicit owner grant for that role and propagate
it through each workflow attempt. This proposal is not a claim that those owner
extensions have shipped. Direct coordination with the dispatch owner remains
unconfirmed: no native collaborator-message tool or verified Dirac inbox mapping
was available to the finite-leader implementation session.

### Team-native qualification boundary

The finite-leader binding is part of one governed team-native effort start. A
real effort must first have a finite delivery team, a validated operating
contract, a registered Source Ledger scope, an exact accepted effort/revision,
and a disposable qualification of the selected owner route. Qualification must
show one controlled owner admission and reconcile PM's admission identity with
the Agent Manager task, run, work reference and handoff/receipt. It must also
exercise disable, pause, restart, uncertain dispatch, retirement and explicit
completion/reopen boundaries without changing global heartbeat policy or
another effort.

The disposable qualification does not establish recurring continuation. A
heartbeat tick may retain and observe an owner wait or uncertain identity, but
it cannot choose a fresh run, invent authority, or replace a lost dispatch.
Fresh-run recovery, accepted continuation authority, useful-progress evidence,
and autonomous recurrence remain separate gates until their owner contracts are
qualified. Keep the real finite effort disabled and unapproved when any gate is
unqualified. The standing effort supervisor remains read-only observation of
owner state; it is not the finite coordinator.

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
