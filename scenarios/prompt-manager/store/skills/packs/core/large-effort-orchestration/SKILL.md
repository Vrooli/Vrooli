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
  revision: 25
  createdAt: "2026-09-10T00:00:00Z"
  updatedAt: "2026-10-03T13:13:18Z"
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
   (§4.2) and set its gates. If the queue cannot reach the destination, plan
   first (§2.1).
3. **Spawn.** If the active epoch has no live worker:
   `agent-manager task create --title "<goal> E<n>" --scope-path <scenario path>`, then
   `agent-manager run create --task-id <task> --profile-id <worker profile> --parent-run-id <your run ID> --until "<epoch outcome and exit gate>" --model <luna> --effort medium --workload-key <goal>/E<n>`.
   The prompt names the epoch file, your run ID and the worker card. The worker
   profile runs on the interactive substrate, whose native `/goal` loop keeps it
   going across turns. Record the run ID in the brief's `Workers` field.
4. **Park.** `agent-manager run park <your run ID> --producer children --key <your run ID> --timeout 1h`.
   You wake when a child run ends, something calls
   `agent-manager run wake --key <your run ID>` (the worker's epoch-check step-back
   wake, new feedback), or the timer expires. The timer is only a backstop for a
   worker that hangs without ending. After 20 parked minutes Agent Manager
   compacts your session, so a later wake starts from a summary: re-read the goal
   home rather than relying on memory, and keep check-ins short, because each
   wake costs a re-orientation. Never claim to be parked from prompt text.
5. **On wake.** Read new slice-log lines and run
   `agent-manager effort epoch-check <epoch file> --runs <worker run IDs>`. Stop an
   idle worker that has handed off but still shows as running
   (`agent-manager run stop <id>`). Then do exactly one of: park again, write a
   directive, direct a step-back, spawn a successor, or accept (§4.4).
6. After acceptance, move the slice to done in `QUEUE.md`, update the Forecast
   (§2.1) and go to step 2.

#### 2.1 Forecast and planning

The queue serves the destination in `GOAL.md`; an empty queue is not a finished
goal. Keep one `## Forecast` section at the top of `QUEUE.md`, rewritten at every
acceptance:
- the destination metric now, its target and the gap;
- per accepted epoch of the last five: actual result against the brief's estimate;
- the sum of the queued slices' estimates, scaled by the recent actual/estimate
  ratio, against the gap.

Plan new slices when the scaled queue cannot close the gap or fewer than two
admissible slices remain. Measure the modules against the budgets in `GOAL.md` or
`QUEUE.md`, start with the largest gaps, and prefer slices that delete whole
features, files or duplicate surfaces over slices that only move ownership. Give
every new slice a deletion list and an estimate from measurement, not from hope.

Three accepted epochs in a row below half their estimate is a planning finding:
record it in the Forecast with your explanation, re-estimate the queued slices,
and wake the supervisor. So is a destination the queue can no longer reach.

**Waking the supervisor.** After recording a step-back, a workaround logged a
second time, a spend spike or a planning finding (§2.1) in the goal home, run
`prompt-manager team heartbeat-trigger effort-supervision effort-supervisor`
(delivery members may wake a supervision member; at most one event wake per
schedule window). The supervisor reads the goal home, so record the event first.

**Children.** With the orchestrate scope you may `run stop`, `run wake` and
`run continue --message` your own direct child runs; nothing wider.

A parked epoch never idles the goal: after recording a park, switch or shrink,
go straight to step 2 and admit the next slice. Only when `QUEUE.md` has no
admissible slice left (every remaining one waits on an operator decision or an
outside repair), park with `--timeout 12h` instead of ending; an ended run is
relaunched and rereads the goal home for nothing. Never poll children in a loop.

**Steering.** Append a directive to the epoch file's `## Directives`
(`- D<n> <ISO time> <text>`). The worker acknowledges it in its next slice-log
line. `agent-manager run continue <worker run ID> --message "..."` also types the
text into a running session; the Directives section stays the record.

