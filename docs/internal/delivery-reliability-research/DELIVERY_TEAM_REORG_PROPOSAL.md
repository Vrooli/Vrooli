# Delivery-team reorganization proposal

Status: IMPLEMENTED · 2026-10-23 recovery · nine product homes repaired; native parallel admission and timed handback deployed; real teams deliberately disabled.

**Current handoff: [§11 — consolidation](#11-consolidation-target--2026-10-23).**
[§10](#10-implementation-and-qualification--2026-10-23) retains the prior implementation receipts.
Sections 1–8 preserve the proposal and earlier execution notes. Section 9 is the pre-repair audit;
its open findings are superseded by the receipts and dispositions in §10.

**Decisions (operator-approved 2026-10-23):** root = `.vrooli/work/`; migration non-destructive
with tombstones; pressure supervisor extends effort-supervision; experiments/plans relocation
deferred. team.json **stays** in the central prompt-manager store (moving it would fork the
all-teams store) and links to its home via a new `homePath` field. Execution is sequenced:
**Phase 1** relocate (safe), **Phase 2** file-set modernization (control-plane rewrite),
**Phase 3** parallel workers + external pressure supervisor.

**Phase 1 execution state: DONE (2026-10-23).**
- Scaffold `.vrooli/work/` + READMEs; `.gitignore` rules; `homePath` on the Go `Team` struct and
  UI Zod schema.
- All 9 goal homes moved to `.vrooli/work/delivery/<id>/`; 6 enrolled efforts repointed via
  `ReconcileEffortMetadata` (owner-approved `effort-metadata` grant), 3 non-enrolled moved directly.
- All 9 `team.json` updated: internal path refs repointed + `homePath` added.
- Affected Go tests updated/green (teamcontract canonical-location guard rewritten to the new
  invariant); canonical docs (DELIVERY_RELIABILITY, packaging-strategies, external-knowledge-search,
  large-effort-orchestration SKILL) repointed. Research-tree evidence/sources left as historical.
- Enabler: fixed the web-console authorization binding bug (harness vs agent field) that had
  blocked agent self-authorization; wildcard authorization targets (`effort/*`) also added.
- Remaining: rebuild/install the `vrooli` CLI + agent-manager so the binding & wildcard fixes are
  permanently live (used a /tmp build + explicit targets this run). Then Phase 2/3.

This proposes **subtraction and relocation**, not new steering. It reorganizes how delivery-team
data is stored and how a team is pushed to an honest finish. It does **not** change the delivery
*model* (goal-based, orchestrator-driven, multiple parallel workers) — that model is kept intact
and deliberately distinct from Plan Manager's phased model.

---

## 1. Why delivery teams never get close to the target

Audited notification-hub + four other delivery teams (desktop-packaging, browser-automation-studio,
personal-planner, experience-quality). **All five show the same structural failure, none ever
shipped, all are disabled.** Root causes, in order of impact:

1. **The product goal was colonized by a measurement instrument.** notification-hub and
   desktop-packaging literally share `delivery-reliability-research/follow-through/BENCHMARK.md`
   as their plan-of-record — a file that says *"measures delivery capability, not visual asset
   production"* and spends ~130 of 143 lines on clock/anti-cheat accounting. The teams were
   commissioned as *trials of the delivery mechanism*; shipping the product was never the scoreboard.
   **A program must reference the teams it observes, never write into their homes.**

2. **Binary, all-or-nothing acceptance.** "Done" = a single gate stack (e.g. 10 gates, all must
   close) with no partial-credit gradient. 1-of-5 surfaces finished reads identically to zero.
   No signal rewards getting closer.

3. **Serialized single worker + micro-epochs.** Every team is `maxConcurrentRuns: 1`. Epochs are
   tiny (E15 = "a footer"; E16 = "one card's dark-surface color"). Per-chunk overhead (admission,
   independent review, evidence folders, handoff) dwarfs the work. It cannot converge.
   **The fix is parallel workers, which is also why the Plan Manager phase-list model is wrong here.**

4. **The target is kept out of the worker's view** ("avoid reinjecting the dossier every wake"),
   so the agent builds toward visuals it can't see.

5. **Infra time burns the product clock.** notification-hub's E16 can't even be admitted (host
   memory PSI 13.36% ≥ 10%), yet the benchmark counts that against the 24h budget.

6. **Rigid-then-toothless deadline.** Hard-fails at 24h (MISSED), then runs forever anyway —
   disciplines nothing.

### Storage is also convoluted (wider than delivery teams)

Three independent systems model the same shape — *target → work units → acceptance* — and never
reference each other:

| System | Store | Model | Used by delivery teams? |
|---|---|---|---|
| Plan Manager | git-ignored SQLite (`scenarios/plan-manager/data/plan-manager.db`) | plan → phases → acceptance | **No** |
| Effort + Goal-Home | git-**tracked** markdown in `scenarios/<x>/docs/internal/goal/` | effort → epochs → ACCEPTANCE.md | **Yes** |
| Swarm backlog | proto, `plan_ref` | item → acceptance_criteria | No |

The richest, queryable model (Plan Manager) is the one delivery teams bypass. Worse, a single
delivery team's data is smeared across **three locations with three git postures**:

- team **definition** → `scenarios/prompt-manager/store/teams/<id>/team.json` (git-tracked, nested in prompt-manager)
- team **working ledger** → `scenarios/<product>/docs/internal/goal/` (git-tracked, nested in the product scenario — operational markdown checked in as if it were source)
- effort **supervision** → `.vrooli/plan-artifacts/efforts/`, `.vrooli/agent-manager/` (git-ignored, central — the *right* place, but only partial)

A team that touches five scenarios still has to squat in one. The nesting is a fiction, and it
drags non-source files into product git. ("goal" is also overloaded 3×: goal-home ≠ harness goal
≠ goal-loop skill.)

---

## 2. Principles

- **Reference, don't colonize.** Programs (delivery-reliability, supervision) point at teams and
  read their `STATUS.md`. They never author a team's charter or home.
- **One central root for all work-drivers, grouped by type.** `.vrooli/` is the anchor (it already
  holds `plan-artifacts/efforts/`, `agent-manager/`, `state/`).
- **Standardize within each type, for that type's own model.** Delivery = goal-based + parallel.
  Do not borrow Plan Manager's phases; do borrow its *storage discipline*.
- **A team is one folder** — definition + home together — not split across two scenarios.
- **Honest gradient over binary gate.** Partial delivery must be legible at any stopping point.
- **External clock drives pressure.** The team cannot self-report its own pace.
- **Hybrid git:** durable, reviewable intent tracked; churny working state ignored.

---

## 3. Proposed layout

```
.vrooli/work/                        ◀── one place all work-drivers live
├── plans/                           phased, pre-planned   → Plan Manager's store (unchanged)
├── experiments/                     hypothesis / A-B       → its own model (unchanged)
└── delivery/                        goal-based, parallel   → delivery-team model (this proposal)
    └── notification-hub-delivery/   ONE folder = the whole team
        ├── team.json                identity, members, coordination, effortRef   [git-TRACKED]
        ├── CHARTER.md               what we deliver, target (links to assets),
        │                            acceptance INTENT, time budget               [git-TRACKED]
        ├── STATUS.md                the ONE scoreboard: parallel surfaces + clock [git-IGNORED]
        ├── LOG.md                   append-only work journal + evidence links     [git-IGNORED]
        ├── FEEDBACK.md              the one place the operator writes             [git-TRACKED]
        └── evidence/                captures, receipts                           [git-IGNORED]
```

`.gitignore` (hybrid): track `team.json`, `CHARTER.md`, `FEEDBACK.md`; ignore `STATUS.md`,
`LOG.md`, `evidence/`. Rationale mirrors how `.vrooli/` already splits config (tracked) vs state
(ignored). Charter/feedback carry durable, reviewable intent; status/log/evidence are churn.

> The exact root name (`.vrooli/work/` vs a dedicated sibling) is the one cosmetic choice left open;
> `.vrooli/` is recommended for precedent.

### File responsibilities (fixed files, free content within)

- **team.json** — slimmed. Identity, members, coordination pattern, `homePath`, and an
  `effortRef` pointer. **No charter prose** (that moves to CHARTER.md). Replaces today's
  `operatingContract.planOfRecord` + 6–9 `sharedState` pointers.
- **CHARTER.md** — operator-owned, rarely changes. The target (links to the design atlas/dossier
  by reference), the acceptance *intent* (what "good" means, not a 10-gate checklist), the
  anticipated time **budget**, and non-goals.
- **STATUS.md** — the single source of "how close + how long." A per-surface gradient plus the
  clock line (see §4/§5). The external supervisor reads this; the operator reads this.
- **LOG.md** — append-only journal replacing `epochs/` + `evidence/` handoffs + `archive/`.
- **FEEDBACK.md** — operator inbox.

Everything today split across `GOAL/QUEUE/TARGETS/ACCEPTANCE/CONTRACT/WORKAROUNDS/INSTALLS/COVERAGE`
folds into these as sections.

---

## 4. Parallel workers + the gradient scoreboard

The orchestrator fans out **multiple workers concurrently**, each owning a surface end-to-end
against its mockups. `STATUS.md` tracks them in flight at once:

```
Surface      Target     State                 Owner      Updated
─────────    ───────    ──────────────────    ───────    ────────
Needs you    NH13.x     ███████░░  close       worker-A   14:02
Updates      NH15.x     ██░░░░░░░  rough        worker-B   13:58   ← concurrent
History      NH16.x     █████████  accepted ✓   —          11:20
Delivery     NH17.x     ░░░░░░░░░  queued        —          —
Settings     NH18.x     ░░░░░░░░░  queued        —          —
─────────────────────────────────────────────────────────────────
Overall: 1/5 accepted, 1 close, 1 rough — clock 14h/24h — phase ON TRACK
```

States are a gradient (`queued → rough → close → accepted`), not a binary gate. Stopping at any
moment yields an honest, useful picture. This is the precondition for the pressure schedule.

---

## 5. The delivery pressure schedule (external clock)

Replaces the rigid-then-toothless deadline. The team pursues full acceptance while it has time;
as it runs long, an **external supervisor** progressively shifts it from *letter* → *intent* →
*honest stop + hand back to operator*.

```
elapsed / budget →  0%        75%        100%       150%       200%
                    │          │          │          │          │
phase:         ON TRACK ─▶ APPROACHING ▶ OVER ──▶ CONVERGE ─▶ HARD STOP
behavior:      pursue      stop gold-   intent      lock in     STOP. write
               full        plating;     over        done work;  honest terminal
               acceptance  triage by    letter;     no new      STATUS; hand to
               as written  value        drop nice-  scope;      operator for a
                                        to-haves    secure a    decision
                                        (logged)    coherent
                                                    partial
scoreboard:    the honest gradient — pressure NEVER edits it
```

Contract (the properties that keep it trustworthy):

- **Budget + clock live outside the team** (in `team.json` / the supervisor), not self-reported.
  Each wake, the supervisor computes the phase from wall-clock and **injects** it into the team's
  context. The team cannot fool itself; the operator sees the same phase.
- **Pressure changes priority and stopping behavior — never the truth of the scoreboard.**
  Loosening is always explicit and logged in LOG.md. The team may *stop chasing* low-value bars;
  it may never *claim* a bar it didn't meet.
- **Infra/blocked time is tracked separately** and may pause the clock (a host outage must not
  silently eat the product budget — the opposite of today's benchmark).
- **HARD STOP is a feature.** It produces a terminal honest STATUS and hands to the operator, who
  decides: good enough / re-commission with more budget / re-scope / change acceptance intent.
  This is the human decision point; it replaces silent forever-running.

Because the clock is external and the scoreboard is an honest gradient pressure can't touch, the
~130 lines of anti-cheat prose in today's benchmark become unnecessary.

---

## 6. What gets deleted (subtraction ledger)

- The benchmark-as-charter colonization (program references, doesn't author).
- 6–9 per-team working registers → 4 fixed files + evidence/.
- Scenario-nesting of operational markdown (→ central `.vrooli/work/delivery/`).
- `maxConcurrentRuns: 1` serialization → parallel workers.
- The binary gate stack → gradient scoreboard.
- The rigid 24h deadline + anti-cheat prose → external pressure schedule + honest stop.
- The split between team.json (prompt-manager) and goal-home (product scenario) → one folder.

---

## 7. notification-hub, before → after

| | Before | After |
|---|---|---|
| Charter | shared `BENCHMARK.md` ("measures capability, not product") | own `CHARTER.md`: deliver the 5 surfaces to the atlas |
| "Done" | 10 gates, all open, no gradient | per-surface `queued→rough→close→accepted` |
| Workers | 1 serialized, micro-epochs (E15 footer, E16 one card) | N parallel, surface-sized |
| Target visibility | dossier withheld to save context | mockups are the primary artifact the worker opens |
| Clock | 24h → MISSED → runs forever | budget + pressure phases → honest HARD STOP → operator decides |
| Storage | team.json in prompt-manager + goal-home in notification-hub + effort in .vrooli | one folder: `.vrooli/work/delivery/notification-hub-delivery/` |

---

## 8. Open questions for the operator

1. Confirm root name: `.vrooli/work/` (recommended) vs a dedicated top-level sibling.
2. Migration: move existing (disabled) teams folder-by-folder, non-destructively? (Proposed: yes;
   leave a tombstone pointer at the old `scenarios/<x>/docs/internal/goal/` for one cycle.)
3. Does the external pressure supervisor live in the existing effort-supervision layer, or a new
   thin owner? (Proposed: extend effort-supervision — it already reads goal-homes and runs the board.)
4. Should experiments relocate under `.vrooli/work/experiments/` in the same pass, or later?

---

## 9. Verified status and remaining work — 2026-10-23

This audit answers the operator's request to recover the previous conversation and identify all
remaining work. It inspected the current working tree, the Plan Manager record, the nine migrated
homes, live effort enrollments and selected live boards. It changed this status document only;
it did not repair implementation, activate teams, wake retained runs, or restart scenarios.

### Verified current state

| Area | Observed state | What that establishes |
|---|---|---|
| Central delivery homes | All nine have CHARTER/STATUS/LOG/FEEDBACK; previous files survive under archive/v1 | Physical relocation and conversion happened |
| Team linkage | Nine team records have homePath; six live enrollments point into the new root | Registry/enrollment location plumbing exists; enrollment filenames still say GOAL.md |
| v2 support | Parser, converter, board projection, proto fields and CLI changes exist; live RCL board demonstrates v2 reading | Substantial Phase-2 code is implemented and in use |
| Progress gradient | Zero of nine STATUS files has a Scoreboard section | The capability exists in code but has not been commissioned for the actual teams |
| Time budget | Zero of nine CHARTER files has a Budget field | No team has the new pressure-clock input populated |
| Execution policy | Nine of nine migrated teams are disabled and serialized with maxConcurrentRuns=1 | Disabled state is intentional; parallel delivery is not configured |
| Clock supervisor | budget_seconds is parsed/projected; no delivery pressure classifier, schedule or hard-stop controller was found in the owning code | Phase 3 remains implementation work |
| Git posture | STATUS, LOG and a representative evidence path are not ignored for any migrated home | The promised hybrid tracking policy is not implemented |
| Formal plan | phase-2-delivery-home-format-rewrite is DRAFT, all seven phases TODO; no execution row for its plan ID | The earlier chat's completion claims were not reconciled into Plan Manager |

Plan ID: `03484695-2a4f-4183-8a79-b2bb7e501011`. Read it with
`plan-manager plans get phase-2-delivery-home-format-rewrite --json`.
The absence of execution rows was checked through a read-only SQLite query after inspecting
the CLI's available read surfaces. No plan state was changed by this audit.

The registry choice is deliberate in the later implementation notes: team.json remains in Prompt
Manager's common team store and homePath links to the delivery home. Update the older “one folder
including team.json” claims to describe this choice. Plans and experiments were recorded as
deferred in the local proposal; their stores were not relocated.

### Defects and incomplete migration work

1. **Unit identity changes during reads.** `goalhome/home_v2.go` assigns every parsed unit a new
   sequential Number rather than preserving its ID. At 21:04 UTC, the live RCL board reported
   activeUnit=E4 and lastAcceptedUnit=E3, while its actual STATUS says active E1 and LOG's last
   accepted entry is E5. This is a live correctness defect, not a hypothetical risk. Preserve
   stable IDs through parsing, projection, reopen handling and attribution. The migration
   comparator in convert.go does not compare these IDs, so its equivalence result misses this.

2. **Migration verification is incomplete.** It compares selected parsed facts, not every
   promised behavior or the persisted enrollment/document references. Park-policy equivalence
   is not compared. `TestConvertHomeIdempotentShape` compares projectFacts(v2) with itself, which
   cannot establish idempotence. Replace that assertion with a real repeated-operation check
   and verify the actual migrated homes, including IDs, workers, acceptance/reopen records,
   park decisions and unknown/partial behavior. Preserve pre-existing malformed worker data
   as unknown unless it can be reconciled against owner evidence.

3. **Conversion is not recoverable after every partial failure.** applyMigration writes three
   files individually and then archives the old files. A retry declares success whenever
   CHARTER exists, without checking the remaining files or archive completion. Make publication
   and recovery coherent before relying on the converter as retained tooling. No interrupted
   conversion was induced during this audit; this failure path is established by source review.

4. **References were not fully migrated.** Eight of nine team definitions contain missing
   repo-root paths, including required shared-state files. Update operatingContract documents,
   sharedState and member write paths. Update HEARTBEAT/RESPONSIBILITIES and supervisor guidance,
   not just team.json: Notification Hub's HEARTBEAT still directs the agent to the old scenario
   directory and GOAL/QUEUE/WORKAROUNDS files. large-effort-supervision still reads old registers.
   Repair relative links that broke when homes left product scenarios. Reconcile the six
   enrollment destinationRef values from the GOAL.md sentinel to CHARTER.md through the owner
   API; the current adapter makes them readable but they name files that no longer exist.

5. **Archiving did not consolidate every live responsibility.** The converter moves TARGETS,
   INSTALLS and ACCEPTANCE, where present, into archive/v1 without merging their operative
   contents into the live files. Notification Hub's archived TARGETS has 75 lines; its current
   charter has seven. Recover applicable validation targets, acceptance intent, install records
   and active briefs into the agreed live sections or explicit authoritative references. Existing
   STATUS text still names epochs/ files that were moved into archive/v1/epochs/. Distinguish
   historical evidence from the working briefs the current orchestration skill still requires.

6. **The product-first charter change is unfinished.** Notification Hub's CHARTER and team
   heartbeat still make BENCHMARK governing, keep the expired fixed clock and require all old
   gates before closure. Its STATUS still says one serialized worker at a time. Give each team
   a clear product target, visible design references and acceptance intent; let delivery research
   reference the team without governing its product home. Populate an evidence-backed surface
   gradient. An absent assessment must remain explicitly unknown/unassessed; conversion alone
   cannot justify declaring a surface queued, accepted or complete.

7. **Resume friction remains.** Current `agent-manager effort lint` fails seven of nine homes:
   BAS, desktop packaging, Nooch, Notification Hub, Planner and RCL exceed the 30,000-byte resume
   budget; marketing has an idle-queue finding. Planner also has ten unacknowledged-feedback
   findings. Experience quality and external knowledge pass. Reconcile outstanding obligations
   and compact resolved history without deleting unresolved feedback or treating acknowledgement
   as resolution. Do not clear a preparation-only team's idle finding by dispatching it.

8. **Hybrid git remains open.** Add the intended ignore rules for mutable status/log/evidence
   while preserving durable intent/feedback and useful evidence manifests. Inventory archives
   and bespoke retained files instead of silently adding the relocated operational tree to git.
   The central work root, proposal/spec, and new v2 parser/converter files remain untracked in
   this checkout. Review and commit the coherent effort without sweeping up unrelated changes.

### Phase-7 disposition needs an explicit resolution

The previous handoff called v1 removal “unsatisfiable.” That conclusion is too strong. Keeping a
reusable importer requires legacy parsing somewhere; it does not require a permanent v1 runtime
board-reader or unchanged dual-format CLI writers. Conversion can also finish before retiring
the converter. The current code retains both runtime reading paths and selects writer filenames
according to the detected format, so “v1 is import-only” does not describe its full behavior.

Inventory actual remaining consumers, then either remove obsolete runtime support as authored,
or isolate a justified importer and record an explicit amendment to the spec/plan. Do not mark
the existing phase complete by changing comments. The plan/spec still demand one parsing path
and net reduction; current goalhome Go sources including tests total 4,794 lines versus 3,515 at
HEAD (+1,279). This is a checkout comparison, not attribution of every line to this conversation.

### Behavioral work still required after migration repair

1. **Bounded parallel delivery.** Reuse the existing bounded-parallel infrastructure. Establish
   the actual concurrency boundary for heartbeat/team runs versus Agent Manager child workers;
   changing maxConcurrentRuns alone does not prove child parallelism. Preserve one orchestrator,
   allow independent surface workers, account for leases/file conflicts/host admission, and
   support multiple active work units in STATUS and the board. The current parseActive keeps
   one unit, taking the first value for each field. Update the orchestration/worker/supervision
   contract and acceptance accounting together. Prove overlapping independent worker execution.

2. **Authoritative delivery clock.** Define commissioning start, budget authority, deliberate
   disable/pause/resume behavior, blocked-time accounting and restart durability. Avoid starting
   a new commission against an already expired historical benchmark clock. The current budget
   is only a parsed charter field; there is no separate authoritative clock lifecycle yet.

3. **Pressure and honest stopping.** Extend effort supervision with ON TRACK / APPROACHING /
   OVER / CONVERGE / HARD STOP at the proposed 0/75/100/150/200 percent thresholds. Expose the
   same computed signal to owner and workers. Change priorities explicitly and log scope
   dispositions; preserve acceptance truth and human-only decisions. Enforce hard stop through
   execution/admission lifecycle, checkpoint active work, publish accepted/partial/unverified
   outcomes and remaining work, and prevent the heartbeat from relaunching the stopped commission.
   Return the decision through the existing operator channel: accept the partial result,
   recommission, or change scope. A board label or prose nudge alone does not meet this outcome.

4. **Qualification and controlled adoption.** Run focused regressions for migration defects and
   the new timing/concurrency behaviors, then the relevant Test Genie phases for changed
   interfaces. Qualify a bounded disposable commission covering partial delivery, deadline
   transitions, restart recovery, blocked time, independent acceptance and terminal handback.
   Keep the real delivery teams disabled until deliberate activation; reconcile retained parked
   owners before recommissioning so old instructions do not silently resume.

5. **Close the records honestly.** Amend contradictory proposal/spec text, reconcile the formal
   plan and its actual validation evidence, update canonical skills/docs/CLI help, and leave one
   current handoff. Decide separately whether plans/experiments need further reorganization;
   this delivery repair does not require relocating a working managed database.

### Notification Hub specifically

At 21:02 UTC the live board read the new home and still reported 10 open whole-redesign gates
with a current measurement dated 2026-10-21. LOG has no accepted units. The retained orchestrator
`11175250-c50d-44e9-9af5-cd6eba79bab2` is PARKED; disabled team state did not remove that retained
run. These are current owner/ledger observations, not a fresh visual audit of the product.

After the framework repair, refresh the product assessment against its actual served build and
selected atlas. The existing handoff still names E16's History dark-surface remainder and
unestimated Updates/Delivery/Settings/Needs-you work. Reconcile the disposable-recipient journey
through supported isolation/fixture owners and preserve genuine human decisions. The recorded
host-pressure refusal is from Oct 21; this audit did not repeat product build admission, so it
does not establish today's host blocker or today's visual state. Historical green checks and
the earlier seven-phase completion claim do not accept the redesign.

### Validation performed for this audit

- `cd packages/proto && make verify-committed-gen` — passed.
- Agent Manager API: `go test ./internal/goalhome ./internal/supervision -count=1` — passed.
- Agent Manager CLI: `go test ./... -count=1` — passed across its packages.
- Read-only inspection of all nine homes and team records; lint of all nine; git-ignore checks;
  live enrollment list; exact Notification Hub, BAS and RCL boards; exact Hub retained run; formal
  plan and empty plan log; read-only execution-row lookup.

No comprehensive certification or new product/E2E acceptance is claimed. The passing focused
suites coexist with the live identity defect above and therefore do not establish migration
completion. Resume with migration correctness and reference repair, then implement clock and
parallelism against the clarified product contract.


## 10. Implementation and qualification — 2026-10-23

The operator authorized all recovery work from §9. The delivery infrastructure and source handoff
are now implemented. This is not acceptance of the nine products, and no real delivery team was
enabled, commissioned or deliberately woken.

### Current organization and behavior

| Owner | Authoritative responsibility |
|---|---|
| Prompt Manager team registry | One team identity and member contract; homePath links to its product home |
| `.vrooli/work/delivery/<team-id>/CHARTER.md` | Product destination, selected design/requirements, acceptance intent, constraints and proposed budget |
| STATUS / LOG / FEEDBACK | Honest surface gradient and active units / stable accepted-unit history / operator directions |
| Agent Manager effort enrollment | Existing home destination plus operator-commissioned clock, worker limit, lifecycle, terminal snapshot and decision |
| Delivery reliability research | Observes outcomes; does not replace the product charter |
| Plan Manager and experiment owners | Keep their managed stores; no duplicate writable store was introduced under `.vrooli/work/` |

All nine homes now have product-first charters, explicit unassessed surface scoreboards and a
proposed 24-hour budget. Their definitions use bounded-parallel member queues. Agent Manager
separately admits one native orchestrator and the commissioned independent-worker limit (prepared
default: two), including commissioned descendants. Multiple active units retain distinct authored
IDs. The registry stays common to all teams; it is not duplicated into each home.

Native active-time pressure is ON TRACK before 75%, APPROACHING at 75%, OVER at 100%, CONVERGE
at 150%, and HARD STOP at 200%. Start/pause/resume/stop are operator transitions through
`agent-manager effort delivery`. A paused clock requires a reason and parked/terminal owners.
Disabling heartbeat alone does not freeze active execution. Service restarts preserve elapsed and
paused time. At the hard deadline, new admissions and first execution of queued product work stop.
Retained owners receive a checkpoint opportunity of two minutes from actual owner observation,
including delayed observation after restart. Remaining commissioned runs are stopped through the
native controller; terminal readback precedes the frozen outcome and operator decision. Elapsed
time never accepts product work. Unfinished commissions cannot be withdrawn or silently restarted.

### Audit findings closed

- Stable unit IDs and all active-worker sets survive parsing, conversion, reopening, projection and
  attribution. The live RCL owner now reports active E1 and last accepted E5, matching its files.
- Runtime boards and ordinary CLI operations use v2 only. Legacy parsing is retained solely for
  explicit import and verification. Missing or unsafe evidence remains partial/unknown.
- Import publication has a durable recovery snapshot, exclusive file publication, operator-edit
  checks and non-destructive archive collision handling. Repeated import does not duplicate LOG.
  Existing authored epoch briefs remain usable; operative retained targets, installs and acceptance
  are explicitly linked. All nine actual homes preserve their prior bytes and unresolved duties.
- Team documents, member reading/write routes, archived navigation and current source manifests
  were reconciled. Removed historical code locators are explicitly historical-missing. Resolved
  feedback moved to linked verbatim history; unresolved directions remain visible.
- All nine lint checks pass. Every raw CHARTER+STATUS+FEEDBACK resume set is below 30,000 bytes.
  Hybrid git policy ignores STATUS, LOG, operational evidence and temporary publication files;
  durable intent, feedback, acceptance sources and source maps remain eligible for tracking.
- All six enrolled destinationRefs were changed from deleted GOAL.md sentinels to existing CHARTER
  files through ordinary revision-checked ReconcileEffortMetadata, then read back. Every other
  enrollment field was compared and preserved. Three preparation-only homes remain unenrolled.
- The operator approved consent `0900e1c4-97eb-4903-88bc-b32c63550c87` for those six exact metadata
  targets. A handler bug initially discarded separate paired operation proofs. The regression was
  reproduced and fixed using the canonical evaluator, without widening the grant or ordinary
  identity scopes; the same approved grant then completed all six updates.
- homePath now traverses the store, API response and mutation DTOs, generated Team/TeamDetails/
  TeamInput contracts, CLI JSON/human reads and explicit `--home-path` changes/clears. Readback of
  the disabled Notification Hub team displays its current home. Canonical orchestration and
  supervision skills, generated sidecars, heartbeat section vocabulary and current docs agree.

### Evidence and practical limits

Plan: `phase-2-delivery-home-format-rewrite` (`03484695-2a4f-4183-8a79-b2bb7e501011`).
Execution: `ce330872-3cfe-4280-b35d-4edbc253667e`. All ten phases are recorded done;
`exec complete` returned the canonical `COMPLETENESS_FULL` handoff under the advisory policy.
Durable receipts and their SHA-256 manifest are under
`/home/matthalloran8/.vrooli/plan-artifacts/phase-2-delivery-home-format-rewrite/recovery-20261023/`.

- Governed protobuf generation and the final `make verify-committed-gen` passed. No generated
  output was hand-edited. The Prompt Manager final artifact is
  `d83e7664855cb110f66cc41d6dc89b0af1920fb1d2c977e6c4ec6449c5b280fc`.
- Focused Agent Manager goalhome, supervision, decision, orchestration, wiring, handler and
  contract suites passed. Both full scenario CLI suites passed. Prompt Manager heartbeat, team
  contract/config/store, typed team CRUD, member declarations and skill corpus checks passed.
- A disposable native in-process runner test proved overlapping execution of two independent
  workers under one orchestrator, with a third refused before persistence. Controlled-owner
  tests proved thresholds, pause/restart/resume, stale-owner rejection, late checkpointing,
  queued-work fencing, withdrawal protection, hard stop, terminal readback, unchanged scoreboard
  and an honest pending operator decision. These use no external model/provider work.
- Agent Manager Test Genie unit recheck `20261023-215224-ef7bc847` passed. Its first scoped run
  `20261023-214413-1f2e8d44` passed dependencies and retained an API scanner finding at tasks.go:246:
  the flagged 200 is the successful creator-orchestrator fallback after all error exits. Existing
  authority regressions passed; the scanner issue was filed through the canonical report-bug skill.
- Prompt Manager `20261023-214417-70c2f4df` passed API and dependencies. Current unit recheck
  `20261023-215810-28a2e085` has no Go execution failure, but its UI workspace reports eight failed
  files and 20 failed tests (2,470 passed) in editor/tree/world suites outside this change. That
  limitation is retained in Plan Manager and Scenario QA. No whole-scenario green or comprehensive
  certification is claimed, and attribution to another contributor was not inferred.
- Independent receiving/code review found no material remaining source or lifecycle issues,
  including the paired-grant and typed-home-path delta. This is source review, not product acceptance.
- Both owning services were adopted through supported lifecycle restarts and returned healthy.
  The source checkout contains unrelated changes; no blanket stage or git commit was performed.

### Notification Hub and the next operator action

The selected manifest still contains 240 originals, 102 task compositions and 71 state IDs. Its
byte/hash/dimension/decode verification and durable commissioning source map were retained.
Delivery detail is NH16.4–NH16.6; NH17 answer/default/late-answer states were not mislabelled as
Delivery. Original human decisions, unresolved NH-DEC-26 and selected acceptance remain operative.

A fresh read-only observation found the Hub healthy at serving build
`sha256:e158fa32f4affd5fe13b632cca295a47efa5e6d3c372de7e2b45d89c30773ee3`.
BAS captures `5e29a7cf-2e1f-4d9d-9d20-f5a5ded95645` (desktop) and
`9ca46d45-651a-4932-9ff3-a6b7a2c6d743` (phone) show the ordinary empty Needs you route. Phone comparison
with selected NH13.4 confirms content intent but still shows surface-color, spacing, button and
navigation differences. No surface was accepted. Populated/response journeys and the other four
destinations need product qualification and delivery. The Oct 21 memory-pressure refusal is a
historical observation; current build admission was not repeated or declared blocked.

All real delivery teams remain disabled and uncommissioned. Before deliberate activation, read the
chosen charter and open feedback, reconcile retained native ownership (parked is not terminal),
qualify current product routes/fixtures, and commission the chosen budget and worker limit. Then
enable only the intended team. Existing human-only approvals and product acceptance decisions
remain intact. The infrastructure repair is complete; shipping each selected product is its next
explicit commission, not an implicit consequence of this reorganization.

## 11. Consolidation target — 2026-10-23

Operator-authorized follow-through: account for every finite delivery team, remove unused
legacy runtime code, consolidate parsing and test fixtures, correct current documentation,
and expose the native product scoreboard and commission in the existing Efforts UI.
Keep retained evidence, importer recovery, independent acceptance and genuine human decisions.
Teams remain disabled; this maintenance does not commission or wake product work.

Work shape: bounded task. Work until the current entry point explains each owner and work
model, all finite teams have an explicit home/disposition, obsolete runtime paths and
duplicated tests are removed, and the Efforts view renders existing owner facts without
inventing acceptance or zero-valued unknowns. Prove this through focused goalhome/CLI/UI
regressions, team contract checks, scoped Test Genie phases and two final source reviews.
Boundary: Agent Manager goalhome/effort presentation and tests, Prompt Manager delivery-team
declarations and canonical orchestration guidance, existing delivery docs and navigation.
Validation is targeted; preserve unrelated findings. Blocked means a missing decision,
credential or approval, not diagnosable implementation friction. Checkpoint changed, verified,
remaining and unverified work here before an interruption. Do not relocate managed plans or
experiments, weaken required behavior, discard unique tests, reset clocks or enable teams.

### Consolidation result

Implemented the authorized consolidation. The migration cohort is accounted for in the
work entry point: nine central product homes, six retained managed owners with explicit
existing README links, and three archived records (the prior launch plus two disposable
fixtures). No team was enabled, no native commission started, and retained runs were not
retired by source metadata. The unreturned historical fixture outcome remains unknown.

Removed the unused legacy tolerant reader, its unreachable tolerance branches, the runtime
board forwarding wrapper and constant filename adapter. Legacy import is explicitly named
and isolated. Import verification and runtime reads share parsing/file accounting; tests
share fixture editing. Overlapping converter assertions and timestamp-based unit lookup
were simplified without removing interrupted/repeated import, attribution, reopening,
partial-evidence or concurrency protection. V2 resume accounting counts STATUS workarounds
once, with exact-boundary regressions. New Efforts UI displays current surfaces, independent
units, native pressure and frozen handback without another store or clock; partial owner
rows retain uncertainty.

Current charters explicitly supersede historical scheduling/benchmark authority while
preserving product checks and human decisions. Hub duplicate feedback was consolidated only
after verifying both original entries in its verbatim archive. A unique ten-file historical
repark packet was relocated from Agent Manager docs to its existing research owner, with
byte hashes and original Git revision/path recovery in its source map. Current docs point
at one work entry point; the old relocation recipe is historical. Git policy excludes
operational receipts and imported visual bundles while retaining intent/source maps.

Validation: focused goalhome/supervision, full Agent Manager CLI, Prompt Manager
teamcontract/teamconfig/store, 18 Efforts UI cases and TypeScript pass. Test Genie
`20261024-011539-50913eb0` passed unit/structure; Prompt Manager
`20261024-011538-a9ea00e4` passed structure. Their broader docs findings are retained;
the later Agent Manager docs recheck removed four broken trimmed-fixture links and the
misplaced historical packet was subsequently relocated. Owner-reference revision,
retrieval, preservation and reference verification passed with no new scoped findings.
Agent Manager was adopted through its supported restart; the served Efforts chunk contains
the current uncertainty handling and scoreboard. Two final source reviews checked runtime
versus import boundaries, owner/scheduling precedence, attribution/unknowns, retained bytes
and the Git boundary; no material scoped issue remains.

Limits: no unattended real product commission or product acceptance was performed. An
exact managed-team RPC read was denied by the current agent grant; source contracts and
FileTeamStore's per-read disk loading were verified without changing identity or authority.
The canonical orchestration skill was synced. It is not a selected native base-pack skill,
so native projection refresh correctly refused it; registry source remains authoritative.
Evidence, hashes and source/test counts: `~/.vrooli/plan-artifacts/delivery-consolidation-20261023/`.

Final preservation review: goalhome production changed from 3,596 to 3,574 lines (-22),
and its tests from 1,284 to 1,252 (-32), including the new budget-boundary cases. New UI
behavior/tests are a separate feature cost. Git attributes preserve imported delivery
source bytes; all 490 staged delivery files were checked byte-for-byte against disk,
including original CRLF sources. Historical whitespace is retained rather than silently
rewriting evidence. The scoped commit includes the remaining Desktop home deletions and
migrated team/member declarations, so a checkout does not restore old homes or serialized
contracts alongside the new structure. Unrelated working-tree edits remain separate.
The post-relocation Agent Manager docs run `20261024-013357-37daaad1` retains five broader
absolute-path errors; the current owner guide adds none. Scenario QA receipts:
`knw-1792805778075771777` (Prompt Manager manifest) and `knw-1792805778914052524`
(documented Knowledge Observatory append refusal). No full-scenario docs green is claimed.

Commit prerequisite: the repository-wide hygiene hook initially refused 105 findings,
primarily empty capture directories plus a missing ownership comment in the existing
`internal/safeguards/graphical-session/handler.go`. Safe directory cleanup uses the
control-plane `vrooli hygiene --fix-safe --contract-only` owner. The package received
only its missing boundary comment; its unrelated untracked implementation is not staged
with this delivery change. The hook remains enabled and its result is retained.

### Git preservation status

The implementation and scoped validation are complete; a Git commit has not yet been
created. The first hook refusal was reduced through the supported safe hygiene fixer
(138 empty directories removed) and the package ownership comment. The remaining whole-
checkout dependency check is driven by unrelated root go.mod/go.sum changes and includes
captured baseline modules and a disposable RCL fixture as live surfaces. SDA's exact
freshness report and current hook output are retained with the reviewed patch. Do not
mutate frozen evidence, repair unrelated module graphs wholesale, or bypass the hook to
manufacture commit success. The scoped staged changes and full patch remain reviewable;
no unrelated staging was included and no real team was enabled.

Final code follow-through: commissioning now shares the existing authorization and response
renderer directly, eliminating temporary request files, recursive CLI parsing and duplicate
local-owner eligibility checks. The complete CLI suite retains all 494 passing test events,
including exact owner revision, caller-clock exclusion and identified-agent refusal. No new
mock framework was introduced. The repository contract now explicitly permits `work`, and
its compatibility guide distinguishes the approved hybrid authoring home from native runtime
state. Only that declaration is staged from the otherwise modified root contract.

The second commit attempt remained refused by seven whole-checkout findings, including
runtime-home/configuration drift, captured manifest/schema identities, a root lockfile/UI
boundary and dependency freshness. Its report predates the corrected work declaration.
SDA and contract-inventory evidence bugs were filed; no gate was bypassed. Structure Health's
stale CLI/API checksums were repaired only through SDA's two exact module freshness commands,
and the owner was normally restarted healthy. These prerequisite metadata changes remain
separate from this delivery patch. Frozen source modules were not tidied. The exact staged
patch and its hash are retained for review; Git commit remains pending the wider repository
hygiene repair. Current source/CLI checks do not claim whole-project contract certification.

Final post-refactor Test Genie unit run `20261024-021507-13b36949` passed (172 seconds).
The owner still reports two scenario-wide missing-injectable-seam warnings and 77 advisory
coverage findings; this consolidation is not a claim that all Agent Manager test debt is
removed. The latest contract-only hygiene call found no empty directories or missing
package documentation, but its Structure Health provider timed out again; a warning-only
aggregate is not a successful contract verification. The previous full commit rejection
and its concrete contract/dependency findings remain the preservation limit.
