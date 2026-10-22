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
  revision: 56
  createdAt: "2026-09-10T00:00:00Z"
  updatedAt: "2026-10-21T02:00:07Z"
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

Delivery teams use Agent Manager for execution and record state in the goal home.
Repairs inside the approved effort destination stay in that effort; a second
proposal is not required. Work outside its approved scope or a new operator
disposition follows the existing Swarm route in
`path:docs/agent-system/SWARM_MANAGER_WORK.md`. This distinction does not change
the orchestrator's Agent Manager execution or acceptance authority.

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

1. **Resume.** Run `agent-manager effort lint <goal home>` first. Each open
   FEEDBACK entry it names is new steering, and it outranks the handoff: act on
   it this check-in. Then read `GOAL.md`, `QUEUE.md`, the active epoch file and
   `WORKAROUNDS.md`. Your run ID is `.claims.run_id` in
   `agent-manager run identity --json`.
2. **Admit.** When no epoch is active, take the next `QUEUE.md` slice, write its
   brief (§4.2) and set its gates. If an epoch has a live worker, do not replace
   or broaden its admitted scope to plan future slices. Use §2.1 to prepare
   remaining work during a due or changed check-in.
   Dispatch another slice only when the effort/team policy permits it and no
   file, lease, host/device, or other exclusive-resource conflict remains. A
   serialized queue permits no second worker until the active worker returns.
   Do not add a competing concurrent implementation worker to the same epoch
   or a second effort leader. After the active worker returns, use the normal
   review, recovery or next-slice route for that epoch.
   Run `git-control-tower proposal anchor --effort
   effort:<slug> --epoch E<n> --scope '<scenario path>/**' --json`, which
   fingerprints files that are already dirty. If the queue cannot reach the
   destination, plan first (§2.1).
Before dispatch, recheck the cited failure and relevant source hash; reconcile fixes already present, and refresh the brief so it names only work still required.

3. **Spawn.** If the active epoch has no live worker:
   Put the complete worker prompt in the task description. This CLI accepts
   `--description` as a string and has no file flags for task descriptions or
   run prompts. From Bash, read the brief file into one quoted argument:

   ```bash
   agent-manager task create --title "<goal> E<n>" --scope-path <scenario path> --description "$(cat -- /path/to/worker-brief.md)"
   agent-manager run create --task-id <task> --profile-id <worker profile> --parent-run-id <your run ID> --until "<short engine-owned completion predicate>" --model <canonical model ID from the selected profile/catalog> --effort medium --workload-key <goal>/E<n>
   ```

   Use the exact canonical model ID accepted by the selected profile/catalog;
   for the current catalog, that is `gpt-6-luna`, not the shorthand `luna`.
   Correct only a deterministic request-validation refusal that identifies a
   noncanonical spelling. Retry through ordinary admission with only the model
   ID corrected; keep the caller, task, profile, parent, scope and all other
   request fields unchanged. The corrected ID must remain within that profile's
   already permitted model/tier, and all normal admission checks still apply.
   Authentication, authorization, budget and policy denials remain stop
   conditions; do not retarget or route around them.

   Shell command-substitution output is passed as argument text; the shell does
   not re-parse Markdown or shell syntax inside the brief. Keep the quotes around
   the substitution. If the task already exists without a description, set it
   first with `agent-manager task update <task> --description "$(cat -- /path/to/worker-brief.md)"`.
   Do not also set `--prompt` on this path: it replaces the task description as
   instructions; it does not append to it. Keep `--until` separate and short:
   it is the engine-owned completion test, limited to 2,048 characters, not the
   worker brief. The worker profile runs on the interactive substrate, whose
   native `/goal` loop keeps it going across turns. Record the returned run ID
   in the brief's `Workers` field.
   Name the full repository-relative epoch path, your run ID and the worker card
   in the prompt. Make the assignment self-contained: include the applicable
   directive IDs, authority, gates, source references and bounded resume set.
   Keep the complete epoch as the authoritative ledger for `epoch-check`; the
   worker reads other epoch sections or older evidence when an assigned
   requirement depends on them.
   Use owner-returned IDs, not predicted suffixes. Preserve a verified handle
   for the admitted owner and asset; reconcile derived references in this goal
   instead of creating another. Stop before writes only on owner, asset or scope
   mismatch, or an unverified identity.
   Assign one meaningful behavior with implementation, focused self-validation
   and handoff together when available. Before setting the brief's scope, inspect
   the existing tests, fixtures and workflow paths that directly validate the
   behavior; include only those needed for focused self-validation when
   authorized and available. Keep independent review separate. A held
   check needs an explicit return condition and retained evidence, rather than
   another goal solely to repeat orientation. Follow the worker card's coherent
   scope rule; task completion is not epoch acceptance, and duration is not a gate.
