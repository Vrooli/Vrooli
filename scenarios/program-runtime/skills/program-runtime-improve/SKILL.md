---
name: "program-runtime-improve"
description: "Regulate program-runtime against its setpoint: discovery and authoring floors, agent program failure rate, program adoption, portfolio maturity and health, unexercised contracts, governed share, Act coverage, delegation liveness, recurring uncovered shapes, attribution, and caller skill-set coverage. Routes each out-of-band row to a curation move, a work-ladder rung, or an owner."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["program-runtime", "improve", "self-improvement", "control-loop", "setpoint", "act-projection", "meta-optimization"]
  icon: "gauge"
  status: "active"
  revision: 4
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-07T17:35:00Z"
  requires:
    scenarios: ["program-runtime", "prompt-manager", "agent-manager", "vrooli-memory"]
    commands: ["program-runtime discovery eval", "program-runtime authoring eval", "program-runtime programs mine", "program-runtime programs governance-share", "program-runtime bindings act", "program-runtime bindings condition", "program-runtime sessions delegations", "program-runtime library list", "program-runtime shapes list", "program-runtime programs submit", "prompt-manager skill read", "vrooli-memory journal note"]
  origin:
    kind: "authored"
---
## Practice focus: Program Runtime Improve

Regulate program-runtime, the Act instrument, against the setpoint below. The plant is the governed execution surface: bindings, sessions, programs, the library, and the two evaluation corpora. This skill is read by an agent whose task is program-runtime itself (`goal-loop`, or the meta-optimization run-introspector heartbeat). It never acts on another scenario; it files.

Required reading:
- `prompt-manager skill read program-runtime` — the usage skill; every command this skill names is documented there or in the CLI's own help.
- `prompt-manager skill read improvement-do-and-dont` — anti-gaming, cited by section below.
- `prompt-manager skill read scenario-work-ladder` — where code routes go.
- `path:scenarios/program-runtime/docs/spaces/act-space.md` — the Act denominator this scenario owns.

### 1. Focus and scope

**In scope:** the setpoint rows below; curation of the library; corpus repair; filing ladder rungs against program-runtime; filing `skill-set-authoring` runs against the scenarios program-runtime's callers bind to.

**Out of scope:** editing any other scenario's skills, manifests, or programs (file instead); changing a floor without a recorded derivation; the usage skill's content; agent-manager's subscription work (filed against agent-manager).

### 2. Setpoint

Bands are targets. Readings are dated observations; re-read them every cycle with `run program-runtime.setpoint-read`.

