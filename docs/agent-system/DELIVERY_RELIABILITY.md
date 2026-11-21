# Delivery reliability program

**Plain summary.** Long delivery goals (orchestrator, epoch workers and a
supervisor) have produced accepted increments, while unattended recovery and
operator attention still require qualification under the current commission. In
the Browser Automation Studio (BAS) goal, most lost time came from a few
repeating causes:

- rules that existed only as skill prose, which agents drifted from;
- a destination nobody had measured;
- evidence requirements that grew after admission;
- operator decisions that sat unseen for days;
- shared infrastructure that broke under real goals.

This program moves those rules into tools, gives the operator a real decision
channel, and adds a canary that catches infrastructure breaks before real goals
do. It also makes commits informative and linked to the work that produced them.

Status: **active** · Owner: operator, with a Claude Code coordinating session ·
Started 2026-10-07 · Tracker: [§7](#7-tracker).

Current corrective commission (October 16, 2026):
[delivery reliability goal](../internal/delivery-reliability-research/CHARTER.md)
and [execution queue](../internal/delivery-reliability-research/STATUS.md).
The operator confirmed implementation and qualification using Luna agents.
Historical defaults below do not override [OWNER-INTENT R01–R27](../internal/delivery-reliability-research/OWNER-INTENT.md): token targets
are advisory and stopping caps remain off by default. Completion requires named
required outcomes; source tests alone do not establish serving adoption,
operator attention, receiver recovery or accepted useful delivery.

Terms used here:
- **Goal**: a long delivery effort with a destination, run as epochs.
- **Epoch**: one bounded work unit with a brief, deletion list and exit gate.
- **Goal home**: the folder of files that is the goal's only ledger.
- **Orchestrator**: the team leader run that admits, steers and accepts epochs.
- **Worker**: the run that implements one epoch.
- **Supervisor**: the Sol run that audits and repairs from above.
- **Canon**: `large-effort-orchestration` (LEO) and `large-effort-supervision`
  for method, and [EFFORT_SUPERVISION.md](EFFORT_SUPERVISION.md) for authority.

---

## 1. Context and history

### 1.1 Before epochs (to 2026-09-29)

A week of BAS rehabilitation moved runtime code by −0.4%. In the same week,
agents added about 65k lines of process documents and 24k lines of supervision
infrastructure. The diagnosis found three causes:
- Ratchets were asymmetric: agents added code and tests easily and never removed
  them.
- A stall never led to a park.
- The protocol itself was heavy.

The operator approved the epoch model (decisions D1–D28, plan
`epoch-based-delivery-orchestration-and-bas-rehabilitation`):
- one Luna-high orchestrator per goal;
- one Luna-medium worker per epoch on the interactive substrate;
- a daily Sol supervisor;
- all canon kept in the LEO skill.

### 1.2 What worked (2026-09-29 → 2026-10-03)

E0–E22 were accepted in four days. Runtime fell from 283,309 to 254,178 lines
(−29.1k, −10.3%). The journey suite grew from 5 to 12 and passed daily on an
isolated shadow. Consolidations included:
- one session broker;
- one recording journal;
- one frame transport;
- one AI engine;
- one timeline shape;
- one replay spec;
- a schema-driven node editor (35 components removed).

Step-backs, remainder requeues and supervisor event wakes all worked at least
once.

### 1.3 What went wrong

| # | Observation (dated, measured) | Cost |
|---|---|---|
| O1 | 2026-10-03 → 10-06: no product epoch. The orchestrator woke every 12 h, appended a "nothing changed" handoff, and parked again. That made 8 handoffs; `STATUS.md` reached 57 KB and the goal home 93 KB against a 20 KB budget. | 3.5 days idle; context cost on every wake |
| O2 | The censuses found no large dead or duplicate code. The orchestrator read "admit only slices with a measured deletion list" as "admit nothing". | Root of O1 |
| O3 | The 205k destination came from module budgets that nobody had reconciled with an owner map. Redesign slices E23–E26 then delivered −328, −66, −332 and −801 lines. At that rate the gap needs 100+ epochs. | About 1 week of chasing an unreachable target |
| O4 | The Record/Export retirement decision (FB-062) waited 3.5 days. Nothing notified the operator, who found out only by asking. | Same stall as O1 |
| O5 | 2026-10-06: skill revision 26 added "replace the handoff" and "no 12 h timer park". By 10-07 the orchestrator was appending a growing handoff again and planned a 12 h park. It is a long-lived run, so it holds the skill text it read days earlier. | Rules ignored within a day |
| O6 | Evidence ceremony grew after admission. E22 was reopened because a worker's acknowledgement line was malformed, which needed a successor run. E26's brief (written by the orchestrator) required "compare every changed path to the pre-write baseline". The worker then could not prove the identity of the generated `ui/dist` bundle, overran its spend trigger twice (673k vs 500k weighted), and parked a passing −801 change. | One successor run; one passing epoch held; about 2× epoch spend |
| O7 | Shared infrastructure broke under real goals: the Codex `rate_limits` decode (no token data for any interactive run), the park/wake writer race, cross-team supervisor wake, the orchestrator unable to stop its own children, supervisor wake admission, missing run identity in resumed shells (logged twice), the engaged-live restore-point layout (live BAS down 10-04 → 10-06), the `run quiesce` drain blocking promotion on shadow-only work, and the live UI build failing after promotion. | Each cost hours of delivery and needed the coordinating session to diagnose |
| O8 | One Sol supervisor daily check (2026-10-06) cost 1.42M weighted non-cache tokens, about 5× a whole epoch, mostly to refile a known defect. | Supervision cost above delivery cost |
| O9 | The `bas-goal` shadow engagement lasted a week. Shared `packages/proto` changed under it, so the frozen live copy could not compile. Promotion became one big step. | Live outage; risky promotion |
| O10 | `agent-manager effort board` did not show the BAS goal. All status reached the operator only through the coordinating session. | Operator time |
| O11 | The operator commits by hand with short names ("Bas rewrite p16"). The working tree holds weeks of uncommitted work from many sessions, and commits carry no link to the goal, epoch or run. | Weak history; hard review; risk of loss |
| O12 | Agents in other sessions (for example a 2026-10-04 recovery task) edited the canonical supervision skill and worker reference. Canon drifts from more than one writer. | Unreviewed rule changes |
| O13 | 2026-10-07: a freshly launched orchestrator (current skill) followed the old `## Handoff` text ("park for 12 hours") over the open BAS-FB-064 directive, and parked after 3 minutes. | Prose in the goal home outranked both skill and feedback |
| O14 | `vrooli scenario restart` stops live before building; only `start` builds first. A failed UI build on restart would have taken live BAS down again. | Latent outage |
| O15 | 2026-10-07 01:50Z: the E26 worker ran `git-control-tower baseline start --name bas-goal`. It overwrote the open engagement's restore point with the working tree, then failed with "engagement is live". | The "frozen" live copy silently held unaccepted code |
| O16 | The restore point copies about 2.8 GB including BAS's live SQLite database. `abandon` would restore stale data over today's; `abandon` and `gc` delete the serving tree while live still runs from it. | Latent data loss and outage |
| O17 | "Shadow data population skipped — safety backup unavailable" is a parse bug: the backup CLI prints `run_id`, GCT reads `runId`. No promote has had a rollback snapshot. | Silent loss of a safety net |
| O18 | Lifecycle stage logs are deleted about 70 seconds after a run, so the original live UI build failure left no log. | Lost diagnostics |
| O19 | notification-hub has never delivered a notification (128 of 128 unroutable). Its event intake has rejected 51,209 deliveries (401) since 2026-08-25 because it reads an unset environment variable instead of the stored secret. | No operator channel exists at all |
| O20 | Park ends the agent process mid-turn, so that turn's usage is never emitted. The BAS orchestrator recorded usage for 11 of 66 turns, so `usage run` (then `run tokens`) reads unknown. | Spend invisible |
| O21 | Resumed-shell "no identity token" is not a long-lived-run problem. When a CLI notices stale sources it rebuilds and re-executes itself through `envkit`, which drops `VROOLI_AGENT_IDENTITY_TOKEN` (`packages/cli-core/cliutil/stalechecker.go:223`, `envkit.go:114-116`). | Recurring workaround in every goal |
| O22 | BAS epoch workers run in place with the sandbox off. Agent Manager has no per-run file-change records for them, so a commit proposal cannot take its file list from Agent Manager. | Shapes DL-10 |
| O23 | A research helper agent ran `git log -1`. Subagents that spawn helpers do not pass operator rules on. | Rule breach |
| O24 | 2026-10-07 14:15: another session marked two develop-time WebKit tools in `scenario-to-desktop` as required. Preflight enforces develop-time host requirements over every scenario's whole dependency tree, and restart stops first, so restarts failed fleet-wide and notification-hub went down. | Fleet-wide outage risk from one manifest edit |
| O25 | api-core `eventbus.NewDiscoveredClient` resolves the events address once and keeps an empty one forever. `PublishDomainEvent` then reports success without sending (system-monitor, storage-manager, deployment-manager, scenario-to-cloud). | Silent event loss |
| O26 | 2026-10-07 ~14:54Z: the shared Playwright browser cache lost pinned Chromium 1169, most likely through a Playwright install or uninstall run by an agent, which garbage-collects browsers. The BAS shadow driver then failed to start. Workers were barred from installing browsers and restarting the shadow, and Search Hub listed no recovery route, so E27/E28 went unverified for about 1.5 h. Yet BAS has a supported route (`pnpm run browsers:ensure` plus a shadow restart), which fixed it in minutes. | An epoch stalled on a missing route; supervisor repair (DL-6) and route discoverability are the fix |
| O27 | 2026-10-07 19:05Z: on its third wake the orchestrator again missed a new FEEDBACK entry (FB-065: browser restored) and kept waiting for a repair that had already happened. Interim fix: LEO rev 27 makes `effort lint` the first resume step and requires `effort park`/`effort handoff set`; the orchestrator was relaunched fresh. | Third O13-type miss; tool-led resume now in canon |
| O28 | 2026-10-07 19:09Z: the coordinator restarted agent-manager to deploy track B. The restart also deployed another session's AUTH-01 caller checks from the working tree (then marked rollout-held; the gate contradiction that caused this was later reconciled by the auth-consolidation effort — see `docs/concepts/AUTHORIZATION.md`); prompt-manager followed at 19:31Z. Since then every unattended `CreateRun` is refused: keep-alive leader relaunch, scheduled heartbeats, and the orchestrator's supervisor wake. The BAS leader sat in `dispatch-uncertain` for 3.5 h while `heartbeat-list` showed healthy. | All unattended delivery blocked; restarts deploy whatever is in the shared tree |
| O29 | 2026-10-07 23:26Z: the P-18 deploy restart again shipped other sessions' source-only work. A `code.delivery` exact-effort gate refused the high-effort leader. A 2026-09-28 `resource-codex` CLI emitted no `effort` because its freshness inputs omitted `packages/agentharness`. The real error was relabelled "all models … excluded by resource policy". Fixed: the error now passes through, `agentharness` is added to five resource freshness inputs, and a `VROOLI_DELIVERY_EFFORT_ENFORCE` switch defaults off. BAS leader `9871d148` running 00:08Z. | Second O28 repeat in a day |
| O30 | 2026-10-07: BAS finished E33 with gates met but could not record acceptance: GCT refused a sixth open proposal for the effort, and four of the five open ones had drifted because later epochs touched the same files. A slow operator review stopped delivery. | Track P (wave 2) |
| O32 | 2026-10-08: after the task-create repair, the BAS orchestrator ran `run create` with a truncated task ID (35 characters), then filed bug `knw-1791485193193951732` saying it had sent the exact UUID. The validation error was correct. A generic error plus unchecked transcription produced a false defect claim and a park. | `run create` now echoes a malformed ID with its length (deployed with the next agent-manager CLI build). BAS-FB-069 corrected it, and E34 was dispatched at 18:53Z. Open: evidence lines in agent bug reports should quote the command, not paraphrase it. |
| O33 | 2026-10-09: after accepting E34 the BAS orchestrator parked 72h on outside conditions with an empty `## Next` while BAS-FB-036 stayed open and actionable (the O31 pattern again). Its GCT proposal step failed on a tool defect and was logged as a workaround, so E34 reached no review. Separately, the coordinator's 1h grant lapsed and every `agent-manager` command, health included, failed as "Unable to reach the API"; and `effort board` took 62s. | Fixed and deployed 2026-10-09: a lapsed grant returns the session to its pre-grant identity (revocation still refuses), and a refused credential is reported as an authorization error without auto-start. Grant default 24h, maximum 7 days, no longer cut short by the approver's login session (docs/concepts/AUTHORIZATION.md). GCT: `check-ignore` ran under `--literal-pathspecs`, which it rejects for every path; and an `--effort` carrying `@<revision>` produced an invalid epoch trailer. Board: supervision read each run through `GetRun`, which rescans an interactive run's event history; it now reads stored state (62s → 3.8s; worst row 22.5s → 0.18s). BAS-FB-070 directs FB-036 as the next epoch. Park-while-actionable remains a DL-2 item. |
| O31 | 2026-10-08: after E33 the BAS orchestrator parked 72h on two non-blockers (a proposal awaiting review, a 24-vs-25 journey count). The coordinator answered both (BAS-FB-067), but could not wake it: the operator's own session now carries an attached-run identity and `run wake` is refused 403, and `team trigger` does not wake a parked leader. Agent Manager was also restarted at about 04:36Z with partial wave 2 source (cause unknown). | DL-2 removes parking; F2 item 9. 2026-10-08: the operator's session acts through owner-approved `vrooli agent authorize` grants. `agent-management` scoped to `agent-manager:run/<id>` covers wake. The new `effort-metadata` policy (`effort.reconcile` on `agent-manager:effort/<name>`) covers goal-home registration; it was deployed 17:57Z. First live use 2026-10-08: combined grant df6bbfd4 (agent-management plus effort-metadata, one consent) approved; `effort reconcile-metadata` without `--local-owner` moved BAS to revision 12 with destination `path:.vrooli/work/delivery/browser-automation-studio-delivery/CHARTER.md`. `effort board --json` timed out at 60s during the check (not investigated). Friction found, then fixed and deployed 18:39–18:44Z:
- The selector now accepts the scope name or PID.
- One request carries several policies, each paired with its own targets (`OperationGrant.Rules`).
- A new request supersedes a pending one instead of being refused.
- Expired requests read `expired`, and approving them says why.
- Browser approval shows a readable page instead of JSON.

The 15-minute window was the pre-17:34Z build and is not a bug now.

Also found: the AUTH rollout's agent-authority middleware refused orchestrator `task create` (E34 parked; `knw-1791484033866508554`) and worker `run wake --key`. Both were admitted at 18:44Z: task create requires a running run whose profile declares `agent-manager:orchestrate`, and wake-by-key keeps its lineage rule. BAS was woken with BAS-FB-068. |

### 1.4 Root causes

1. **Prose rules without enforcement.** Every rule that failed (handoff
   replacement, park policy, size budget, Needs operator) lived only in skill
   text (O1, O5).
2. **Long-lived runs keep stale instructions** and pay re-orientation and
   compaction cost on every wake (O5, part of O8).
3. **Uncalibrated destinations** and no rule for diminishing returns (O2, O3).
4. **Gates are open-ended.** Briefs can invent evidence requirements, and
   audits reward more evidence, never less (O6).
5. **No operator channel** and no default for reversible decisions (O4).
6. **Infrastructure is validated only by real goals** (O7, O9).
7. **Spend lacked useful outcome and repeated-work attribution** (O6, O8).
   The October 16 correction makes optimization targets advisory; missing caps
   are not a delivery defect.
8. **Visibility depends on asking** (O10, O11).
9. **Safety nets were never exercised** (restart order, restore-point scope,
   rollback snapshot, log retention, notification delivery), so each was broken
   when first needed (O14–O19).

---

## 2. Principles

- **A rule that fails twice in prose moves into a tool.** The skill then says
  "use command X" instead of describing the rule. This is the three-speed stack
  (skills → governed programs → scenarios) applied to the delivery loop.
- **The goal home is the only memory.** A check-in that does not need prior
  conversation should not carry one.
- **Destinations are measured before they are committed.**
- **Gates are fixed at admission.** Acceptance checks only what the brief
  named. Any amendment is recorded with a reason.
- **The operator decides through one channel, with a recommendation and a
  default.**
- **Infrastructure faults surface in a canary first.**
- **Spend targets are advisory per role.** Keep stopping caps off by default.
  Measure repeated investigation, supervision share and cost per accepted
  outcome; finish coherent useful work when it exceeds a target.
- **Current operating rules stay in force:** follow the repository's AGENTS.md
  and the operator's session instructions. EFFORT_SUPERVISION.md defines the
  repair boundary: approved in-scope repair stays with its owner; proposed new
  scope is filed to its owner, not routed through Swarm. Historical rules below do not
  replace that current authority.

---

## 3. Recommendations (workstreams)

Each workstream has an ID, the problem it fixes, the intended behavior, and a
Gherkin acceptance. Design detail from research goes into §5.

### DL-0 Immediate repairs (BAS)

**Problem.** E26 is parked on an invented gate, the BAS goal is chasing an
unreachable destination, the orchestrator runs old skill text, and live BAS
failed a replacement UI build after promotion.

**Behavior.**
- Accept E26 under a recorded gate amendment: generated `ui/dist` identity is
  not a gate, and source changes plus journeys are.
- End the line-count phase at about 253k. Re-aim the BAS goal at product
  quality: journeys 12 → 24, E22 R1/R2, recording reliability and desktop
  portability. (Default chosen 2026-10-07; the operator may still choose to
  close the goal instead.)
- Relaunch the orchestrator so it reads the current canon.
- Find and fix the live UI build failure before anything restarts live BAS.

```gherkin
Given BAS-FB-064 is recorded in the goal home
When the orchestrator next checks in
Then E26 has an ACCEPTED line citing the gate amendment
And CHARTER.md states the quality destination
And `vrooli scenario restart browser-automation-studio` builds the UI and reports healthy
```

### DL-1 Loop rules enforced by commands

**Problem.** Handoff, park and size rules fail in prose (O1, O5).

**Behavior.** Agent Manager `effort` commands own the rules:
- `effort handoff set <goal home>` replaces the single `## Handoff` section.
- `effort park` picks the timeout from goal state. It refuses a park longer than
  the backstop unless a child is live or `## Needs operator` holds a decision.
- `effort epoch-check` (or a sibling `effort lint`) fails on a resume set over
  its byte budget, more than one handoff section, or an empty admissible queue
  with no Needs operator item.
- The skill names the commands instead of restating the rules.

```gherkin
Given a goal home whose STATUS.md has two handoff sections
When the orchestrator runs effort lint
Then the command fails and names the duplicate section
Given no child is live and Needs operator is empty
When the orchestrator requests a 12 h park
Then effort park refuses and states the admissible-slice rule
```

### DL-2 Event-driven, short-lived orchestrator check-ins (proposed; unavailable)

**Status.** Agent Manager has the durable effort signal source. Prompt Manager does
not consume it or implement event check-ins; the supported operating mode remains
the long-lived parked orchestrator. DL-2 is proposed work, not deployed behavior.

**Problem.** A long-lived parked orchestrator keeps stale skill text, pays
compaction and re-orientation on every wake, and needed park/wake race fixes (O5, O7, O8, O20). The identity-token
workaround has its own cause (O21) and is fixed in cli-core.

**Behavior.** Each check-in is a fresh run started by an event: a child run
ending, new feedback, an operator answer, or the backstop timer. The run reads
the goal home and current canon, acts once, writes the handoff and ends. The
team heartbeat or an Agent Manager trigger owns event-to-run dispatch with
deduplication, so two check-ins never run at once.

```gherkin
Given a goal with an active epoch worker
When the worker run ends
Then exactly one orchestrator check-in run starts within 5 minutes
And it reads the canon revision current at its start
And it ends after writing the handoff, holding no parked session
```

### DL-3 Destination calibration and diminishing returns

**Problem.** An unmeasured destination drove a week of low-yield work (O3).

**Behavior.**
- A new goal's first epoch is a calibration spike. It maps the owners of the
  largest modules and returns a destination with a confidence range. The
  operator approves the destination after the spike.
- `epoch-check` flags diminishing returns: three accepted epochs in a row under
  25% of estimate, or two censuses in a row with no admissible slice. The flag
  requires a re-aim decision under Needs operator, with options.

```gherkin
Given three consecutive accepted epochs each delivered under 25% of estimate
When the orchestrator runs epoch-check
Then the report contains a diminishing-returns finding
And the orchestrator cannot admit a further slice until Needs operator holds a re-aim decision
```

### DL-4 Fixed evidence menu

**Problem.** Briefs invent gates, and audits reward ceremony (O6).

**Behavior.**
- An exit gate is a list of typed items from a short menu: `test` (command),
  `journey` (IDs), `inventory` (metric and bound), `review` (named check).
  `custom` is allowed only with a stated reason, checked at admission.
- Acceptance checks exactly the admitted items. A later amendment needs a dated
  reason line.
- The supervisor's audit fails gate inflation (evidence demanded outside the
  brief) as it fails partial work.
