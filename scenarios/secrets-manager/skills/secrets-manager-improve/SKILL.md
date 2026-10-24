---
name: secrets-manager-improve
description: Regulate Secrets Manager delivery evidence against its measured contracts and route gaps to the owning work ladder.
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: ["practice"]
  tags: ["secrets-manager", "improve", "password-manager", "evidence"]
  icon: gauge
  status: active
  revision: 1
  createdAt: "2026-09-06T00:00:00Z"
  updatedAt: "2026-09-06T00:00:00Z"
  requires:
    scenarios: ["secrets-manager", "program-runtime", "prompt-manager", "vrooli-memory"]
    commands: ["measures-health validate scenario", "business-health matrix show", "program-runtime bindings condition", "test-genie runs findings", "cli-health validate scenario", "vrooli-memory journal note"]
  origin: {kind: authored}
---

# Focus and scope

This skill regulates Secrets Manager's password-manager delivery evidence: encrypted custody, human assurance, bounded delegation, typed API/CLI parity, recovery evidence, and the product's required UI and operational proof. It may inspect declared sensors and route work, but it does not edit measurements, weaken requirements, suppress findings, or change another scenario's implementation.

The scenario has a CLI manifest and currently has no governed broker invocations in the observed seven-day window. The declared improve role is therefore owed by the scenario's CLI surface and its measured delivery obligations; dormant exercise is evidence, not a success claim.

## Setpoint

The readings below are dated observations. The band is the target for the next cycle.

| row | sensor | band | today |
|---|---|---|---|
| measures-contract | `measures-health validate scenario secrets-manager` | `status = VALIDATION_STATUS_PASSED` | passed, 2026-09-06; one architecture-fallback advisory remains |
| business-traceability | `business-health matrix show secrets-manager --format summary` | `unproven claims = 0` | 0 unproven, 2026-09-06; 9 targets in progress and 3 planned |
| broker-exercise | `program-runtime bindings condition --scenario secrets-manager --window-seconds 604800` | `dormant_bindings = 0` | 4 dormant, 2026-09-06 |
| broker-freshness | `program-runtime bindings condition --scenario secrets-manager --window-seconds 604800` | `drift_status != CONDITION_STATUS_DEGRADED` | degraded because the CLI manifest is newer than generated binding evidence, 2026-09-06 |
| comprehensive-suite | `test-genie runs findings <latest-run> --scenario secrets-manager --json` | `error findings = 0` | 11 failed phases/findings groups in run `20260906-101428-3fe448d5`, 2026-09-06 |
| generated-contract | `cli-health validate scenario secrets-manager --json` | `status = VALIDATION_STATUS_PASSED` and `findings = 0` | passed at L4, 2026-09-06 |

No local `setpoint-read` program is declared yet. A future cycle may add one only after a governed binding exists for each composed read; until then the binding rows remain observable sensors and the improvement route is `no_governed_binding`.

## Sensors

Use the exact commands in the setpoint table. The two fleet sensors are also required on every cycle:

- `program-runtime bindings condition --scenario secrets-manager --window-seconds 604800` reports exercise and freshness for the typed broker bindings.
- The `agent-manager.friction-digest` program with inputs `scenario=secrets-manager` and `window_days=7` reports recurring operator friction when its governed binding is available.

External provider results outrank scenario self-reporting. A command that cannot produce a governed receipt is recorded with its closed reason and routed; it is never converted into an estimated green reading.

## Golden corpora

No `evals/*.json` corpus with a declared floor exists for this scenario. The comprehensive Test Genie suite is the current comparison surface, with a target of zero error findings; its latest reading remains below that target. Do not lower that target or create a floor from one run. Derive a corpus floor only from at least two comparable, artifact-backed runs.

## Routes

| condition | route | next sensor |
|---|---|---|
| measures contract is not passed | scenario-work-ladder W1 for the measure declaration or W3 for an implementation defect | measures-contract |
| business traceability has unproven claims | scenario-work-ladder W2, then W3 when evidence is missing from implementation | business-traceability |
| broker binding is dormant or `no_governed_binding` | file the missing exercise/binding work through the scenario work ladder; do not infer broker safety from unit tests | broker-exercise |
| broker freshness is degraded | regenerate the owned CLI evidence after the manifest change, then rerun the binding condition sensor | broker-freshness |
| comprehensive suite has errors | repair the highest broken layer identified by the suite and rerun the owned scenario tests | comprehensive-suite |
| typed CLI validation fails | repair the manifest, generated primitive evidence, or command binding at the CLI owner | generated-contract |
| another scenario owns the defect | use `report-bug` against that scenario and retain the Secrets Manager reading | the affected external sensor |

## Anti-gaming

Apply D1: do not loosen or delete a requirement-linked test; fix the contract. Apply D2: do not delete a known issue from `docs/internal/PROBLEMS.md`; resolve or migrate it. Apply D3: do not suppress a scanner, health finding, or coverage rule; fix the underlying defect or scope a proven false positive with a regression test. The scenario-specific gaming moves are removing an acceptance row to make the matrix pass, converting dormant broker exercise into a security claim, adding a waiver solely to hide a persisted domain, widening a performance floor, or treating a mocked external provider as live acceptance.

Use the skeptic test from `improvement-do-and-dont`: the diff must change behavior or evidence quality, not only the measurement.

## Evidence

Before a cycle, recall the relevant record with `search-hub query "secrets-manager password manager evidence" --type record,skill,doc`. After a route, write one `vrooli-memory journal note --kind work-record` with the trigger, approach, before and after readings on the same sensor, and outcome. Include the Test Genie run ID or provider receipt when available. Keep unresolved cross-scenario defects in the owner channel rather than rewriting this role to absorb them.

## Stop rules

- `no_governed_binding` or `pending_telemetry`: route on the first read; do not wait or estimate.
- `scenario_unreachable`: record the reason and wait one cycle for the owner/runtime to recover.
- `read_elsewhere:<program>`: run the named program under its own contract and preserve its status.
- `unreliable:<why>` or `kernel_invoke_budget`: report the reason and do not assign a band.
- If a comparable corpus falls below its floor, refuse to lower the floor and route the failing evidence.
- If a route needs a grant, request it through the governed session path.
- After two cycles in band, propose close-out; do not close the goal from this skill.
- Cycle budget: 20 minutes wall-clock; stop when the budget expires and preserve the partial receipt.
