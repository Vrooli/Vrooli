---
name: "offer-desk"
description: "Read the offer board as the monetization team's single address, choose between catalog, gate and posture operations, and keep unreadable actuals visible instead of estimated."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  status: "active"
  revision: 1
  createdAt: "2026-09-08T00:00:00Z"
  updatedAt: "2026-09-08T00:00:00Z"
  requires:
    scenarios: ["offer-desk", "money-ledger", "program-runtime", "vrooli-memory"]
    commands: ["offer-desk offers", "program-runtime library run", "vrooli-memory learning"]
  learning:
    scope: "offer-desk-usage"
    capture: "every attempt"
  origin:
    kind: "authored"
---
## Tools focus: Offer Desk

Offer Desk is the monetization team's declared instrument: the offer graph, its promotion
gates, and the financial posture beside them. Reading it answers *what is the state of the
offers I own, and what should I do next*. It never answers *is this worth doing* — that
denominator is authored outside the board, and a board that set its own target would be
confirming itself.

Every command sits under the `offers` group: `offer-desk offers <command>`.

### Before acting

Recall prior advice for this scope before the first operation:
`program-runtime library run vrooli-memory.prepare-attempt --input scope=offer-desk-usage --input task_id=<id> --input operation=<operation> --input context_key=<key>`. **[S3]**
No match is a real answer; proceed without inventing advice. Memory mechanics belong to
`prompt-manager skill read vrooli-memory` and are not restated here.

### Choosing the operation

| Task | Step |
|---|---|
| Learn the current team state, or what changed since last time | Run `program-runtime library run offer-desk.board-read --input baseline=<prior snapshot>`. Store the returned `snapshot` in your continuity record; it is the next run's baseline. **[S3]** |
| Decide what ships next and what blocks it | Run `program-runtime library run offer-desk.release-path`. Add `--input stream_node_id=<id>` only when you need one stream's prerequisite tree. **[S3]** |
| Find the rows that warrant attention this cycle | Run `program-runtime library run offer-desk.attention-triage --input max_items=<n>`. One batched call labels a deterministic shortlist; it never labels the whole board. **[S3]** |
| Read this instrument's own condition rows | Run `program-runtime library run offer-desk.setpoint-read`. **[S3]** |
| Read whether past attempts in this scope are comparable | Run `program-runtime library run offer-desk.learning-read`. **[S3]** |
| Inspect one node's identity or edges | Run `offer-desk offers catalog-list` or `offer-desk offers catalog-edges`. **[S1]** |
| Reconcile declared catalog sources against live records | Run `offer-desk offers catalog-verify --source-path <path>`. Slow; see Troubleshooting. **[S1]** |
| Inspect meter vocabulary and graph conformance | Run `offer-desk offers meters`. **[S1]** |
| Inspect the monetization obligation cells | Run `offer-desk offers space`. **[S1]** |
| Record an observed fact against a trigger | Run `offer-desk offers gates-fact --name <n> --value <v> --stale-after-days <d>`. **[S1]** |
| Evaluate candidate triggers after facts change | Run `offer-desk offers gates-evaluate`. Takes no flags. **[S1]** |
| Propose a promotion the operator will decide | Run `offer-desk offers gates-promote --node-id <id> --actor <id> --role <role>`. **[S1]** |
| Make a node's earnings readable | Run `offer-desk offers catalog-map-account --node-id <id> --actor <id> --reason <why>`; `--account-id` is optional. **[S1]** |
| Improve this instrument, its programs or its skills | Read `prompt-manager skill read offer-desk-improve`. **[S0]** |

Prefer a program over a raw read when one exists. `board-read`, `release-path` and
`setpoint-read` each replace three to four manual reads plus a join an agent gets wrong
differently each time.

### What the readings mean

**The board owns ranking.** `release_rank` and `rank_reason` are the board's. Reorder your
own presentation if you must; never recompute a rank and never present a derived order as
the board's.

**An unreadable actual is not a zero.** A board entry carrying
`availability: [{source: money-ledger.actuals, reason: "no ledger account mapping"}]` means
money-ledger holds no account for that node, so its earnings are *unknown*. Report it as
unknown and name `catalog-map-account` as the fix. Never substitute an estimate, and never
read the absence of revenue as no revenue. As of 2026-09-08 this is the majority condition
on the board, so the correct posture is to say so once, not to re-derive it per node.

**An empty response is a real zero.** Protojson omits a field at its type's zero, so
`gates-proposals` returns `{}` when no proposal is open. That is zero open proposals, not a
failed read. Treat a genuinely failed read differently: it carries a reason.

