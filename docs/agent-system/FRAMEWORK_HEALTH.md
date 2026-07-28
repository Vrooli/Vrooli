# Framework Health Targets

**Status:** canon. The agent system's yardstick for itself: is the framework the teams run on getting more coherent, more checkable, and less entropic over time?

This file covers **framework health only** — contract conformance, declaration integrity, coupling visibility, canon coherence, and skill conditioning quality. It deliberately does **not** state what Vrooli is working toward. What the system is for lives in `path:docs/director-swarm/strategy/OBJECTIVES.md`; how progress toward it is observed lives in `path:docs/director-swarm/evidence/OUTCOMES_CHARTER.md`; per-team reliability targets live in `path:docs/infra-health/strategy/RELIABILITY_TARGETS.md`. A target that answers "is the work worth doing" belongs there, not here. A target that answers "can the system still be reasoned about" belongs here.

One target below is the exception that proves the boundary. **Objective coverage** does not score the objectives — that is not this file's business — it scores whether the *join* between stated intent and team structure is intact in both directions. That is a framework property: a roster that no longer derives from intent is exactly the kind of incoherence this file exists to catch, and it is invisible to every other sensor here.

Honesty flags use the same vocabulary as `path:docs/infra-health/strategy/RELIABILITY_TARGETS.md` §"Honesty flags" — `measured`, `estimate`, `aspirational`, `pending-baseline`, `pending-telemetry`. Read the definitions there; they are not restated here.

## Why this file exists

Every team in the swarm is supervised by some loop. The framework the teams run on was not. Its quality was assessed only when the operator noticed something felt wrong, which meant improvements arrived by chance rather than by measurement — the same open-loop failure the infra-health control-loop work named for the platform, applied one layer up.

The rule that makes this file load-bearing is the same one infra-health adopted: **you cannot regulate what you cannot observe.** A target with an empty sensor cell is open-loop and is honest about it.

## Sensor map

Every target names its **sensor** (the exact command that observes it), its **deadband** (the band inside which no finding is raised), and its **actuator** (what work fires when out of band). Observed values carry the date they were read; they are dated observations, not live state, per `OPERATING_GRAPHS.md` §"State belongs to scenarios".

`prompt-manager graph audit` reads every sensor below in one call. It collects the graph-API sensors in process and reports the rest as `external` or `no-sensor` rather than dropping them, so a clean sweep never reads as fuller coverage than it has.

| Target | Sensor | Deadband | Actuator | Observed 2026-07-25 |
|---|---|---|---|---|
| Operating-model contract validity | `prompt-manager graph operating-model validate` | 0 errors, 0 warnings | `framework-update`, or the owning team's decision context | 0 errors, 0 warnings (`measured`) |
| Contract-mode coverage | `prompt-manager graph operating-model list` — count of teams at `mode: contract` | every team with a PoR is `contract`; an exemption needs a dated rationale in that team's `## Current Implementation Gaps` | `framework-update` | 6 of 6 (`measured`) |
| Graph/runtime drift | `prompt-manager graph operating-model diff` | 0 diff items | the owning team's decision context | 0 items (`measured`) |
| Topic-flow declaration integrity | `prompt-manager graph topics` | 0 errors | `framework-update` or a member `topics.json` fix | 9 errors, all `actual_writer_undeclared`, one per (member, topic family) (`measured`) |
| Member-document conformance | `prompt-manager graph topics` — `member_doc_*` findings | 0 errors; recommended-section gaps are reported, not banded | `framework-update` for errors; the owning team's decision context for recommended-section gaps | 0 errors, 21 recommended-section gaps (`measured`) — 15 `HEARTBEAT.md` without `## Stop Conditions`, 6 `RESPONSIBILITIES.md` without `## Primary Duties` |
| Prose/declaration coherence | `prompt-manager graph topics` — `prose_topic_leak` warnings | 0 warnings on declaration-bearing surfaces | `framework-update` — tighten the inferred-backtick matcher, then triage the residue | 235 warnings (`measured`); the count **over-reports drift** — the inferred-backtick pattern still matches path-shaped references in skill prose (Go import paths, test-helper directories), so the reading is not yet a defect count (`pending-baseline` on precision) |
| Cross-team coupling visibility | `prompt-manager graph map --json` | 0 runtime-only relationship rows; retain every composed edge | the producing team's decision context | 14 composed edges (`measured`) |
| Canon coherence | `path:scenarios/prompt-manager/test/agent_system_canon_test.sh` | all assertions pass | `framework-update` | **8 pass, 1 fail** — `proto-contract-audit` is tagged `audit-technique` with no paired PoR doc in `path:docs/scenario-qa/methods/audit/` (`measured`) |
| Skill reachability | `prompt-manager graph orphaned-skills` | 0 unreachable candidates | `skill-deprecation` or `skill-improvement` | 54 candidates (`measured`) — a standing triage backlog, not a clean reading |
| Objective coverage | `path:docs/director-swarm/strategy/OBJECTIVES.md` §"The coverage rule" — read both directions against the charter's team contribution map | 0 objectives unserved without a dated gap marker; 0 teams or outcome categories tracing to no objective | `outcome-direction` or `capability-gap` in `director-swarm` | 2 unserved, both declared: `T2` (no team, no evidence source), `I3` (no owning lane); 0 unattached teams or categories (`measured`) |
| Skill conditioning quality | per-skill only: the divergence probe (`skill-validation` §3.3). No corpus-wide aggregation exists — the instrument is built, the sweep is not | 0 unreviewed divergence regressions once the corpus sweep exists | `skill-improvement` | `pending-baseline`; dated gap marker **2026-07-27**: build a corpus-wide divergence-probe sweep, blocked on a stable per-skill evaluation inventory |
| Skill-experiment loop liveness | `prompt-manager experiment list` | at least one concluded experiment per audit cycle once the loop is live | `skill-experiment-promotion` | 0 experiments (`measured`) |
| PoR entropy | state-in-prose telemetry (not implemented; the defect class is audited by judgment today) | 0 unclassified state-in-prose findings once telemetry exists | `framework-update` | `pending-telemetry`; dated gap marker **2026-07-27**: build state-in-prose telemetry, blocked on a stable document-state classification contract |

