---
name: "scenario-to-desktop-improve"
description: "Regulate the desktop ramp against its behavior contract, platform evidence, performance baselines, and engineering targets; route measured gaps to scenario-owned skills, programs, and implementation."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["desktop", "improvement", "evidence", "performance", "control-loop"]
  icon: "gauge"
  status: "active"
  revision: 2
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  requires:
    scenarios: ["scenario-to-desktop", "program-runtime", "agent-manager", "vrooli-memory"]
    commands: ["program-runtime", "scenario-to-desktop", "measures-health", "business-health", "vrooli-memory"]
  origin:
    kind: "authored"
---
## 1. Focus and scope

Regulate `scenario-to-desktop`, the desktop ramp consumed by deployment-manager
and scenario-to-android (two declared dependents, observed 2026-09-04). Route
usage failures to its skill, repeated composition failures to its programs, and
missing stable operations to the scenario work ladder.

Run this skill for scenario improvement, not ordinary packaging. Read the active
goal before choosing a target. Preserve the distinction between implementing a
capability, validating it in an available environment, and proving target-native
support. This loop does not grant release, credential, or recovery authority.

Required reading:
- `prompt-manager skill read scenario-to-desktop`
- `prompt-manager skill read improvement-do-and-dont`
- `prompt-manager skill read scenario-work-ladder`
- `path:scenarios/scenario-to-desktop/PRD.md`
- `path:scenarios/scenario-to-desktop/docs/internal/PROBLEMS.md`
- `path:scenarios/scenario-to-desktop/docs/internal/PROGRESS.md` — skill/program inventory and filed obligations

Use the current goal, PRD, and canonical desktop evidence contract as the
maturity destination. Compare them in both directions at W0. Do not silently
promote P1/P2 ideas to release requirements or drop a goal requirement because
its adapter is absent. Route contradictions to contract repair before code work.

In this control cycle, read, route, file, and journal. Execute substantial code
work through the routed work-ladder and implementation workflow under the task's
authority. `goal-loop` supplies cadence and work tracking; it does not execute
filed implementation work by itself. No cadence is implied by loading this skill.

## 2. Setpoint

Run `scenario-to-desktop.setpoint-read` first. The following nine rows match its
contract. Dated readings are observations; replace them with the next live read.
A board status of `ok` is not a maturity verdict. A missing baseline has a null
target and cannot be classified in band.

| Row | Sensor | Band | Today (2026-09-04) |
|---|---|---|---|
| binding-condition | `program-runtime bindings condition --scenario scenario-to-desktop --window-seconds 604800`; program filters its five usage read bindings | all five usage-program read bindings healthy over 7d | `unreliable:unexercised_bindings` |
| external-friction | `agent-manager.friction-digest`, scenario=scenario-to-desktop, window_days=7 | zero recurring fingerprints in a representative window | `read_elsewhere:agent-manager.friction-digest` |
| desktop-behavior | Pending artifact/target/journey/profile outcome measure | all selected contract cells evidenced; unavailable cells retained | `pending_telemetry` |
| runtime-performance | Pending comparable native startup/CPU/RSS/shutdown measure | pending-baseline; target=null | `pending_telemetry` |
| pipeline-performance | Pending comparable build duration/size/cache measure | pending-baseline; target=null | `pending_telemetry` |
| engineering-quality | Pending aggregate over provider-owned maturity and regression evidence | selected provider maturity targets met without regressions | `pending_telemetry` |
| learning-failure-recurrence | Vrooli Memory learning measure by operation/context | null; reduce against comparable baseline | live; no eligible attempts stays unreliable |
| learning-success-effort | Same sensor; completed and unresolved tasks, attempts/time to success | null; pending comparable baseline | live; incomplete histories have no median |
| learning-advice-outcomes | Same sensor; applied/rejected and supported/contradicted/unknown advice | null; pending comparable baseline | live; evidence-linked reports, no causal claim |

The desktop-behavior denominator comes from the canonical evidence contract and
the agreed goal. It covers scaffolding/API/native surfaces, bundled and
external-server operation, broker leases and private fallback, peer communication,
network/credential/provider failures, updater/recovery profiles, signing and
release trust, and routed semantic journeys. Read target capabilities from the
owner inventory. Preserve required unsupported and unavailable cells in that
denominator; never replace a missing profile with a normal run.

