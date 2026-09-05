# Infra Health Plan of Record

This folder is the plan of record for the platform's own health: Vrooli's internal code, lifecycle, runtime reliability, capacity arbitration, and cross-platform readiness. It is owned by the `infra-health` team in prompt-manager and curated through approved operator decisions.

> **The instrument, not this folder, holds the numbers.**
> Reliability targets, sensor definitions, cell status and instrumentation gaps live in
> [`scenarios/infrastructure-manager`](../../scenarios/infrastructure-manager/) and are
> **computed**, not written down here. Read them with
> `infrastructure-manager coverage status`, `condition status`, and `focus next`.
> This folder holds what an instrument cannot compute: how the team operates, what it may
> not do, and which portability debts are tracked. See [§ The instrument](#the-instrument).

> **Loop status:** paused since ~2026-06; recorded in the heartbeat control plane as `paused-manual` on 2026-07-24 (resume via `prompt-manager team heartbeat-control infra-health resume`). Honesty flags reflect the pause. First heartbeat after resume follows the resume protocol in the scanner's `HEARTBEAT.md`.

The local contract is [`manifest.json`](manifest.json), which instantiates the shared plan-of-record shape from [`docs/agent-system/team-plan-of-record.manifest.json`](../agent-system/team-plan-of-record.manifest.json).

## Start here for agents

Use this README first, then choose the module that matches the work:

| Question | Start with |
|---|---|
| What should I work on next? | `infrastructure-manager focus next` |
| What is the state of the platform I own? | `infrastructure-manager coverage status` and `condition status` |
| What reliability targets are we measuring against? | The setpoint: [`scenarios/infrastructure-manager/setpoint/`](../../scenarios/infrastructure-manager/setpoint/) |
| Which cells have no sensor at all? | `infrastructure-manager coverage open-loop` |
| Can I believe a reading? | `infrastructure-manager condition trust` and `condition explain <cell>` |
| How does the infra-health team operate end to end? | [`operating/OPERATING_MODEL.md`](operating/OPERATING_MODEL.md) |
| Which capabilities are not yet built on macOS or Windows? | `vrooli capability ledger` and `vrooli capability fleet blocked` |
| How are approved doc changes applied? | [`governance/editing.md`](governance/editing.md) |

## The instrument

`infrastructure-manager` is this team's instrument. It joins each control layer's
owner-authored reliability space against the checked-in operator setpoint, takes
trust-qualified readings, and emits one ranked error surface.

| Axis | Question | Read it with |
|---|---|---|
| Coverage | Is this dimension instrumented at all? | `infrastructure-manager coverage status` |
| Condition | For what we can see, is it in band? | `infrastructure-manager condition status` |
| Trust | Can the reading be believed? | `infrastructure-manager condition trust` |
| Focus | What should be worked first? | `infrastructure-manager focus next` |

Two rules follow from the split, and both are load-bearing:

- **Owners define the space; the operator defines the bar.** Which cells exist is
  authored by each control layer in its own `docs/spaces/<projection>-space.md`. The
  deadbands live in the instrument's `setpoint/`, which has no write path.
- **Findings are ranked by cascade order**, innermost first: sensor-channel integrity,
  host/process substrate, capability availability, efficiency, measurement improvement.
  Do not chase an outer-tier reading while an inner tier holds an unresolved excursion.

### Retired documents

Three files in this folder were the hand-maintained ancestors of a computed surface. Each
was retired for the same reason: a list maintained by hand can disagree with the surface
computed beside it, and this folder has no way to tell which one is wrong. **No row may be
added to any of them.**

| Retired file | Retired on | Superseded by |
|---|---|---|
| `strategy/RELIABILITY_TARGETS.md` (deleted 2026-08-20) | 2026-08-20 | `setpoint/reliability-setpoint.json` and the owner space documents |
| `evidence/INSTRUMENTATION_ROADMAP.md` (deleted 2026-08-20) | 2026-08-20 | The computed open-loop set (`coverage open-loop`) |
| `evidence/CROSS_PLATFORM_LEDGER.md` (deleted 2026-08-20) | 2026-08-20 | `vrooli capability ledger` / `vrooli capability fleet`, plus the instrument's `portability` domain for the judgment half |

`RELIABILITY_TARGETS.md` and `INSTRUMENTATION_ROADMAP.md` were **deleted on
2026-08-20**, after the out-of-folder links this plan of record held their deletion
on were repointed: five marked `path:` references in `docs/agent-system/` and
`docs/director-swarm/`, one Go comment in `scenarios/prompt-manager/cli/graph/audit.go`,
and three pointers in the `morning-vision-walk` skill. Each now points at the surface
that actually owns the concept — the honesty-flag vocabulary at the instrument's
`SETPOINT-MODEL.md`, the open-loop rule at its `COVERAGE-MODEL.md`, and per-team
reliability targets at the checked-in setpoint. One of those references was already
broken before this pass: `FRAMEWORK_HEALTH.md` deferred to a `RELIABILITY_TARGETS.md`
§"Honesty flags" section that did not exist.

`CROSS_PLATFORM_LEDGER.md` was **deleted on 2026-08-20** in the same pass. Its stated
condition was met — the instrument's `portability` domain shipped that day and now owns
the judgment half of cross-platform debt — and its two remaining routes, in the team's
`graph-presentation.json` and `team.json`, were pruned first. Read cross-platform state
with `vrooli capability ledger` or `vrooli capability fleet`, both of which delegate to
that domain.

## Organizing principle

Everything the instrument can compute belongs to the instrument. This folder keeps only
what it cannot:

1. **HOW the team operates** - [`operating/OPERATING_MODEL.md`](operating/OPERATING_MODEL.md), the Platform Under Control map, the routing rules, and "supervise, don't operate".
2. **WHO may change what** - [`governance/`](governance/editing.md), editing authority and adoption validation.

What we are aiming for, and what we are missing, are no longer written down: they are
`coverage status` and `coverage open-loop`. A hand-maintained copy of either would
immediately be able to disagree with the grid beside it, which is why both were retired.

The team that authors these is leaderless: `runtime-health-scanner` watches the runtime, `platform-code-auditor` audits internal code, `infra-contrarian` challenges. Findings flow into the morning vision walk (Phase 5.7); approved decisions update these docs.

## Folder map

| Folder | Purpose |
|---|---|
| [`operating/`](operating/README.md) | Team operating contract and validation commands. |
| [`governance/`](governance/editing.md) | Editing authority, adoption validation, and changelog. |
| [`evidence/`](evidence/README.md) | Retired. Held the instrumentation roadmap and the cross-platform ledger before either was computed. |
| [`strategy/`](strategy/README.md) | Retired. Held the sensor map before the instrument existed. |

## Cross-references

| Consumer | Use case |
|---|---|
| `runtime-health-scanner` | Compare observed service/process behavior against reliability targets; propose instrumentation gaps when evidence is missing. |
| `platform-code-auditor` | Audit internal code and lifecycle assumptions against portability and instrumentation requirements. |
| `infra-contrarian` | Challenge whether reliability, instrumentation, and portability findings are material enough to promote. |
| `director-swarm` | Pull infrastructure risks into morning vision walk decisions and portfolio sequencing. |

## Boundaries

These docs cover the *platform* (internal code, lifecycle, and the layered control stack — commissioning/setup, capacity broker, autoheal, system-monitor — as observed from outside; see the "Platform Under Control" map in [`operating/OPERATING_MODEL.md`](operating/OPERATING_MODEL.md)). They do NOT cover:
- Per-scenario reliability — that's each scenario's PRD and scenario-qa's audit lane
- Skill / agent / team optimization — meta-optimization owns that
- Live alerts and immediate-incident response — system-monitor + agent-manager handle the moment; infra-health watches the aggregate
- External observability dashboards (Grafana, etc.) — out of scope for v1

## Editing rules

- The operator curates these files. Agents propose diffs via decisions (`reliability-target-update`, `instrumentation-gap`, `cross-platform-debt`); approved decisions cite the decision id in the change line.
- No member of the infra-health team may edit these files directly.
- Honesty flags are mandatory on every metric: `measured`, `estimate`, `aspirational`, `pending-baseline`, `pending-telemetry`. Unflagged numbers are a guardrail violation.

Decision-context detail lives in [`governance/editing.md`](governance/editing.md).

## Future PoR work

- Add `catalogs/` only if infra-health grows stable registries for audit dimensions, signal tiers, or remediation patterns that are more durable than member task parameters.
- Add `taxonomies/` only if runtime findings, platform-code findings, or cross-platform-debt entries need machine-readable sidecars beyond the operating model.
- Add PoR manifest validation once prompt-manager consumes `manifest.json`.
- Retire `evidence/` and `strategy/` as folders. Their last banner-first files were deleted on 2026-08-20, so both hubs now hold nothing but a pointer — and a folder that exists only to say "look elsewhere" is the next instance of the pattern this cycle has now removed three times. This is the remaining step.
- Ratify the provisional substrate bars in a setpoint review; they were authored with the projection on 2026-08-20 and carry `decision_ref: 2026-08-20-substrate-provisional`.
