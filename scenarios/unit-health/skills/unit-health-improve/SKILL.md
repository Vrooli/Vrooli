---
name: "unit-health-improve"
description: "Regulate Unit Health, the unit-test instrument, against its setpoint: self-maturity by repair, calibration and corpus floors, mutation signal, holdout agreement before any rule leaves advisory, unknown share, declared-versus-emitted codes, reviewed evidence, traceability, fleet adoption, and external friction. Routes each out-of-band row to a repair inside unit-health, a corpus move, or an owner."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["unit-health", "improve", "self-improvement", "control-loop", "setpoint", "calibration", "holdout", "test-quality"]
  icon: "gauge"
  status: "active"
  revision: 3
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-09T00:00:00Z"
  requires:
    scenarios: ["unit-health", "program-runtime", "test-genie", "agent-manager", "vrooli-memory"]
    commands: ["unit-health validate scenario", "program-runtime library run", "vrooli-memory journal note", "prompt-manager skill read"]
    skills: ["unit-health", "improvement-do-and-dont", "scenario-work-ladder", "test", "seam-discovery-and-enforcement"]
  origin:
    kind: "authored"
---
## Practice focus: Unit Health Improve

Regulate Unit Health against the setpoint below. The plant is the instrument
itself: the maturity contract, the seven-rule test-quality catalog and its
adapters, the calibration corpus and holdout discipline, the sampled-review
program, and the trust the Test Genie `unit` phase places in all of it. This
skill is read by an agent whose task is Unit Health (`goal-loop`, or a
`scenario-improvement-campaign` mandate). It never edits another scenario's
tests, manifests, or programs; it files.

Required reading:
- `prompt-manager skill read unit-health` — the usage skill; every read this skill names is documented there.
- `prompt-manager skill read improvement-do-and-dont` — anti-gaming, cited by id in §6.
- `prompt-manager skill read scenario-work-ladder` — where code routes go.
- `path:docs/reference/test-quality-rules.md` — the catalog, its support profiles, and each rule's promotion prerequisites.
- `path:docs/reference/maturity.md` — the capability ladders and the declared-but-not-emitted codes.
- `path:docs/internal/PROBLEMS.md` — the ledger; §5 reads it every cycle.

### 1. Focus and scope

**Purpose.** Every rule Unit Health emits is calibrated against independent
evidence, honest about what it cannot observe, cheap enough to run on every
applicable target, and adopted by the phase that gates on it. Unit Health reaches
its own top rung by repair, never by relaxing what it measures.

**In scope:** the setpoint rows below; repair of unit-health's own tests,
adapters, analyzers, catalog, corpus, and docs; curation moves on the corpus
that the owner already exposes; filing ladder rungs against unit-health; filing
`report-bug` against the owners of sensors this board reads elsewhere.

**Out of scope:** editing any other scenario's tests or policy profile to move a
row (file instead); lowering a coverage floor, widening a waiver, or changing a
rule's enforcement without a recorded owner decision; the usage skill's content;
Test Genie's phase policy (filed against test-genie).

Dependents: Test Genie's `unit` phase shells every applicable target to this
scenario; that dependency is not declared in Test Genie's manifest (ledger
2026-09-09). The improve role was selected by operator request on 2026-09-08.

### 2. Setpoint

Bands are targets. Readings are dated observations; re-read them every cycle with
`program-runtime library run unit-health.setpoint-read --provenance operator --json`
and regenerate the Today column with
`python3 scenarios/unit-health/scripts/regenerate-setpoint-readings.py <envelope>`.
A row marked `unavailable` remains unknown and is routed by its exact reason in §5.

