# Framework Health Targets

**Status:** canon. The agent system's yardstick for itself: is the framework the teams run on getting more coherent, more checkable, and less entropic over time?

This file covers **framework health only** — contract conformance, declaration integrity, coupling visibility, canon coherence, skill conditioning quality, and team orientation cost. It deliberately does **not** state what Vrooli is working toward. What the system is for lives in `path:docs/director-swarm/strategy/OBJECTIVES.md`; how progress toward it is observed lives in `path:docs/director-swarm/evidence/OUTCOMES_CHARTER.md`; per-team reliability targets live in `path:docs/infra-health/strategy/RELIABILITY_TARGETS.md`. A target that answers "is the work worth doing" belongs there, not here. A target that answers "can the system still be reasoned about" belongs here.

Two targets below sit closest to that boundary, and both stay on this side of it. **Objective coverage** does not score the objectives — that is not this file's business — it scores whether the *join* between stated intent and team structure is intact in both directions. That is a framework property: a roster that no longer derives from intent is exactly the kind of incoherence this file exists to catch, and it is invisible to every other sensor here.

**Open-loop target count** is the second. It counts how many targets in this very sensor map have no working instrument, which is a property of the framework's own observability and belongs here by the same logic. `path:docs/director-swarm/strategy/OBJECTIVES.md` cites this count as the measure for objective `I3` (Enablement), but the citation runs one way: the measurement is defined and owned here, and the objective borrows it. Nothing about `I3`'s priority or worth is stated in this file.

Honesty flags use the same vocabulary as `path:docs/infra-health/strategy/RELIABILITY_TARGETS.md` §"Honesty flags" — `measured`, `estimate`, `aspirational`, `pending-baseline`, `pending-telemetry`. Read the definitions there; they are not restated here.

## Why this file exists

Every team in the swarm is supervised by some loop. The framework the teams run on was not. Its quality was assessed only when the operator noticed something felt wrong, which meant improvements arrived by chance rather than by measurement — the same open-loop failure the infra-health control-loop work named for the platform, applied one layer up.

The rule that makes this file load-bearing is the same one infra-health adopted: **you cannot regulate what you cannot observe.** A target with an empty sensor cell is open-loop and is honest about it.

## Sensor map

Every target names its **sensor** (the exact command that observes it), its **deadband** (the band inside which no finding is raised), and its **actuator** (what work fires when out of band). Observed values carry the date they were read; they are dated observations, not live state, per `OPERATING_GRAPHS.md` §"State belongs to scenarios".

`prompt-manager graph audit` reads every sensor below in one call. It collects the graph-API sensors in process and reports the rest as `external` or `no-sensor` rather than dropping them, so a clean sweep never reads as fuller coverage than it has. Its printed tally names all four states — out of band, unsensored, not collected, total — because a summary that counted only the first two let an uncollected target read as a clean one.

**Trend targets need the previous cycle.** Several deadbands here are comparisons ("downward trend", "no team's cost rises"), and one reading cannot decide them. Pass the prior cycle's artifact to band them:

```bash
prompt-manager graph audit --out cycle-N.json          # this cycle's readings
prompt-manager graph audit --baseline cycle-N-1.json   # banded against the last one
```

Without `--baseline` those targets report their reading and stay open-loop, which is honest but permanent — three of them sat at `pending-baseline` indefinitely while the readings they needed were being written to an audit record nothing read back. A malformed or missing `--baseline` path fails the sweep rather than silently reverting to no comparison.

Each open-loop target also carries `gap_opened_on` and `gap_open_days`, parsed from the leading date of its gap marker, so an intentionally visible hole can be told apart from an overdue one.