| Row | Sensor | Band | Today (generated) |
|---|---|---|---|
| discovery-floor | `program-runtime discovery eval --suite evals/discovery.primary.json --mode judged --json` → `met` vs `floor` | met ≥ floor | 2026-09-07: unavailable (no_governed_binding) |
| authoring-floor | `program-runtime authoring eval --json` → `met` vs `floor` | met ≥ floor; wall time under the 120 s sync bound | 2026-09-07: unavailable (kernel_invoke_budget) |
| agent-failure-rate | `programs list` filtered in-kernel to `PROVENANCE_AGENT`, failed ÷ total | < 0.15, and zero `kernel_runtime` failures whose detail names a forbidden import | 2026-09-07: {"failed":30,"rate":0.15,"total":200,"window":"last-30-days"} (out of band) |
| program-adoption | `programs portfolio` window plus `programs list --provenance agent`; caller run id proves adoption | ≥ 0.02 of eligible agent runs; daemon harness excluded | 2026-09-07: {"adopted_agent_runs":0,"daemon_runs_excluded":0,"eligible_agent_runs":200,"rate":0.0,"unattributed_agent_runs":200,"window":"last-30-days"} (out of band) |
| portfolio-maturity | `program-runtime.portfolio-audit` → deterministic portfolio score | ≥ 0.8 | 2026-09-07: {"score":0.5472222222222223,"scored":90} (out of band) |
| program-health | `programs portfolio` → declared rows with failed executions | 0 unhealthy declared programs | 2026-09-07: {"declared_programs":6,"examples":[],"unhealthy":0} (in band) |
| unexercised-contracts | `programs portfolio` → `never_executed` | 0 | 2026-09-07: {"count":84,"examples":["agent-manager.conversation-recall","agent-manager.friction-digest","agent-manager.investigate","agent-manager.investigation-evidence","agent-manager.setpoint-read","agent-manager.supervision-case-read","agent-manager.supervision-experiment-read","ai-gateway.setpoint-read","browser-automation-studio.author-flow","browser-automation-studio.do-task"]} (out of band) |
| governance-share | `program-runtime programs governance-share --window-seconds 604800 --json` → `governed_share` | 1.0; every observed name filed | 2026-09-07: {"governed_calls":7307,"governed_share":0.9967262310735234,"observed_calls":24,"observed_names":4} (out of band) |
| act-coverage | `program-runtime bindings act --json` → cells by verdict | 0 cells `ACT_VERDICT_AUTHORED` | 2026-09-07: {"ACT_VERDICT_AUTHORED":2,"ACT_VERDICT_IN_REACH":1,"ACT_VERDICT_NOW":25} (out of band) |
| binding-condition | `program-runtime bindings condition --json` → dormant and degraded-sustained counts | 0 degraded-sustained; dormant reviewed each cycle | 2026-09-07: {"bindings":33,"by_status":{"CONDITION_STATUS_DEGRADED":2,"CONDITION_STATUS_DORMANT":18,"CONDITION_STATUS_HEALTHY":13}} (out of band) |
| delegation-live | `program-runtime sessions delegations --json` → count | ≥ 1 succeeded per 7 days | 2026-09-07: {"delegations":35,"window":"all-time"} (in band) |
| uncovered-recurring-shapes | `program-runtime shapes list --uncovered --min-occurrences 3` → nominated count | 0 nominated shapes with no declared contract | 2026-09-07: {"nominated":8} (out of band) |
| attribution | `bindings.exercise-unattributed` measure; agent-manager episodes naming a program id | agent-manager subscribed; program id on every fact from a submit | 2026-09-07: unavailable (pending_telemetry) |
| external-friction | `run agent-manager.friction-digest` with inputs `scenario=program-runtime`, `window_days=7` → `recurring_count` | 0 recurring fingerprints with owner confidence `manifest-derived` | 2026-09-07: unavailable (read_elsewhere:agent-manager.friction-digest) |
| fleet-improve-coverage | `run prompt-manager.skill-set-read` per scenario with ≥ 50 binding invocations in 30 d (`binding_invocations` grouped by target scenario) → usage present, improve present | every such scenario has a usage and an improve skill registered | 2026-09-07: unavailable (read_elsewhere:prompt-manager.skill-set-read) |

### 3. Sensors

Read all rows through `run program-runtime.setpoint-read` (contract: `.vrooli/program-runtime/setpoint-read.json`). Section 2's Today column is generated from that envelope by `python3 scenarios/program-runtime/scripts/regenerate-setpoint-readings.py`; the date and unavailable reason are retained. Rows the program marks `unavailable` are read by hand only with the exact command in the table, and the hand reading is journaled as such. Two rows remain unavailable inside a program by construction: the eval bindings exist and require confirmation, but their corpus runs exceed the kernel's per-invoke budget. That is a W3 runtime-budget limitation (§5), not a missing binding and not a reason to estimate.

Fleet sensors every scenario has: `program-runtime bindings condition` for this scenario's own bindings, and `run agent-manager.friction-digest` (inputs `scenario`, `window_days`) for `program-runtime` commands.

### 4. Golden corpora

| Suite | Floor | Derivation |
|---|---|---|
| `evals/discovery.primary.json` | 43 of 45 positive cases | one below the observed minimum of four comparable runs (44, 44, 44, 45), recorded in the suite's `floor_reason` on 2026-08-18 |
| `evals/authoring.primary.json` | 9 of 12 | recorded with corpus v3 |

A run below floor is a stop: no other route runs until the corpus route (§5) has been taken and the re-read is at or above floor. A floor is never lowered by this skill. A floor is raised only with a new derivation from at least two comparable runs, recorded in the suite.

### 5. Actuators and ladder routing

`Actuator` rows are curation moves the agent running this skill performs in-cycle without a diff. `Filing` rows hand off: a work-ladder rung against program-runtime, or `report-bug` against another owner.