Engineering targets come from the selected providers and `.vrooli/testing.json`.
Use PRD operational targets for P0/P1/P2 scope. Include scenario UI accessibility
and experience obligations where the goal names them. Do not invent a composite
quality score or infer completeness from requirement status labels.


## 3. Sensors

Use `program-runtime library run scenario-to-desktop.setpoint-read` for the board.
Use `program-runtime library run agent-manager.friction-digest --input scenario=scenario-to-desktop,window_days=7`
for the external row. A truncated window, failed episode reads, or no attributable
scenario exercise does not prove zero friction; record `unreliable` and retain
the digest's sampling evidence. External observations outrank self-reports.

The binding row includes only the five read bindings declared by the two usage
programs. Never invoke a destructive command to increase exercise counts.
Dormant or absent bindings remain unproven; use actual usage records to decide
whether binding repair is necessary.

Read `measures-health validate scenario scenario-to-desktop` for measure adoption.
The 2026-09-04 inventory has seven tier-fallback warnings. Existing snapshot
measures (build, preflight, signing, state, tasks, telemetry, and captures) do not
compute the missing goal-level outcomes. The governed measures-health validation
response currently has a descriptor decoding defect; use its working CLI for this
inspection and retain the owner bug reference in the inventory.

Use `business-health matrix show scenario-to-desktop --format summary` for target
linkage. Use the gates selected by `scenario-work-ladder` for contract, evidence,
and implementation. Run suites through `vrooli scenario test scenario-to-desktop`;
follow `path:docs/TESTING.md` and wait once on a server-owned run.

Read `vrooli-memory learning measure --scope scenario-to-desktop-usage` for the full
learning window. Use `--from`, `--to`, `--operation`, and `--context-key` for
comparable windows. The setpoint reader projects at most ten cohorts and exposes
sampling limits. Scope is fixed to this scenario; comparison never pools different
contexts. Missing history, legacy records, and capped scans remain unreliable.

Keep all three learning targets null until at least two comparable windows support
a baseline. Retain sample sizes, unresolved tasks, recall-unavailable counts, and
the baseline derivation in the cycle record. These reports do not establish causal
benefit or independent evidence verification. Check the referenced owner evidence
before claiming an improvement. A fixture proves the sensor, not operator benefit.

## 4. Golden corpora

No `evals/*.primary.json` floor is declared. Do not manufacture one from a
successful build. The historical native reference is Hello Desktop; the
canonical evidence contract also requires representative real scenario journeys.
Fixtures must include both platform lifecycle and provider-owned semantic evidence.

`path:scenarios/scenario-to-desktop/docs/internal/performance-baseline.json`
contains five cold and five warm historical samples. It is a baseline candidate,
not a live sensor or an approved performance budget. Check artifact/build inputs,
host fingerprint, display, mode, profiler state, and cold/warm class before
comparison. Preserve the sample spread. Resolve inconsistent cohort metadata
before deriving a floor. Add a provider-owned comparable measurement and its
floor derivation before banding either performance row.

When an approved corpus floor exists, a regression below that floor blocks
other optimization routes. Keep unavailable environments separate from failures;
neither can be counted as a passed corpus case.

## 5. Routes

Route permanent missing sensors and bindings on their first read. Reuse the
stable backlog references in `docs/internal/PROGRESS.md` before filing duplicates.
For measured out-of-band rows, use table order. For multiple causes within a row,
use the highest failed work-ladder gate. Take one attributable repair per cycle.

| Reading or finding | Route | Next evidence |
|---|---|---|
| Binding read fails or has drifted | W1 for a missing contract; W3 for an existing operation that fails | binding-condition |
| Agent selects the wrong valid operation | Repair the usage decision predicate; supersede contrary memory guidance | external-friction |
| Valid operations recur with the same joins | Repair or author a program through `prompt-manager skill read program-runtime` | external-friction and its program execution references |
| Program contains matrix policy, private-state parsing, recovery rules, or retries | W1/W3 for the missing owner primitive; then delete compensation | affected behavior row and program result |
| desktop-behavior is pending | File/reuse its `measures-adoption` obligation and the governed matrix binding obligation | desktop-behavior |
| A measured contract cell fails | W0/W1 for missing promises; W2 for missing attributable proof; W3 for a failing implementation | same immutable cell and predecessor comparison |
| Target capability is absent | W1/W3 for the desktop adapter; file bridge/provider defects against their owners | same retained unavailable/unsupported cell |
| runtime-performance or pipeline-performance is pending | File/reuse the named measurement obligation; derive a comparable baseline before a target | same performance row |
| Comparable performance exceeds its approved budget | W3 in the owning desktop operation; preserve behavior and identity checks | same cohort and measurement method |
| engineering-quality is pending | File/reuse the provider-aggregate obligation; retain direct provider evidence | engineering-quality |
| Provider evidence reports architecture, maintainability, accessibility, or regression debt | Route through the first failed work-ladder gate and its owning skill | same provider assessment |
| The cause belongs to another scenario or the shared delivery spine | File `report-bug` with run/commit references against its owner | owner repair plus affected desktop evidence |

