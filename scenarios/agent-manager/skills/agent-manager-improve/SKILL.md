---
name: "agent-manager-improve"
description: "Regulate Agent Manager against its setpoint: episode ownership attribution, recurring-friction publish cadence, findings with measured effectiveness, investigation completion, skill and program identity on invocation facts, and the durable friction measures. Routes each out-of-band row to a curation move, a work-ladder rung, or an owner."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["agent-manager", "improve", "self-improvement", "control-loop", "setpoint", "friction", "episodes", "findings", "meta-optimization"]
  icon: "gauge"
  status: "active"
  revision: 2
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T20:00:00Z"
  requires:
    scenarios: ["agent-manager", "program-runtime", "prompt-manager", "vrooli-memory"]
    commands: ["agent-manager run publish-recurring-friction", "agent-manager findings list", "agent-manager run list", "agent-manager run episodes", "agent-manager run episode-cohort", "agent-manager run invocation-facts", "agent-manager measures external-tool-share", "agent-manager measures retry-rate", "agent-manager measures tool-failure-rate", "agent-manager measures repeated-work-rate", "agent-manager measures run-success-rate", "agent-manager measures finding-recurrence-rate", "program-runtime bindings condition", "program-runtime programs submit", "prompt-manager skill read", "vrooli-memory journal note"]
  origin:
    kind: "authored"
---
## Practice focus: Agent Manager Improve

Regulate Agent Manager, the run and friction instrument, against the setpoint below. The plant is the evidence chain from a run to an owner: runs, episodes, invocation facts, findings, and the durable friction measures. This skill is read by an agent whose task is Agent Manager itself (`goal-loop`, or the meta-optimization heartbeat that owns the friction inbox). It never edits another scenario; it files.

Required reading:
- `prompt-manager skill read agent-manager` — the usage skill; every command named here is documented there or in the CLI.
- `prompt-manager skill read improvement-do-and-dont` — anti-gaming, cited by section below.
- `prompt-manager skill read scenario-work-ladder` — where code routes go.
- `path:scenarios/agent-manager/docs/reference/classifier-accuracy.md` — the golden corpus this scenario owns.

### 1. Focus and scope

**In scope:** the setpoint rows below; publishing recurring friction; filing ladder rungs against Agent Manager; filing `report-bug` against the scenarios whose commands the episodes name.

**Out of scope:** editing another scenario's manifest, skills, or programs (file instead); the meta-optimization inbox's own triage (that owner drains it); the usage skill's content; changing a detector threshold without a recorded derivation.

### 2. Setpoint

Bands are targets. Readings are dated observations; re-read them every cycle with `run agent-manager.setpoint-read`.

| Row | Sensor | Band | Today (2026-09-02) |
|---|---|---|---|
| ownership-attribution-share | `run agent-manager.friction-digest` (or `setpoint-read`) → episodes with `ownerConfidence == manifest-derived` ÷ all episodes over the 25 most recent runs | ≥ 0.90 | 0.706 over the 25 most recent runs (`setpoint-read` live run 2026-09-02) |
| recurring-friction-publish-cadence | `agent-manager run publish-recurring-friction --json` → fingerprints published this run; `findings list --severity recurring` | every fingerprint at ≥ 3 distinct runs is published within 7 days of reaching 3 | unavailable, reason `no_governed_binding`: the publisher has no program binding. Read it from the CLI; no scheduler calls it and `findings list` is empty |
| findings-with-resolved-effectiveness | `agent-manager findings list --json` → share of findings older than 14 days whose `effectiveness` is not `not_yet_measurable` | ≥ 0.50 | unavailable, reason `no_governed_binding`: no findings exist because `publish-recurring-friction` has never run. The effectiveness writer has callers (`api/internal/orchestration/investigation_findings.go:137` and `:175`, the latter writing the after value from `metrics.RecurrenceRate`), so the row reads once findings exist |
| investigation-completion-rate | `agent-manager run list --tag-prefix agent-manager-investigation --json` → `RUN_STATUS_COMPLETE` ÷ terminal | ≥ 0.80 | unavailable, reason `unreliable:no_terminal_runs_with_tag_prefix` |
| skill-and-program-identity-on-facts | `agent-manager run invocation-facts <id>` → facts carrying a `skill_id` or `program_id` | every fact whose executable is `program-runtime` or `prompt-manager skill read` carries the id | unavailable, reason `pending_telemetry`: the fact schema has neither field |
| friction-measure-validity | `agent-manager measures retry-rate --window last_7d` → `validity.state` (same field on `tool-failure-rate`, `repeated-work-rate`, `external-tool-share`) | `available` on all four | 0 of four measures `available` (last_7d): classified share 83.4 percent against a 90.0 percent minimum |
| external-tool-share | `agent-manager measures external-tool-share --window last_7d` → `share` | pending-baseline; direction falling once the row is `available` | unavailable, reason `unreliable:classified share 83.4% is below the minimum 90.0%` |
| run-success-rate | `agent-manager measures run-success-rate --window last_7d` → `rate` | ≥ 0.85 | 0.892 last_7d (`setpoint-read` live run 2026-09-02) |
| retry-rate | `agent-manager measures retry-rate --window last_7d` → `rate` | ≤ 0.03 | unavailable, reason `unreliable:classified share 83.4% is below the minimum 90.0%` |
| tool-failure-rate | `agent-manager measures tool-failure-rate --window last_7d` → `rate` | ≤ 0.01 | unavailable, reason `unreliable:classified share 83.4% is below the minimum 90.0%` |
| repeated-work-rate | `agent-manager measures repeated-work-rate --window last_7d` → `rate` | ≤ 0.05 | unavailable, reason `unreliable:classified share 83.4% is below the minimum 90.0%` |

