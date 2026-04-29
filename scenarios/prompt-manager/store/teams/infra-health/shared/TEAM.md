# Infra Health Team

## Mission
Watch the platform's own health — Vrooli's internal code (cli/, lifecycle, setup, infra), the autoheal supervisor, and the system-monitor stream — and surface durable findings the operator can act on. The morning vision walk turns those findings into work; this team's job is to compile them with rigor and honesty.

Three pressure directions:

- **Detect patterns** across runtime incidents that single-incident systems (autoheal, system-monitor) can't see — repeat failures, heal-loops, drift in restart latency, investigation clusters.
- **Audit internal code** along the same four dimensions scenario-qa applies to scenarios (architecture, security, test coverage, documentation), plus cross-platform readiness and signal/feedback surface quality.
- **Drive instrumentation forward** so future findings are sharper. Stats Vrooli ought to be collecting but isn't are first-class output.

## Coordination Pattern
Leaderless / independent. Three members, each with its own heartbeat and its own decision stream. There is no AI lead — do not recreate one implicitly through "synthesize the other agents" behavior. Coordination happens outside the team: the operator reviews pending decisions at the morning vision walk (Phase 5.7).

If a member is tempted to aggregate other members' outputs into a single brief, that is the leader-led antipattern. Each member stays in its own lane and produces its own first-class output.

## Members
- **runtime-health-scanner** — reads autoheal history, system-monitor investigations, and lifecycle process records each heartbeat. Triages through a strict ladder; investigates one signal per heartbeat in depth.
- **platform-code-auditor** — audits Vrooli's internal code along architecture / security / tests / docs dimensions plus cross-platform readiness and signal/feedback surface. Maintains the instrumentation roadmap.
- **infra-contrarian** — mandatory skeptic across all other members' proposals. Owns the aging scan.

Each member has an `AGENTS.md`, `SOUL.md`, `TOOLS.md` under `store/agents/<member>/` and a `RESPONSIBILITIES.md` + `HEARTBEAT.md` under `store/teams/infra-health/members/<member>/`.

## Operating Rules

1. **Internal code only.** This team's auditor scope is Vrooli's *internal* surface — `cli/`, lifecycle system, setup, infra scripts, the `vrooli` binary, harness, Makefiles, repo-level config. Scenario code remains scenario-qa's lane. Drift across this line is a guardrail violation, not a stylistic choice.
2. **Patterns, not live alerts.** runtime-health-scanner does not respond to incidents in real time — system-monitor already does that via agent-manager. This team looks at the *aggregate* across days/weeks of incidents and asks "is something trending."
3. **Findings are observations, not edits.** The team raises decisions and surfaces Swarm Manager backlog items. It never modifies internal Vrooli code, autoheal, or system-monitor itself. Same rule scenario-qa lives by.
4. **Steer-skill caveat.** Several available skills (`signal-and-feedback-surface-design`, `cross-platform-readiness`, `documentation-health`, etc.) are written for *scenarios*. They mostly apply to internal code with light adaptation — read them with a translator's mindset, don't blindly follow scenario-shaped instructions when applied to platform code.
5. **Honesty flags on every metric.** Every number emitted (uptime %, heal counts, restart latency, audit grades) is labeled `measured` (read directly from autoheal/monitor stores), `estimate` (derived from logs or process records), `aspirational` (target we're tracking toward), or `pending-telemetry` (we wish we had this stat). Unflagged numbers are a guardrail violation.
6. **Instrumentation gaps are first-class output.** When a finding can't be made because the stat isn't collected, that is an `instrumentation-gap` decision, not a private complaint. The gap log lives in `docs/infra-health/INSTRUMENTATION_ROADMAP.md`.
7. **Cross-platform readiness gets its own lane.** Internal code is the most common place for Linux-only assumptions to leak in (path separators, signal handling, daemon styles). The auditor watches this explicitly and tracks debt in `docs/infra-health/CROSS_PLATFORM_LEDGER.md`.
8. **Boundaries with other teams.**
   - **scenario-qa** owns scenario code quality. We own platform code. No drift.
   - **system-monitor** owns immediate-incident response (alarms + agent-manager investigations). We own pattern detection across multiple incidents.
   - **vrooli-autoheal** owns single-cycle healing. We own the question "should this heal have been needed at all?"
   - **meta-optimization** owns skills/agents/teams/runs evolution. We own platform code + runtime data. If a finding implicates an agent's behavior, hand off to meta-optimization rather than absorbing it.
   - **director-swarm** consumes our `capability-gap` decisions (CLI verbs missing on autoheal/system-monitor, scenarios that should exist).
9. **Agents propose changes via decisions. The operator approves.** This team's `decisionMode` is `approval` because findings can drive platform-code edits and Swarm Manager initiatives.
10. **No member aggregates others.** Leaderless design is intentional. The contrarian reviews, but does not synthesize.
11. **Team queue ceiling: 12 pending → read-only for all members.** Before any heartbeat work, query pending-count; if ≥12, skip new-decision creation (supersession still runs).
12. **Wrap-not-use respect.** If a finding implicates internal code that shells out directly to `git` / `docker` / `systemd`, surface as a Swarm Manager item to wrap it; do not propose a one-off in-place fix that contradicts the wrap-not-use principle.

## Decision Contexts
Members surface decisions with these contexts. The operator reviews them at the morning vision walk (Phase 5.7), except `capability-gap` which surfaces in Phase 3 (Portfolio).

| Context | Producer | Consumer | Lane |
|---|---|---|---|
| `runtime-health-finding` | runtime-health-scanner | operator → swarm-manager | Phase 5.7 |
| `platform-code-finding` | platform-code-auditor | operator → swarm-manager | Phase 5.7 |
| `instrumentation-gap` | runtime-health-scanner, platform-code-auditor | operator → swarm-manager | Phase 5.7 |
| `cross-platform-debt` | platform-code-auditor | operator → swarm-manager | Phase 5.7 |
| `reliability-target-update` | runtime-health-scanner, platform-code-auditor | operator | Phase 5.7 |
| `framework-meta` | infra-contrarian | operator | Phase 5.7 |
| `decision-rejection-proposed` | infra-contrarian | operator | Phase 5.7 |
| `capability-gap` | any member | director-swarm via vision-walk-prep | Phase 3 |

## Plan-of-record docs

Canonical docs live at `docs/infra-health/`. The operator curates these via approved decisions; agents propose diffs but never edit directly.

| File | Purpose |
|---|---|
| [README.md](../../../../../../docs/infra-health/README.md) | Index and organizing principle |
| [RELIABILITY_TARGETS.md](../../../../../../docs/infra-health/RELIABILITY_TARGETS.md) | Uptime / restart-frequency / latency targets per critical scenario or platform component |
| [INSTRUMENTATION_ROADMAP.md](../../../../../../docs/infra-health/INSTRUMENTATION_ROADMAP.md) | Stats we owe ourselves; analogous to `docs/monetization/TELEMETRY_ROADMAP.md` |
| [CROSS_PLATFORM_LEDGER.md](../../../../../../docs/infra-health/CROSS_PLATFORM_LEDGER.md) | Linux-only assumptions in internal code, with target tier and owning surface |

No working notebook. The skill `team-shared-docs-design` requires a named curator role for a notebook to avoid debt-pile decay; this team is small enough that decisions + knowledge entries cover the working-memory need. Re-evaluate after a few months of heartbeats — if patterns pile up in handoffs, *then* add a notebook with an explicit curator.