This skill's cycle does not edit other scenarios or shared packages. Shared
matrix policy stays in `delivery-ramp-go`; host remediation stays in `internal/`;
BAS owns semantic execution, Test Genie owns test runs/storage leases, Bridge owns
remote transport, and Deployment Manager owns release approval.

Learning routes are selected before interpreting a trend: missing or legacy
attempt records → repair capture coverage; capped/invalid sensor reads → report
the Vrooli Memory limitation with the window and scope; valid readings without
a baseline → retain the window and collect a comparable window during subsequent
use. An adverse supported trend routes the affected usage decision or scenario
failure through the existing routes above. No route drops failed attempts.

## 6. Anti-gaming

Apply `improvement-do-and-dont` D1, D2, D3 and its skeptic test.
Do not remove a failing target/profile, weaken an assertion, widen freshness,
delete history, or lower a quality floor to clear the board. Do not count package
success, capture presence, a relay call, or test-provenance program success as
native release or operator success. Do not obtain faster startup by skipping
readiness, credential, migration, or integrity checks. Preserve platform-native
and emulator evidence as separate claims. Simplify programs and skills after
the durable owner absorbs their workaround.

## 7. Evidence

Write one `vrooli-memory journal note --kind work-record` in
`scenario-to-desktop-usage` per cycle. Include the goal and row, before reading,
route and work reference, after reading on the same sensor, and owning layer.
Record files or evidence IDs rather than log bodies. A filed item is not a
completed capability. Missing sensors need `skill-set-authoring`'s backlog filing
recipe; another owner's defect needs `report-bug`.

After three consecutive `scenario_unreachable` cycles, append the references to
the existing `docs/internal/PROBLEMS.md` and route W3. Preserve earlier entries.

## 8. Stop rules

- `no_governed_binding` or `pending_telemetry`: file/reuse W1 or measurement work on the first read; do not wait for it to appear.
- `scenario_unreachable`: record the unknown reading and wait until the caller's next cycle.
- `read_elsewhere:<program>`: run that program and retain its validity limits.
- `unreliable:<why>` or `kernel_invoke_budget`: report the reason; do not classify the row in band.
- A comparable corpus misses its floor: refuse to lower the floor; prioritize its repair.
- A route requires mutation authority absent from the task: request the exact missing grant through the owning session path.
- After two comparable cycles meet every selected target, propose close-out. Keep pending sensors, unset targets, and required unavailable cells explicit; they prevent a full-maturity claim. Only the operator can accept a narrower milestone or close the goal.
- One cycle has a 30-minute wall-clock budget. Stop at expiry; this is a ceiling, not a performance target.

### Troubleshooting & Edge Cases

| Symptom | First check | Response |
|---|---|---|
| Board is `ok` with pending rows | Each row's reason | Complete the measurement work; do not report the desktop ramp mature |
| Available Linux cells pass but Windows/macOS remain unavailable | Target inventory and native evidence identities | State the local milestone and remaining target obligations separately |
| The same obligation is filed repeatedly | Existing backlog reference and prior work records | Reuse it; execute routed work under an implementation workflow when authorized |
| Friction digest says zero but its window is truncated | `window_truncated_by_run_limit`, attribution, and failed reads | Preserve the lower-bound observation; do not claim a representative zero |
| Historical performance is faster on a different host | Cohort identity and baseline derivation | Do not compare or change the floor until cohorts are comparable |
| No work occurs after setting a goal | Cadence and executor assignment | `goal-loop` records/routes work; an authorized implementation executor must perform it |