- Acknowledgement formatting can never reopen an accepted epoch on its own.

```gherkin
Given an admitted brief whose exit gate lists test, journey and inventory items
When the worker meets those items
Then epoch-check reports the gate met, whatever other evidence is missing
```

### DL-5 Operator decision channel with defaults

**Problem.** Decisions sat unseen for days (O4).

**Behavior.**
- A `## Needs operator` item has a fixed shape: question, options, a
  recommendation and an optional default with a deadline.
- Notification Hub adds a recipient-authored `Other` response to every decision
  and supports dismissal without an answer. The callback preserves free text;
  dismissal closes the matching decision as unanswered and never grants
  authority.
- Writing one notifies the operator once (push notification and/or the Command
  Center inbox), with a one-action answer path that records the answer
  verbatim in `FEEDBACK.md` and wakes the orchestrator.
- For a reversible choice, the stated default applies after the deadline unless
  the operator objects.

```gherkin
Given the orchestrator writes a Needs operator item with a 24 h default
When 24 h pass without an operator answer
Then the default is recorded in FEEDBACK.md as "applied by default"
And the orchestrator continues with it
```

### DL-6 Delivery-loop canary

**Problem.** Real goals were the only test of shared infrastructure (O7).

**Behavior.** A small synthetic goal runs daily through the whole loop:
- spawn a worker;
- child-end wake;
- `epoch-check` triggers;
- accept;
- token accounting;
- supervisor wake;
- shadow start, promote and re-create on a toy scenario.

