# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario infrastructure-manager`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Be the infra-health team's instrument — the one address a member reads to answer "what is the state of the platform I own, and what should I do next?" It joins the team's authored setpoint (the sensor map in `docs/infra-health/strategy/RELIABILITY_TARGETS.md`) against live readings from the platform's control layers, qualifies every reading with a trust verdict, and emits one ranked error surface. It surfaces and ranks; it never decides, never actuates, and never operates the loops it observes.
- **Primary users/verticals**: The `infra-health` team members (runtime-health-scanner, platform-code-auditor, infra-contrarian) as the daily readers; the operator at the morning vision walk; peer boards and agents that need a typed read of platform reliability.
- **Deployment surfaces**: Go API (Connect-RPC) and Go CLI as the primary programmatic surface; a read-only operator board UI as a later surface; declared widgets and tools so agents can discover it through `cli-health`.
- **Value promise**: The team's reliability targets are authored, dated and honest, but nothing computes them — five of fourteen targets have no sensor, the alarm channel is out of band, and the supervised-set figure is a hand-run diff. This scenario turns a document that must be read and reasoned over into a number that can be queried, so reliability work is driven by a measured error signal instead of by whichever signal an agent happened to look at that heartbeat.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Typed setpoint read | Parse the infra-health sensor map into typed targets and validate its integrity: every target names a sensor, a deadband and an actuator, or is an honest empty cell. The setpoint stays owned by the team's plan of record and is read, never authored here.
- [ ] OT-P0-002 | Live reading join | Read each named sensor live and join it against its deadband to report in-band or out-of-band per target. Every source degrades independently to an explicit UNAVAILABLE with a stated reason, never to a zero and never to a dropped row.
- [ ] OT-P0-003 | Trust-qualified readings | Attach a closed-vocabulary trust verdict to every reading (valid, ghost, saturated, shelved, unit-mismatch, unavailable). An untrusted reading never contributes to an aggregate, and an uninstrumented target is never reported as healthy.
- [ ] OT-P0-004 | One ranked error surface | Merge out-of-band readings, supervision gaps, untrusted readings and uninstrumented targets into a single ranked surface a member reads with one call, with each source named and independently degradable.
- [ ] OT-P0-005 | Open-loop self-report | Count and date the targets that have no working sensor, including this scenario's own blind spots, so an honest hole is visible and ageing rather than silent.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Supervised-set reconcile | Report the two-direction diff between the autoheal check registry and the derived should-be-supervised set: ghost checks and unsupervised plant. The set is computed at read time from the core-set closure and load-bearing declarations, never held as an enumerated roster.
- [ ] OT-P1-002 | Reading history and trend | Persist readings and never verdicts, so in-band status is recomputed against the current deadband at query time and a target change re-grades history instead of stranding stale judgments.
- [ ] OT-P1-003 | Actuation efficacy | Re-read a finding's named sensor after its downstream work lands and record whether the reading returned in band. A fix that does not move its sensor re-opens the finding, so the sensor grades the fix rather than its author.
- [ ] OT-P1-004 | Setpoint drift detection | Diff the sensor map against the fleet's live command surface and report targets whose named sensor no longer resolves, plus open cells whose gap a shipped verb could already close.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Operator board UI | A read-only operator panel presenting the ranked error surface, the trust distribution across readings, and per-target drilldown to the evidence behind each verdict.
- [ ] OT-P2-002 | Capability availability history | Consume each capability owner's persisted availability aggregate once owners expose it, replacing the interim read-time reachability proxy with owner-derived history.
- [ ] OT-P2-003 | Watchdog supervision | Supervise a watchdog tier's liveness, enumerated-action counts and claim-suppression rate, so a watchdog repeatedly restarting the same element surfaces as a heal-loop finding on the slow loop. Deferred until that tier is authorized.

## 🧱 Tech Direction Snapshot