4. **Park.** `agent-manager effort park <goal home> --json` defaults to the
   verified calling run; omit `--run`. If you supply `--run`, it must match the
   verified identity. Inspect the returned park status and `await_handle` before
   ending the turn, and preserve any refusal or nonzero command result (do not
   mask the exit). This applies the park policy below; do not call `run park`
   directly.
   You wake when a child run ends, an authorized caller runs
   `agent-manager run wake --key <your run ID>`, or the timer expires. A raw
   `FEEDBACK.md` edit is passive. A recorded eligible signal for the active
   effort may also cause Agent Manager's bounded dispatcher to issue a neutral
   check-in to this same parked orchestrator on a later scheduler tick. This is
   not child completion, approval, or scope change. The check-in consumes the
   previous await and clears its wait handle. Reread canonical feedback and
   signal state, then inspect your current run and the awaited condition. If the
   same producer is still unresolved and the goal still waits on it, park this
   run again with the same producer and key before ending the turn. A prior
   successful park does not mean this run is currently parked. Continue only
   within approved scope. The timer remains a backstop for a worker that hangs
   without ending. After 20 parked minutes Agent Manager compacts your
   session, so a later wake starts from a summary: re-read the goal
   home rather than relying on memory, and keep check-ins short, because each
   wake costs a re-orientation. Never claim to be parked from prompt text.
5. **On wake.** Treat every wake as a resume: repeat step 1 and reconcile open FEEDBACK entries,
   `QUEUE.md`/handoff, and parked-slice return conditions against current native child and
   external-owner state before choosing an action; a timer wake with no change writes nothing
   and parks again. Before assessing source drift, select the exact admitted worker run from
   current Agent Manager state and read its terminal return. Compare that attempt's final source
   identities, when available, with its own prewrite baseline and current source; treat older
   final tables as historical. A hash of handoff Markdown identifies that document, not the
   product files it describes. If the exact return or source identities are unavailable, keep
   attribution unknown. Read new slice-log lines and run
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
admissible slices remain. Measure progress against the destination metric in
`GOAL.md`; measure module budgets and start with the largest gaps. For goals that
target simplification or debt reduction, prefer removing whole features, files,
duplicate surfaces or unnecessary ownership over moving ownership alone. Every
slice names its intended end state, a deletion list with an explicit reason when
no deletion applies, and a measured estimate. Keep deletion and consolidation
work visible in product slices when it is needed to leave the touched area in a
maintainable state. Estimate the destination metric, including justified code
additions; line reduction is evidence only when lines are the stated metric.

Do not defer remaining-scope planning until a live worker returns. If the
remaining destination has no ordered, estimated path, use a due or changed
check-in and current goal, queue, benchmark and owner evidence to plan coherent
remaining slices and their critical path. Record proposed slices and estimates
in `QUEUE.md` as unadmitted work. Preserve the active brief and gates. Do not
repeat unchanged reads or checks. Reconcile the active worker's actual return
before dispatching a prepared slice, and obey the configured concurrency and
resource policy.

