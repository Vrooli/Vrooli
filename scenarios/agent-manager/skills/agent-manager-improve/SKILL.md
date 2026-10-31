---
name: "agent-manager-improve"
description: "Regulate Agent Manager against its setpoint: command outcome coverage, investigation completion, skill and program identity on invocation facts, and the tool friction rates. Routes each out-of-band row to a curation move, a work-ladder rung, or an owner."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["agent-manager", "improve", "self-improvement", "control-loop", "setpoint", "friction", "findings", "meta-optimization"]
  icon: "gauge"
  status: "active"
  revision: 3
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  requires:
    scenarios: ["agent-manager", "program-runtime", "prompt-manager", "vrooli-memory"]
    commands: ["agent-manager usage recommendations", "agent-manager usage friction", "agent-manager run list", "agent-manager run report", "agent-manager watch policy-get", "agent-manager watch policy-outcomes", "agent-manager conversation index status", "agent-manager measures get", "search-hub evals runs", "search-hub federation status", "search-hub insights insights", "program-runtime bindings condition", "program-runtime programs submit", "prompt-manager skill read", "vrooli-memory journal note"]
  origin:
    kind: "authored"
---
## Practice focus: Agent Manager Improve

Regulate Agent Manager, the run and friction instrument, against the setpoint below. The plant is the evidence chain from a run to an owner: runs, invocation facts, tool-friction profiles, findings, and the durable friction measures. This skill is read by an agent whose task is Agent Manager itself (`goal-loop`, or the meta-optimization heartbeat that owns the friction inbox). It never edits another scenario; it files.

Required reading:
- `prompt-manager skill read agent-manager` — the usage skill; every command named here is documented there or in the CLI.
- `prompt-manager skill read improvement-do-and-dont` — anti-gaming, cited by section below.
- `prompt-manager skill read scenario-work-ladder` — where code routes go.
- `path:scenarios/agent-manager/docs/reference/classifier-accuracy.md` — the golden corpus this scenario owns.

### 1. Focus and scope

**In scope:** the setpoint rows below; reporting owner-level friction (`usage friction-report`); filing ladder rungs against Agent Manager; filing `report-bug` against the scenarios the tool findings name as owners.

**Out of scope:** editing another scenario's manifest, skills, or programs (file instead); the meta-optimization inbox's own triage (that owner drains it); the usage skill's content; changing a detector threshold without a recorded derivation.

### 2. Setpoint

Bands are targets. Readings are dated observations; re-read them every cycle with `run agent-manager.setpoint-read`.

