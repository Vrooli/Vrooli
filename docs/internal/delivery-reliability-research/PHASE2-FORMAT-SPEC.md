# Phase 2 — delivery-home format spec (full rewrite)

Status: IMPLEMENTED · operator authorized the audit's remaining work on 2026-10-23.
The recovery contract below supersedes the original one-cycle bridge/removal assumptions.
Current operation: [.vrooli/work](../../../.vrooli/work/README.md) and the
[Agent Manager owner reference](../../../scenarios/agent-manager/docs/reference/effort-supervision.md).
This document retains migration rationale and field mapping, not a second operating manual.

Phase 2 replaces the goal-home file set (`GOAL.md` / `QUEUE.md` / `ACCEPTANCE`-in-epochs /
`epochs/E*.md` / `WORKAROUNDS.md` / `INSTALLS.md` / `TARGETS.md`) with a modern set
(`CHARTER.md`, `STATUS.md`, `LOG.md`, `FEEDBACK.md`). The hard constraint: the current files
carry **load-bearing** data the agent-manager supervisor depends on — run→effort attribution,
token-spend/cap accounting, acceptance+yield, active-work selection, park-policy inputs, and lint
gates. This spec maps **every** such field to the new format so nothing is lost. (Source of the
inventory: `effort_goalhome.go`, `goalhome/{home,legacy,home_v2,epoch,decision,park}.go`.)

## This is refactoring + debt reduction, not just a file reorg

The end state must be **cleaner and more maintainable than today**, not v1+v2 coexisting. Explicit
acceptance principles for every phase:
- **Net-simpler.** The rewrite should *remove* more than it adds where possible: one parsing path,
  not two; fewer bespoke per-file parsers; shared markdown helpers reused, never forked. Remove unused runtime branches and repeated assembly; retain only the explicit importer.
  Report production/test changes separately from new lifecycle and UI features.
- **One runtime format.** Boards and ordinary CLI writers use CHARTER/STATUS/LOG/FEEDBACK.
  Retain legacy parsing only for explicit import and its fixtures. A missing runtime charter
  reports migration required; it must not silently select the legacy writer.
- **Consolidate, don't duplicate.** Collapse the scattered concepts (GOAL/QUEUE/ACCEPTANCE/epochs/
  WORKAROUNDS/INSTALLS/TARGETS registers → CHARTER/STATUS/LOG/FEEDBACK). Remove dead code, unused
  fields, and now-pointless indirection the old epoch model required.
- **Cite deletions.** Each phase names the symbols/files/registers it removed or merged, not just
  LOC — the same bar the reorg proposal set for itself.
Legacy import code must have an actual import caller. New runtime features can add code; report
their size separately from migration cleanup rather than treating fewer lines as product proof.

## Design principle

The "epoch" was two things fused: a **progress unit** (sequential work) and an **attribution +
acceptance record** (workers, yield, gates). Phase 2 splits them:

- **STATUS.md** = the human-facing **gradient scoreboard**: per-surface progress + the active
  work + clock/phase + park-policy inputs (needs-operator / handoff / next). A *view*.
- **LOG.md** = the append-only **structured ledger of accepted units**: each unit keeps its id,
  workers (run UUIDs), acceptance timestamp, yield, gates. This is where attribution + spend +
  acceptance live — the epoch's load-bearing half, preserved verbatim-equivalent.
- **CHARTER.md** = target metric, acceptance intent, **time budget** (Phase-3 input), non-goals.
- **FEEDBACK.md** = unchanged (operator/supervisor inbox).

STATUS is derived/maintained next to LOG; LOG is the authority for anything billing-adjacent.

## File-by-file schema

### CHARTER.md  (replaces GOAL.md target role; operator-owned)
```
# <effort title>
- Destination: <metric> <op> <target>        # was GOAL.md "- Destination:"
- Budget: <duration>                          # NEW (Phase-3 pressure clock); optional
- Acceptance intent: <prose>                  # NEW; what "done" means, not a gate stack
- Non-goals: <prose>
- Design: <links to atlas/dossier by ref>
```
Reader extracts `destination_metric/op/target` from `- Destination:` (same regex as today), and
`budget_seconds` from `- Budget:` (new proto field).

