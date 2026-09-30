---
name: large-effort-orchestration
description: Run a long delivery goal as an orchestrator that admits epochs, spawns one goal worker per epoch, parks between check-ins, steers through the epoch file, applies the step-back rule and accepts epochs through a mechanical gate. Use for orchestrated long-running efforts only; phased plans, bounded tasks and direct requests do not use epochs.
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [practice]
  tags: [orchestration, multi-agent, effort, epochs, continuity, recovery]
  icon: network
  status: active
  revision: 18
  createdAt: "2026-09-10T00:00:00Z"
  updatedAt: "2026-09-30T18:00:00Z"
  requires:
    scenarios: [prompt-manager, agent-manager]
    commands: [prompt-manager skill read, agent-manager]
  origin:
    kind: authored
---

## Practice focus: Large effort orchestration

Deliver a long goal (days to weeks) through **epochs**: large, verifiable units of
work, each executed by one long goal run and steered by an orchestrator that keeps
its context for hours. This skill is the only home of the epoch standard. Phased
plans, bounded tasks and direct requests do not use epochs.

Delivery teams never use Swarm Manager; they dispatch through Agent Manager and
record state in the goal home.

Supporting files (repository paths; native projection does not install them):
- Worker card — the only protocol an epoch worker loads:
  `path:scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`.
- Lost runs, runner limits and route changes:
  `path:scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/recovery.md`.

### 1. Roles

| Role | Model | Does | Never does |
|---|---|---|---|
| Supervisor | Sol, rare | See `large-effort-supervision`. | Select or accept epochs. |
| Orchestrator | Luna-high | Team leader and one long-lived Agent Manager run. Owns the goal home, admits briefs, sets gates, spawns and steers workers, accepts or reopens epochs, picks the next slice, turns operator feedback into directives, authorizes installs, manages validation targets. | Implement product work. |
| Epoch worker | Luna-medium | One goal run per epoch under the worker card. | Read `FEEDBACK.md`, install external tools, accept its own epoch. |

Record the model on every run you create. Escalate a worker to a stronger model
only with a written reason in the epoch file.

### 2. The orchestrator loop

The team heartbeat (about every 30 minutes) only keeps the orchestrator alive: it
skips while the orchestrator run is running or parked, and relaunches one from the
goal home, with backoff and a cap, when it has ended.

1. **Resume.** Read `GOAL.md`, `QUEUE.md`, the active epoch file, `WORKAROUNDS.md`
   and new `FEEDBACK.md` entries. Your run ID is `.claims.run_id` in
   `agent-manager run identity --json`.
2. **Admit.** If no epoch is active, take the next `QUEUE.md` slice, write its brief
   (§4.2) and set its gates.
3. **Spawn.** If the active epoch has no live worker:
   `agent-manager task create --title "<goal> E<n>" --scope-path <scenario path>`, then
   `agent-manager run create --task-id <task> --profile-id <worker profile> --parent-run-id <your run ID> --until "<epoch outcome and exit gate>" --model <luna> --effort medium --workload-key <goal>/E<n>`.
   The prompt names the epoch file, your run ID and the worker card. The worker
   profile runs on the interactive substrate, whose native `/goal` loop keeps it
   going across turns. Record the run ID in the brief's `Workers` field.
4. **Park.** `agent-manager run park <your run ID> --producer children --key <your run ID> --timeout 15m`.
   You wake when a child run ends, the timer expires, or something calls
   `agent-manager run wake --key <your run ID>` (the worker's epoch-check friction
   wake). Never claim to be parked from prompt text.
5. **On wake.** Read new slice-log lines and run
   `agent-manager effort epoch-check <epoch file> --runs <worker run IDs>`. Stop an
   idle worker that has handed off but still shows as running
   (`agent-manager run stop <id>`). Then do exactly one of: park again, write a
   directive, direct a step-back, spawn a successor, or accept (§4.4).
6. After acceptance, move the slice to done in `QUEUE.md` and go to step 2.

**Waking the supervisor.** After recording a step-back, a workaround logged a
second time, or a spend spike in the goal home, run
`prompt-manager team heartbeat-trigger effort-supervision effort-supervisor`. It
admits at most one event wake per day; the supervisor reads the goal home, so
record the event there first.

When nothing is admissible (every remaining slice waits on an operator decision
or an outside repair), park with `--timeout 12h` instead of ending; an ended run is
relaunched and rereads the goal home for nothing. Never poll children in a loop.

**Steering.** Append a directive to the epoch file's `## Directives`
(`- D<n> <ISO time> <text>`). The worker acknowledges it in its next slice-log
line. `agent-manager run continue <worker run ID> --message "..."` also types the
text into a running session; the Directives section stays the record.

**Operator and supervisor feedback.** Record it verbatim in `FEEDBACK.md` (open
items at the top), then translate it into directives or brief amendments. Workers
see only those. This is the only feedback path.

### 3. Goal home

Scenario goals live in `scenarios/<scenario>/docs/internal/goal/`; other goals in
one dedicated folder. The goal home is the only ledger for the effort.

