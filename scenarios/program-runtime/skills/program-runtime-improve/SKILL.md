---
name: "program-runtime-improve"
description: "Regulate program-runtime against its setpoint: discovery and authoring floors, agent program failure rate, program adoption, portfolio maturity and health, unexercised contracts, learning-loop durability and advice application, fragment cache and promotion health, governed share, Act coverage, delegation liveness, recurring uncovered shapes, attribution, and caller skill-set coverage. Routes each out-of-band row to a curation move, a work-ladder rung, or an owner."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["program-runtime", "improve", "self-improvement", "control-loop", "setpoint", "act-projection", "meta-optimization"]
  icon: "gauge"
  status: "active"
  revision: 7
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-11-20T00:00:00Z"
  requires:
    scenarios: ["program-runtime", "prompt-manager", "agent-manager", "vrooli-memory"]
    commands: ["program-runtime discovery eval", "program-runtime authoring eval", "program-runtime programs mine", "program-runtime programs governance-share", "program-runtime bindings act", "program-runtime bindings condition", "program-runtime sessions delegations", "program-runtime library list", "program-runtime shapes list", "program-runtime programs submit", "prompt-manager skill read", "vrooli-memory journal note"]
  origin:
    kind: "authored"
---
## Practice focus: Program Runtime Improve

Regulate program-runtime, the Act instrument, against the setpoint below. The plant is the governed execution surface: bindings, sessions, programs, the library and the two evaluation corpora. An agent whose task is program-runtime itself (`goal-loop` or the meta-optimization run-introspector heartbeat) reads it. It never acts on another scenario; it files.

Required reading: `prompt-manager skill read program-runtime improvement-do-and-dont scenario-work-ladder` (usage skill, anti-gaming, where code routes go) and `path:scenarios/program-runtime/docs/spaces/act-space.md` (the Act denominator this scenario owns).

### 1. Focus and scope

In scope: the setpoint rows, library curation, corpus repair, filing ladder rungs against program-runtime, and filing `scenario-skill-setup` runs against scenarios its callers bind to. Out of scope: editing any other scenario's skills, manifests or programs (file instead), changing a floor without a recorded derivation, the usage skill's content, and agent-manager's subscription work (file it).

### 2. Setpoint

Current readings are generated into `path:scenarios/program-runtime/docs/reference/setpoint-readings.md` from `run program-runtime.setpoint-read` (refresh: `path:scenarios/program-runtime/docs/guides/setpoint-regeneration.md`); re-read them every cycle and treat a stale date as unknown.

