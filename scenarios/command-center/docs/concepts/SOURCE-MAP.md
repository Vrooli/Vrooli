# Source Map

**Status:** living reference. The declarations below are read live at runtime; this document explains what they mean for the board and records the observed state at authoring time. Where this file and a team's record disagree, the record wins.

## The principle

**A room cannot be more instrumented than the team behind it.**

The board's honesty is inherited from its sources. When Broadcast shows nothing real, that is not a Command Center defect — it is `team:marketing-crew` having no aggregator and one capability owned by no scenario at all. Rendering those as six unrelated pipeline gaps would imply six pieces of work where there is one, and would point them at the wrong owner.

This is why every reading carries its source team's declared instrument state (`CC-P0-008`), and why the ranked surface distinguishes *no pipeline* from *no instrument*.

## The fleet, as declared

Read live from each team's `instrument` block. Observed 2026-09-01.

| Team | Instrument | Status | Archetype | What the board can read |
|---|---|---|---|---|
| `meta-optimization` | `meta-optimization-manager` | `live` | coverage-board | The reference case. One address, named projections, denominator-confidence, the two attestation axes. Readable cleanly, with confidence attached to every ratio. |
| `monetization` | `offer-desk` | `live` | production-ledger | Live but ledger-shaped. The release ladder is readable; revenue and subscription attribution remain LPBS-owned projections. |
| `infra-health` | `infrastructure-manager` | `partial` | coverage-board | Wired, with a long stated gap marker: setpoint confidence `SKETCH`, most peer sources untyped, capability availability history unavailable, and the team loop paused since 2026-07-24. Readable with confidence attached. |
| `marketing-crew` | `landing-page-business-suite` | `partial` | production-ledger | LPBS provides dimensioned visitor, referrer, campaign, device, path, and variant projections. Social reach and SEO remain declared gaps. |
| `director-swarm` | *two addresses* | `partial` | production-ledger | The team this board serves. `swarm-manager` holds portfolio state and is read as a source; this scenario becomes the address. |
| `scenario-qa` | *none* | `none` | coverage-board | "test-genie, scenario-auditor, scenario-completeness-scoring and tidiness-manager each hold part of the answer and no denominator is authored." No control loop, no room today. |

## What each room inherits

| Room | Sources | Inherited limit |
|---|---|---|
| Mission Control | Vrooli control plane, `swarm-manager` | Mostly `NOW`. The best-instrumented room, because scenario health and swarm throughput are both live. |
| The Hive | Vrooli control plane, `meta-optimization-manager` | Mostly `NOW`. Usage frequency is `MISSING` — nothing anywhere collects invocation counts. |
| The Forge | `swarm-manager` | Fully `NOW`-capable. The only room whose sources are all live today. |
| Ledger | `offer-desk`, `money-ledger`, `landing-page-business-suite` | The monetization instrument is `live` but exposes no revenue surface, so readings are `IN-REACH`, not `MISSING`. A pipeline, not a control loop, is what is absent. |
| Broadcast | `landing-page-business-suite` | LPBS provides the traffic projections; unimplemented social and SEO capabilities remain explicit gaps. |
| Panorama | Composed from the five above | Inherits the worst state of its inputs, per input. Cannot be more honest than what feeds it. |

## Reading a source

Three rules, in order of preference:

1. **Read the team's instrument** if it declares one with status `live` or `partial`, over its standard verb.
2. **Read named scenarios directly** where a team's instrument is absent but `coversScenarios` names real sources — accepting that this is a partial read, and saying so through denominator-confidence.
3. **Report the absence** where neither is possible. A team with no instrument and no readable source is a `MISSING` finding owned by that team, dated, and ranked as a control-loop gap rather than a plumbing gap.

Never re-implement a source's measurement. Never cache a derived set. Never let an unreadable source silently become a zero.

## Why this table will change, and should

Three of six teams have a partial or absent instrument. As they mature, rooms fill in without a Command Center release — that is the property the derived board shape exists to provide ([OUTCOME-TAXONOMY.md](OUTCOME-TAXONOMY.md)).

The inverse is also true and worth stating plainly: **this board makes the fleet's instrumentation maturity visible.** Broadcast being empty is a legible, dated finding against a named team, sitting on a wall where people look at it. That is a more useful artefact than a full room would be, and it is the mechanism by which the board generates pressure to build the sensors it says are missing.

## Cross-references

- [INSTRUMENT-MODEL.md](INSTRUMENT-MODEL.md) — what an instrument is and what it owes
- [COVERAGE-MODEL.md](COVERAGE-MODEL.md) § The third axis — how source maturity qualifies a coverage status
- [OUTCOME-TAXONOMY.md](OUTCOME-TAXONOMY.md) — how these declarations become the board's shape
- `path:docs/agent-system/TARGET_MODEL.md` § 9 — the deviation catalogue, including D1 "no instrument"
# Operational source projection

Command Center treats `.vrooli/service.json` as the runtime dependency
authority. The API projects its enabled scenario dependencies through
`capability-registry-go` and adds the Vrooli control plane as a distinct
`control_plane` integration. Current read paths are intentionally narrow:

| Integration | Read surface | Feature evidence | Failure meaning |
|---|---|---|---|
| `vrooli-core` | typed `ScenarioControlPlaneService.ListScenarios` (Connect); `/health` only for liveness | scenario inventory and health | control-plane availability |
| `swarm-manager` | typed `StatsService.GetPortfolioStats` (Connect); `/health` only for liveness | throughput and agent statistics | producer availability and independently proven feature state |
| `landing-page-business-suite` | typed `MetricsService.GetAnalyticsSummary` (Connect); `/health` only for liveness | funnel and revenue selectors | producer availability; absent surface is not guessed |
| `prompt-manager` | typed `MemberflowService.GetInstruments` transmitter | team instrument | owner guidance when the typed transmitter is unavailable |

The Swarm and LPBS read paths above remain producer-owned aggregate projections.
Their canonical aggregate reads use generated Connect contracts through
producer-owned typed projections; REST is retained only for the operational
health probe. A typed adapter rejects undeclared paths instead of falling back
to an untyped producer surface. Swarm and LPBS carry producer-owned observation
time through their generated responses. Prompt Manager's objective and
instrument reads also use its generated Connect contract.

The browser calls Command Center only. Explicit integration reads use the
generated TypeScript Connect client and shared protobuf response; the remaining
board, room, Focus, and open-loop projections use Command Center's read routes.
Source strips and the Focus panel read the same integration snapshot exposed by
Command Center; they do not infer readiness from a metric binding. Metric coverage remains authored in
`config/outcome-registry.json`, while trust is qualified from producer-owned
observation time. A response without observation metadata is `UNTRUSTED`, not
freshened with the consumer fetch time.