**Deduplicate before proposing.** Check the board and the decline history in
`gates-proposals` for the same node before raising a promotion or a work item. A previously
declined proposal needs new evidence, not a resubmission.

### Writes

Thirteen of the twenty-three commands write. Before any of them:

- **Attribution is per-command, not universal.** `--actor` is required on `catalog-import`,
  `catalog-map-account`, `catalog-merge` and `gates-promote`, and those four also want a
  `--reason` or `--role`. The gate-recording paths (`gates-fact`, `gates-evaluate`) take no
  actor at all. Read `--help` for the command you are about to run rather than assuming the
  shape; do not borrow another member's identity where one is required.
- **Rehearse before applying.** `catalog-import` rehearses unless `--apply` is passed, and
  `catalog-merge` offers `--dry-run`. Read the rehearsal output before applying; an audited
  merge cannot be undone by re-running it.
- **A write is not a decision.** Promotion, retirement and release rank are operator calls.
  Record the evidence and propose; do not transition a node because the trigger fired.

### After acting, always

Capture the attempt whether it succeeded or not:
`program-runtime library run vrooli-memory.finish-attempt --input scope=offer-desk-usage --input attempt=<attempt>`. **[S3]**
Entry kinds are `task-record` and `attempt-observation`. Capture failure is separate from
task outcome: a failed capture does not make a successful operation a failure, and it is
reported rather than retried in place. Pin advice after a third confirmation; supersede it
when it fails; propose a rule only after the pattern repeats.

### What this scenario cannot answer yet

Four questions have no command behind them, so do not hunt for one: *what did we decide about
this node and why*, *which facts are stale*, *which triggers are unmet*, and *how long has
this waited*. The data exists and is correct — `catalog_audit`, `facts` and `triggers` are
written faithfully and read by nothing. Report the limit and move on; the register and the
wanted read surface are in `scenarios/offer-desk/docs/internal/PROBLEMS.md`, and the repair is
routed by `prompt-manager skill read offer-desk-improve` §5.

If you hit a *fifth* such question, that observation is the valuable output of your run. It
costs one typed entry and requires no decision — the Observe exit in
`path:docs/agent-system/TARGET_MODEL.md` §8. Add it to the ledger rather than working around
it silently, and do not file a work item for it unless a decision is genuinely needed.

### In-use settings

| Symptom | Move |
|---|---|
| The board diff covers fewer rows than the board holds | `board-read` reports `truncated: true` and sets `partial`. Raise the contract's `materialize_limit`; do not read the partial diff as complete. Journal the new bound. |
| Triage costs more than it returns | Lower `--input max_items`. The shortlist stays deterministic; only the labelled subset shrinks. Journal the value and why. |
| A ladder read includes retired deliverables you do not want | Pass `--input include_retired=false` (the default) to `release-path`. |
| Learning reads report `comparable: false` | Expected until the scope has eligible attempts. Do not present an empty cohort as a measured comparison; keep capturing. |

### Troubleshooting & Edge Cases

| Situation | Response |
|---|---|
| `catalog-verify` does not return | Observed exceeding 90 s on 2026-09-08, which is why `setpoint-read` reports `catalog-conformance` as `kernel_invoke_budget` rather than calling it. Run it directly with a generous timeout, once, outside any loop. |
| `rows must be one of: <none>` | The response carried no repeated field because it is empty. Read it as zero, not as drift. Any other `ambiguous_response` text is real drift: probe the binding and fix the contract. |
| `release-prerequisites` returns `sql: no rows in result set` | The stream node id does not exist. `release-path` degrades to `partial` and still reports the ladder join; correct the id rather than dropping the read. |
| The instrument is unavailable | Record the reason and the age of your last good reading in the continuity record, fall back to your lane's declared durable evidence, and keep conclusions qualified. Never silently skip the board. |
| A reading disagrees with a Command Center revenue tile | Both can be true; they measure different things. As of 2026-09-08 the tiles are UNTRUSTED on a producer contract mismatch (`revenue-summary.v1` against an expected `legacy.v1`). Report the disagreement; do not reconcile it by picking one. |
| You want a target for a row | Targets are authored outside this scenario. An instrument that authors its own denominator is deviation D6 in `docs/agent-system/TARGET_MODEL.md`. Raise the gap; do not fill it here. |

Promotion note: the `[S1]` leaves above are single commands whose `--help` already carries
their flags, so this skill names the decision and not the flag list. `catalog-map-account`
is the strongest Action candidate — one command, deterministic, and currently the highest-
volume unmet need on the board — but it is not wrapped yet, so it stays a leaf.