| Target | Sensor | Deadband | Actuator | Observed 2026-07-25 |
|---|---|---|---|---|
| Operating-model contract validity | `prompt-manager graph operating-model validate` | 0 errors, 0 warnings | `framework-update`, or the owning team's work type | 0 errors, 0 warnings (`measured`) |
| Contract-mode coverage | `prompt-manager graph operating-model list` — count of teams at `mode: contract` | every team with a PoR is `contract`; an exemption needs a dated rationale in that team's `## Current Implementation Gaps` | `framework-update` | 6 of 6 (`measured`) |
| Graph/runtime drift | `prompt-manager graph operating-model diff` | 0 diff items | the owning team's work type | 0 items (`measured`) |
| Topic-flow declaration integrity | `prompt-manager graph topics` | 0 errors | `framework-update` or a member `topics.json` fix | 9 errors, all `actual_writer_undeclared`, one per (member, topic family) (`measured`) |
| Member-document conformance | `prompt-manager graph topics` — `member_doc_*` findings | 0 errors; recommended-section gaps are reported, not banded | `framework-update` for errors; the owning team's work type for recommended-section gaps | 0 errors, 21 recommended-section gaps (`measured`) — 15 `HEARTBEAT.md` without `## Stop Conditions`, 6 `RESPONSIBILITIES.md` without `## Primary Duties` |
| Prose/declaration coherence | `prompt-manager graph topics` — `prose_topic_leak` warnings | 0 warnings on declaration-bearing surfaces | `framework-update` — tighten the inferred-backtick matcher, then triage the residue | 235 warnings (`measured`); the count **over-reports drift** — the inferred-backtick pattern still matches path-shaped references in skill prose (Go import paths, test-helper directories), so the reading is not yet a defect count (`pending-baseline` on precision) |
| Cross-team coupling visibility | `prompt-manager graph map --json` for composed edges, plus `prompt-manager graph operating-model coverage` for `runtime_only` | 0 runtime-only relationship rows; retain every composed edge | the producing team's work type | 13 composed edges across 6 teams, 0 runtime-only rows (`measured`, 2026-08-10). Until 2026-08-10 this row banded on `len(edges) > 0`, which could only detect total collapse of the map — the runtime-only clause was stated and unchecked |
| Canon coherence | `path:scenarios/prompt-manager/test/agent_system_canon_test.sh`, run by `graph audit` when a repository root is in reach | all assertions pass | `framework-update` | **9 pass, 1 fail** — `proto-contract-audit` is tagged `audit-technique` with no paired PoR doc in `path:docs/scenario-qa/methods/audit/` (`measured`, 2026-08-10). This target was `external` and reported "not collected" while failing, which reads exactly like a pass; the sweep now runs it and degrades to `external` only when the script or the repo root is unreachable |
| Statically unreferenced skills | `prompt-manager graph orphaned-skills` — skills with zero incoming reference-graph edges | none; this target is **reported, not banded**, until the sensor joins all three reachability classes (below) | `skill-deprecation` or `skill-improvement`, and only after the join confirms the skill is also undiscovered and unread | 53 statically unreferenced (`measured`); **not a defect count** — three of them were served 129, 41, and 25 times in the last 7 days. Cross-check against `prompt-manager skill-usage` before treating any as dead (`pending-baseline` until a full read-telemetry window has accumulated) |
| Discovery budget pressure | `prompt-manager discovery-metrics --json` — `overBudgetRate` | under 25% of budgeted calls over budget | `skill-improvement`, owned by `skill-optimizer` | 71% of 630 budgeted calls (`measured`) |
| Prompt structure invariant | `go test ./heartbeat/ -run TestAssembledPromptEmitsOnlyRegisteredSectionHeadings` | every level-one heading in an assembled prompt is a registered prompt section | `framework-update`; repair the prompt builder, never the section registry alone | `pending-telemetry` — enforced in-process, no corpus-wide CLI sensor |
| Catalogued rules with no finding | `prompt-manager graph rules` — `silent` | downward trend against the previous cycle; a single reading is a baseline, not a finding | screen each silent rule on whether a test makes it fire and whether a failure names something specific to change | `pending-baseline` — needs the previous cycle's reading to band |
| Objective coverage | `prompt-manager graph objectives` — joins `path:docs/director-swarm/strategy/OBJECTIVES.md` against `team.json::objectivesServed` in both directions | 0 objectives unserved without a dated gap marker; 0 teams tracing to no objective; 0 declaration errors | `outcome-direction` or `capability-work` in `director-swarm` | 1 unserved, declared: `T2` (no team, no evidence source); 0 unattached teams; 0 errors, 1 warning (`measured`, 2026-08-09). `I3` was the second unserved entry through 2026-07-30 and is now owned by `director-swarm` as ranking authority |
| Open-loop target count | `prompt-manager graph audit` — count of targets reported `no-sensor`; the target names are carried in that row's `detail` | downward trend across audit cycles; a single reading is a baseline, not a finding. Not banded against a fixed count — a target enters this set the moment it is honestly declared, so growth can mean new honesty rather than new decay | `capability-work` in `director-swarm`, sequenced by the instrument rule (`path:docs/director-swarm/strategy/PORTFOLIO_PHILOSOPHY.md`) | `pending-baseline` — **7 of 18** (`measured`, 2026-08-09): 2 `pending-telemetry` (prompt structure invariant, PoR entropy) and 5 `pending-baseline` (statically unreferenced skills, skill conditioning quality, team orientation cost, catalogued rules with no finding, and this target). This row is a trend target with no prior cycle, so it is `no-sensor` and **counts itself** — an open-loop count with nothing to diff against is open-loop. First baseline; the band needs a second cycle. Dated gap marker **2026-08-09** |
| Skill conditioning quality | per-skill only: the divergence probe (`skill-validation` §3.3). No corpus-wide aggregation exists — the instrument is built, the sweep is not | 0 unreviewed divergence regressions once the corpus sweep exists | `skill-improvement` | `pending-baseline`; dated gap marker **2026-07-27**: build a corpus-wide divergence-probe sweep, blocked on a stable per-skill evaluation inventory |
| Skill-experiment loop liveness | `prompt-manager experiment list`, collected in-process from the same API | at least one concluded experiment per audit cycle once the loop is live | `skill-experiment-promotion` | `pending-baseline` — 0 experiments exist, so the deadband's "once the loop is live" precondition is unmet and there is no per-cycle rate to band (`measured`, 2026-08-10). Dated gap marker **2026-08-10**: band once the first experiment is created |
| PoR entropy | state-in-prose telemetry (not implemented; the defect class is audited by judgment today) | 0 unclassified state-in-prose findings once telemetry exists | `framework-update` | `pending-telemetry`; dated gap marker **2026-07-27**: build state-in-prose telemetry, blocked on a stable document-state classification contract |
| Team orientation cost | `prompt-manager graph orientation-cost` — `members×10 + topics×2 + canonLines÷50`, read against declared scenario coverage | no team's orientation cost rises across an audit cycle in which its scenario coverage grew | `team-capability-consolidation` | `measured` 2026-08-09 (`director-swarm` 62, `infra-health` 51, `marketing-crew` 113, `meta-optimization` 206, `monetization` 135, `scenario-qa` 48). The 2026-07-30 readings are **not comparable** and are retained only as history: they were taken while the sensor description claimed a fourth "declared work types" term that no team ever declared and `orientationComposite()` never computed, and `meta-optimization` has since had its canon root declared (2,396 → 4,823 lines), which is a measurement fix rather than growth. 2026-08-09 is the first baseline the band can diff against |