## Deadband rule

**A deadband states the target, not the current reading.** A deadband set equal to whatever the sensor last observed reports in-band while the defect stands, and can only ever detect growth — never the standing problem. Two targets here carried that shape (prose coherence at "519 or fewer", skill reachability at "54 or fewer") and both read `ok` for it. This is the same failure `infra-health`'s contrarian rubric names as **dead-sensor evidence** and **target drift**, applied one layer up to the framework's own sensor map.

When a target is out of band and the work to close it is not scheduled, that is what the Observed column and the audit record are for. An out-of-band reading with a named actuator is a healthier state than a green one that means nothing.

Where a reading is not yet a defect count — the instrument fires, but its precision is not established — say so with `pending-baseline` in the Observed cell rather than widening the deadband to swallow it. The deadband is the target; the honesty flag carries what stands between the reading and the target.

**One band per actuator.** When a sensor produces two readings whose fixes route to different actuators, band only the one this file's actuator can close and report the other in Observed. Member-document conformance is the worked example: a missing required heading is mechanical and closes through `framework-update`, while a missing recommended section needs the owning team to author judgment no validator can infer. Banding both would put the wrong fix in the actuator column; dropping the second would let the band read as fuller coverage than it has.

## Reading rule

Cite the sensor, never copy its answer. The Observed column is a dated observation refreshed by an approved decision; anything that needs a current number runs the command. This is the same read-time rule that governs team PoR content in `OPERATING_GRAPHS.md` §"State belongs to scenarios".

## Audit record

A single reading answers "is the framework healthy now". It cannot answer "is the framework getting healthier", which is the question this file exists to settle. Each audit cycle therefore writes its full sensor readings and findings to `topic:framework-health-audit/<YYYY-MM-DD>`, owned by `team-agent-optimizer` (`path:docs/meta-optimization/operating/OPERATING_MODEL.md` §"Topic Catalog"). The topic is the trend; the Observed column above is only the latest spot value.

Produce the record from the sweep rather than by hand:

```bash
prompt-manager graph audit --json
bash scenarios/prompt-manager/test/agent_system_canon_test.sh   # external sensor
prompt-manager experiment list                                   # external sensor
```

The third external sensor is a document read, not a command: score `path:docs/director-swarm/strategy/OBJECTIVES.md` §"The coverage rule" in both directions against the charter's team contribution map. `prompt-manager skill read agent-system-audit` §"Phase 4" is the procedure.

A target outside its deadband in that record is a declared trigger for the `framework-update` decision.

## Ownership

`meta-optimization` owns this file, as it owns the rest of `path:docs/agent-system/`. Targets change by decision, not by drift. When a sensor ships for a `pending-telemetry` row, the flag moves to `pending-baseline` and the first audit that reads it records the baseline.

## Validation rule reference

The operator-facing validation-rule reference is generated from
`memberflow.DefaultRuleCatalog`; it is not a second hand-maintained list of
identifiers or severities. The catalog requires an identifier, group, default
severity, description, and actuator for every registered rule, and registry
construction rejects missing, duplicate, or orphaned entries.

Run the catalog test and renderer from the API module when changing a rule:

```bash
cd scenarios/prompt-manager/api
go test ./memberflow/... -run 'TestDefaultRuleCatalogExactlyMatchesDefaultRegistry|TestRuleCatalogMarkdownIsStableAndComplete'
```

The generated Markdown table is exposed by `RuleCatalog.Markdown()`. Consumers
that need the current reference must render it from that source rather than
copying rule metadata into a document or CLI surface.
