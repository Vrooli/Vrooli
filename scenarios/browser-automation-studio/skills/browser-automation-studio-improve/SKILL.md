---
name: "browser-automation-studio-improve"
description: "Regulate Browser Automation Studio against its setpoint: execution pass rate, flake rate, selector failure rate, p95 step duration, failed-run evidence, and external friction. Routes each out-of-band row to a curation move, a work-ladder rung, or an owner."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["browser-automation-studio", "bas", "improve", "self-improvement", "control-loop", "setpoint", "executions", "selectors", "evidence"]
  icon: "gauge"
  status: "active"
  revision: 2
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T20:00:00Z"
  requires:
    scenarios: ["browser-automation-studio", "program-runtime", "prompt-manager", "vrooli-memory", "measures-health", "workflow-health"]
    commands: ["browser-automation-studio executions list", "browser-automation-studio executions screenshots", "browser-automation-studio uxmetrics workflow-aggregate", "browser-automation-studio executions retention-preview", "program-runtime programs submit", "program-runtime bindings condition", "measures-health validate scenario", "workflow-health validate scenario", "prompt-manager skill read", "vrooli-memory journal note"]
  origin:
    kind: "authored"
---
## Practice focus: Browser Automation Studio Improve

Regulate browser-automation-studio (BAS), the typed browser-execution instrument, against the setpoint below. The plant is the execution surface: persisted workflows, executions and their evidence, the playwright-driver, session profiles, and the usage skill's program steps. This skill is read by an agent whose task is BAS itself (`goal-loop` or a heartbeat). It never edits a scenario under test, a `bas/` asset owned by another scenario, or a requirements registry; it files.

Required reading:
- `prompt-manager skill read browser-automation-studio` — the usage skill; every command this skill names is documented there or in `--help`.
- `prompt-manager skill read improvement-do-and-dont` — anti-gaming, cited by section below.
- `prompt-manager skill read scenario-work-ladder` — where code routes go.
- `prompt-manager skill read measures-adoption` — how a missing sensor is filed.

### 1. Focus and scope

**In scope:** the setpoint rows below; curation of BAS-owned data (retention sweeps, candidate workflows in the `candidates` folder, the `bas-usage` memory scope); filing ladder rungs against BAS; filing against owners of scenarios whose `bas/` assets fail.

**Out of scope:** editing another scenario's workflows, selectors, or skills (file instead); changing a band without a recorded derivation; the usage skill's content; workflow-health's validation rules (filed against workflow-health).

### 2. Setpoint

Bands are targets. Readings are dated observations; re-read them every cycle with `run browser-automation-studio.setpoint-read`.

| Row | Sensor | Band | Today (2026-09-02) |
|---|---|---|---|
| pass-rate | `browser-automation-studio executions list --limit 100` → completed ÷ (completed + failed), counted in-kernel | ≥ 0.9 | 97 completed and 1 failed among the last 100 rows (2 rows non-terminal): 97 ÷ 98 = 0.99, in band (`setpoint-read`, 2026-09-02) |
| flake-rate | none: executions carry no run-group or re-run key | ≤ 0.05 | pending_telemetry (unmeasurable until a run-group key exists) |
| selector-failure-rate | `executions list` failed rows whose `error` names a selector or locator ÷ failed, classified in-kernel | ≤ 0.2 of failures | 1 of 1 failures = 1.0, out of band on a sample of one (same read) |
| p95-step-duration | `browser-automation-studio uxmetrics workflow-aggregate <workflow-id>` per workflow; no fleet sensor | ≤ 5000 ms per step | pending_telemetry (per-workflow only; Pro-tier gated; untyped Struct) |
| failed-run-evidence | `browser-automation-studio executions screenshots <execution-id>` over the five most recent failed executions → share with ≥ 1 screenshot | 1.0 | pending-baseline: 1 of 1 sampled failed executions had a screenshot (same read), and one sample establishes no precision; `docs/PROBLEMS.md` 2026-07-27 recorded failed runs retaining none. The row reads when the sample reaches five |
| external-friction | `run agent-manager.friction-digest` with inputs `scenario=browser-automation-studio`, `window_days=7` → `recurring_count` | 0 recurring fingerprints | 0 recurring, 0 episodes for this scenario across the last 40 runs (all created 2026-09-02; the program reads `run_limit=40` runs) |