### STATUS.md  (the gradient scoreboard + active work + park inputs)
```
## Scoreboard                                 # per-surface gradient
| surface | target | state | owner |          # state ∈ unassessed|queued|rough|close|accepted
| History | NH16.x | accepted | - |
| Settings| NH18.x | rough  | <run-uuid> |    # owner = active run UUID for that surface

## Current: <value> @ <ISO>                    # was GOAL.md "- Current:"
## Active                                      # replaces active-epoch selection
- unit: <unit-id>  status: <text>  workers: <uuid[,uuid]>   # workers → active_workers (cap usage)

## Needs operator                              # was QUEUE.md ## Needs operator (verbatim move)
## Handoff                                     # was QUEUE.md ## Handoff (park policy reads this)
## Next                                        # was QUEUE.md ## Next ([ready] items; lint idle-queue)
## Workarounds                                 # was WORKAROUNDS.md open entries (open_workarounds)
```
`active_workers` comes from `## Active` `workers:`. Park policy reads `## Handoff` / `## Needs
operator` / `## Next` from STATUS (same semantics as today's Queue object).

### LOG.md  (append-only; structured accepted units = attribution + acceptance authority)
```
## Accepted units                             # newest-last; each line is one accepted unit
- <unit-id> | ACCEPTED <ISO> | workers=<uuid[,uuid]> | yield=<±n> <unit> | gates=G1,G2 | <summary>
## Reopened                                   # unit-id + ISO; reopen if ts > its ACCEPTED ts
```
This preserves exactly what the spend/acceptance code needs:
- `last_accepted_*` = last `## Accepted units` line not superseded by a `## Reopened` entry.
- `weighted_tokens_per_accepted_epoch` → `..._per_accepted_unit`: average over the **last 5**
  accepted units, summing `WeightedRunTokens(workers)` — identical algorithm, new field name.
- reopening logic = a `## Reopened` entry with ts after the unit's ACCEPTED ts (was epoch Directives).

### FEEDBACK.md — unchanged.

## Load-bearing field map (old → new) — nothing dropped

| Old (file → pattern) | Proto field | New home |
|---|---|---|
| GOAL `- Destination:` | destination_metric/op/target | CHARTER `- Destination:` |
| GOAL `- Current:` | current_value/as_of | STATUS `## Current:` |
| epoch `Status:` (priority select) | active_epoch/active_status | STATUS `## Active` unit/status |
| epoch `Workers:` (active) | **active_workers** (cap usage) | STATUS `## Active` `workers:` |
| epoch ACCEPTED ts/yield/gates | last_accepted_epoch/at/yield | LOG `## Accepted units` line |
| accepted epochs' `Workers:` (last 5) | **weighted_tokens_per_accepted_unit**, accepted_units_measured | LOG accepted-unit `workers=` |
| epoch Directives > ACCEPTED ts | (reopen gate) | LOG `## Reopened` |
| QUEUE `## Needs operator` | needs_operator | STATUS `## Needs operator` |
| QUEUE `## Handoff` (park) | (park policy) | STATUS `## Handoff` |
| QUEUE `## Next` `[ready]` | (lint idle-queue) | STATUS `## Next` |
| WORKAROUNDS open entries | open_workarounds | STATUS `## Workarounds` |
| FEEDBACK open `### <id>` | open_feedback_ids | FEEDBACK (unchanged) |
| NEW | budget_seconds, surface_scoreboard[] | CHARTER `- Budget:`, STATUS `## Scoreboard` |

## Proto EffortGoalHome deltas
- Rename `..._epoch` → `..._unit` where it means a work unit (keep wire-compat by adding new
  fields and retaining old as deprecated for one release, OR hard-cut since enrollments are
  data-only and re-read each board — decide in the plan).
- Add `budget_seconds`, repeated `SurfaceProgress { surface, target, state, owner_run }`.

### D1 RESOLVED (2026-10-23): hard cut, no deprecation window
At the original rename, `EffortGoalHome` was recomputed by the board and had no
persisted consumer. That justified the hard cut without compatibility aliases.
Native delivery now persists a frozen `EffortGoalHome` in `EffortDelivery.outcome`;
future schema changes must account for those retained terminal snapshots. Live board
facts remain projections. The current unit field names were introduced before these
snapshots and need no legacy epoch aliases. Applied in `effort.proto`:
`active_epoch→active_unit` (9), `last_accepted_epoch→last_accepted_unit` (12),
`weighted_tokens_per_accepted_epoch→weighted_tokens_per_accepted_unit` (19),
`accepted_epochs_measured→accepted_units_measured` (20). Added `budget_seconds` (25) and repeated
`SurfaceProgress surface_scoreboard` (26, new message). Regenerated via the governed owner
(`cd packages/proto && make generate SCENARIO=agent-manager`); `make verify-committed-gen` clean;
api+cli build and the goalhome/supervision/cli suites green. **Population of `budget_seconds` and
`surface_scoreboard` is deferred to Phase 3** (the v2 reader switch), which is where the v2 facts
become available; Phase 2 only plumbs the proto surface.

## Lint gates — re-expressed, not dropped
`handoff-count`/`resume-budget`/`idle-queue`/`feedback-unacknowledged` re-point at STATUS sections
(`## Handoff`, `## Next`, `## Needs operator`) and FEEDBACK. Same thresholds; new anchors.

## Migration of the 9 existing homes
A one-time converter reads each old home and writes the new files:
- GOAL `- Destination:` → CHARTER; GOAL `- Current:` → STATUS `## Current`.
- Each `epochs/E*.md` ACCEPTED line → one LOG `## Accepted units` line (id, workers, yield, gates);
  reopening directives → `## Reopened`. Non-accepted active epoch → STATUS `## Active`.
- QUEUE sections → STATUS sections; WORKAROUNDS open → STATUS `## Workarounds`.
- `archive/`, `evidence/`, slice-logs → `LOG.md` prose appendix / retained under `evidence/`.
Converter must be **idempotent** and **lossless for attribution** (every accepted unit's worker
UUIDs preserved, or the home is flagged `partial` exactly as today on parse failure).

## Phased plan (ordered; each phase independently testable)
1. **New parser** `goalhome/v2` reading CHARTER/STATUS/LOG → same internal `Home`/`Goal`/facts
   structs the board already consumes (so the proto/board layer is untouched first). Unit tests
   with new testdata fixtures mirroring `testdata/bas`.
2. **Proto** add new fields (budget, surface scoreboard, `_unit` names); regen via governed owner.
3. **Reader switch** `effort_goalhome.go` to v2 only. Keep legacy reading solely in the explicit importer; unimported runtime homes are partial and cannot be written.
4. **CLI** repoint `effort decision/feedback/park/lint/handoff` writers at STATUS/LOG; update tests.
5. **Converter** + migrate the 9 homes; verify each board reads identical attribution/spend/accepted
   facts before vs after (golden diff).
6. **Canon skill** `large-effort-orchestration` rewritten to author the new files; worker contract.
7. **Isolate legacy import** after migration. Runtime boards and writers use v2 only; importer recovery remains tested.

Acceptance: preserve authored identity, UUID attribution, acceptance/reopen records, park decisions and explicit unknowns. Correct known renumbering and unknown-as-zero defects. Product-first charter/scoreboard edits intentionally change benchmark summaries; they must preserve every operative acceptance obligation and never manufacture acceptance. Converter equivalence is tested against legacy parsed facts, not against itself.

## Recovery and delivery lifecycle contract (operator-approved scope, 2026-10-23)

- Preserve authored work-unit IDs. Multiple independent active units remain distinct; the board
  exposes every unit and its workers. The single active-unit summary remains a selected view.
- Publish imported files with a recoverable transaction marker. Validate all required files before
  reporting completion. Retry an interrupted import without overwriting operator edits. Merge
  operative targets, installations and acceptance references; retain old bytes as history.
- Keep team identity in Prompt Manager's common registry. The delivery home holds product intent
  and working records. homePath and enrollment destinationRef name that home and its CHARTER.
- The product charter and governing product/design sources define acceptance. Research observes
  the result. A scoreboard state is unassessed, queued, rough, close or accepted, supported by
  evidence. Unassessed is explicit; migration never manufactures acceptance.
- Agent Manager owns a persisted delivery commission in the existing effort enrollment. The
  operator commissions a new ID, time budget and worker limit through the existing owner enrollment
  operation. Metadata reconciliation cannot change that policy or its clock. No commission starts
  merely because files were migrated, a team was enabled, or a historical benchmark expired.
- The commission stores start, pause, accumulated paused time and terminal disposition. An operator
  can pause for deliberate disablement or a named infrastructure hold, then resume with accounted
  time. Blocked time is visible; an agent cannot silently stop or reset the clock. Recommissioning
  uses a new ID after old owners have become terminal; it does not erase prior results.
- Elapsed active time determines ON TRACK (<75%), APPROACHING (75%), OVER (100%), CONVERGE (150%)
  and HARD STOP (200%). The same owner computation appears on the board and in execution context.
  Priorities change; acceptance truth, safety controls and human decisions do not.
- Native admission allows one orchestrator and a bounded number of independent workers. It rejects
  paused, ended or expired commissions and conflicts with retained live ownership. Global resource,
  qualification, permission, file/lease and host admission remain in force.
- At HARD STOP, freeze admission and queued product starts, allow two minutes from actual owner observation for a checkpoint (including after restart), and then stop
  remaining owned runs through the existing lifecycle controller. Persist an honest terminal
  handback and raise the existing operator decision with accept-partial/recommission/rescope choices.
  A stopped commission cannot be relaunched by heartbeat. It is never marked product-accepted by time.
- Real teams stay disabled during qualification. Use a disposable commission with an injected
  clock and controlled run owner to prove concurrency, deadlines, restarts, pauses and handback.

Harness handoff: Work until this contract and proposal §9 are implemented and verified. Repair
the existing plan and record phase evidence. Use focused regressions and scoped Test Genie phases.
Extend scope with an owner log when needed. Blocked means a decision, credential, or approval you
lack; diagnose engineering friction. Preserve shared-worktree edits and disabled real teams.
Checkpoint in Plan Manager and the proposal on interruption; report changed, verified, remaining,
and unverified outcomes. Do not weaken acceptance or claim product delivery from infrastructure tests.

Implementation and validation receipts: proposal §10 and the Plan Manager execution `ce330872-3cfe-4280-b35d-4edbc253667e`. An unfinished commission must finalize before withdrawal; enforcement also covers retained withdrawn data. Separate paired operation proofs use the canonical authorization evaluator without merging identity scopes.