## Deadband rule

**A deadband states the target, not the current reading.** A deadband set equal to whatever the sensor last observed reports in-band while the defect stands, and can only ever detect growth — never the standing problem. Two targets here carried that shape (prose coherence at "519 or fewer", skill reachability at "54 or fewer") and both read `ok` for it. This is the same failure `infra-health`'s contrarian rubric names as **dead-sensor evidence** and **target drift**, applied one layer up to the framework's own sensor map.

When a target is out of band and the work to close it is not scheduled, that is what the Observed column and the audit record are for. An out-of-band reading with a named actuator is a healthier state than a green one that means nothing.

Where a reading is not yet a defect count — the instrument fires, but its precision is not established — say so with `pending-baseline` in the Observed cell rather than widening the deadband to swallow it. The deadband is the target; the honesty flag carries what stands between the reading and the target.

**Three reachability classes, one sensor.** A skill is reachable three different ways, and only the first is currently instrumented:

| Class | How an agent arrives | Instrument |
|---|---|---|
| Static reference | a member file, skill, doc, or Action names it | `prompt-manager graph orphaned-skills` |
| Discovery | AI search returns it for an intent | `prompt-manager skill-usage` — `returned` |
| Read | an agent runs `prompt-manager skill read` on it | `prompt-manager skill-usage` — `reads` / `demandReads` |