For a simplification or debt-reduction goal, when a census finds no large dead
or duplicate code, consider a **redesign slice** only when measured remaining
gap is primarily caused by ownership or coupling and a simpler owner can preserve
the required behavior. Its brief carries the new owner, behavior it keeps, the
journeys that prove it and the owners it replaces. Name those owners in the
deletion list and estimate progress against the goal's metric, including any
necessary additions. If no such redesign advances the destination, remeasure
the gap and plan the next slice from its actual cause. Only the operator may
retire a live feature.

Use two distinct activities to maintain code quality:

| Activity | Scope and result |
| --- | --- |
| Independent review | Review coherent outcomes for behavior, ownership and complexity before acceptance. Batch dependent corrections and reuse valid same-risk reviews. A bounded handoff alone does not require a new reviewer task or epoch acceptance. Never accept work you implemented. |
| Preexisting debt reduction | When maintainability is part of the approved destination, assign bounded Luna implementation work alongside delivery. Select existing duplication, obsolete paths or unnecessary complexity within the authorized scenarios. It may target a different module or earlier changes; it need not concern the current repair. Deliver the refactor, remove superseded code and callers, and verify preserved behavior. A findings report alone does not complete this work. |

Look for both local and scenario-wide simplification opportunities. Before a
systematic change, assign a bounded Luna research pass to compare concrete
alternatives against current code and existing shared infrastructure. Require
measured duplication or ownership problems, preserved behavior, migration and
validation scope, and expected delivery benefit. Select the strongest justified
change; do not limit the search to the current diff or invent a framework to meet
a deletion target. Research is decision input, not completed debt reduction.

Route parallel implementation and research through the current epoch worker's
permitted delegation and admitted scope. This does not authorize an additional
goal worker for the same epoch or a second effort leader. Work outside that brief
needs the normal scope amendment or admission as a later slice/epoch. Independent
acceptance remains a separate review; the implementer cannot accept its own work.
Coordinate write ownership and validation. Record cleanup scope, removed
responsibilities, retained contracts, validation and cost in the existing brief
and queue. Review the result independently. Avoid
cosmetic churn, arbitrary deletion quotas and unrelated scope expansion. Keep the
cleanup bounded so useful delivery continues; the orchestrator retains acceptance.

An outside-owner dependency may delay its required check or affected slice, not
other admissible work. Keep the check and slice open under `## Waiting on others`
in `QUEUE.md`, with the unmet gate and return condition. Continue other admissible
slices, then resume the remainder when that condition changes. Do not accept the
epoch or mark its slice done until its required gates and assigned outcome pass.
Scope changes use the authorized amendment route and preserve the approved
destination; they do not silently waive required work.

Three accepted epochs in a row below half their estimate is a planning finding:
record it in the Forecast with your explanation, re-estimate the queued slices,
and wake the supervisor. So is a destination the queue can no longer reach.

**Waking the supervisor.** After recording a step-back, every 3rd acceptance, a
workaround logged a second time, a spend spike, a planning finding (§2.1) or a new
`## Needs operator` item in the goal home, run
`prompt-manager team heartbeat-trigger effort-supervision effort-supervisor`
(delivery members may wake a supervision member; at most one event wake per
schedule window). The supervisor reads the goal home, so record the event first.

**Children.** With the orchestrate scope you may `run stop`, `run wake` and
`run continue --message` your own direct child runs; nothing wider.

Parking does not make the remaining goal idle. At a due or changed check-in, use
§2.1 to plan any unplanned remainder and complete independent reads that do not
conflict with active work. Keep prepared slices unadmitted until the configured
concurrency policy permits dispatch. If all due work is complete and only a child
or external return remains, park on its real condition. Do not keep the
orchestrator live for future work or repeat unchanged planning. Only when every
remaining slice really waits on an operator decision or an outside repair:
1. Put the decisions needed, with options and your recommendation, under
   `## Needs operator` at the top of `QUEUE.md`.