Report figure, not a setpoint row: the share of `[S3]`/`[S4]` leaves among rung-labelled leaves in the usage skill was 8 of 39 = 0.21 on 2026-09-02 (hand count). No sensor produces it; `skill-improvement-suggestions` E10 names the promotion candidates.

### 3. Sensors

Read every row through `run browser-automation-studio.setpoint-read` (contract: `.vrooli/program-runtime/setpoint-read.json`). Rows the program marks `unavailable` are read by hand only with the exact command in the table, and the hand reading is journaled as a hand reading. Two rows are unavailable in-program by construction: flake-rate has no key and p95-step-duration has no fleet binding. That is a `measures-adoption` finding (§5), not a reason to estimate.

BAS declares `measures.domains: []` in `cli/manifest.json` with three waivers (selectors, telemetry, session_checkpoints). The `executions` domain is not waived and not covered. Until it is, pass-rate and selector-failure-rate are program-computed readings, not measures.

Fleet sensors every scenario has: `program-runtime bindings condition` for BAS bindings, and `run agent-manager.friction-digest` (inputs `scenario`, `window_days`) for `browser-automation-studio` commands. External sensors outrank self-reported ones.

### 4. Golden corpora

BAS has no `evals/*.json` corpus with a floor. Its closest fixed corpus is its own validation catalog: `workflow-health validate scenario browser-automation-studio` over `bas/` (57 cases on 2026-07-27; 25 passed). No floor is recorded, so the row is `pending-baseline`. Derive a floor from two comparable full runs before any route treats a catalog result as a stop. A floor is never lowered by this skill.

### 5. Actuators and ladder routing

`Actuator` rows are curation moves the agent running this skill performs in-cycle without a diff. `Filing` rows hand off: a work-ladder rung against BAS, a `measures-adoption` item, or `report-bug` against another owner.

| Kind | Row out of band | Route | Sensor that should move |
|---|---|---|---|
| Filing | Standing item, filed once and not per cycle: no `executions` measures domain exists, so pass-rate, selector-failure-rate, and failed-run-evidence are program-computed | `measures-adoption` item against BAS: an `executions` measures domain with pass rate, selector failure rate, and evidence presence | the row becomes a measure reading |
| Filing | pass-rate below band, most failures `selector_not_found` and all on one scenario's `bas/` assets | `report-bug` against that scenario with the execution ids; do not edit its selectors | pass-rate |
| Filing | pass-rate below band, failures `timeout` across two or more scenarios | ladder W3 against BAS: default navigate wait and step timeout in the executor | pass-rate |
| Filing | pass-rate below band, failures `selector_not_found` across two or more scenarios | ladder W3 against BAS: selector resolution and element-wait defaults in the executor; not the scenarios' selectors | pass-rate |
| Filing | flake-rate pending | ladder W3 against BAS: a run-group key on executions (re-run of the same workflow version) | flake-rate |
| Actuator | selector-failure-rate above band, same selector recurring in `bas-usage` site-notes | Curation: `vrooli-memory facets pin <entry-id> --scope bas-usage` on the confirmed site-note; propose a rule through `run vrooli-memory.scope-bootstrap` | selector-failure-rate |
| Filing | p95-step-duration pending | `measures-adoption` item: a fleet-level step-duration aggregate | p95-step-duration |
| Filing | failed-run-evidence below 1.0 | ladder W3 against BAS: retain the last screenshot on a failed step (PROBLEMS.md 2026-07-27) | failed-run-evidence |
| Filing | README or PRD claims self-healing workflows while no executor-validated AI path exists (README: AI generation "lacks validation against a runnable executor") | ladder W0 against BAS: contract finding; the claim is narrowed or the executor validation is built | none directly; PRD text |
| Actuator | executions and artifacts grow past the host budget | Curation: `browser-automation-studio executions retention-preview --max-age-days 14 --keep-latest 5`, then `executions retention-run --max-age-days 14 --keep-latest 5 --confirm`; retention env vars are not settable through `observability config-set` (W3 if a runtime knob is wanted) | failed-run-evidence unchanged; disk |
| Actuator | `candidates` folder holds workflows never executed after authoring | Curation: run `browser-automation-studio.smoke-flow` on each; delete failing candidates older than one cycle with `workflows delete <id>` | pass-rate |
| Filing | external-friction recurring fingerprint | Read the fingerprint's episode; if the command is BAS's, W3 here; if the fix is skill prose, `skill-improvement-suggestions` on the usage skill | external-friction |
| Filing | S3-share report figure below 0.5 | Promote the next `[S1]` leaf that recurs across agents into a program (candidate: the debug order as one read program); `workflows create` needs a governed binding first (W1 against BAS) | the report figure; pass-rate once the program runs |

