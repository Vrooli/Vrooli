# Infra Health

Plan-of-record for the platform's own health — Vrooli's internal code, lifecycle, and runtime reliability. Owned by the `infra-health` team in prompt-manager; authored and curated by the human operator.

## Start here for agents

Use this hub first, then follow the table below to the relevant spoke. Do not treat the presence of an observed problem as permission to edit these docs directly; capture evidence in team working state and propose durable updates through decisions.

## Organizing principle

Three orthogonal lenses, one file each:

1. **WHAT we're aiming for** — [reliability targets](RELIABILITY_TARGETS.md) per critical scenario or platform component.
2. **WHAT we're missing** — [instrumentation roadmap](INSTRUMENTATION_ROADMAP.md), the stats we owe ourselves so future findings are sharper.
3. **WHERE the platform isn't yet portable** — [cross-platform ledger](CROSS_PLATFORM_LEDGER.md), Linux-only assumptions tracked against tier-2+ deployment.

The team that authors these is leaderless: `runtime-health-scanner` watches the runtime, `platform-code-auditor` audits internal code, `infra-contrarian` challenges. Findings flow into the morning vision walk (Phase 5.7); approved decisions update these docs.

## Consumers

| Consumer | Use case |
|---|---|
| `runtime-health-scanner` | Compare observed service/process behavior against reliability targets; propose instrumentation gaps when evidence is missing. |
| `platform-code-auditor` | Audit internal code and lifecycle assumptions against portability and instrumentation requirements. |
| `infra-contrarian` | Challenge whether reliability, instrumentation, and portability findings are material enough to promote. |
| `director-swarm` | Pull infrastructure risks into morning vision walk decisions and portfolio sequencing. |

## Files

| File | Purpose |
|---|---|
| [RELIABILITY_TARGETS.md](RELIABILITY_TARGETS.md) | Uptime / restart-frequency / latency targets per critical scenario or platform component, with current-state snapshot and gap-to-target |
| [INSTRUMENTATION_ROADMAP.md](INSTRUMENTATION_ROADMAP.md) | Stats Vrooli should be collecting but isn't, with proposed shape and most-likely host scenario (analogous to `docs/monetization/TELEMETRY_ROADMAP.md`) |
| [CROSS_PLATFORM_LEDGER.md](CROSS_PLATFORM_LEDGER.md) | Linux-only assumptions in internal code, target deployment tier, owning surface, and resolution path |

## Boundaries

These docs cover the *platform* (internal code, lifecycle, autoheal/system-monitor as observed from outside). They do NOT cover:
- Per-scenario reliability — that's each scenario's PRD and scenario-qa's audit lane
- Skill / agent / team optimization — meta-optimization owns that
- Live alerts and immediate-incident response — system-monitor + agent-manager handle the moment; infra-health watches the aggregate
- External observability dashboards (Grafana, etc.) — out of scope for v1

## Curation rules

- The operator curates these files. Agents propose diffs via decisions (`reliability-target-update`, `instrumentation-gap`, `cross-platform-debt`); approved decisions cite the decision id in the change line.
- No member of the infra-health team may edit these files directly.
- Honesty flags are mandatory on every metric: `measured`, `estimate`, `aspirational`, `pending-telemetry`. Unflagged numbers are a guardrail violation.
