---
name: "program-runtime-improve"
description: "Regulate program-runtime against its setpoint: discovery and authoring floors, agent program failure rate, governed share, Act coverage, delegation liveness, library hygiene, attribution, and the share of its callers that own a conformant skill set. Routes each out-of-band row to a curation move, a work-ladder rung, or an owner."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["program-runtime", "improve", "self-improvement", "control-loop", "setpoint", "act-projection", "meta-optimization"]
  icon: "gauge"
  status: "active"
  revision: 2
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T20:00:00Z"
  requires:
    scenarios: ["program-runtime", "prompt-manager", "agent-manager", "vrooli-memory"]
    commands: ["program-runtime discovery eval", "program-runtime authoring eval", "program-runtime programs mine", "program-runtime programs governance-share", "program-runtime bindings act", "program-runtime bindings condition", "program-runtime sessions delegations", "program-runtime library list", "program-runtime library promote", "program-runtime programs submit", "prompt-manager skill read", "vrooli-memory journal note"]
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

| Row | Sensor | Band | Today (2026-09-02) |
|---|---|---|---|
| discovery-floor | `program-runtime discovery eval --suite evals/discovery.primary.json --mode judged --json` → `met` vs `floor` | met ≥ floor | 41/45, floor 43 — below floor |
| authoring-floor | `program-runtime authoring eval --json` → `met` vs `floor` | met ≥ floor; wall time under the 120 s sync bound | suite `floor_reason` records 11/12 and 10/12 (2026-08-19, corpus v3), floor 9; run exceeds 240 s |
| agent-failure-rate | `programs list` filtered in-kernel to `PROVENANCE_AGENT`, failed ÷ total | < 0.15, and zero `kernel_runtime` failures whose detail names a forbidden import | 29/87 = 0.33 (all-time; the binding has no window) |
| governance-share | `program-runtime programs governance-share --window-seconds 604800 --json` → `governed_share` | 1.0; every observed name filed | 0.9986 (7 d: 722 governed, 1 observed) |
| act-coverage | `program-runtime bindings act --json` → cells by verdict | 0 cells `ACT_VERDICT_AUTHORED` | 25 NOW / 1 IN-REACH / 2 AUTHORED |
| binding-condition | `program-runtime bindings condition --json` → dormant and degraded-sustained counts | 0 degraded-sustained; dormant reviewed each cycle | pending-baseline |
| delegation-live | `program-runtime sessions delegations --json` → count | ≥ 1 succeeded per 7 days | 0; bridge fails on workflow schema drift since 2026-08-06 |
| library-hygiene | `program-runtime library list --json` → `origin == agent-authored` count, and duplicate binding sets among promoted entries | 0 unpromoted candidates older than one cycle; 0 duplicate binding sets | 147 `agent-authored` candidates and 13 promoted rows; fixture runs harvest into the candidate tier, so the count grows with every test |
| attribution | `bindings.exercise-unattributed` measure; agent-manager episodes naming a program id | agent-manager subscribed; program id on every fact from a submit | 0 references to program-runtime in agent-manager — pending_telemetry |
| external-friction | `run agent-manager.friction-digest` with inputs `scenario=program-runtime`, `window_days=7` → `recurring_count` | 0 recurring fingerprints with owner confidence `manifest-derived` | 0 recurring, 0 episodes for this scenario across the last 40 runs (the program reads `run_limit=40` runs; all 40 were created 2026-09-02) |
| fleet-improve-coverage | `run prompt-manager.skill-set-read` per scenario with ≥ 50 binding invocations in 30 d (`binding_invocations` grouped by target scenario) → usage present, improve present | every such scenario has a usage and an improve skill registered | 7 of 7 named targets present on 2026-09-02; waiver grading and program-reference resolution are planned, so presence is the only graded leg today |

### 3. Sensors