**Operator and supervisor feedback.** Record it verbatim in `FEEDBACK.md` (open
items at the top), then translate it into directives or brief amendments. Workers
see only those. This is the only feedback path. Whoever adds an entry while the
orchestrator is parked wakes it: `agent-manager run wake --key <orchestrator run ID>`.

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
under about 3 hours; fold small follow-ups into the next related slice instead. Split an oversize brief along module boundaries. Compaction,
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
## Deletion list        (refactor: each concrete file, symbol or pattern from the
                        QUEUE slice with its end state: deleted, generated, or ≤ N lines)
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
- the growth budget is met and every deletion-list item has reached its stated
  end state, checked item by item (file absent, symbol gone, size), with no
  remaining callers or matches (`improvement-do-and-dont` D7);
- test changes follow the test rules the worker card links: no new duplicate fakes,
  sleeps for state, or tests of private helpers, and any net test growth is
  explained by behavior newly covered;
- validation targets marked `required` pass;
- selected source/material versions remain accessible and mapped for the epoch;
  any required visual gate uses the actual-build comparison evidence in §6.1;
- `epoch-check` reports no unacknowledged directive.

Rerun the gate yourself; a worker's report is not evidence. The `QUEUE.md` slice
is done only when its whole deletion list is done. If you accept an epoch with
items left, put them back at the top of `QUEUE.md` as a named remainder slice,
and state in the `ACCEPTED` line how the result compares with the brief's
estimate. Never mark a slice done on partial work. Record
`ACCEPTED <time> <summary>` as the final log line, with the changed files and a
suggested commit message; the operator commits. That line is the denominator for
cost per accepted epoch.

Gates are set at admission. Amend one only with a recorded reason (too strict,
incorrect, unverifiable); never lower a gate so a worker can pass, and steer a
struggling worker toward a verifiable path instead. A gate whose tool is still
unavailable after one authorized attempt to provide it is unverifiable: amend it
to record the check as unverified, log the tool in `WORKAROUNDS.md`, and decide
the epoch on its remaining gates. A missing tool never holds an epoch.

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

#### 6.1 Commission an accurate, accessible handoff

Before creating a team, resolve current native outcome/goal ownership. If a team
already owns it, amend that team's handoff through §2 instead of creating another.
Before commission or extension, verify exclusions against the whole approved
outcome, accepted requirements and required shared contracts/generated outputs.
Distinguish observer read-only limits, per-worker slice exclusions and constraints
on the whole outcome. The excluded worker still obeys its slice; required approved
work outside that slice returns to the native owner for a corrected assignment or
next slice, not automatically to a human approval gate.

Use normal governed tools in the established shared checkout. Preserve genuinely
overlapping authored edits and native locking, staging, source-drift and scoped-
publication safeguards. Shared paths or dirty generated output alone do not require
worktrees, a separate artifact root or extra isolation; only an actual native
contract does. For example, a Planner child brief's “Never edit packages/proto”
does not prohibit required Planner contract work across the approved outcome.
Its owner may assign the remainder through the governed Proto route documented in
`path:packages/proto/README.md` and `path:docs/package-governance.md`:
`cd packages/proto && make generate SCENARIO=personal-planner`. Verify freshness
with `make verify-committed-gen`; never hand-edit generated output or overwrite
conflicting authored changes.

Label each blocker as an actual enforcement/admission denial, assignment wording,
or reviewer inference, retaining attempted-action evidence or stating that no
action was attempted. No attempted admission is not proof of denial. Never
override or route around a real denial. Native budgets/security and genuine
action-time confirmations remain intact; this scope check does not expand profiles.

For a new finite delivery team, complete this checklist while the team remains
disabled:

1. **Integrate the target.** Reconcile the applicable canonical PRD, requirements,
   experience and design records with authorized selections. Preserve source
   precedence; distinguish desired from implemented state, selected from candidate
   references, and historical observations from current facts. Write `GOAL.md`
   with the destination, constraints and decisions linked to these canonical
   sources; it must not become a competing specification.
2. **Deliver the materials.** Make every required image and other reference file
   available through the receiving executor's actual route at the selected version
   and usable original resolution. In the project's existing source/asset manifest,
   record identity/version, executor-readable location, bytes, hash, format and
   image dimensions, and requirement-to-screen/state/source mappings. Verify bytes
   and successful decoding/opening. Label governing, illustrative, historical and
   rejected references; preserve originals and identify derivatives. A link, ID,
   contact sheet, overview PDF or older image folder does not prove access to the
   selected originals. If no such materials are required, record that scope.