| Row | Sensor | Band | Reading |
|---|---|---|---|
| command-outcome-coverage | `agent-manager measures get tool-friction --window last_7d` → `coverage.vrooli_outcome_known` ÷ `coverage.vrooli_commands` (Vrooli CLI calls whose outcome is known) | ≥ 0.95 | read it with `setpoint-read`; below the band the failure and shape rates are not banded |
| investigation-completion-rate | `agent-manager run list --tag-prefix agent-manager-investigation --json` → `RUN_STATUS_COMPLETE` ÷ terminal | ≥ 0.80 | `unreliable:no_terminal_runs_with_tag_prefix` until investigation runs with that tag finish |
| skill-and-program-identity-on-facts | `agent-manager run report <id> --facts` → facts carrying a `skill_id` or `program_id` | every fact whose executable is `program-runtime` or `prompt-manager skill read` carries the id | unavailable, reason `pending_telemetry`: the fact schema has neither field |
| tool-friction-validity | `agent-manager measures get tool-friction --window last_7d` → `validity` | `available` | `unreliable` for a small sample; the row carries the reason |
| bypass-calls | `agent-manager measures get tool-friction --window last_7d` → `bypass_calls` (calls that reached a scenario's database, port or state directory around its CLI) | pending-baseline; direction falling | descriptive until a week of profiles sets the baseline |
| run-success-rate | `agent-manager measures get run-success-rate --window last_7d` → value (a rate) | ≥ 0.85 | read with `setpoint-read` |
| shape-guess-rate | `agent-manager measures get tool-friction --window last_7d` → `shape_guess_rate` (Vrooli CLI calls that hit an unknown command, flag or missing argument, or looked up help to recover) | ≤ 0.03 | `unreliable:outcome_coverage_below_band` while command-outcome-coverage is below its band |
| vrooli-cli-failure-rate | `agent-manager measures get tool-friction --window last_7d` → `vrooli_failure_rate` (failed ÷ succeeded-or-failed Vrooli CLI calls; no-match, refusals, verify verdicts excluded) | ≤ 0.05 (provisional: re-derive after a week of v26 facts) | `unreliable:outcome_coverage_below_band` while command-outcome-coverage is below its band |
| polled-share | `agent-manager measures get tool-friction --window last_7d` → `polled_share` (calls in a series of one read command run three or more times within ten minutes) | pending-baseline; direction falling | descriptive until a week of profiles sets the baseline |
| supervision-safety | `run agent-manager.friction-digest` → `supervision_safety_violations` | 0 | read from durable supervision outcomes; any non-zero value blocks promotion |
| supervision-calibration | `run agent-manager.supervision-experiment-read` → owner replay gates and evidence coverage | non-regressing against the active policy replay | compare candidates offline; never tune the policy during a family run |
| supervision-outcome-coverage | `agent-manager watch policy-outcomes` → outcomes with decision, action, child, family, and evidence | 1.0 | incomplete linkage is an Agent Manager outcome-contract defect |

### 3. Sensors

Read base rows and supervision routing rows through `run agent-manager.setpoint-read` (contract: `.vrooli/program-runtime/setpoint-read.json`). Rows the program marks `unavailable` are read by hand only with the exact command in the table, and the hand reading is journaled as such. Rows whose sensor has no program binding, or whose fact carries no identity field to read (`run report <id> --facts`), are unavailable in-program by construction. Measures bindings take `window={"token": "TIME_WINDOW_TOKEN_LAST_7D"}` in-kernel; the CLI takes `--window last_7d`.

Fleet sensors every scenario has: `program-runtime bindings condition` for Agent Manager's own bindings, and Agent Manager's own friction for `agent-manager` commands through `run agent-manager.friction-digest` with `scenario=agent-manager`.

External sensors outrank self-reported ones: an owner's `report-bug` filing about a wrong attribution outranks `ownerConfidence`.

Supervision rows route explicitly to `agent-manager.friction-digest`. Read its
assessed/unassessed counts and capped-sample flag. Empty or capped samples cannot
prove calibration or zero safety violations. Outcome coverage remains
pending_telemetry until the decision denominator is supplied.

For a policy change, follow `agent-manager-plan-family-supervision`: assess linked
outcomes, create a candidate, replay actual retained inputs, run an explicitly
bounded candidate cohort, and promote only through the owner gates. An evaluator
code change requires a new pinned artifact. Never replace replay with stored
predictions or caller-supplied rollout counts.

### 4. Golden corpora

| Suite | Floor | Derivation |
|---|---|---|
| `api/internal/runsignal/testdata/classification/all-detectors.labels.json` (42 labelled windows) | every shipped detector at or above its committed per-detector precision and recall threshold | thresholds are committed beside the corpus; scored only by `runsignal.ClassificationAccuracy` through `GOWORK=off go test ./internal/runsignal/...` from `scenarios/agent-manager/api` and the `agent-conformance` test-genie phase |

A detector below its threshold is a stop: no other route runs until the corpus route (§5) has been taken and the re-read is at or above threshold. A threshold is never lowered by this skill. A threshold is raised only with a new derivation over the same corpus version, recorded in the labels file.

#### 4.1 Conversation-recall evidence

Read this evidence set when a recall failure, privacy report, or repeated manual search enters the cycle. Do not infer the owning layer from one empty result.

| Evidence | Read | Decision |
|---|---|---|
| Projection coverage and freshness | `agent-manager conversation index status` | A missing generation, lag, orphan count, or degraded leg is Agent Manager evidence. |
| Direct retrieval quality | `search-hub evals runs agent-manager.runs.primary --tier provider_direct --limit 2` | A comparable direct regression is Agent Manager indexing, access, or retrieval evidence. |
| Federated selection and ranking | `search-hub evals runs agent-manager.runs.primary --tier federated --limit 2`; then `search-hub federation status` | Direct pass plus federated fail is Search Hub routing, ranking, mapping, or eval evidence. |
| Live behavior | Use the operator-local `agent-manager.runs.live-overlay` when registered | The overlay checks retained personal data without adding personal identifiers to the committed corpus. Absence is unknown, not failure. |
| Query outcomes | `search-hub insights insights --window 7d`; inspect the `agent-manager.runs` provider row | Read no-result, degradation, and latency as aggregate evidence. Do not record query text or snippets. |
| Governed recall outcome | Run `agent-manager.conversation-recall` through Program Runtime and inspect its typed status, degradation, and next-action category | A partial leg keeps its evidence. Repeated successful multi-query recovery supports keeping orchestration in the program. |

Route one defect to one owner:

| Observed defect | Owner route |
|---|---|
| The caller chose the wrong surface, mode, filters, or interpreted copied text as unique | Patch and validate `agent-manager`; this is a usage-judgment defect. |
| Agents repeatedly hand-run complementary queries, merge duplicates, and fetch context | Patch and validate `agent-manager.conversation-recall`; this is governed orchestration. |
| Coverage, freshness, access, deletion, context, or direct retrieval is wrong | Run `scenario-work-ladder` against Agent Manager. |
| Direct evidence passes but provider selection, mapping, ranking, or an eval verdict is wrong | File `report-bug` against Search Hub with the suite and run IDs. |
| Search Hub evidence is correct but Answer cell 37 or its condition is wrong | File `report-bug` against Meta Optimization Manager with the provider status and eval run IDs. |

Remove an upper-layer workaround after the lower owner exposes the stable behavior. A direct-search fix removes compensating query advice from the usage skill. A repeated multi-step flow moves into the program and leaves one program-selection leaf in the skill. A Search Hub or projection fix removes parallel routing or condition logic from Agent Manager artifacts.

The supervision behavior corpus is
`api/internal/supervision/testdata/supervision_behavior.py`, executed by the Go
supervision test. It covers productive event batches, legitimate parked waits,
missing friction, repeated friction, terminal results, reset and isolated failure.
`replay_test.go` proves candidate execution, unknown exclusion, attribution and
artifact immutability. These are correctness gates, not an operator-benefit floor.

### 5. Actuators and ladder routing

One route per row per cycle. When two rows' predicates hold for the same reading, take the first matching row from the top; re-read next cycle before taking the next.

`Actuator` rows are moves the agent running this skill performs in-cycle without a diff (an investigation request, `usage friction-report`). `Filing` rows hand off: a work-ladder rung against agent-manager, or `report-bug` against the scenario a tool finding names. `None` rows are honest readings with no move.

| Kind | Row out of band | Route | Sensor that should move |
|---|---|---|---|
| Filing | command-outcome-coverage below band | W3 against agent-manager: name the runner whose commands lack an outcome (`measures get tool-profiles` → `unknown` per profile); a codec that drops the command's result is the defect, never an inferred outcome | command-outcome-coverage |
| Filing | investigation-completion-rate below band | Read the failed investigation runs with `run report <id>`; a preflight or resource-role failure is W3 here; a prompt-contract failure is `skill-validation` on `agent-manager-process-investigation` | investigation-completion-rate |
| Filing | skill-and-program-identity-on-facts pending | W3 against agent-manager: carry `skill_id` when the executable is `prompt-manager skill read` and `program_id` when it is `program-runtime`; subscribe to program-runtime and ai-gateway typed events so a submit and a route carry the run id | skill-and-program-identity-on-facts |
| None | tool-friction-validity `unreliable` for the sample-size reason | No route; journal the sample size | tool-friction-validity |
| Actuator | bypass-calls above its baseline | `usage friction` → the `bypass` finding names the scenario and method (database, port, state directory); `usage friction-report <fingerprint>` files it so the owner can add the read agents reach for | bypass-calls |
| Actuator, then operator | run-success-rate below band, failures concentrated in one profile (`measures get profile-breakdown`) | `investigation start --file request.json` naming the top failures as the subject run_ids; apply only with an operator decision | run-success-rate |
| Filing | run-success-rate below band, failures spread across profiles | W2 against agent-manager: the runner or sandbox layer; read `run report` for the exit code class first | run-success-rate |
| Actuator | shape-guess-rate, vrooli-cli-failure-rate, or polled-share above band | `usage friction` lists owner-level findings ranked by their share of the window's harm; `usage friction-report <fingerprint>` files the top open one with its owner (the inbox deduplicates on the fingerprint); if the owner is agent-manager, W3 here | the row that moved |
| Actuator | quota burn or tool friction worth acting on, and no single row above names its owner | `usage recommendations` ranks quota levers, tool findings, opportunities and gated operator corrections by the share of their own kind of harm; act on or file the top action with its owner, and use its `verify` command as the check | the action's own effect next window |
| None | any measure row `unreliable` for the sample-size reason | No route; journal the sample size. Do not widen the window to manufacture a sample | none |

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
--outcome  "<in band | filed <ref> | reported <n> friction problems | unavailable: <reason>>"
```

A sensor unavailable for three cycles is a `docs/internal/PROBLEMS.md` entry with the three dated readings. Filings against other owners use `report-bug` with the fingerprint, `representativeRunIds`, and `ownerConfidence` as the observation.

### 8. Stop rules

| Condition | Action |
|---|---|
| A detector below its corpus threshold | Only the corpus route runs this cycle |
| A row reads `unavailable` | Journal; do not estimate; after three cycles, PROBLEMS.md and W2 |
| A measure reads `unreliable` | Journal the validity reason; the row's band is not evaluated this cycle |
| A route needs a grant or an operator decision (`apply-investigation`) | Stop and request it through the session path |
| Every required row has reliable evidence and is in band for two comparable cycles | Propose close-out to the operator; stop |
| The session's inference or delegation ceiling is reached | Stop; journal; do not open a new session to continue |

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `setpoint-read` reports every row unavailable | program-runtime restarted and the CLI resolved a stale port | `vrooli scenario status program-runtime`; `PROGRAM_RUNTIME_API_PORT` | Use control-plane discovery and readiness; preserve unavailable evidence until the owner resolves the stale route |
| `setpoint-read` measures rows fail with `proto: syntax error ... unexpected token` | The window was passed as a string | the program's `window=` argument | Pass `{"token": "TIME_WINDOW_TOKEN_LAST_7D"}` |
| `run/report` binding is unavailable | The typed RPC or agent-manager service is not reachable | `program-runtime bindings describe agent-manager/run/report` and `agent-manager run report <run-id> --json` | Confirm the service is healthy, then inspect the typed not-found or transport error |

### Supervision evidence validity

Use `agent-manager.supervision-experiment-read` for a candidate's owner gates,
pinned code/inference identity and population denominators. The friction digest's
supervision sample spans scenarios and is labelled separately from scenario
friction. Do not use that sample as a scenario-specific efficacy estimate.

Keep completion impact unknown unless an assessment explicitly sets
`completion_impact_observed` and cites its measurement. A completed child is not
proof that a nudge caused the completion. Compare independent family outcomes.
Count accepted and applied actions separately. Preserve unassessed observations
until a justified assessment supersedes them. Do not relabel fixtures as live
rollout evidence or promote a candidate to remove an insufficient-evidence gate.

Prospective policy comparison is separate from training memory. Freeze the candidate
before collecting held-out incumbent families. Inspect the owner-reported selection,
five-family minimum, candidate/incumbent errors and safety violations. Only a passing
comparison permits an explicit candidate watch; admission caps the rollout at five
families. Re-evaluate after assessment, and promote only through the owner gate.
A missing sensor remains an open measurement obligation, even when other rows improve.
