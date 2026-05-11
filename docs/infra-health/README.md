# Infra Health Plan of Record

This folder is the plan of record for the platform's own health: Vrooli's internal code, lifecycle, runtime reliability, instrumentation gaps, and cross-platform readiness. It is owned by the `infra-health` team in prompt-manager and curated through approved operator decisions.

The local contract is [`manifest.json`](manifest.json), which instantiates the shared plan-of-record shape from [`docs/agent-system/team-plan-of-record.manifest.json`](../agent-system/team-plan-of-record.manifest.json).

## Start here for agents

Use this README first, then choose the module that matches the work:

| Question | Start with |
|---|---|
| How does the infra-health team operate end to end? | [`operating/OPERATING_MODEL.md`](operating/OPERATING_MODEL.md) |
| What reliability targets are we measuring against? | [`strategy/RELIABILITY_TARGETS.md`](strategy/RELIABILITY_TARGETS.md) |
| Which stats are missing before findings can become measured? | [`evidence/INSTRUMENTATION_ROADMAP.md`](evidence/INSTRUMENTATION_ROADMAP.md) |
| Which Linux-only assumptions are tracked for future deployment tiers? | [`evidence/CROSS_PLATFORM_LEDGER.md`](evidence/CROSS_PLATFORM_LEDGER.md) |
| How are approved doc changes applied? | [`governance/editing.md`](governance/editing.md) |

## Organizing principle

Three orthogonal lenses, grouped by purpose:

1. **WHAT we're aiming for** - [`strategy/RELIABILITY_TARGETS.md`](strategy/RELIABILITY_TARGETS.md), targets per critical scenario or platform component.
2. **WHAT we're missing** - [`evidence/INSTRUMENTATION_ROADMAP.md`](evidence/INSTRUMENTATION_ROADMAP.md), the stats we owe ourselves so future findings are sharper.
3. **WHERE the platform isn't yet portable** - [`evidence/CROSS_PLATFORM_LEDGER.md`](evidence/CROSS_PLATFORM_LEDGER.md), Linux-only assumptions tracked against tier-2+ deployment.

The team that authors these is leaderless: `runtime-health-scanner` watches the runtime, `platform-code-auditor` audits internal code, `infra-contrarian` challenges. Findings flow into the morning vision walk (Phase 5.7); approved decisions update these docs.

## Folder map

| Folder | Purpose |
|---|---|
| [`operating/`](operating/README.md) | Team operating contract and validation commands. |
| [`strategy/`](strategy/README.md) | Reliability targets and durable platform-health goals. |
| [`evidence/`](evidence/README.md) | Instrumentation roadmap and cross-platform debt ledger. |
| [`governance/`](governance/editing.md) | Editing authority, adoption validation, and changelog. |

## Cross-references

| Consumer | Use case |
|---|---|
| `runtime-health-scanner` | Compare observed service/process behavior against reliability targets; propose instrumentation gaps when evidence is missing. |
| `platform-code-auditor` | Audit internal code and lifecycle assumptions against portability and instrumentation requirements. |
| `infra-contrarian` | Challenge whether reliability, instrumentation, and portability findings are material enough to promote. |
| `director-swarm` | Pull infrastructure risks into morning vision walk decisions and portfolio sequencing. |

## Boundaries

These docs cover the *platform* (internal code, lifecycle, autoheal/system-monitor as observed from outside). They do NOT cover:
- Per-scenario reliability — that's each scenario's PRD and scenario-qa's audit lane
- Skill / agent / team optimization — meta-optimization owns that
- Live alerts and immediate-incident response — system-monitor + agent-manager handle the moment; infra-health watches the aggregate
- External observability dashboards (Grafana, etc.) — out of scope for v1

## Editing rules

- The operator curates these files. Agents propose diffs via decisions (`reliability-target-update`, `instrumentation-gap`, `cross-platform-debt`); approved decisions cite the decision id in the change line.
- No member of the infra-health team may edit these files directly.
- Honesty flags are mandatory on every metric: `measured`, `estimate`, `aspirational`, `pending-telemetry`. Unflagged numbers are a guardrail violation.

Decision-context detail lives in [`governance/editing.md`](governance/editing.md).

## Future PoR work

- Add `catalogs/` only if infra-health grows stable registries for audit dimensions, signal tiers, or remediation patterns that are more durable than member task parameters.
- Add `taxonomies/` only if runtime findings, platform-code findings, or cross-platform-debt entries need machine-readable sidecars beyond the operating model.
- Add PoR manifest validation once prompt-manager consumes `manifest.json`.
- Promote measured targets from `pending-telemetry` after the roadmap gaps ship and 30+ days of data exists.