3. **Verify independently.** A fresh reader, without the author's conversation or
   coached answers, must retrieve the canonical sources and required materials
   through the normal receiving execution route. Check coverage as well as access:
   each relevant requirement and screen/state resolves to its correct source and
   image version; conflicts and missing items are explicit. Retain a dated receipt
   naming the route, identities/hashes, what was opened and mapped, and remaining
   limits. Resolve required handoff gaps before enablement. This verifies the
   handoff, not product implementation or broader authority. Include a concrete
   required shared-contract case: the reader must distinguish a child exclusion
   from whole-outcome authority, recover the native owner/next-slice and governed
   generation/verification route, and identify genuine overlap or denial evidence.
   Resolve lost required work or invented project-wide owner/isolation gates.
4. **Enable the qualified commission.** Configure the native orchestrator leader
   member using the delivery-orchestrator profile and supported disabled
   contract/binding path. Verify the complete contract and accepted source revision,
   then enable under the existing native permissions, budgets and safeguards.
   These checks do not copy the sender's tool restrictions or add a parent-release
   or accounting layer.

For visual delivery, put close fidelity to selected governing images in the
admitted acceptance gate. Compare captures of the exact running build with the
matching screen/state/appearance and intended viewport or device composition.
Preserve required behavior, truthful data and accessibility; document justified
responsive, accessibility and real-data adaptations. Retain source/runtime
versions, captures, differences and unresolved limits with native acceptance
evidence. Decoding images, passing code tests or viewing a contact sheet does not
prove visual acceptance. Do not invent pixel-identity guarantees or numerical
visual budgets that the product contract does not set.

For an active team, send a source/material correction through short verbatim
`FEEDBACK.md` (§2), with changed versions, requirement/acceptance impact and the
receipt. The leader reconciles `QUEUE.md` and epoch directives; use the supported
native wake when parked and retain its acknowledgment. Keep the existing owner
and native safeguards. Do not stop, restart or replace a team to deliver materials.

Source clarification: [Verify specifications and assets before team enablement](https://docs.google.com/document/d/152DaKTk-lBAiy2GXHf-1Bpp3GalY0xXj8EolzhYECrc/edit#heading=h.aig16ectqkdt),
owner-requested 2026-10-03. The operative native checklist is here; following it
does not require recovering the author's cloud conversation.

#### 6.2 Recover and close

For a lost run, runner exhaustion or an uncertain dispatch, read the recovery
reference. A terminal orchestrator run is not goal completion.

Close only when the destination in `GOAL.md` is met, or every remaining gap is
parked on an operator decision with the Forecast saying so; an empty queue with an
open gap means plan (§2.1), not close. On close: stop external
processes, clean up installs, list open workarounds and unverified targets, disable
the team, and record a work record through the Memory contract.

### CreateRun caller identity and refusal

CreateRun requires caller proof at both the HTTP and service boundaries, before
reservation or dispatch. The shared CLI transport forwards an existing run
credential in `X-Agent-Identity-Token`; each adapter still needs qualification.
Delegated requests must name that verified run as their exact `parentRunId` and
retain qualification and execution restrictions. An existing human owner
credential belongs in `Authorization: Bearer`, where the owner verifier must
validate it as a live human with the declared `agent-manager:write` capability.
A run credential in the human channel is not human authorization. If both
channels are offered, both must validate for the same owner; scopes and expiry
are intersected. An invalid offered channel cannot be ignored.

Public CreateRun selects existing profiles. Inline defaults that would create
or change a profile must be reconciled through the existing authorized profile
owner route first; identical existing declarations remain no-ops. An accepted
public replay requires the original persisted caller, task, parent and scope
ceiling. Historical anonymous admissions are refused by the public route;
renewed proof does not renew the original run's authority expiry.

After an authentication or policy refusal, preserve the redacted request and
exact response and stop the denied action and dependent work. Do not remove an
identity header, substitute another principal, attach a run, mint credentials,
or retry anonymously. Body lineage, localhost origin, task/profile names and
harness labels do not prove authority. Use an already authorized direct session
only when the owner explicitly supports that work shape; record its actual
scope without claiming native admission.

The AUTH-01 receiving-boundary source correction is not rollout qualification.
Supported caller migration, service-level replay binding, positive verified
operator/exact-parent tests and independent evidence remain deployment gates.
Other lifecycle endpoints retain their existing contracts.