Read all rows through `run program-runtime.setpoint-read` (contract: `.vrooli/program-runtime/setpoint-read.json`). Rows the program marks `unavailable` are read by hand only with the exact command in the table, and the hand reading is journaled as such. Two rows are unavailable inside a program by construction: `discovery eval` is a local CLI command with no binding, and `authoring eval` is an RPC without a program binding. That is a W1 finding (§5), not a reason to estimate.

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
| Actuator, then Filing | governance-share below 1.0 | Curation: for each observed name, `programs mine-unresolved`; a typo of a governed name is a preflight suggestion (W3); a real capability with no binding is W1 against its owner | governance-share |
| Filing | act-coverage with an AUTHORED cell | Read the cell's `unresolved_operations`; if the operation's scenario exists, W1 against it; if it does not (A10 names symbol-search), record the cell as blocked in `docs/spaces/act-space.md` notes with the date | act-coverage |
| Filing | binding-condition degraded-sustained > 0 | `report-bug` against the binding's scenario with the condition row | binding-condition |
| Filing | delegation-live at 0 | W3 against program-runtime for the bridge (`schema_mismatch` on workflow input), and `report-bug` against development-toolchain-validator for the 3 workflow files under its `.vrooli/agent-manager/` that still carry the removed `budgets.maxCostUsd` field | delegation-live |
| Actuator | library-hygiene: candidates | Curation: `run program-runtime.library-curate` to group candidates by called-binding set; `program-runtime library promote --program-id <id> --name <name> --description <text> --reason "dedupe: same binding set as <name>@<v>"` for one per group; leave the rest unpromoted (retention removes them) | library-hygiene |
| Actuator | library-hygiene: duplicate promoted names with the same binding set | Curation: `library set-current` to the newest validated version; do not delete history | library-hygiene |
| Filing | attribution pending | `report-bug` against agent-manager: subscribe to program-runtime and ai-gateway events; carry `program_id` on invocation facts when the executable is `program-runtime` | attribution |
| Filing | external-friction recurring fingerprint | Read the fingerprint's episode; if the command is program-runtime's, W3 here; if the fix is skill prose, `skill-improvement-suggestions` on the usage skill | external-friction |
| Filing | fleet-improve-coverage below band | For each scenario missing a role, file one `skill-set-authoring` run against its owner with the invocation count as the reason; never author into that scenario from here | fleet-improve-coverage |

### 6. Anti-gaming

`improvement-do-and-dont` §1 and its three DON'T subheadings (tagged test, known-issue ledger, suppression) and §2 (the skeptic test) apply verbatim. Program-runtime's own gaming moves, each worth zero credit and a review flag:

- Lowering a floor, or re-deriving it from runs that are not comparable (different mode, different corpus version).
- Editing `requirements/*/module.json` status fields to match PROGRESS.md prose instead of the validation refs.
- Marking a failure `unclassified` when its cause is in the closed vocabulary, or the reverse.
- Promoting a candidate to make `library-hygiene` read in band without checking its binding set.
- Counting operator- or test-provenance programs in `agent-failure-rate`.
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
| A route needs a grant (`refused_no_grant`) | Stop and request the grant through the session path |
| Every readable row in band for two consecutive cycles | Propose close-out to the operator; stop |
| The session's inference or delegation ceiling is reached | Stop; journal; do not open a new session to continue |

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `setpoint-read` reports every row unavailable | The API restarted and the CLI resolved a stale port | `vrooli scenario status program-runtime`; `ss -ltnp \| grep program-runtime` | Set `PROGRAM_RUNTIME_API_PORT` to the listening port for this cycle; file W3 for auto-detect if the CLI names a wrong base |
| `authoring eval` exceeds the sync bound | The corpus authors twelve programs through a model | none | Run it with `--async --wait-timeout 600s` when the CLI supports it; otherwise treat the row as unavailable for this cycle and file W3 |
| `programs mine` shows only test provenance | Agent programs are rare this window | `programs list` in-kernel count by provenance | The row is honest; do not widen the window silently; journal the count |
| `library promote` refused | The source program did not succeed, or the name exists at that version | `programs get <id>`; `library get <name>` | Promote a succeeded run only; use a new version |
| `bindings act` shows `UNAVAILABLE` for a cell | The owning scenario is stopped | `vrooli scenario status <scenario>` | Unavailable is not AUTHORED; journal and re-read next cycle |