It is cheap: bounded turns and a fixture scenario. A failure is the
supervisor's first repair target that day and is reported through DL-5.

```gherkin
Given the canary schedule fires
When any loop step fails
Then the canary records the failing step and evidence
And the operator channel reports one infrastructure item
```

### DL-7 Small promotions and safe baselines

**Problem.** Long shadow engagements drifted from shared packages, and
promotion was risky (O7, O9).

**Behavior.**
- Promote after every N accepted epochs (default 3) when journeys are green,
  then re-create the shadow at once. TARGETS and LEO state the cadence.
- Baseline capture freezes the shared packages a scenario uses (Go `replace`,
  pnpm `file:` and Vite alias targets), so the engaged live copy always compiles.
  (The 2026-10-06 serving-tree fix made the layout resolve; freezing makes it
  stable.)
- `run quiesce` drains only runs that use the live instance. Its errors describe
  the real cause.
- The live UI build after promotion passes (DL-0).

```gherkin
Given an engaged shadow and a change to packages/proto in the working tree
When live BAS restarts during the engagement
Then live builds from its frozen shared packages and reports healthy
```

### DL-8 Advisory token targets by role

**Problem.** Reporting spend alone did not explain repeated orientation,
stalls or supervision overhead (O6, O8).

**Behavior (operator correction, October 16).**
- Role targets guide optimization for supervisor checks/audits, orchestrator
  check-ins and epoch workers. Stopping caps remain off by default.