```
goal/
  GOAL.md          destination, stakes, constraints, links (≤ 2 KB)
  QUEUE.md         ordered next slices, parked slices with return conditions, done list
  epochs/E<n>.md   brief + Directives + Slice log (one file per epoch)
  FEEDBACK.md      operator and supervisor feedback verbatim (orchestrator-owned)
  WORKAROUNDS.md   broken-infrastructure log for the supervisor
  INSTALLS.md      external tools installed for the goal, with cleanup rule
  TARGETS.md       validation targets and tiers
```

Archive history. Never delete a ledger.

### 4. The epoch standard

#### 4.1 Scope

An epoch finishes exactly one of:
- one user-visible outcome, such as a journey that now works end to end; or
- one whole replace-and-delete slice: a concept goes from N owners to 1 and the
  old paths are deleted.

It normally runs several hours to about a day and spans many compactions. Reject
a brief that is one defect, test, fence, sensor or receipt, or that is estimated
under about 3 hours. Split an oversize brief along module boundaries. Compaction,
a status request, a green test or elapsed time is never an epoch boundary.

#### 4.2 Epoch file

`epochs/E<n>.md`. `epoch-check` parses the `- Field: value` lines and the two
append-only sections.

```
# E<n> — <outcome in one sentence>

- Outcome: <journey or queue slice ID and sentence>
- Kind: refactor | feature
- Started: <ISO time>
- Estimate: <work units>
- Expected size: <net runtime lines; feature epochs only>
- Metric: <exit metric name and the command that reads it>
- Targets: <platform: required | occasional | unavailable, ...>
- Workers: <run IDs, comma separated>
- Status: admitted | in-progress | parked | accepted

## Modules and budgets
## Deletion list        (refactor: old paths, shims and callers gone at close)
## Exit gate            (exact commands, affected journeys, growth budget)
## Non-goals
## Directives
## Slice log
```

Only the orchestrator writes the brief, Directives and the `ACCEPTED` line; only
the worker writes other slice-log lines. Work units, the slice-log line, the
growth budget and the step-back triggers are defined once, in the worker card.
Terms: a **slice** is a `QUEUE.md` entry that becomes an epoch; a **work unit**
is one checked change, logged as one slice-log line.

#### 4.3 Step-back backstop

If a worker misses a fired trigger, write the step-back as a directive. Record
every park, switch or shrink in `QUEUE.md` with its reason and return condition.
Calibrate K and the spend threshold from the first three epochs' logs. Never stop
the goal; park instead.

#### 4.4 Acceptance

Accept an epoch only when all of these hold:
- the exit-gate commands and the affected journeys pass;
- the growth budget is met and the deletion list has no remaining callers or
  matches (a cleanup's retirement manifest, `improvement-do-and-dont` D7);
- test changes follow the test rules the worker card links: no new duplicate fakes,
  sleeps for state, or tests of private helpers, and any net test growth is
  explained by behavior newly covered;
- validation targets marked `required` pass;
- `epoch-check` reports no unacknowledged directive.

Rerun the gate yourself; a worker's report is not evidence. Record
`ACCEPTED <time> <summary>` as the final log line, with the changed files and a
suggested commit message; the operator commits. That line is the denominator for
cost per accepted epoch.

Gates are set at admission. Amend one only with a recorded reason (too strict,
incorrect, unverifiable); never lower a gate so a worker can pass, and steer a
struggling worker toward a verifiable path instead.

### 5. Infrastructure, installs and targets

Shared tools are aids, never gates; the worker card states the fallback, and the
supervisor owns repair. Workspace Sandbox is per-run change tracking, not
containment.

**External tools.** Only the orchestrator authorizes an install outside the
project's scenarios and resources, and records the tool, size, reason and cleanup
rule in `INSTALLS.md`. Dependency installs still go through Scenario Dependency
Analyzer. Before the goal completes, stop everything it started outside the
project and uninstall large goal-only installs.

**Validation targets.** At goal start, find which platforms and builds can
realistically be validated. Record each in `TARGETS.md` as `required`,
`occasional` or `unavailable`, and change tiers on new evidence. An unavailable
target never blocks the goal; log it for the supervisor.

**Daily qualification.** Once a day, run the goal's qualification command from
`TARGETS.md` against the goal's shadow copy and record the result in the active
epoch's log. A failure becomes a `QUEUE.md` item, never a reopened epoch.

### 6. Start, recover and close

Start a goal by writing the goal home (preserve the operator's destination,
constraints and decisions in `GOAL.md` with links to their sources), configuring
the delivery team with an orchestrator leader member (delivery-orchestrator
profile), and enabling the team.

For a lost run, runner exhaustion or an uncertain dispatch, read the recovery
reference. A terminal orchestrator run is not goal completion.

Close when `QUEUE.md` has no remaining slice for the destination: stop external
processes, clean up installs, list open workarounds and unverified targets, disable
the team, and record a work record through the Memory contract.