### 3. Sensors

Read all rows through `run agent-manager.setpoint-read` (contract: `.vrooli/program-runtime/setpoint-read.json`). Rows the program marks `unavailable` are read by hand only with the exact command in the table, and the hand reading is journaled as such. Three rows are unavailable in-program by construction: `publish-recurring-friction` and `findings list` have no program binding, and `invocation-facts` carries no identity field to read. Measures bindings take `window={"token": "TIME_WINDOW_TOKEN_LAST_7D"}` in-kernel; the CLI takes `--window last_7d`.

Fleet sensors every scenario has: `program-runtime bindings condition` for Agent Manager's own bindings, and Agent Manager's own friction for `agent-manager` commands through `run agent-manager.friction-digest` with `scenario=agent-manager`.

External sensors outrank self-reported ones: an owner's `report-bug` filing about a wrong attribution outranks `ownerConfidence`.

### 4. Golden corpora

| Suite | Floor | Derivation |
|---|---|---|
| `api/internal/runsignal/testdata/classification/all-detectors.labels.json` (50 labelled friction windows) | every shipped detector at or above its committed per-detector precision and recall threshold | thresholds are committed beside the corpus; scored only by `runsignal.ClassificationAccuracy` through `GOWORK=off go test ./internal/runsignal/...` from `scenarios/agent-manager/api` and the `agent-conformance` test-genie phase |

A detector below its threshold is a stop: no other route runs until the corpus route (§5) has been taken and the re-read is at or above threshold. A threshold is never lowered by this skill. A threshold is raised only with a new derivation over the same corpus version, recorded in the labels file.

### 5. Actuators and ladder routing

One route per row per cycle. When two rows' predicates hold for the same reading, take the first matching row from the top; re-read next cycle before taking the next.

`Actuator` rows are moves the agent running this skill performs in-cycle without a diff (the publisher, an investigation request). `Filing` rows hand off: a work-ladder rung against agent-manager, or `report-bug` against the scenario an episode names. `None` rows are honest readings with no move.

| Kind | Row out of band | Route | Sensor that should move |
|---|---|---|---|
| Filing | ownership-attribution-share below band, `unknown` episodes name a command with no manifest entry | W3 against agent-manager: extend the manifest-derived owner resolver (`runsignal/capability_detectors.go`) to read the command's `cli/manifest.json` when the scenario directory exists; until then `report-bug` against each named scenario whose manifest lacks the command | ownership-attribution-share |
| None | ownership-attribution-share below band, `unknown` episodes name a shell builtin or non-Vrooli tool (`grep`) | No route: these are honest unknowns. Record the share of unknowns that are non-Vrooli in the journal so the band can be re-derived | ownership-attribution-share |
| Actuator, then Filing | recurring-friction-publish-cadence pending | Curation, in-cycle: `agent-manager run publish-recurring-friction --cap 20`; journal the fingerprints published. W3 against agent-manager: a scheduler (heartbeat member or `maintenance`) that calls the publisher daily and exposes the last published time | recurring-friction-publish-cadence |
| Filing | findings-with-resolved-effectiveness below band or pending | W2 against agent-manager: prove the effectiveness writer fires on a completed investigation. First take the cadence route so findings exist; then, on one completed investigation with an `applied` decision, confirm `investigation_findings.go:175` wrote the after value; if it did not fire, W3 with the run id | findings-with-resolved-effectiveness |
| Filing | investigation-completion-rate below band | Read the failed investigation runs with `run report <id>`; a preflight or resource-role failure is W3 here; a prompt-contract failure is `skill-validation` on `agent-manager-process-investigation` | investigation-completion-rate |
| Filing | skill-and-program-identity-on-facts pending | W3 against agent-manager: carry `skill_id` when the executable is `prompt-manager skill read` and `program_id` when it is `program-runtime`; subscribe to program-runtime and ai-gateway typed events so a submit and a route carry the run id | skill-and-program-identity-on-facts |
| Filing | friction-measure-validity `unreliable`, `unclassifiedShare` above 0.10 | W3 against agent-manager: sample 10 unclassified facts with `run invocation-facts <id>`; each is a missing detector case; add it to the labelled corpus with a reason before adding detector logic | friction-measure-validity |
| Filing | external-tool-share above its baseline once `available` | Read the top external commands with `run episode-cohort --limit 50`; if one Vrooli scenario owns them, `report-bug` against it with the fingerprint; if none does, the row is descriptive and stays | external-tool-share |
| Actuator, then operator | run-success-rate below band, failures concentrated in one profile (`measures profile-breakdown`) | `run investigate --run-ids <top failures> --depth standard`; apply only with an operator decision | run-success-rate |
| Filing | run-success-rate below band, failures spread across profiles | W2 against agent-manager: the runner or sandbox layer; read `run report` for the exit code class first | run-success-rate |
| Filing | retry-rate, tool-failure-rate, or repeated-work-rate above band while `available` | `run episode-cohort --limit 50`; the top fingerprint's `suspectedOwnerScenario` gets a `report-bug` with the fingerprint and `representativeRunIds`; if the owner is agent-manager, W3 here | the row that moved |
| None | any measure row `unreliable` for the sample-size reason | No route; journal the sample size. Do not widen the window to manufacture a sample | none |
| Filing | a detector below its corpus threshold | Corpus route: read the failing cases, fix the detector or add labelled near-miss cases with a reason; never edit a threshold | classifier accuracy |