2. For each genuine operator choice that blocks the outcome, use the native
   `agent-manager effort decision raise` route described in
   `path:scenarios/agent-manager/docs/reference/effort-supervision.md` under
   Decisions (DL-5). Use a stable unique ID, factual question and viable options;
   keep any recommendation tied to evidence. Record the returned ID, pending
   state and event-publication status. An item in `QUEUE.md` alone is not a
   notified decision; if publication is pending or failed, retain that state and
   follow the owner contract. Do not raise decisions for plain status notes or
   waits on an external repair, repeat a decision to send another notification,
   invent an answer, or set a default unless the choice is reversible and its
   automatic application is explicitly appropriate under the decision contract.
3. Wake the supervisor once.
4. Park with `--timeout 72h`; follow §2 for same-run check-ins. A raw
   `FEEDBACK.md` edit does not itself emit a signal. Do not end the run, because
   an ended run is relaunched and rereads the goal home for nothing.

Never poll children in a loop.

**Handoff.** `QUEUE.md` holds one `## Handoff` section (what changed, what was
verified, what remains, the exact next action). Replace it only with
`agent-manager effort handoff set <goal home> --file <path>` at each check-in
that changed something; name every open FEEDBACK entry in it. A timer wake that finds no
change writes nothing and parks again.

**Steering.** Append a directive to the epoch file's `## Directives`
(`- D<n> <ISO time> <text>`). The worker acknowledges it in its next slice-log
line. Use `agent-manager run continue <worker run ID> --message "..."` to send a
message to a running interactive child. That path accepts message text only and
rejects `--reinstall-goal`.

For a completed interactive child with a nonblank `Until` and a native goal
verdict of `complete` or `blocked`, omit `--reinstall-goal` for a one-turn
status or report follow-up. The prior goal verdict stays unchanged. Add
`--reinstall-goal` only when you intend to reinstall that retained objective
before continuing it, and only for an eligible terminal run. Do not treat a
follow-up run's completion as acceptance of the retained goal. Use the existing
Agent Manager continuation contract for other run states. The `Directives`
section stays the record.

Apply an amendment to the next authorized dispatch or continuation it affects.
Before claiming the amended outcome delivered, verify that the brief, directive
acknowledgment, and installed completion predicate agree about its requirements
and source pointers. A continuation message does not prove the predicate changed.
Use the existing native amendment route. If it cannot synchronize the affected
continuation, record the exact constraint and carry the requirement into the next
properly admitted assignment. Do not rewrite historical accepted goals or restart
a run merely to manufacture agreement. Missing source-load receipts establish
uncertainty, not noncompliance.

**Operator and supervisor feedback.** The orchestrator owns `FEEDBACK.md`; the
person who supplied feedback does not thereby maintain the ledger. Record each
item verbatim (open items at the top), then translate it into directives or
brief amendments. The orchestrator may move genuinely resolved entries verbatim
to the resolved archive while preserving any operative conditions; unresolved
entries, including unanswered human-only decisions, stay active. Acknowledgment
alone does not resolve an item. Workers see only the applicable directives and
brief amendments. This is the only feedback path. Writing an entry does not wake
a parked run. An existing authorized caller may run
`agent-manager run wake --key <orchestrator run ID>` for an immediate resume;
otherwise the run resumes through its normal producer or deadline wake.

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

Keep the files you read on every resume (`GOAL.md`, `QUEUE.md`, open
`FEEDBACK.md`, open `WORKAROUNDS.md`) under 30 KB together. When they grow past
that, move done slices, resolved entries and census evidence to
`goal/archive/`, leaving a one-line pointer. Archive history; never delete a
ledger.

### 4. The epoch standard

#### 4.1 Scope

An epoch finishes exactly one of:
- one user-visible outcome, such as a journey that now works end to end; or
- one whole replace-and-delete slice: a concept goes from N owners to 1 and the
  old paths are deleted.

