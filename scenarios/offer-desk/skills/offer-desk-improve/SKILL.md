---
name: "offer-desk-improve"
description: "Regulate Offer Desk against its condition rows — ledger-account coverage, posture readability, meter and obligation coverage — without grading an unreadable actual as a zero."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  status: "active"
  revision: 1
  createdAt: "2026-09-08T00:00:00Z"
  updatedAt: "2026-09-08T00:00:00Z"
  requires:
    scenarios: ["offer-desk", "money-ledger", "program-runtime", "vrooli-memory", "measures-health"]
    commands: ["offer-desk offers", "program-runtime library run", "measures-health validate scenario", "vrooli-memory learning"]
  origin:
    kind: "authored"
---
## Practice focus: Offer Desk Improvement

### 1. Focus and scope

The plant is Offer Desk's own readability: whether the monetization team can learn the state
of the offer graph, its promotion gates and the financial posture beside them from one
address. Three scenarios declare Offer Desk as a dependency (`command-center`,
`compute-manager`, `scenario-to-plugin`) and one team reads it as its instrument, which is
what makes this role owed.

This skill never authors a monetization target, never transitions a catalog node, and never
edits money-ledger. An instrument that authored its own denominator would be deviation D6 in
`path:docs/agent-system/TARGET_MODEL.md`.

### 2. Setpoint

Read every row with `program-runtime library run offer-desk.setpoint-read`. The board's
`ok` means it reported its evidence and its declared gaps — not that the product is healthy.
Copy `reading` and `reason` verbatim into the "today" column; never print a number for a row
the board declined to evaluate.

| Row | Target | Today (2026-09-08) |
|---|---|---|
| `ledger-account-coverage` | Every ACTIVE or TRIGGER_MET node maps to a ledger account | 68 nodes, 2 mapped; 3 earning nodes, 2 mapped — out of band |
| `default-alive-posture` | A readable default-alive gap | `unavailable: revenue and burn observations are incomplete` — unreliable, `in_band` null |
| `meter-coverage` | No deliverable meter gaps and no undeclared streams | 4 meters, 2 deliverable meter gaps, 0 undeclared streams — out of band |
| `obligation-confidence` | Denominator confidence above `sketch` | 3 cells at `sketch` — out of band |
| `learning` | A comparable cohort exists | 0 eligible attempts, `unreliable:no_eligible_attempts` — expected while the scope is new |
| `external-friction` | — | `read_elsewhere:agent-manager.friction-digest` |
| `catalog-conformance` | Declared catalog sources reconcile against live records | `kernel_invoke_budget` — `catalog-verify` exceeded 90 s on 2026-09-08 |
| `promotion-latency` | No promotion proposal waits on the operator beyond one review cycle | `pending_telemetry` — no sensor exists |

The last three rows carry permanent reasons and do not lower the board's status. They are
still unmet obligations; a cycle that reports `ok` has not closed them.

### 3. Sensors

| Sensor | Command | Validity rule |
|---|---|---|
| Board coverage and posture | `offer-desk offers board-show` | An entry's `availability` entry is the board's own statement that money-ledger holds no account. It is unknown, never zero. |
| Meter conformance | `offer-desk offers meters` | `deliverable_meter_gaps` and `undeclared_streams` are counts; protojson omits an empty list, so absence is zero. |
| Obligation denominator | `offer-desk offers space` | `denominator_confidence` is operator-authored. `sketch` is an honest low-confidence reading, not a defect to edit away. |
| Attempt learning | `program-runtime library run offer-desk.learning-read` | vrooli-memory owns the reliability verdict. An empty scope is comparable false, never a measured result. |
| Measure coverage | `measures-health validate scenario offer-desk` | Local maturity L1 with 5 blocking domain-coverage findings as of 2026-09-08. |
| Cross-scenario friction | `program-runtime library run agent-manager.friction-digest` | Applicable diagnostic, not a universal gate; it may hold no sample for this owner. |

### 4. Golden corpora

`scenarios/offer-desk/evals/` does not exist, so no suite and no approved floor governs this
scenario today. Do not synthesise one to have a number: an unestablished floor is an open
target decision, recorded here and routed under §5, and it is separate from sensor validity.
Until a suite exists, no cycle may claim a corpus result.

### 5. Routes