### 6. Anti-gaming

`improvement-do-and-dont` §1 and its three DON'T subheadings (tagged test, known-issue ledger, suppression) and §2 (the skeptic test) apply verbatim. Agent Manager's own gaming moves, each worth zero credit and a review flag:

- Lowering a detector threshold, or relabelling a corpus case to make a detector pass.
- Setting `ownerConfidence` to `manifest-derived` from a substring match instead of a manifest entry.
- Writing `after_value` equal to `before_value` so a finding reads `ineffective` instead of `not_yet_measurable`.
- Publishing a fingerprint below 3 distinct runs to move the cadence row.
- Excluding heartbeat runs from `run-success-rate` without declaring the filter in the reading.
- Widening a measure window to reach the minimum sample.

### 7. Evidence

One `vrooli-memory journal note --kind work-record` per cycle:

```
--trigger  "<goal> cycle <n>: <row> <reading> vs <band>"
--approach "<route row text>"
--evidence "<before> -> <after> on <sensor command>"
--outcome  "<in band | filed <ref> | published <n> fingerprints | unavailable: <reason>>"
```

A sensor unavailable for three cycles is a `docs/internal/PROBLEMS.md` entry with the three dated readings. Filings against other owners use `report-bug` with the fingerprint, `representativeRunIds`, and `ownerConfidence` as the observation.

### 8. Stop rules

| Condition | Action |
|---|---|
| A detector below its corpus threshold | Only the corpus route runs this cycle |
| A row reads `unavailable` | Journal; do not estimate; after three cycles, PROBLEMS.md and W2 |
| A measure reads `unreliable` | Journal the validity reason; the row's band is not evaluated this cycle |
| A route needs a grant or an operator decision (`apply-investigation`) | Stop and request it through the session path |
| Every readable row in band for two consecutive cycles | Propose close-out to the operator; stop |
| The session's inference or delegation ceiling is reached | Stop; journal; do not open a new session to continue |

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `setpoint-read` reports every row unavailable | program-runtime restarted and the CLI resolved a stale port | `vrooli scenario status program-runtime`; `PROGRAM_RUNTIME_API_PORT` | Set the port for this cycle; file W3 for auto-detect |
| `setpoint-read` measures rows fail with `proto: syntax error ... unexpected token` | The window was passed as a string | the program's `window=` argument | Pass `{"token": "TIME_WINDOW_TOKEN_LAST_7D"}` |
| `publish-recurring-friction` returns `friction intake is unavailable` | The meta-optimization inbox owner is not reachable | `vrooli scenario status meta-optimization-manager` | The row is unavailable; journal; do not write to the inbox by hand |
| `findings list` is empty while investigations completed | Investigations write findings only through the projection store, and no `publish` ran | `run list --tag-prefix agent-manager-investigation` | Run the publisher; if still empty, W2 against agent-manager |
| `run/report` binding returns `501 not implemented` | The RPC is declared in the manifest and not served | `program-runtime bindings describe agent-manager/run/report` | Read reports with the CLI; file against agent-manager |
| `group_by("suspectedOwnerScenario")` raises `key is missing` in a program | `unknown` episodes omit the field | the episode row keys | Filter with `r.get(...)` before grouping |