| Row | Sensor | Band | Today (generated) |
|---|---|---|---|
| self-maturity | `unit-health validate scenario unit-health` → `assessment.local` level, next level, blocking codes, per-capability levels | every capability at its top rung; no blocking codes | 2026-09-09: {"blocking":[],"capabilities":{"coverage_quality":"L2","execution_readiness":"L2","framework_config":"L3","stability_traceability":"L2","surface_discovery":"L2","test_architecture":"L3"},"level":"L3","next":null} (in band) |
| self-execution | `unit-health validate scenario unit-health --execution` → `evidence_stages.executed`, `LOW_COVERAGE` count | executed=passed; 0 `LOW_COVERAGE` | 2026-09-09: {"executed":"passed","low_coverage":0} (in band) |
| unknown-share | `test_quality.results` grouped by rule: unknown ÷ assessed, excluding `not_applicable` and `NOT_EXECUTED` | worst rule ≤ 0.05; every unknown reason in the closed vocabulary | 2026-09-09: {"not_executed":3,"per_rule":{"assertion-observation":0.0334,"async-assertion":0.0,"focused-test":0.0,"malformed-expectation":0.0,"skip-declaration":0.0},"worst":0.0334} (in band) |
| rules-promoted | `test_quality.results[].enforcement` per rule | undecided: a rule leaves advisory only through §5's promotion route with a dated owner decision; the program keeps `target` null | 2026-09-09: {"promoted":0,"promoted_rules":[],"rules_observed":5} (no band; target undecided) |
| reviewed-evidence | `evidence_stages.reviewed` | supplied | 2026-09-09: not_supplied (out of band) |
| requirement-traceability | `traceability.unavailable_reason` and `requirement-link` violations | owner reachable; 0 untagged | 2026-09-09: {"owner":"reachable","untagged":0} (in band) |
| calibration-floor | `unit-health calibrate run --partition inventory` → matched ÷ implemented development cases and the recorded development floor | matched = implemented; floor recorded in `development.json` | 2026-09-09: {"floor":"42/42@2026-09-09","implemented":42,"matched":42} (in band) |
| corpus-implemented-share | `unit-health calibrate corpus` → implemented, retired, and specified cases | implemented + retired = specified | 2026-09-09: {"implemented":42,"retired":39,"specified":81} (in band) |
| holdout-agreement | `unit-health calibrate run --partition reviewed-holdout --holdout <id> --rule <rule>` → false positives, false negatives, unknowns per rule | FP rate within the rule's declared budget on a reviewed holdout of at least `num[threshold]:30` cases | 2026-09-09: {"budget":0.05,"fn_rate":0.0,"fp_rate":0.0,"labelled":30,"observed":30,"promotion_allowed":false,"unknown":0} (in band) |
| declared-vs-emitted | maturity spec finding codes minus codes with an analyzer emitter | 0 | 2026-09-09: {"spec_codes_without_emitter":0} (in band) |
| mutation-signal | `unit-health.mutation-pilot` through `unit-health/mutation/pilot` → killed ÷ valid mutants on the sampled set | kill rate ≥ 0.95; floor is derived from comparable runs and is not lowered by a later read | 2026-09-09: {"invalid":0,"kill_rate":1.0,"killed":20,"survived":0} (in band) |
| fleet-adoption | bounded roster validation plus `lib.test_genie.validation_digest(caller_scenario=<target>, limit=20)` → applicable, covered, uncovered, and child failures | uncovered = 0; child failures remain explicit | 2026-09-09: {"applicable":1,"child_failures":[],"covered":0,"uncovered":["unit-health"]} (out of band) |
| external-friction | `lib.agent_manager.friction_digest(scenario="unit-health", window_days=7)` → recurring count and top fingerprints | no recurring fingerprints on unit-health commands | 2026-09-09: {"recurring_count":0,"top_fingerprints":[]} (in band) |

Rows that read in the program today: self-maturity, self-execution (only with
`execution=true`), unknown-share, rules-promoted, reviewed-evidence,
requirement-traceability, calibration-floor, corpus-implemented-share,
declared-vs-emitted, mutation-signal (only with `execution=true`),
fleet-adoption, and external-friction. Fleet applicability uses the explicit
bounded `roster` input while the registry-sweep binding is unavailable.

### 3. Sensors

Read every row through `run unit-health.setpoint-read` (contract:
`.vrooli/program-runtime/setpoint-read.json`; inputs `target`, `execution`). The
program makes one validation read, one inventory calibration read, one reviewed
holdout read, and the bounded child reads for fleet rows; it derives the
readable rows from those results.
Validity rules:

| Row | Valid when | Comparable when |
|---|---|---|
| self-maturity | `assessment.local.current_level` present; plan-only is sufficient because required findings are static | same target and spec version |
| self-execution | `execution=true` and `evidence_stages.executed` is not `not_requested`. The default read declines the row with `kernel_invoke_budget`: the decline is conditional on the input, because an executed run on an arbitrary target can exceed the invoke budget and the caller accepts that budget by passing `execution=true` | same policy profile; a changed coverage floor is a new baseline |
| unknown-share | at least one assessed result; results with reason `NOT_EXECUTED` are excluded from the denominator and counted separately, never as clean | same read mode (plan-only or executed) |
| rules-promoted | results present | catalog version unchanged |
| reviewed-evidence | `evidence_stages` present | same run |
| requirement-traceability | `unavailable_reason` is `NONE`; any other reason is `unreliable:<reason>`, the row keeps `in_band` null, and the board's status is unchanged | requirements registry unchanged |

Supporting calibration and catalog checks for the declared rows:

- **calibration-floor, corpus-implemented-share:** The setpoint program reads `unit-health/calibrate/run` with `partition=inventory`. The RPC returns the bounded inventory, development floor, and case outcomes. Unknown and mismatched cases remain visible; no expectation is treated as an observation.
- **holdout-agreement:** Read the committed `assertion-observation-go-v1` holdout through `unit-health/calibrate/run`; the setpoint stays in band only when at least `num[threshold]:30` cases are observed within the catalog budget and `promotion_allowed` remains false.
- **declared-vs-emitted:** `pending_telemetry`. Hand read: codes in the `maturity.findings` block of `.vrooli/test-genie.json` minus codes that any analyzer under `api/internal/validation/` emits. The quality rollup emitter covers `TEST_SKIPPED_OR_ONLY` and `TEST_UNTAGGED_REQUIREMENT`; the non-observable legacy codes were retired on 2026-09-09.
- **mutation-signal:** Read `unit-health.setpoint-read` with `execution=true`, `mutation_package=./internal/testquality/...`, `mutation_seed=pilot-1`, and `mutation_floor=0.95`. The comparable pilot runs were program `prog_a76228a3-004a-4ab7-8bf4-db861cc745ca` / owner run `mutation-c3a8b6d0a6e23a9b` and program `prog_fdd5a584-ba8b-4aa1-9ce7-a2be923d8a6b` / owner run `mutation-b7838dd4ad9f8655`; both generated `num[sot]:20` valid mutants and killed `num[sot]:20`. The floor is `min(1.0, 1.0) - 0.05 = 0.95`. A non-executed read is declined with `kernel_invoke_budget`; a failed pilot is unreliable, not a zero score.
- **fleet-adoption:** The setpoint program reads the bounded roster through `unit-health/validate/scenario`, treats a scenario with at least one workspace as applicable, then calls `lib.test_genie.validation_digest(caller_scenario=<that scenario>, limit=20)`. Coverage requires child status `ok`, a terminal receipt, and evidence; child failures are listed separately and never counted as uncovered or covered. When the registry-sweep binding is available, replace the explicit roster fallback through a reviewed contract change.
- **external-friction:** The setpoint program calls `lib.agent_manager.friction_digest(scenario="unit-health", window_days=7)` and reads `recurring_count` plus the top fingerprints. A failed child is `unreliable:<class>`, never recurring count zero.

An independent applicable measurement outranks a self-report: a Test Genie
receipt of an executed unit phase outranks this scenario's own plan-only read of
the same target.

### 4. Golden corpora

| Suite | Location | Floor | Derivation |
|---|---|---|---|
| development cases | `api/internal/testquality/testdata/development.json` (`num[sot]:42` of the `num[sot]:81` cases in `case-specification.json`) | matched = implemented; floor `num[sot]:42/42@2026-09-09` | every implemented case has passed inside the Go suite and the governed development read |
| native Vitest conformance | `api/internal/testquality/testdata/native/vitest-v2/` via `run-native.mjs`, expectations in `native-expected.json` | all expectations match for Vitest 2.1.9 | version-scoped; another installed version is a new profile, not a floor change |
| reviewed holdout | `api/internal/testquality/testdata/holdouts/assertion-observation-go-v1/` (`num[sot]:30` Go tests from the selected scenarios) | `assertion-observation` FP budget 0.05; current comparison `num[sot]:30/30` observed, `num[sot]:0` unknown | labels precede observations; the single-reviewer limitation remains explicit and the catalog decision is `requested` |

A run below floor is a stop: no other route runs until the corpus route in §5 has
been taken and the re-read matches. A floor is never lowered by this skill. A
development fixture is never relabelled as a holdout; the harness rejects mixed
partitions and this skill does not work around it.