| Kind | Row out of band | Route | Sensor that should move |
|---|---|---|---|
| Filing | discovery-floor below floor, `wrong_selection` on adjacent operations | Corpus repair: read the four cases' `selected_binding_ids`, tighten the descriptor of the intended binding in its owning manifest by filing `report-bug` against that scenario with the case ids; if the case itself is ambiguous, propose the case change in the suite with a new derivation | discovery-floor |
| Filing | discovery-floor, `null_verdict` on positive cases | `program-runtime bindings unbound` for the expected id; file W1 (obligation) against the owning scenario | discovery-floor |
| Filing | authoring-floor below floor | Read the `missed` cases; if the brief rule is present in `api/internal/harness/contract.json` but absent from the skill or guide, file W2 (evidence: drift check); if the rule is missing, file W0 (contract) | authoring-floor |
| Filing | authoring-floor wall time over bound | File W3: run the eval asynchronously or shard the corpus | authoring-floor |
| Filing | agent-failure-rate with `kernel_runtime` naming `vrooli` | File W3: preflight already catches `import vrooli` at the line; classify it `UNRESOLVED_NAME` instead of `unclassified` and add attribute-level resolution so a misspelled command fails preflight; then retire the skill paragraph that warns about it (`PROMOTION_LADDER.md` step 2) | agent-failure-rate |
| Filing | agent-failure-rate with `kernel_syntax` at line 1 | File W3: argv-passed source loses quoting; the skill already prefers `--source-file`; add a preflight hint naming the fix | agent-failure-rate |
| Filing | agent-failure-rate with `unclassified` | File W2: every `unclassified` is a missing failure cause; sample three, name the cause, extend the closed vocabulary | agent-failure-rate |
| Filing | program-adoption below 0.02 | W9 placement: inspect the caller path for missing program identity/run metadata, then file the smallest owner repair; daemon, operator, and test harness activity never supplies adoption credit | program-adoption |
| Filing | portfolio-maturity below 0.8 | Run `program-runtime.portfolio-audit`; repair the lowest deterministic contract dimension through the owning work-ladder route, preserving unknown portfolio reads | portfolio-maturity |
| Filing | program-health above 0 | Inspect the failed declared-program rows and file W3 against the program-runtime owner; do not hide failures by changing the band | program-health |
| Actuator, then Filing | unexercised-contracts above 0 | For each never-executed declaration, run its fixture or file a retirement/owner decision; stale contracts are not counted as healthy | unexercised-contracts |
| Actuator, then Filing | governance-share below 1.0 | Curation: for each observed name, `programs mine-unresolved`; a typo of a governed name is a preflight suggestion (W3); a real capability with no binding is W1 against its owner | governance-share |
| Filing | act-coverage with an AUTHORED cell | Read the cell's `unresolved_operations`; if the operation's scenario exists, W1 against it; if it does not (A10 names symbol-search), record the cell as blocked in `docs/spaces/act-space.md` notes with the date | act-coverage |
| Filing | binding-condition degraded-sustained > 0 | `report-bug` against the binding's scenario with the condition row | binding-condition |
| Filing | delegation-live at 0 | W3 against program-runtime for the bridge (`schema_mismatch` on workflow input), and `report-bug` against development-toolchain-validator for the 3 workflow files under its `.vrooli/agent-manager/` that still carry the removed `budgets.maxCostUsd` field | delegation-live |
| Filing | uncovered-recurring-shapes with nominations | File a W1 obligation against the dominant scenario named by `program-runtime shapes list`. Author the declared contract in that scenario. | uncovered-recurring-shapes |
| Filing | coverage miss telemetry for a recurring shape | File a discovery-failure report against program-runtime naming the failed binding-shape query and the covering contract; use the report-bug skill for the durable finding. | discovery-coverage-miss |
| Filing | uncovered-recurring-shapes with a covered recurring shape | File a discovery finding against program-runtime. Name the failed intent query and the covering contract. | uncovered-recurring-shapes |
| Filing | attribution pending | `report-bug` against agent-manager: subscribe to program-runtime and ai-gateway events; carry `program_id` on invocation facts when the executable is `program-runtime` | attribution |
| Filing | external-friction recurring fingerprint | Read the fingerprint's episode; if the command is program-runtime's, W3 here; if the fix is skill prose, `skill-improvement-suggestions` on the usage skill | external-friction |
| Filing | fleet-improve-coverage below band | For each scenario missing a role, file one `skill-set-authoring` run against its owner with the invocation count as the reason; never author into that scenario from here | fleet-improve-coverage |