- Exceeding a target remains visible with accepted outcome and repeated-work
  evidence. It does not terminate coherent useful work or restart investigation.
- Existing provider, security and separately approved operational limits retain
  their own semantics; they do not convert these targets into stopping caps.

```gherkin
Given a worker exceeds its advisory token target while making useful progress
When usage is recorded
Then the overrun remains visible for optimization
And the advisory target does not stop the worker
```

### DL-9 Effort board from goal homes

**Problem.** Goals were invisible without asking (O10).

**Behavior.**
- `agent-manager effort board` and the Efforts view read registered goal
  homes. They show: destination metric, current value, active epoch, last
  acceptance, open Needs operator items, open workarounds, spend per accepted
  epoch, and canary status.
- Missing facts read as unknown, never zero.

```gherkin
Given the BAS goal home is registered
When the operator opens the effort board
Then BAS appears with its runtime metric, the active epoch and open decisions
```

### DL-10 Informative, linked commits

**Problem.** Commits carry no information or links, and review is manual (O11).

**Existing work this builds on, which must be reconciled rather than duplicated:**
- Git Control Tower (GCT) plan `git-control-tower-advisory-maturity-20260905`
  (status draft), Phase 6 (trailer parsing and work-reference resolution) and
  Phase 7 (pending and committed AI provenance). Its trailer intent:
  `Vrooli-Plan:`, `Vrooli-Backlog:`, `Vrooli-Continues:`. Keys derive from
  current owner identities, unknown keys are preserved, and history is never
  rewritten.
- `scenarios/git-control-tower/api/internal/trailers` (parser) and
  `internal/provenance`.
- The `git-control-tower-draft-commit` skill (editable message plus supported
  trailer suggestions for an exact subject; never commits).
- Plan `generic-provenance-blame-knowledge-observatory-cleanup` (complete): the
  generic work-reference contract in Agent Manager and Vrooli Events.
- The Workspace Sandbox auditability contract (pending provenance promoted to
  committed).

**Behavior.**
- At each epoch acceptance, the orchestrator records a commit proposal. The file
  list comes from Agent Manager file-change records of the epoch's runs, and
  files shared with other in-flight work are flagged. The message is a subject,
  a body (outcome, deletion list result, gates, metric delta) and trailers that
  link the goal, epoch and runs through the work-reference contract.
- GCT shows the proposal queue. The operator reviews and approves each with one
  action. GCT stages and commits under authenticated human intent; agents never
  run git.
- For the operator's own commits, GCT drafts a message with trailers for any
  selected subset.

```gherkin
Given E27 is accepted
When the operator opens the GCT commit view
Then a proposal for E27 lists its files, an informative message and Vrooli work-reference trailers
And approving it creates one commit whose trailers resolve to the goal and epoch
```

### DL-12 Safe deploys from a shared tree

**Problem.** Restarting a shared service deploys everything in the working
tree, including other sessions' unfinished, rollout-held work. This happened
twice in one day (O28, O29). Installed CLIs can also be stale against their
real sources.

**Behavior.**
- Staged security or policy gates ship behind a switch that defaults off.
- Before a coordinator restart, a pre-deploy report lists the source changes
  since the running build (by fingerprint, without git) and the switches they
  add.
- Resource CLI freshness covers local `replace` targets
  (`CanonicalResourceGoModuleFreshnessSpec`), and Agent Manager checks that a
  resource CLI is current before it trusts its evidence.

```gherkin
Given another session has changed agent-manager source since the running build
When the coordinator prepares to restart agent-manager
Then the pre-deploy report lists those changed packages before the restart
```

### DL-11 Smaller, protected canon

**Problem.** The skills grew as rules were added, and more than one writer
changes them (O5, O12).

**Behavior.**
- After DL-1, DL-2 and DL-4 land, cut LEO and the supervision skill to the
  judgment that tools cannot hold. Target: LEO under 15 KB (23 KB now).
- Canon edits come only from the operator, the coordinating session, or a
  supervisor FEEDBACK route. A change record states who changed what.
  Prompt Manager or a lint may enforce this.

---

## 4. Approach

The work runs in four stages. The coordinating Claude Code session manages
subagents and owns this document and the tracker.

1. **Document** (this file). Done 2026-10-07.
2. **Research.** One read-only subagent per area maps current code, owners,
   tests and gaps, and proposes a design. Results go into §5. Areas:
   - Agent Manager effort and park mechanics (DL-1/2/3/4/8);
   - Prompt Manager heartbeats and teams (DL-2/5);
   - operator notification surfaces (DL-5);
   - GCT baseline and lifecycle (DL-7);
   - GCT commits, trailers and work references (DL-10);
   - the effort board (DL-9);
   - canary feasibility (DL-6).
3. **Implement.** Subagents work in dependency order, each on a disjoint set of
   packages, with focused tests per [docs/TESTING.md](../TESTING.md). Shared
   service restarts happen only when no delivery run is mid-turn.
4. **Validate.** Each workstream meets its Gherkin acceptance with a receipt. The
   canary (DL-6) and the live BAS goal then serve as the end-to-end check. The
   qualification table in [EFFORT_SUPERVISION.md](EFFORT_SUPERVISION.md) gains
   rows for the new behaviors.

Ordering (superseded by the wave plan below): DL-0 first. Then DL-1 + DL-4 (same Agent Manager area) and DL-5 in
parallel. Then DL-2 (needs DL-1). Then DL-3, DL-8 and DL-9. Then DL-6 (exercises
all of them). DL-7 and DL-10 run in parallel from the start because they touch
the control plane and GCT. DL-11 comes last.

---

### 4.1 Wave plan