It often spans several hours and many compactions, but duration is not an
admission gate. Reject a brief that is only one defect, test, fence, sensor or
receipt unless it completes a coherent user-visible outcome or whole
replace-and-delete slice with a clear exit gate. Admit a smaller coherent outcome
when it has that gate; do not stretch it to meet a runtime or token target. Combine
follow-ups only when they belong to the same behavior. Split an oversize brief
along module boundaries. Compaction, a status request, a green test or elapsed
time is never an epoch boundary.

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
- test changes follow the authoring rules the worker card links: retire obsolete
  checks and inert support, preserve unique protection and valid helper/lifecycle
  and special-kind oracles, keep domain-local builders/fakes local, and share
  suitable mechanics. Avoid unexplained duplicate setup and sleeps used as
  evidence of completion. Explain net test growth by a concrete gain in
  protection, qualification, isolation, or readability. Scale evidence to the
  effect under `path:docs/TESTING.md`; counts and lines alone prove no improvement;
- validation targets marked `required` pass;
- selected source/material versions remain accessible and mapped for the epoch;
  any required visual gate uses the actual-build comparison evidence in §6.1;
- `epoch-check` reports no unacknowledged directive.

Record justified retention as assessment complete, not simplification delivered.
Record an implemented change with proportionate evidence as delivered, not yet
independently accepted. Acceptance closes the scoped outcome; it does not prove
fleet adoption or completion of the whole goal. Preserve material gaps beside the
affected claim.

Independently inspect retained owner artifacts, source applicability and actual
proof for each gate; a worker's report alone is not evidence. Rerun gates only
when inputs changed or evidence is missing, stale, partial or leaves material
risk unresolved, following `path:docs/TESTING.md`; reuse valid owner evidence for
unchanged inputs. The `QUEUE.md` slice is done only when its full assigned
outcome and deletion list pass. If work must
be repartitioned, use the authorized amendment route and put the unfinished
remainder at the top of `QUEUE.md` with its unmet gates and return condition.
Keep the original outcome open until every partition passes; do not claim the
original outcome accepted or mark its slice done on partial work. Then run
`git-control-tower proposal create --effort effort:<slug> --epoch E<n>
--subject '<scenario>: <outcome> (E<n>)' --body-file <body> --path <file>…
--run <worker run> --run <your run> --json`. The body states the outcome, the
deletion-list result, the gates and the metric delta. Record
`ACCEPTED <time> <summary>` as the final log line and cite the returned
`gctp-…` ID; the operator approves the proposal in Git Control Tower. That line is the denominator for
cost per accepted epoch.

Gates are set at admission. Amend one only with a recorded reason (too strict,
incorrect, or requiring a different proof route); never lower a gate so a worker
can pass. When a tool is unavailable, use equivalent authorized evidence that
proves the same required behavior when available. If no equivalent proof is
available, keep the gate and affected outcome open, log the tool and return
condition in `WORKAROUNDS.md` and `QUEUE.md`, and continue other admissible work.
Tool availability can change the proof route; it does not waive required proof
or the approved outcome.

### 5. Infrastructure, installs and targets

Tool brands and availability are not acceptance gates; required behavior and
proof are. The worker card states the fallback so implementation can continue
where possible, and the supervisor owns shared-tool repair. Workspace Sandbox
is per-run change tracking, not containment.

**External tools.** Only the orchestrator authorizes an install outside the
project's scenarios and resources, and records the tool, size, reason and cleanup
rule in `INSTALLS.md`. Dependency installs still go through Scenario Dependency
Analyzer. Before the goal completes, stop everything it started outside the
project and uninstall large goal-only installs.

**Validation targets.** At goal start, find which platforms and builds can
realistically be validated. Record each in `TARGETS.md` as `required`,
`occasional` or `unavailable`, and change tiers on new evidence. An unavailable
target does not block other admissible work. If it prevents proof of a required
outcome and no equivalent authorized evidence is available, keep that outcome
open with its return condition; log it for the supervisor.

**Daily qualification.** Once a day, run the goal's qualification command from
`TARGETS.md` against the goal's shadow copy and record the result in the active
epoch's log. A new failure becomes a `QUEUE.md` item. If it shows that current
evidence for an accepted outcome is invalid or no longer applies, reopen that
affected claim through the existing route and keep it unverified until current
proof passes. Preserve historical receipts and accepted snapshots.

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