### 6. Anti-gaming

`improvement-do-and-dont` §1 and its three DON'T subheadings (tagged test, known-issue ledger, suppression) and §2 (the skeptic test) apply verbatim. Program-runtime's own gaming moves, each worth zero credit and a review flag:

- Lowering a floor, or re-deriving it from runs that are not comparable (different mode, different corpus version).
- Editing `requirements/*/module.json` status fields to match PROGRESS.md prose instead of the validation refs.
- Marking a failure `unclassified` when its cause is in the closed vocabulary, or the reverse.
- Loosening the nomination gate to make `uncovered-recurring-shapes` read in band.
- Counting operator-, test-, or daemon-harness programs in `program-adoption` or `agent-failure-rate`.
- Treating a caller run id from an operator, test, or daemon harness as agent adoption.
- Excluding the two eval rows from the setpoint because they are unavailable in-program.

### 7. Evidence

One `vrooli-memory journal note --kind work-record` per cycle:

```
--trigger  "<goal> cycle <n>: <row> <reading> vs <band>"
--approach "<route row text>"
--evidence "<before> -> <after> on <sensor command>"
--outcome  "<in band | filed <ref> | reverted | unavailable: <reason>>"
```

A sensor unavailable for three cycles is a `docs/internal/PROBLEMS.md` entry with the three dated readings. Filings against other owners use `report-bug` with the sensor row as the observation.

### 8. Stop rules

| Condition | Action |
|---|---|
| discovery-floor or authoring-floor below floor | Only the corpus route runs this cycle |
| A row reads `unavailable` | Journal; do not estimate; after three cycles, PROBLEMS.md and W2 |
| program-adoption below band | Stop and route W9 placement; do not compensate with operator, test, or daemon runs |
| portfolio-maturity below band | Stop and run the portfolio-audit route before opening another improve route |
| program-health above zero | Stop and repair or file the failed declared programs before claiming portfolio health |
| unexercised-contracts above zero | Stop and fixture-exercise or retire each unexercised declaration |
| A route needs a grant (`refused_no_grant`) | Stop and request the grant through the session path |
| Every readable row in band for two consecutive cycles | Propose close-out to the operator; stop |
| The session's inference or delegation ceiling is reached | Stop; journal; do not open a new session to continue |

For an authorized repair with a purpose-scoped Visited Tracker campaign and
explicit relevant run IDs, compose `lib.program_runtime.improvement_evidence`.
It reuses the owners' attention and investigation-evidence programs without
inference. Check both child statuses and artifact identities. Candidates are
unreserved and run evidence is not repair authority: claim before parallel
work, follow the owning authoring guide for edits, and validate the changed
behavior independently before completing a review. A missing child or partial
read remains unknown; do not promote the parent submission's success into a
correctness claim.

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `setpoint-read` reports every row unavailable | The API restarted and the CLI resolved a stale port | `vrooli scenario status program-runtime`; `ss -ltnp \| grep program-runtime` | Set `PROGRAM_RUNTIME_API_PORT` to the listening port for this cycle; file W3 for auto-detect if the CLI names a wrong base |
| `authoring eval` exceeds the sync bound | The corpus authors twelve programs through a model | none | Run it with `--async --wait-timeout 600s` when the CLI supports it; otherwise treat the row as unavailable for this cycle and file W3 |
| `programs mine` shows only test provenance | Agent programs are rare this window | `programs list` in-kernel count by provenance | The row is honest; do not widen the window silently; journal the count |
| `library promote` refused | The source program did not succeed, or the name exists at that version | `programs get <id>`; `library get <name>` | Promote a succeeded run only; use a new version |
| `bindings act` shows `UNAVAILABLE` for a cell | The owning scenario is stopped | `vrooli scenario status <scenario>` | Unavailable is not AUTHORED; journal and re-read next cycle |