Counting only the first class and calling the result "unreachable" is what made this target a false-positive generator: `skill-authoring` is canon's designated Steer authoring guide and appears in the list. **A skill is a deprecation candidate only when all three classes are zero over the window.** The row reports and does not band until a full window of read telemetry has accumulated, because a deadband nothing can reach trains readers to skip the row.

Two properties of the read class are load-bearing:

- **`demandReads` counts `agent-member` reads only.** An optimizer reading a skill to audit it is not demand, and counting it would make a usage-weighted selection ladder reinforce its own choices — visiting a skill would raise its rank. Total `reads` stays visible beside it.
- **`returned` and `reads` are separate numbers on purpose.** Their ratio is search precision. A skill returned often and read rarely is not popular; it is a skill discovery keeps offering and agents keep declining, and it spends budget on every call it loses. `prompt-manager skill-usage` reports that set as "returned but never read".

**The objective join is one sensor reading two documents.** Objective coverage is the only target whose sensor spans a plan of record and the runtime store, because the coverage rule is a join and a join has two ends. `OBJECTIVES.md` stays operator-authored prose — it is the single operator-specific layer in the system and must not become a config file. What is declared is the *edge*: `team.json::objectivesServed` names the ids, and the sensor parses the objective table to check the ids resolve and that both directions agree. One half of the upward direction stays outside this sensor: outcome categories are Command Center dashboard ids in the outcomes charter, not a store surface, so `agent-system-audit` §"Phase 4" still reads that one by hand.

**One deadband may state a direction instead of a level.** Team orientation cost is the only target here whose band is a trend. An absolute ceiling on roster size or canon length would be arbitrary — a team that owns more is allowed to carry more. What is never allowed is for a team to get harder to orient in the same cycle its scenarios absorbed more of its work, because that is the ratchet running backwards: capability was added and none of it paid for itself in simplification. The band therefore reads two quantities against each other over one cycle, and the audit record is what makes the trend readable at all.

**One band per actuator.** When a sensor produces two readings whose fixes route to different actuators, band only the one this file's actuator can close and report the other in Observed. Member-document conformance is the worked example: a missing required heading is mechanical and closes through `framework-update`, while a missing recommended section needs the owning team to author judgment no validator can infer. Banding both would put the wrong fix in the actuator column; dropping the second would let the band read as fuller coverage than it has.

## Reading rule

Cite the sensor, never copy its answer. The Observed column is a dated observation refreshed by an approved work item; anything that needs a current number runs the command. This is the same read-time rule that governs team PoR content in `OPERATING_GRAPHS.md` §"State belongs to scenarios".

## Audit record

A single reading answers "is the framework healthy now". It cannot answer "is the framework getting healthier", which is the question this file exists to settle. Each audit cycle therefore writes its full sensor readings and findings to `topic:framework-health-audit/<YYYY-MM-DD>`, owned by `team-agent-optimizer` (`path:docs/meta-optimization/operating/OPERATING_MODEL.md` §"Topic Catalog"). The topic is the trend; the Observed column above is only the latest spot value.

Produce the record from the sweep rather than by hand:

```bash
prompt-manager graph audit --json
bash scenarios/prompt-manager/test/agent_system_canon_test.sh   # external sensor
prompt-manager experiment list                                   # external sensor
```

The sweep now carries the objective join and the orientation-cost composites, so the record is self-sufficient for both. One document read remains: score every outcome category in the charter's team contribution map against an objective. `prompt-manager skill read agent-system-audit` §"Phase 4" is the procedure.

**Record the orientation-cost composites in full.** They are the only target whose value is worthless in isolation — the band compares this cycle against the last one, so a record that summarises them has destroyed the sensor rather than reported it.

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