| Row | Sensor (`program-runtime` CLI) | Band |
|---|---|---|
| discovery-floor | `discovery eval --suite evals/discovery.primary.json --mode judged --json` → `met` vs `floor` | met ≥ floor |
| authoring-floor | `authoring eval --json` → `met` vs `floor` | met ≥ floor; wall time under the 120 s sync bound |
| agent-failure-rate | `programs list` filtered in-kernel to `PROVENANCE_AGENT`, failed ÷ total | < 0.15, and zero `kernel_runtime` failures whose detail names a forbidden import |
| program-adoption | `programs portfolio` window plus `programs list --provenance agent`; a caller run id proves adoption | ≥ 0.02 of eligible agent runs, daemon harness excluded |
| portfolio-maturity | `program-runtime.portfolio-audit` → portfolio score | ≥ 0.8 |
| program-health | `programs portfolio` → declared rows with failed executions | 0 unhealthy declared programs |
| unexercised-contracts | `programs portfolio` → `never_executed` | 0 |
| learning-findings | `programs learning-findings`, `program-runtime.learning-maintain` | 0 unresolved |
| blocked-deliveries-aged | `vrooli-memory` learning task delivery rows older than 24 h | 0 |
| advice-application-ratio | Memory `compare-outcomes` applied decisions ÷ exposed candidates over 30 days | ≥ 0.2 |
| fragment-cache-hit-rate | fragment cached runs ÷ Act runs over 7 days, once a verified fragment exists | ≥ 0.5 |
| act-adoption | `library list` `verbs` naming `learn.act` (or analyzer `learning_verbs` naming bare `act`) vs retained verified fragments | ≥ 1 program with a verified fragment |
| promotable-fragments | verified fragment candidates without a filed promotion route (a fragment already declared as a `learning.baselines` entry is published, excluded) | 0 |
| governance-share | `programs governance-share --window-seconds 604800 --json` → `governed_share` | 1.0; every observed name filed |
| act-coverage | `bindings act --json` → cells by verdict | 0 cells `ACT_VERDICT_AUTHORED` |
| binding-condition | `bindings condition --json` → dormant, degraded-sustained counts | 0 degraded-sustained; dormant reviewed each cycle |
| delegation-live | `sessions delegations --json` → count | ≥ 1 succeeded per 7 days |
| uncovered-recurring-shapes | `shapes list --uncovered --min-occurrences 3` → nominated count | 0 nominated shapes with no declared contract |
| attribution | `programs list --provenance agent` caller run identity (program-adoption's corpus) | 0 unattributed agent runs |
| external-friction | `run agent-manager.friction-digest` (`scenario=program-runtime`, `window_days=7`) → `recurring_count` | 0 recurring fingerprints with owner confidence `manifest-derived` |
| fleet-improve-coverage | `run prompt-manager.skill-set-read` per scenario with ≥ 50 binding invocations in 30 d (`binding_invocations` by target scenario) | each has a usage and an improve skill registered |

**Portfolio counts have their own grain** (aggregation in `path:scenarios/program-runtime/api/internal/programs/repository.go`): `programs_executed` counts run records; `agent_runs` counts agent provenance, not a verified caller; `rows_without_identity` counts empty `program_name`; `unattributed_agent_runs` counts agent runs with an empty `caller_run_id`, so one run can count in both. Attribute a caller only from a recorded caller run ID (`programs get <id> --json`), and never compare filtered with unfiltered counts.

### 3. Sensors

Read all rows through `run program-runtime.setpoint-read` (contract `.vrooli/program-runtime/setpoint-read.json`); the readings file is generated from its envelope by `python3 scenarios/program-runtime/scripts/regenerate-setpoint-readings.py`, retaining each date and unavailable reason. A row the program marks `unavailable` is read by hand only with the exact command in the table, and the hand reading is journaled as such. The two eval rows stay unavailable inside a program by construction (their corpus runs exceed the kernel's per-invoke budget): a W3 runtime-budget limitation (§5), neither a missing binding nor a reason to estimate.

### 4. Golden corpora

Floors: `evals/discovery.primary.json` 43 of 45 positive cases (one below the observed minimum of four comparable runs 44, 44, 44, 45; recorded in the suite's `floor_reason` on 2026-08-18) and `evals/authoring.primary.json` 9 of 12 (recorded with corpus v3). A run below floor is a stop: no other route runs until the corpus route (§5) is taken and the re-read is at or above floor. This skill never lowers a floor; one is raised only with a new derivation from at least two comparable runs, recorded in the suite.

### 5. Actuators and ladder routing

Actuator rows are in-cycle curation moves; Filing rows hand off as a work-ladder rung or `report-bug`. Codes: W0 contract, W1 obligation, W2 evidence or drift, W3 runtime, W9 placement. `unexercised-contracts` and `governance-share` start as in-cycle curation, then file; every other row files.

| Row out of band | Route |
|---|---|
| discovery-floor, `wrong_selection` | Read the cases' `selected_binding_ids`; `report-bug` against the owning scenario to tighten the binding's descriptor, quoting case ids; an ambiguous case is a suite change with a new derivation |
| discovery-floor, `null_verdict` | `bindings unbound` for the expected id; W1 against the owner |
| authoring-floor below floor | Read `missed`: a rule in `api/internal/harness/contract.json` absent from skill or guide is W2 (drift check), a missing rule is W0 |
| authoring-floor wall time over bound | W3: run the eval asynchronously or shard it |
| agent-failure-rate | `kernel_runtime` naming `vrooli`: W3, classify `UNRESOLVED_NAME` with attribute-level preflight resolution, then retire the skill paragraph warning of it. `kernel_syntax` at line 1: W3, add a preflight hint (prefer `--source-file`). `unclassified`: W2, sample three, name the cause, extend the closed vocabulary |
| program-adoption below 0.02 | W9: inspect the caller path for missing program identity or run metadata and file the smallest owner repair |
| portfolio-maturity below 0.8 | Run `program-runtime.portfolio-audit`; repair the lowest dimension through its ladder route, keeping unknown reads |
| program-health above 0 | W3 for the failed declared rows; never change the band to hide them |
| unexercised-contracts above 0 | Run each never-executed declaration's fixture, or file its retirement |
| governance-share below 1.0 | `programs mine-unresolved` per observed name: a typo of a governed name is a W3 preflight suggestion, a real capability with no binding is W1 against its owner |
| act-coverage with an AUTHORED cell | Read `unresolved_operations`: W1 if the scenario exists, else record the cell blocked, dated, in `docs/spaces/act-space.md` notes |
| binding-condition degraded-sustained above 0 | `report-bug` against the binding's scenario |
| delegation-live at 0 | W3 for the bridge (`schema_mismatch` on workflow input); `report-bug` against development-toolchain-validator for the three workflow files still carrying the removed `budgets.maxCostUsd` |
| uncovered-recurring-shapes or coverage-miss telemetry | W1 against the dominant scenario in `shapes list`; for a covered shape, `report-bug` naming the failed intent query and the covering contract |
| promotable-fragments above 0 | Run the fragment-promotion route per verified candidate, preserving a fixture |
| attribution pending | `report-bug` against agent-manager: subscribe to program-runtime and ai-gateway events, carry `program_id` on invocation facts |
| external-friction recurring fingerprint | Read the finding (`agent-manager usage friction`): a program-runtime command is W3 here, skill prose is `skill-validation` on the usage skill |
| fleet-improve-coverage below band | File one `scenario-skill-setup` run per scenario missing a role, citing its invocation count; never author there from here |

### Durable learning owner queue

Read `program-runtime.learning-maintain` before selecting a new repair; the server routes terminal adaptive failures and later corrections automatically. Filter by owner and preserve truncation and unavailable reads. Claim one routed finding with action=advance, owner, finding_id, expected_state=routed, next_state=claimed and evidence, and keep claim_ref. Apply the owner repair under the active task's authority, then advance through repaired, validated and measured, each with separate evidence (artifact, independent validation, subsequent comparable use). An unavailable claim response carries the recovery receipt; never repeat domain effects to recover delivery. The `learning-findings` row reports active state counts; the routes learning-owner-queue, learning-delivery-recovery, learning-advice-consumption and learning-fragment-eligibility name owner-queue work, outbox recovery, missing knowledge consumption and compatibility or evidence diagnosis, and none certifies success from cache hits, queue routing or nonempty evidence.

### 6. Anti-gaming

`improvement-do-and-dont` §1 (tagged test, known-issue ledger, suppression) and §2 (the skeptic test) apply verbatim. Each of these is zero credit and a review flag: lowering a floor or re-deriving it from non-comparable runs (other mode or corpus version); editing `requirements/*/module.json` status fields to match PROGRESS.md prose instead of the validation refs; marking a failure `unclassified` when its cause is in the closed vocabulary, or the reverse; loosening the nomination gate to bring `uncovered-recurring-shapes` in band; counting operator, test or daemon-harness programs (or their caller run ids) in `program-adoption` or `agent-failure-rate`; excluding the two eval rows because they are unavailable in-program.

### 7. Evidence

One `vrooli-memory journal note --kind work-record` per cycle: `--trigger "<goal> cycle <n>: <row> <reading> vs <band>"`, `--approach "<route row text>"`, `--evidence "<before> -> <after> on <sensor command>"`, `--outcome "<in band | filed <ref> | reverted | unavailable: <reason>>"`. A sensor unavailable for three cycles is a `docs/internal/PROBLEMS.md` entry with the three dated readings; filings against other owners use `report-bug` with the sensor row as the observation.

### 8. Stop rules

Stop when: a floor is breached (only the corpus route runs); program-adoption is below band (route W9 placement, never compensate with operator, test or daemon runs); portfolio-maturity is below band (run the portfolio-audit route first); program-health or unexercised-contracts is above zero (repair, file, fixture-exercise or retire before claiming portfolio health); a route needs a grant (`refused_no_grant`: request it through the session path); the session's inference or delegation ceiling is reached (journal; open no new session). An `unavailable` row is journaled, never estimated, and after three cycles becomes a PROBLEMS.md entry and W2. Every readable row in band for two consecutive cycles: propose close-out to the operator and stop.

For an authorized repair with a purpose-scoped Visited Tracker campaign and explicit run IDs, compose `lib.program_runtime.improvement_evidence` (it reuses the owners' attention and investigation-evidence programs without inference) and check both child statuses and artifact identities. Candidates are unreserved and run evidence is not repair authority: claim before parallel work, follow the owning authoring guide, validate the changed behavior independently before completing a review, and treat a missing child or partial read as unknown.

### 9. Troubleshooting & Edge Cases

| Symptom | Cause and fix |
|---|---|
| `setpoint-read` reports every row unavailable | The API restarted and the CLI resolved a stale port: `vrooli scenario status program-runtime`, set `PROGRAM_RUNTIME_API_PORT` to the listening port for this cycle, file W3 for auto-detect if the CLI names a wrong base |
| `authoring eval` exceeds the sync bound | It authors twelve programs through a model: use `--async --wait-timeout 600s` if supported, else mark unavailable and file W3 |
| `programs mine` shows only test provenance | Agent programs are rare this window: journal the count, never widen the window silently |
| `library promote` refused | The source did not succeed or the name exists at that version (`programs get <id>`, `library get <name>`): promote a succeeded run under a new version |
| `bindings act` `UNAVAILABLE` cell | The owning scenario is stopped (`vrooli scenario status <scenario>`); not AUTHORED: journal, re-read next cycle |