- **Preferred stacks / frameworks**: Go API on Connect-RPC with a Go CLI built on cli-core `ScenarioApp`; React + Vite + Tailwind for the later operator board. Matches the thin-aggregator shape already proven by `meta-optimization-manager`.
- **Data + storage expectations**: SQLite via `api-core/storage`, holding only the reading history and the findings-to-sensor efficacy join. The setpoint is not stored (it is read from the team's plan of record) and verdicts are not stored (they are recomputed at query time against the current deadband).
- **Integration strategy**: Typed read clients over each source's existing CLI/RPC — `vrooli-autoheal`, `vrooli capacity`, `storage-manager`, `test-genie`, `system-monitor`, `data-backup-manager`, `agent-manager` — each bounded by a short deadline and each degrading independently. Never re-run or re-implement a source's measurement; read its derived output.
- **Non-goals / guardrails**: This scenario never actuates. No restarts, no policy-lever changes, no degrade or preempt, no privileged host mutation. It never authors its own setpoint, never enumerates a capability owner's members (the contract-not-roster rule), never stores a verdict, and never reports an uninstrumented target as healthy. Live incident response and remediation remain with autoheal, system-monitor, agent-manager and the operator.

## 🤝 Dependencies & Launch Plan

- **Required resources**: SQLite via `api-core/storage`. No new resource dependencies.
- **Scenario dependencies**: Read-only and degradable — `vrooli-autoheal` (uptime, alarm flood, restarts, heal outcomes, check registry), `storage-manager` (device census and growth slope), `test-genie` (validation cost and cache reliability), `system-monitor` (process attribution and investigations), `data-backup-manager` (backup status, coverage, recovery drills), `agent-manager` (run statistics and error patterns), `scenario-dependency-analyzer` (core-set closure for the derived supervised set). Control-plane reads: `vrooli capacity`.
- **Operational risks**: The setpoint is prose today, so a parser change and a document edit can disagree — integrity validation is a P0 target for this reason. The team loop has been paused since 2026-07-24, so no current-state value is a live baseline until it resumes. Both scenario templates report quarantined status in the registry, which is a generation-path risk rather than a runtime one. Reading a source that is itself part of the plant means outages must degrade legibly rather than reading as clean.
- **Launch sequencing**: Resume the team loop and re-baseline; close the sensor-map rows that shipped verbs already serve and write the target rows that have a sensor but no row; build the setpoint read and the live join as the first vertical slice; add supervision reconcile and reading history; defer the UI and any watchdog supervision to explicit later decisions.

## 🎨 UX & Branding

- **Look & feel**: Instrument-panel rather than dashboard-decorative. Dense, scannable rows over large tiles; state encoded in form as well as number so an out-of-band or untrusted reading reads at a glance. Light and dark themes both first-class through the template's design tokens.
- **Accessibility**: WCAG 2.1 AA as the floor. Status is never carried by colour alone — every verdict pairs a hue with a label or shape, which matters more here than usual because the whole surface is status. Keyboard-navigable tables with visible focus, `aria-*` on live-updating regions, and the template's accessibility primitives and `data-testid` selectors preserved.
- **Voice & messaging**: Plain and unhedged. A verdict names the offending signal and its value — a bare DEGRADED is not a valid verdict. Unavailable states state the reason verbatim rather than apologising. No number appears without the confidence or trust qualifier that makes it honest.
- **Branding hooks**: Inherit the `vrooli-default` design language and token plumbing from the template. No scenario-specific brand identity; this is internal operator tooling and should read as part of the platform, not as a product.

## 📎 Appendix

Canon this scenario implements or depends on:

- `docs/agent-system/TARGET_MODEL.md` — the instrument contract: the control chain, the six invariants, and the deviation catalogue this scenario is built to satisfy.
- `docs/infra-health/strategy/RELIABILITY_TARGETS.md` — the setpoint. Owned by the infra-health team and the operator; read here, never authored here.
- `docs/infra-health/operating/OPERATING_MODEL.md` — the Platform Under Control layer map and the routing rules, including "supervise, don't operate".
- `docs/infra-health/evidence/INSTRUMENTATION_ROADMAP.md` — the capability gaps behind every empty sensor cell.
- `docs/concepts/RECURSIVE_SELF_IMPROVEMENT.md` § Control topology — how this loop sits beside meta-optimization and above the fast platform loops.
- `scenarios/meta-optimization-manager/docs/concepts/{COVERAGE-MODEL,CONDITION-MODEL}.md` — the sibling instrument's models; the worked precedent for denominator-confidence and derived populations.