### 6. Anti-gaming

`improvement-do-and-dont` §1 and its three DON'T subheadings (tagged test, known-issue ledger, suppression) and §2 (the skeptic test) apply verbatim. BAS's own gaming moves, each worth zero credit and a review flag:

- Deleting or skipping a failing `bas/` case, or moving it out of the validation catalog, to raise pass-rate.
- Loosening an ASSERT mode (`visible` → `exists`, `text_equals` → `text_contains`) to make a case pass.
- Waiving the `executions` measures domain instead of adopting it.
- Counting adhoc or seeded-demo executions as the pass-rate population.
- Sweeping failed executions with `retention-run` before the failed-run-evidence row is read.
- Editing `docs/PROBLEMS.md` to close the 2026-07-27 evidence entry without a screenshot on a failed run.

### 7. Evidence

One `vrooli-memory journal note --kind work-record` per cycle:

```
--trigger  "<goal> cycle <n>: <row> <reading> vs <band>"
--approach "<route row text>"
--evidence "<before> -> <after> on <sensor command>"
--outcome  "<in band | filed <ref> | reverted | unavailable: <reason>>"
```

A sensor unavailable for three cycles is a `docs/PROBLEMS.md` entry with the three dated readings. Filings against other owners use `report-bug` with the sensor row as the observation.

### 8. Stop rules

| Condition | Action |
|---|---|
| A row reads `unavailable` | Journal; do not estimate; after three cycles, PROBLEMS.md and `measures-adoption` |
| BAS is stopped or restarting when the read runs | Stop this cycle; do not start it from here; re-read next cycle |
| A route needs a grant (`refused_no_grant`) | Stop and request the grant through the session path |
| Every readable row in band for two consecutive cycles | Propose close-out to the operator; stop |
| A route would edit another scenario | Stop; file instead |
| The session's inference or delegation ceiling is reached | Stop; journal the ceiling and the row in progress; do not open a new session to continue |

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `setpoint-read` returns `unavailable` with `scenario_unreachable` | BAS stopped or restarting; it restarts often under test | `vrooli scenario status browser-automation-studio` | Re-read when `running`; journal the miss |
| pass-rate reads 0/0 | Window has no terminal executions | `browser-automation-studio executions list --limit 10` | The row is unavailable, not zero |
| failed-run-evidence unreadable | `executions screenshots` refused for a purged execution | `executions retention-preview` | Sample the newest failed executions only; do not widen the sample |
| `uxmetrics workflow-aggregate` returns an entitlement error | Pro-tier gate | `browser-automation-studio entitlement status` | The row stays pending_telemetry; not a BAS defect |
| Two routes match one reading | A C4 defect | Run the divergence probe (`skill-validation` §3.3) | Add the discriminating predicate (scenario-local vs cross-scenario failures) |