### 5. Routes

The board rows above and the problems ledger feed this section
(`docs/internal/PROBLEMS.md`). A cycle that reads only the board misses every
wanted capability the ledger holds; read both. A ledger entry has no target and
no sensor, so it never becomes a board row; it is routed from here.

`Repair` rows are changes inside unit-health the agent running this skill makes
in-cycle under the active mandate. `Curation` rows are corpus or catalog moves
the owner already exposes. `Filing` rows hand off.

| Kind | Row out of band | Route | Sensor that should move |
|---|---|---|---|
| Repair | self-maturity below top rung | Read the blocking code's `evidence`. `TEST_UTIL_MISSING` on cli: add the shared root only with a consumer test that uses it. `MISSING_INJECTABLE_SEAM` on api: introduce the clock, env, or logger seam where a test needs to substitute it (`seam-discovery-and-enforcement`). Never edit `.vrooli/testing.json` to clear a row. | self-maturity |
| Repair | self-execution with `LOW_COVERAGE` above 0 | Test the reachable paths in the named files through the authoring standard; a `cmd/` binary at 0% is covered by testing the package it calls, not by a smoke test that imports main. | self-execution |
| Repair | unknown-share above 0.05 for a rule | Read the unknown reasons. `EXTERNAL_HELPER_UNRESOLVED`: extend the adapter's helper resolution with a calibration case first. A reason outside the closed vocabulary is a W2 evidence defect. | unknown-share |
| Repair, then decision | rules-promoted with a rule whose holdout passed | Record the owner decision (date, rule, budget, holdout id) in the catalog entry, then change `defaultEnforcement`. Never the reverse order. | rules-promoted |
| Repair | reviewed-evidence not supplied | Fix the sampling program's input first (ledger 2026-09-09: metadata only). Until a bounded excerpt travels through the governed gateway, the row stays unmet; do not attach triage labels as review. | reviewed-evidence |
| Filing, then Repair | requirement-traceability owner unavailable | `report-bug` against the requirements owner with the `unavailable_reason`. Separately, W2 here: link each of unit-health's `num[sot]:24` `planned` requirements to its existing test or mark it honestly unvalidated. | requirement-traceability |
| Curation | calibration-floor below floor | Only this route runs. Read the mismatched case's forbidden observations; fix the adapter, or with a new `rationale` the case. Record which in the work record. | calibration-floor |
| Repair | corpus-implemented-share below `num[sot]:81/81` | Implement the next family in specification order, adapter first, fixture second, or record an explicit retirement reason when the evidence kind is outside static calibration. | corpus-implemented-share |
| Repair | holdout-agreement pending | Re-read the committed assertion-observation Go holdout through `unit-health calibrate run` and the setpoint program. Keep enforcement advisory until the dated catalog decision is `approved`; a requested or declined decision is not promotion. | holdout-agreement, rules-promoted |
| Repair | declared-vs-emitted above 0 | For each code: implement its emitter from the typed catalog, or remove it from the spec, the PRD target, and `maturity.md` in one change. Never leave a code that tests assert is not produced. | declared-vs-emitted |
| Repair | mutation-signal below 0.95 | Re-run the owner-side pilot on the recorded package with `execution=true`; inspect surviving valid mutants and strengthen the owning tests. Never lower the derived floor or treat invalid, equivalent, out-of-contract, infrastructure-failure, or unknown receipts as killed. | mutation-signal |
| Filing | fleet-adoption with an uncovered applicable scenario or child failure | The target scenario's owner, through `report-bug` with the digest state and receipt id when present; an owner child failure is not silently treated as no coverage. | fleet-adoption |
| Filing | external-friction recurring fingerprint | Read the top fingerprint from the composed child result. A unit-health command defect is W3 here; skill prose is `skill-improvement-suggestions` on the usage skill. | external-friction |
| Filing | calibration-floor or corpus-implemented-share has no governed reading | This was closed on 2026-09-09 by `ValidationService.RunCalibration` and the `unit-health calibrate` commands. | calibration-floor |
| Filing | Test Genie dependency undeclared (ledger) | `report-bug` against test-genie: declare validation providers under `dependencies.scenarios`. | none; ledger closes |