For a new finite delivery team, complete this checklist while the team and its
member heartbeat remain disabled:

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
   image dimensions, and map each applicable requirement to its governing
   source/version and applicable screen/state, or record an explicit
   nonvisual/no-screen disposition.
   Verify bytes and successful decoding/opening. Label governing, illustrative,
   historical and rejected references; preserve originals and identify
   derivatives. A link, ID, contact sheet, overview PDF or older image folder
   does not prove access to selected originals. If no such materials are required,
   record that scope.
3. **Verify independently.** A fresh reader, without the author's conversation or
   coached answers, must retrieve the canonical sources and required materials
   through the normal receiving execution route. Check pre-enable source coverage
   as well as access: each applicable requirement resolves to its governing
   source/version and applicable screen/state, or has an explicit no-screen or
   nonvisual disposition. Verify semantic joins against governing source context,
   not name/prefix similarity or manifest membership alone; keep current
   implemented behavior, preserved behavior and roadmap/deferred items distinct.
   Conflicts and missing items remain explicit. Retain a dated receipt
   naming the route, identities/hashes, what was opened and mapped, and remaining
   limits. Resolve required source/asset handoff gaps before enablement. This
   source-handoff check does not require completed product implementation or
   passing product acceptance tests; the native team owns those delivery proofs.
   It does not establish broader authority. Include a concrete
   required shared-contract case: the reader must distinguish a child exclusion
   from whole-outcome authority, recover the native owner/next-slice and governed
   generation/verification route, and identify genuine overlap or denial evidence.
   Resolve lost required work or invented project-wide owner/isolation gates.
4. **Qualify the existing owner route.** Declare the effort reference and leader
   in the existing Prompt Manager team contract; keep its operating-contract
   member and plan-of-record/source paths with that team. Use Agent Manager's
   existing effort-enrollment route for current revision and qualification; do
   not invent a Prompt Manager bind/accept command or copy a revision into a
   second configuration. Read back the exact enrollment through
   `agent-manager effort board --effort-ref <effort-ref> --json` and confirm it
   is current and not withdrawn. Configure the leader's ordinary member
   heartbeat with its selected profile, and confirm active membership and the
   actual owner route. Prompt Manager admits that finite-lead dispatch only
   when the team contract, member/profile and exact current enrollment match,
   then records the enrollment as a descriptive WorkReference on the frozen run
   request.
   This reference grants no authority. Missing, unavailable or mismatched
   identity fails admission; if a prior run ID exists, preserve that owner
   obligation and do not start competing work. Keep the team and heartbeat
   disabled until the fresh receiver's source and route receipt passes. Then
   enable through the existing operator-supported team and heartbeat controls,
   subject to native permissions, budgets and safeguards. These checks do not
   copy the sender's tool restrictions or add a parent-release or accounting
   layer.

For visual delivery, selected governing images are implementation targets for
their matching screens and states, not inspiration or semantic checklists. Use
the existing `experiential-ui-design` comparison guidance to choose the next
implementation change from a matched current/target view; prioritize shared
composition and style foundations before state-specific polish. At the existing
visual gate, compare captures of the exact served build with the selected
original at matching state, theme and viewport. A capture, traceability join or
passing test proves presence or coverage, not fidelity. Preserve required
behavior, truthful data and accessibility; record only justified adaptations in
the existing brief, and have the existing independent reviewer judge remaining
differences. Do not invent pixel-identity guarantees or numerical visual budgets
that the product contract does not set.

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

After an authentication, authorization, budget or policy refusal, preserve the
redacted request and exact response and stop the denied action and dependent
work. Do not remove an identity header, substitute another principal, attach a
run, mint credentials, or retry anonymously. Body lineage, localhost origin,
task/profile names and harness labels do not prove authority. Use an already
authorized direct session only when the owner explicitly supports that work
shape; record its actual scope without claiming native admission.

The AUTH-01 receiving-boundary source correction is not rollout qualification.
Supported caller migration, service-level replay binding, positive verified
operator/exact-parent tests and independent evidence remain deployment gates.
Other lifecycle endpoints retain their existing contracts.