| Reading | Route | Expected evidence change |
|---|---|---|
| `ledger-account-coverage` out of band | `catalog-map-account` is a curation move the scenario already exposes; run it in-cycle for nodes whose account is known. An unknown account is a money-ledger question, routed to its owner. | Mapped count rises; the board's `availability` entry for that node disappears |
| `default-alive-posture` unreliable | money-ledger owns revenue and burn observations. Report the incomplete-observation reason to that owner; do not compute a posture here. | The board returns a readable gap |
| `meter-coverage` out of band | The gap is a declaration in the owning scenario's `.vrooli/monetization.json`; route to that scenario's owner through `scenario-work-ladder`. | `deliverable_meter_gaps` falls |
| `obligation-confidence` at `sketch` | The space contract itself says confidence stays at sketch until infra-health and the monetization team complete one observed reconciliation cycle. Route the reconciliation; do not raise the confidence value. | `denominator_confidence` rises with a stated rationale |
| `catalog-conformance` at `kernel_invoke_budget` | A performance defect in `catalog-verify`. Select the layer with `scenario-work-ladder`; repair within the mandate, otherwise propose authorized follow-up. | The row becomes readable inside the board's budget |
| `promotion-latency` at `pending_telemetry` | `measures-adoption` plus the owning implementation method. | A sensor exists and the row carries a reading |
| `learning` comparable false with attempts present | Inspect capture, not the board. A capture failure is separate from task outcome. | Eligible attempts rise and the cohort becomes comparable |

Implement-versus-propose follows §5 of `improve-skill-authoring`: a curation move the
scenario already exposes is done in-cycle; anything needing a target amendment, another
scenario's code, or authority this cycle does not hold is reported with its prerequisite.

**Routes are also sourced from the problems ledger, not only from board rows.** Read
`scenarios/offer-desk/docs/internal/PROBLEMS.md` every cycle. A board row needs a target and
a sensor to exist; a *wanted capability* has neither, so it can never appear on
`setpoint-read` and would be invisible to a cycle that reads only the board. The ledger is
where it lives. As of 2026-09-08 it carries three entries that no sensor row can express:

| Ledger entry | What is wanted | Why it is not a board row |
|---|---|---|
| persisted decision history has no read path | `catalog-audit`, `gates-facts`, `gates-triggers` reads, then measures over five tables | The data exists and is correct; nothing reads it. There is no sensor to be out of band |
| the requirement matrix is complete and unproven | an `evals/` suite with an approved floor, then a requirements-sync snapshot | A missing corpus is an absent target, not an unmeasured one |
| catalog-verify exceeds a usable budget | a performance repair | Expressed on the board only as `kernel_invoke_budget`, which says why the row is blank, not what is wanted |

Do not migrate these into the Setpoint table to make them visible. A row with no target and
no sensor invites a band derived from the current reading, which §7's first anti-pattern
forbids. Keep them in the ledger and read both surfaces.

### 6. Anti-gaming

Cite `improvement-do-and-dont` by id: **D1** loosening or deleting a tagged test, **D2**
deleting a known-issue ledger entry, **D3** suppressing a finding. §2's skeptic test applies
to every claimed improvement.

Offer Desk's own gaming moves, all forbidden:

- Raising `denominator_confidence` in the space contract instead of doing the reconciliation
  that would earn it.
- Mapping a node to an arbitrary ledger account so `ledger-account-coverage` rises while the
  earnings it reports are wrong.
- Retiring IDEA nodes to shrink the denominator rather than because they were decided against.
- Recording a `gates-fact` that was not observed so a trigger evaluates.
- Reading an absent actual as zero revenue, which converts an unknown into a favourable number.
- Waiving the measures domain that `ledger-account-coverage` depends on.

### 7. Evidence

Reuse automatic capture. Attempts land in the `offer-desk-usage` scope through
`vrooli-memory.finish-attempt`; do not open a parallel ledger. A cycle's record preserves the
target identity, the before and after readings with their `reason` values, what changed, the
obligations still unmet, and the single next action. A row that stayed unavailable is
recorded as unavailable, not omitted.

### 8. Stop rules

- Stop when a required reading is missing or unreliable and the gap is permanent: route it
  immediately within authority rather than holding the cycle open.
- Stop when the next step needs authority this cycle does not hold; report the prerequisite.
- Completion requires every required outcome to be met under its own validity and sample
  rules. `ok` from `setpoint-read` is not completion: four rows are out of band and three
  carry permanent gaps as of 2026-09-08.
- Never re-measure without a coherent intervention between readings.

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| Every board entry reports unreachable actuals | money-ledger holds no account mapping, not an outage | `offer-desk offers board-show --json` | Map known accounts in-cycle; route unknown ones to money-ledger |
| `setpoint-read` reports `ok` while four rows are out of band | Working as designed; `ok` describes the board's reporting, not the product | The `in_band` column | Read §8 before claiming completion |
| `learning` never becomes comparable | No attempts are being captured | `program-runtime library run offer-desk.learning-read` | Check that callers run `vrooli-memory.finish-attempt`; capture failure is reported, not retried in place |
| A row's target is missing | The target was never authored | The Setpoint table | Keep `target: null` and raise the decision gap; do not derive a target from the current reading |