Choose the smallest intervention that advances an approved outcome without
violating a protected floor. Diagnostic movement alone is not success.

### 6. Anti-gaming

`improvement-do-and-dont` §1 and its three DON'T subheadings (D1 loosened or
deleted tagged test, D2 deleted known-issue ledger, D3 suppressed finding) and
§2 (the skeptic test) apply verbatim. Unit Health's own gaming moves, each worth
zero credit and a review flag:

- Lowering a coverage floor, adding a waiver, or changing a policy class in `.vrooli/testing.json` to clear self-maturity or self-execution. `UNIT_POLICY_WEAKENED` and `UNIT_POLICY_WAIVER_INVALID` exist to catch this in other scenarios; the same rule binds the owner.
- Shrinking a denominator: excluding files from coverage include, skipping directories in discovery, or emitting fewer rule results to lower unknown-share.
- Reclassifying `unknown` as `checked_clean` when an adapter cannot resolve a helper, or extending the closed reason vocabulary to hide a missing cause.
- Relabelling development fixtures as holdout, or writing holdout labels after observations are inspected.
- Deleting or loosening a corpus case that fails, widening a forbidden-observation list to admit the current output, or re-deriving a floor from non-comparable runs.
- Deleting a dead finding code from the spec without the PRD and the reference doc in the same change, or the reverse.
- Changing `defaultEnforcement` without a dated owner decision, or counting metadata-only model labels as reviewed evidence.
- Reading zero violations on unit-health itself as evidence of adequacy.
- Running the board with `execution=true` only for the target where it passes, and citing that as fleet evidence.

### 7. Evidence

One `vrooli-memory journal note --kind work-record` per cycle:

```
--trigger  "<goal> cycle <n>: <row> <reading> vs <band>"
--approach "<route row text>"
--evidence "<before> -> <after> on <sensor command or program>"
--outcome  "<in band | filed <ref> | reverted | unavailable: <reason>>"
```

A sensor unavailable for `num[threshold]:three` cycles is a `docs/internal/PROBLEMS.md` entry
with the dated readings. Filings against other owners use `report-bug`
with the board row as the observation. Do not create a parallel ledger; the
Today column and the work record are the evidence.

### 8. Stop rules

| Condition | Action |
|---|---|
| calibration-floor below floor | Only the corpus route runs this cycle |
| A route would change a floor, a waiver, a forbidden-observation list, a policy class, or a rule's enforcement | Stop; that is a target amendment and needs the owner's decision recorded first |
| A row reads `unavailable` with a transient reason | Journal; do not estimate; after `num[threshold]:three` cycles, ledger entry and W2 |
| The mutation pilot or a calibration run would touch the shared worktree | Refuse; disposable workspace or nothing |
| A route needs a grant the session does not hold (`refused_no_grant`) | Stop and request the grant through the session path |
| Every readable row in band for `num[threshold]:two` consecutive cycles and every pending row has a filed owner | Propose close-out to the operator; stop |
| The session's inference or delegation ceiling is reached | Stop; journal; do not open a new session to continue |

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| Every sensor row reads `unreliable:validate-read-failed` or `scenario_unreachable` | unit-health API stopped, or the CLI resolved a stale port | `vrooli scenario status unit-health` | Start through the lifecycle; re-read; never estimate |
| The board reports `ambiguous_response` from the validate binding | The kernel could not pick a primary response field | the program's `rows="findings"` selector | The selector is part of the program; a regression is W3 here |
| unknown-share reads 1.0 for every Vitest rule in a plan-only read | Vitest profiles need a native run; plan-only results carry reason `NOT_EXECUTED` | `not_executed` in the row's reading | Those results are excluded from the denominator; read with `execution=true` for the Vitest rules |
| self-execution never reads | Default `execution=false` | the row's reason `kernel_invoke_budget` | Pass `execution=true` for unit-health itself (fits the budget) or read from the shell with `--execution` |
| The regeneration script rejects the envelope | Wrong row count or a renamed row | `signals.rows` length is 13 | Keep the program and the table in step; a new row is added to both in one change |
| `TEST_UTIL_MISSING` fires while the projection check for the same root passes | The readers resolve the root differently (ledger 2026-09-09) | the finding's `evidence` | Repair the reader, add a calibration case, then the workspace if a consumer needs the root |