The following rules applied to this historical wave plan; they are not a
standing restriction on current Agent Manager restarts. For the current
recovery-first restart contract, see [scenario configuration](../configuration/scenarios.md#restart-and-maintenance-of-an-execution-owner).
The wave plan used these rules:
- Proto changes go through one lane at a time.
- No agent restarts agent-manager, prompt-manager or BAS. The coordinator
  deploys those between waves, when no delivery run is mid-turn.
- Each agent owns a disjoint set of files.

| Wave | Track | Scope | Files owned |
|---|---|---|---|
| 1 | A | `goalhome` package, typed gates, `ACCEPTED` handling, lint, handoff set, park CLI policy, diminishing returns | `scenarios/agent-manager/api/internal/goalhome/` (new), `scenarios/agent-manager/cli/` effort commands |
| 1 | B | cli-core identity overlay; quiesce fix; park usage flush | `packages/cli-core/`, `scenarios/agent-manager/api/internal/orchestration/` (quiesce, park), api `main.go` timeouts |
| 1 | C | notification-hub Phase 0 and ask API | `scenarios/notification-hub/` |
| 1 | D1 | Control-plane lifecycle and baseline floor: restore-point scope, `.vrooli` config, restart order, serving-tree safety, log retention, freeze resolver, proto skip, test fixtures | `internal/lifecycle/`, `internal/baselinefloor/`, `internal/app/recovery/`, `cmd/vrooli/` |
| 1 | D2 | GCT baseline: `run_id` parse, `start` never overwrites an open engagement, `baseline cycle` with live probe | `scenarios/git-control-tower/` baseline code |
| 1 | E | GCT commit proposals S1 (proto lane holder in wave 1) | `scenarios/git-control-tower/` proposals and trailers, `packages/proto` GCT protos |
| 2 | F | Agent Manager proto-bearing work: board goal-home message, decisions raise/sync/answer, effort signal log, token caps (sequential in one agent) | agent-manager api and `packages/proto` agent-manager protos |
| 2 | G | Prompt Manager check-in dispatcher (polls signals, short-run rule) | `scenarios/prompt-manager/api/internal/heartbeat/` |
| 2 | P | Proposals never block acceptance: fold overlapping/drifted proposals into one, no open-proposal refusal, daily batched review notice (O30) | `scenarios/git-control-tower/` proposals, trailers, CLI, UI |
| 2 | J | DL-12: freshness from go.mod `replace` targets; pre-deploy change report without git | `packages/cli-core/`, `cmd/vrooli/`, `internal/lifecycle/` |
| 3 | H | Canary: fixture scenario, team, `effort canary-check` | new `scenarios/delivery-canary/`, teams, agent-manager cli |
| 3 | I | Canon: LEO and supervision skills use the commands; skill size cut; worker card; BAS migration | skills, team files, BAS goal home |
| 3 | V | Validation: Gherkin acceptance per workstream, canary green, BAS live consumer, EFFORT_SUPERVISION rows | receipts |

## 5. Research findings and designs

Full research notes (2026-10-07), with file:line citations, are in
`~/.vrooli/plan-artifacts/delivery-reliability/research/`:
- `am-effort.md` (DL-1/3/4/9);
- `am-lifecycle.md` (DL-2/8, quiesce);
- `operator-channel.md` (DL-5);
- `baselines.md` (DL-7);
- `commits.md` (DL-10);
- `canary.md` (DL-6).

This section keeps the decisions-relevant summary.

### 5.1 Agent Manager effort commands (DL-1, DL-3, DL-4, DL-9)

**Current state.**
- `epoch-check` runs entirely in the CLI (`cli/cmd_epoch_check.go`). It parses
  only header fields, Directives and the Slice log.
- It ignores `ACCEPTED`, so accepted epochs can still exit 3 and wake.
- Runtime totals are summed per log line, and workers log running totals
  (E23 summed −656, actual −328).
- Server park (`park.go`) has a 30-minute default, no maximum and no goal-state
  check.
- The board has a BAS row, but its `destination_ref` points to a deleted file.
  Discovery only scans `~/.vrooli/plan-artifacts/efforts`, and no code reads a
  goal home.

**Design.**
- One stdlib-only package, `api/internal/goalhome`, holds the parser and rules.
  The CLI imports it through a module require added via Scenario Dependency
  Analyzer.
- **`effort handoff set`:** atomic replace of the single `## Handoff`, at most
  4 KB. No-op when unchanged. Refuses if `STATUS.md` changed underneath.
- **`effort lint`:** separate from `epoch-check`, so goal-home size never fails a
  worker. Checks the 30 KB resume set, exactly one handoff, an admissible queue
  or a Needs operator item, and open operator feedback with no acknowledgement
  in the handoff (O13).
- **`effort park`:** allows ≤1 h with a live child, ≤72 h with a Needs operator
  item, and refuses otherwise. A server-side check in `park.go` follows, so raw
  `run park` cannot bypass it.
- **Typed exit gates:** `G<n> test|journey|inventory|review|custom: …`.
  Amendments are dated with a reason. `epoch-check` reports each gate as met or
  unmet; `--acceptance` exits 4 on an unmet gate. An `ACCEPTED` line carries
  `yield=` and `gates=`.
- **Diminishing returns:** a `Yield estimate:` header, `Destination`/`Current`
  lines in CHARTER.md, and a `## Censuses` section in STATUS.md. The finding fires
  on any of:
  - three epochs in a row under 25% of estimate;
  - two empty censuses in a row;
  - a **gap horizon**: the remaining gap needs more than about 30 epochs at the
    recent yield. This rule is added because the original rule would not have
    caught O3.
- **Board:** a goal home registers as enrollment `destination_ref =
  path:…/CHARTER.md`. The API reads it on its scan cycle into a typed
  `EffortGoalHome` message (missing facts read unknown). Worker runs link to
  the effort.

### 5.2 Run lifecycle, proposed check-ins, caps, quiesce (DL-2, DL-8, DL-7 part)

- **Identity (O21):** a one-line overlay in cli-core so self re-exec keeps
  `VROOLI_AGENT_IDENTITY_TOKEN`.
- **Effort signal return (interim; implementation complete, adoption pending):**
  Agent Manager owns a durable **effort signal log**: child ended, step-back, feedback, operator answer and
  cap stop. Its existing supervision tick can coalesce new signals into a
  neutral check-in for the exact admitted, parked orchestrator, using the
  existing run lifecycle and a per-effort/run durable cursor. The retained run
  rereads current canonical feedback; signal details do not become instructions
  or approval. This resumes the long-lived owner only. DL-2 fresh short-lived
  check-ins and the elapsed backstop remain unavailable.
- **Usage (O20):** emit or tail usage before a park ends the process. A phase 2
  tails the Codex session log so codec-pipe runs report usage mid-turn.
- **Historical token-cap implementation (DL-8):** the defaults and automatic
  enforcement below are superseded by the October 16 advisory-target correction.
  Retaining this implementation history does not authorize enabling stop caps.
  - `MaxWeightedTokens` on profile and run (a run can only lower it), enforced
    where usage events are already counted, modelled on the tool-call limit.
  - A nudge at 85%, then a stop at 100% with `stop_reason=token_cap` plus a
    signal.
  - Report-only for 3 epochs, then defaults: worker 1.0M, check-in 150k,
    supervisor daily 300k, audit 1.0M.
- **Quiesce:**
  - Runs declare their target instance, and the drain covers live plus
    undeclared runs. Exclusions are reported, not rejected.
  - Honest errors. A server-validated wait of at most 30 minutes, with a write
    timeout no shorter than the wait (today 3 min server vs 5 min client gives
    the EOF).
  - A scoped orchestrator permission replaces the 403 for run-token callers.

### 5.3 Operator channel (DL-5)

**Current state.**
- Nothing reaches the operator's phone.
- notification-hub "asks" are the only owner with options, deadlines, web push
  and durable state.
- notification-hub has never delivered anything (O19). Its PWA sign-in sends a
  retired identity header, and its CLI `ask` cannot pass options.
- An agent cannot raise an ask directly; the existing route for agents is a
  signed Vrooli Events event (as autoheal approvals do).

**Design.**
- **Phase 0:** repair notification-hub:
  - event-intake secret (`notification-hub/api/main.go:169`);
  - ask options flag;
  - expiry on read;
  - urgency;
  - requester notification;
  - an ask-answer page in the PWA;
  - owner sign-in: the Cloudflare Access login is the owner (P-19), with
    scenario-authenticator sign-in as a secondary path.

  Then the operator registers the phone once (about 5 minutes).
- **Phase 1:** `agent-manager effort decision raise` stores one record per
  `(goal, decision_id)` and publishes one event. The hub turns it into one ask
  (idempotent on event ID) and one push.
- **Answering:** the operator taps the push, then an option, recommended first.
  The CLI `effort decision answer` covers the desktop.
- **Wake:** an "ask resolved" event wakes, or signals, the orchestrator.
- **Sync:** `effort decision sync` writes the answer verbatim into FEEDBACK.md
  and clears the Needs operator item.
- **Defaults:** reversible decisions only, applied by the hub sweep at least
  12 h (default 24 h) after a *delivered* receipt. An undelivered question
  stays open as "operator not reached".
- **Board:** reads decisions from Agent Manager.

### 5.4 Baselines and promotion (DL-7)

**Ownership.** The control-plane floor (`internal/baselinefloor`) owns the
restore point, manifest and serving tree; `internal/app/recovery` sequences;
GCT orchestrates.

**Fixes.**
- **Freeze:** BAS's build reaches 60 repo directories, because the api replaces
  the whole repo module. A shared resolver `lifecycle.SharedSourceRoots`
  (`go list -deps`, pnpm `file:`, Vite aliases) feeds a capture-time copy of
  those sources plus generated outputs into the engagement (about 100–120 MB).
  The serving tree uses them, and capture fails closed (`--allow-unfrozen`
  overrides).
- **Proto:** skip `ProvisionGeneratedPackages(r.Root)` for serving-tree
  instances; it is the O9 mechanism.
- **Restore-point scope:** exclude runtime data and build outputs (O16). Copy
  the scenario's own `.vrooli` config files that builds read; today a BAS UI
  file read `.vrooli/service.json` (fixed in BAS source on 2026-10-07).
- **Lifecycle safety:**
  - `restart` builds before it stops (O14);
  - `abandon` restarts live after restoring;
  - `gc` refuses a live shadow split without `--force`;
  - `baseline start` never overwrites an open engagement's restore point (O15);
  - the safety-backup `run_id` parse is fixed (O17);
  - lifecycle logs are retained (O18).
- **Cadence:** `git-control-tower baseline cycle` promotes, runs a one-journey
  live probe before teardown, then re-creates the shadow without an anchor run.
  `epoch-check` reports `promotion_due` after N=3 accepted epochs with green
  journeys and no Test Genie run on live.
- **Tests:** stray `/tmp/.git` and `~/.git` break Go VCS stamping. Test fixtures
  set `-buildvcs=false`. 18 of the 23 lifecycle failures pass that way, and 5
  unrelated ones need their own fixes.

### 5.5 Commits (DL-10)

**Current state.**
- The `trailers` package has no consumer, and the advisory commit draft returns
  no trailers.
- The GCT UI "Continue" button turns "p16" into "p17", which is where the short
  subjects come from.
- LEO §4.4 already asks for "changed files and a suggested commit message" in
  prose.

**Design (S1, the minimum slice).**
- **Trailers v1:** dedicated keys `Vrooli-Effort`,
  `Vrooli-Epoch: <effort-ref>#E<n>`, `Vrooli-Run`, `Vrooli-Plan`,
  `Vrooli-Backlog`, `Vrooli-Continues`, `Vrooli-Proposal`, plus a generic
  `Vrooli-Work: <kind> <id>[@rev]`. Each key carries one serialized work
  reference, parsed with git's own trailer rules. Unknown keys are kept, and
  history is never rewritten.
- **Proposals domain in GCT:**
  - `proposal anchor` runs at admission and fingerprints files already dirty in
    scope.
  - `proposal create` runs at acceptance. It hashes each file and flags files
    that are mixed with older work, shared with another proposal,
    sandbox-pending or already staged. Excluded paths are recorded with a reason.
- **Approval:** one action in the UI queue or `proposal approve` through the
  existing human-intent path. A human-only `ApplyProposal`:
  - refuses on content drift, on a moved base touching the paths, or on other
    staged files;
  - stages exactly the paths and verifies the staged blobs;
  - commits, and restores the index on failure.
- **Operator drafting:** the advisory draft suggests trailers from open
  proposals, sandbox provenance and Continue.
- **Later slices:** S2 adds Agent Manager edit facts, resolvers and trailers in
  history views. S3 adds hunk-level proposals and committed provenance.
- **Reconciliation:** this implements GCT plan Phase 6 intent within delivery
  scope; the GCT plan keeps the wider advisory work.

### 5.6 Canary (DL-6)

**Design.**
- A Prompt Manager delivery team `delivery-canary` runs daily at 10:30 UTC on
  the real orchestrator profile and spawns a real interactive worker.
- A new `agent-manager effort canary-check` command asserts 18 steps from
  Agent Manager, Prompt Manager and GCT records, never from the agent's own log.
- A separate canary supervision team with a cheap Luna probe exercises the
  cross-team wake without waking Sol.
- The baseline step targets a new minimal `scenarios/delivery-canary` that
  copies BAS's build shape:
  - it always uses `--no-anchor --ttl 26h` and ends with `abandon`;
  - it never passes `--force`, `--no-drain`, `--replace` or `gc`.

**Reporting.** Results go to a 1 KB `STATUS.md` that the supervisor already
discovers, to one DL-5 item per new failure signature, and to a board column.

**Cost.** The original design estimated 80–135k weighted tokens per run. The
150k figure is now an advisory optimization target under the October 16 owner
correction; stopping caps remain off by default. Finish coherent work when a
target is exceeded and use actual outcome/cost evidence to improve efficiency.

**Expected reds on day one:**
- operator-only quiesce;
- no identity token for interactive runs;
- interactive tokens unknown.

## 6. Decisions

The operator approved the program and asked for minimal check-ins (P-01, P-02).
Research recommendations within that approval are adopted as defaults and
marked **default**; the operator may override any of them. Items marked
**operator action** need the operator.

| ID | Date | Decision | Source |
|---|---|---|---|
| P-01 | 2026-10-07 | Run this program first. It is the highest priority because it slows all delivery work. | Operator |
| P-02 | 2026-10-07 | Document first, then research, then implementation by managed subagents. | Operator |
| P-03 | 2026-10-07 | Include informative, trailer-linked commits; reconcile with existing GCT trailer and provenance work. | Operator |
| P-04 | 2026-10-07 | BAS: accept E26 under a gate amendment, end the line-count phase, re-aim at quality, relaunch the orchestrator, fix the live UI build. | Operator approved the fixes; re-aim is the default |
| P-05 | 2026-10-07 | Shared `goalhome` package imported by the CLI through an SDA module require; goal homes register via `destination_ref = path:…/CHARTER.md`; typed board message. | default (am-effort §5) |
| P-06 | 2026-10-07 | Strict park policy; 30 KB resume set; 4 KB handoff; `[ready]` markers; `## Censuses`; gap-horizon rule added to DL-3. | default |
| P-07 | 2026-10-07 | Proposed check-ins: Prompt Manager would consume the Agent Manager effort signal log, with effort-level lineage and codec-pipe usage reporting. | Future design (LC-1..3); unavailable until the PM consumer is implemented and qualified. Current delivery uses the long-lived parked orchestrator. |
| P-08 | 2026-10-07 | Historical design: report-only for 3 epochs, then role caps with a nudge at 85% and stop at 100%. **Superseded October 16:** role spend targets guide optimization; stopping caps stay off by default and coherent work may exceed a target. | superseded by OWNER-INTENT R02; not an instruction to enable stopping caps |
| P-09 | 2026-10-07 | Quiesce: undeclared runs count as live; 30-minute server-validated maximum; identity fix in cli-core. | default (LC-6..8) |
| P-10 | 2026-10-07 | Operator channel: web push from the notification-hub PWA; scenario-authenticator sign-in; reversible-only defaults at least 12 h after delivery (24 h default); the orchestrator's sync command writes FEEDBACK.md; a late answer supersedes a default. | default (OD-1..5) |
| P-11 | 2026-10-07 | Baselines: freeze fails closed; copy generated outputs at capture; exclude runtime data and build outputs from restore points; promotion every 3 accepted epochs with a live journey probe; declare the baseline cache as a storage root. | default |
| P-12 | 2026-10-07 | The orchestrator promotes through `baseline cycle`, guarded by green journeys, a live probe and auto-rollback. The operator promotes only on failure. | default (canary O-4). This is the most consequential default: agents restart live services. |
| P-13 | 2026-10-07 | Commits: dedicated trailer keys plus `Vrooli-Work`; informative subjects with series in `Vrooli-Continues`; mixed files included and flagged; refuse approval when other paths are staged; one approval per proposal; run IDs public; anchor at admission; the operator is always the author, with no agent `Co-Authored-By`. | default (OD1..8) |
| P-14 | 2026-10-07 | Canary: a dedicated `delivery-canary` fixture scenario; daily 10:30 UTC; advisory 150k target (the original stopping-cap default is superseded); push only on new signatures or stale status; build now, run by hand once, schedule after DL-2/5/7. | default (O-1..7) |
| P-15 | 2026-10-07 | Subagents must pass the operator rules on verbatim to any helper they spawn (O23). | Coordinator |
| P-17 | 2026-10-07 | Parked-turn usage counts as a lower bound until check-ins stop parking; quiesce keeps the no-parent rule; cap report mode starts with the orchestrator profile only. | default (track B) |
| P-16 | 2026-10-07 | Until an incident digest exists, the hub's incident subscription forwards only `incident.remediation_approval_requested.v1`, so the phone gets approvals and decisions, not about 1,400 incident events a day. | default (track C) |
| P-18 | 2026-10-07 | AUTH-01 caller enforcement is OFF at runtime by default (a single reversible switch; code kept). Unattended leader relaunch, scheduled heartbeats and the supervisor wake work as before. Re-enable it once the setup is more mature and AUTH-02 unattended authority exists. | Operator: "I don't think I want that security feature… at this stage it's too much friction" |
| P-19 | 2026-10-07 | notification-hub fully trusts the Cloudflare Access login as the owner, as vrooli-onboarding does. The operator needs no separate Vrooli password; the scenario-authenticator sign-in stays as a secondary path. Supersedes the sign-in part of P-10. | Operator |
| P-20 | 2026-10-08 | The delivery exact-effort gate is off by default (`VROOLI_DELIVERY_EFFORT_ENFORCE`), as for P-18. Re-enable it after moving `delivery-orchestrator` to medium or declaring high on the resource. | Coordinator, under P-18 |
| A-1 | open | Remove the stray `~/.git` and `/tmp/.git` (agent policy blocks deletion). | **operator action** |
| A-2 | done 2026-10-08 | Phone registered as a Home Screen app; a signed `decision_requested` test event was delivered to the lock screen. Three fixes were needed: the Settings button now reads the VAPID key at tap time and names the real failure (iPhone Safari tab vs. key load), and the VAPID contact falls back to the subscription's https origin because Apple answers a `@localhost` contact with 403 BadJwtToken. Push-provider rejections now record the provider's reason. | operator |

## 7. Tracker

| ID | Workstream | Status | Evidence / notes |
|---|---|---|---|
| DL-0 | Immediate BAS repairs | in progress | FB-064 recorded and CHARTER.md re-aimed (2026-10-07). Orchestrator relaunched as `4ea6c8a8`; it first followed the stale handoff (O13), so the handoff was replaced and the run woken. E26 ACCEPTED 14:36Z under FB-064; E27 (journeys J16/J19/J20, 12 → 15) admitted 14:36Z. UI build cause fixed in `ui/src/stores/aiCapabilityStore.ts` (serving-tree build passes, journeys 12/12). The live restart waits for wave 1 D1 (restart order, `.vrooli` in serving tree). |
| DL-1 | Loop rules in commands | A done 2026-10-07 | `api/internal/goalhome` (stdlib) plus CLI `effort lint`, `effort handoff set`, `effort park`, rewritten `epoch-check` (an accepted epoch never wakes; Workers-field parse bug fixed). The CLI go.mod require went through SDA. 21 package tests plus CLI tests pass. Live BAS lint caught a real open FB-063 not in the handoff. Adopted: idle-queue rule stays; duplicate handoffs are refused; the O13 rule is widened to supervisor entries. Wave 2: server-side park check (`goalhome.DecidePark` seam); Unicode-minus line counts. |
| DL-2 | Event-driven check-ins | Interim same-owner signal return implemented; adoption pending; full DL-2 unavailable | Agent Manager owns the durable effort signal log. Its existing supervision tick can wake the exact admitted parked orchestrator using a per-effort/run cursor. Fresh short-lived check-ins and the elapsed backstop remain unavailable. |
| DL-3 | Calibration and diminishing returns | A done (detection) | Three rules implemented. They need goal-home fields (`Yield estimate:`, GOAL `Destination`/`Current` lines, `## Censuses`), added by track I. The calibration spike lives in canon (I). |
| DL-4 | Fixed evidence menu | A done (mechanics) | Typed `G<n>` gates, dated amendments (`unverified` action), and `--acceptance` exit 4. Legacy gates warn. Canon conflict to fix in I: LEO §4.4 "no unacknowledged directive". |
| DL-5 | Operator channel with defaults | C code-complete without proto | Event intake uses the stored secret. Asks have options, recommendation, default, urgency and correlation. `*.decision_requested.v1` becomes one ask, idempotent on event ID. Reads have no side effects. The default applies only ≥12 h after a delivered receipt, otherwise "operator not reached". `ask.resolved` is published through an outbox. The CLI takes `--option`; the service worker deep-links pushes. Tests pass, including -race ×5. Live-checked 2026-10-07 after D1's fix: the bad signature gets 401; stored-secret signature 202; redelivery gives the same ask; a read does not expire; default withheld until delivered; `ask.resolved` published; a real vrooli-events decision event became one ask. P-16 applied (undo: PUT `track-c/event-config.before.json`). No device was notified. Proto follow-up done 2026-10-07 (full `make generate`, verify passes): labels, recommendation, default, urgency, note; GetAsk/ListAsks (recipient-only); same-origin sign-in through scenario-authenticator (HttpOnly cookies); PWA `/asks` list and `/asks/<id>` one-tap page; push re-registration. Live-checked except a real sign-in and answer, which need A-2. Leftover staging dir `packages/.proto-gen-verify-3630407007` to remove. A-2 done 2026-10-07: operator phone registered and a real notification received. Next: wave 2 F (Agent Manager `effort decision raise/sync`). Live-check script: `track-c/livecheck/run-livecheck.sh`. Wave 2 F done and deployed 2026-10-08 14:03Z: `effort decision raise|answer|list|sync` (one record per goal home and ID; one `agent_manager.effort.decision_requested.v1` event; a default needs `--reversible` and a deadline at least 12 h away; `ask.resolved` records the answer, appends an `operator_answer` signal and wakes the parked orchestrator; `sync` writes the answer verbatim into FEEDBACK.md). Subscription `agent-manager-decision-answers` is registered. End-to-end check passed 2026-10-08: test decision `e2e-test-1` was raised at 14:06Z and answered on the phone at 14:09Z. Within a second Agent Manager had recorded `answered via phone` and appended the `operator_answer` signal (seq 1). `sync` on the scratch goal home then wrote the answer verbatim to FEEDBACK.md and removed the Needs operator item. Limitation: a desktop `answer` does not close the phone ask. |
| DL-6 | Delivery-loop canary | wave 3 (H) | |
| DL-7 | Promotions, baselines, quiesce | wave 1 done (B, D1, D2) | D1 done 2026-10-07: the restore point dropped from 2.8 GB to 26 MB and now includes `.vrooli/service.json`; shared sources are frozen (105 MB, 107 roots) and fail closed (`--allow-unfrozen`); frozen instances skip proto provisioning; lifecycle logs are retained (last 10 per instance in `~/.vrooli/state/lifecycle-runs`); capture refuses an open engagement; GCT abandon restarts live before clean, gc refuses live-serving engagements without `--force`. Adopted: live-mode captures always allow unfrozen, and the reaper never uses `gc --force`. Live BAS refresh: `baseline cycle` with the health probe once the shadow journeys are green (E28's J05/J22 pending). Open: the storage-root declaration for the baseline cache. | B deployed 19:08Z (a 31 m wait is refused live): `--exclude-run` is reported not rejected; 30 m server maximum; no EOF (wait wrapper); a root orchestrator run of an enrolled effort may quiesce its scenario; runs may declare `scenario-instance` `<s>@shadow` with no proto change, and shadow runs are not drained. Identity re-exec fixed in cli-core (O21). D2 done 2026-10-07. `baseline start` refuses an open engagement before capture (O15). Safety-backup and drain parse accept `run_id`/`in_flight` (O17). New `baseline cycle` and `promote --probe-cmd`. An interrupted promote now resumes correctly. Focused tests pass and were shown to fail on the old code. Open: 3 trusted-base tests that are not hermetic (pre-existing). The capture-vs-check race needs `vrooli recovery capture` to refuse an open engagement (D1). The first real backup copy and rollback snapshot run on the canary before BAS. BAS TARGETS.md was updated to use `cycle` with `--name bas-goal`. |
| DL-8 | Token caps by role | B deployed 2026-10-07 19:08Z | Cap enforcement in the usage hook (Sol 10/Luna 1): nudge at 85%, `token_cap` stop at 100%. Configured by environment (`AGENT_MANAGER_TOKEN_CAP_MODE`, `AGENT_MANAGER_TOKEN_CAPS` per profile key). Report mode is adopted for `prompt-manager/delivery-orchestrator` at 150k. Parked-turn usage is now recovered from the Codex rollout as a lower bound (O20). Wave 2: profile/run cap fields, agent-visible nudge, atomic stop reason, `cap_stop` signal. F deployed 2026-10-08: `max_weighted_tokens` on profiles and runs (a run can only lower it); `stop_reason=token_cap` is written with the cancelled status; nudge at 85% in enforce mode. Mode is still env-controlled (default off). |
| DL-9 | Effort board from goal homes | wave 2 (F) started 2026-10-07 | F done and deployed 2026-10-08: `effort board` shows a typed goal-home block (unknown, never zero; cached for one scan interval). BAS registration is waiting on the operator (`track-f/bas-goal-home-registration.json`, `reconcile-metadata --local-owner`). The Efforts UI is not updated. |
| DL-10 | Informative, linked commits | E done, live 2026-10-07 | Trailers v1 (8 keys, git rules, agent `Co-Authored-By` refused). ProposalService plus human-only ApplyProposal (drift, base and index refusals; exact staging; operator author). CLI `proposal anchor/create/list/show/edit/approve/withdraw/refresh`. UI Proposals queue on desktop and mobile. Tests pass. LEO rev 28 calls anchor at admission and create at acceptance. Acceptance proof: the operator approves the first real epoch proposal. Open: 3 baseline CLI tests and 9 FileList/DiffViewer UI tests fail (pre-existing, another session). | Wave 2 P done and deployed 2026-10-08 (O30): the open-proposal ceiling is removed; `proposal create` folds open proposals of the effort that share paths or drifted into one combined proposal (one `Vrooli-Epoch` trailer per epoch, folded ones `superseded` with history); no proto change. Daily review digest `GCT_PROPOSAL_DIGEST_ENABLED` (default off) publishes `git_control_tower.proposals.waiting.v1`; it reaches the phone only once Notification Hub's event pattern includes `git_control_tower.**`. Follow-ups for the proto lane: `repeated superseded_ids`, `folds`/`folded_epochs` on Proposal; `proposal consolidate`/`digest` RPCs. BAS woken with FB-066 to create E33's proposal (folds E28–E32).
| DL-11 | Smaller, protected canon | wave 3 (I) | |
| DL-12 | Safe deploys from a shared tree | J done 2026-10-08 | Freshness follows go.mod `replace` targets for scenario and resource CLIs (installer, manifest check and runtime agree; about 10 ms per call). `vrooli scenario deploy-report <s>` and `restart --deploy-report` list changed source files since the running build (shared packages marked) and, for builds made from now on, env switches added or removed. Adopted: `restart --deploy-report` is the coordinator's standard deploy step. First live use: a Prompt Manager restart would ship 56 files, including another session's shared AUTH code (no `_ENFORCE` name). Open: Agent Manager checks a resource CLI is current before trusting its evidence. |

Known open defects that are not workstreams but must be closed or filed:
- Lifecycle start smoke tests in `internal/lifecycle`: 18 of 23 failures come
  from the stray `.git` directories (fixed in D1 fixtures). The other 5 need
  their own fixes: three owner-maintenance tests, repo script content and a
  stale proto fixture.
- Stray empty `~/.git` and `/tmp/.git` make Go builds under them fail without
  `-buildvcs=false`. The operator must remove them.
- BAS WORKAROUNDS: `agent-manager run identity --json` has no identity token
  after a CLI self-rebuild (O21). Fixed in wave 1 track B.
- notification-hub, GCT and control-plane defects found by research are listed
  in the research notes, §9 of each. Each is fixed by its wave-1 track or filed
  through `report-bug` at the end of the wave.
- About 1.7 GB of DL-0 scratch copies sit in the coordinating session's
  scratchpad. They are temporary and cleared with the session.
- The J11 shadow journey needs stub AI gateway injection.
