# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario infrastructure-manager`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Be the infra-health team's instrument — the one address a member reads to answer "what is the state of the platform I own, and what should I do next?" It joins each control layer's authored reliability space against a checked-in operator setpoint, takes live readings on every instrumented cell, qualifies each with a trust verdict, bands it, and emits one ranked error surface. It surfaces and ranks; it never decides, never actuates, and never operates the loops it observes.
- **Primary users/verticals**: The `infra-health` team members (runtime-health-scanner, platform-code-auditor, infra-contrarian) as the daily readers; the operator at the morning vision walk; peer boards and agents that need a typed read of platform reliability.
- **Deployment surfaces**: Go API (Connect-RPC) and Go CLI as the programmatic surface; a polished read-only operator board as a first-class surface shipped vertical-by-vertical alongside each domain; declared widgets and tools so agents can discover it through `cli-health`.
- **Value promise**: The platform's reliability is measured by seven different scenarios that each answer one question, in units that do not compose, behind prose deadbands a human must reason over by hand. Nothing computes how much of the platform is instrumented *at all* — five of fourteen authored targets have no sensor and nothing counts them. This scenario turns that into a cell grid with a live join, an honest denominator-confidence, and a single ranked surface, so reliability work is driven by a measured error signal instead of by whichever signal an agent happened to look at that heartbeat.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Live cell grid | Join each control layer's authored reliability space against the fleet's live surface into a typed cell grid, reporting `NOW` / `IN-REACH` / `MISSING` per cell with per-projection denominator-confidence. Spaces are read live from their owners and never cached; a cell absent from a live join keeps its authored status rather than being fabricated as `MISSING`.
- [ ] OT-P0-002 | Setpoint read and integrity | Parse the checked-in setpoint into typed bars and validate it: every `NOW` cell has a bar, no bar equals the reading it was authored against, every honesty flag is derived rather than hand-set, and every changed bar carries a decision reference. An unparseable setpoint fails loudly instead of reporting zero targets as zero problems.
- [ ] OT-P0-003 | Trust-qualified readings | Attach a closed-vocabulary trust verdict to every reading (`VALID`, `GHOST`, `SATURATED`, `SHELVED`, `UNIT_MISMATCH`, `UNAVAILABLE`, `UNTRUSTED`). Only `VALID` contributes to an aggregate, every trust number is reported as a distribution over checked readings over total readings, and an untrusted reading routes to the instrument's owner rather than becoming plant work.
- [ ] OT-P0-004 | One ranked error surface | Merge out-of-band readings, untrusted readings, open-loop cells, coverage drift and source unavailability into a single ranked surface a member reads with one call. Sources are named and independently degradable, and ranking honours cascade order — sensor-channel integrity first — and states that it is doing so.
- [ ] OT-P0-005 | Open-loop self-report | Count and date every `MISSING` cell, including this scenario's own blind spots, so an honest hole is visible and ageing rather than silent. The instrumentation roadmap becomes the computed open-loop set rather than a hand-maintained list that can disagree with the grid beside it.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Supervision reconcile | Report the two-direction diff between the autoheal check registry and the derived should-be-supervised set: ghost checks on one side, unsupervised plant on the other. The set is computed at read time from the core-set closure and load-bearing declarations, never held as an enumerated roster.
- [ ] OT-P1-002 | Reading history and band recomputation | Persist readings and trust verdicts but never band verdicts, so in-band status is recomputed against the current deadband at query time and tightening a target re-grades its own history. Retention floor is derived from the longest declared window, so a widened target widens retention with it.
- [ ] OT-P1-003 | Actuation efficacy | Re-read a finding's named sensor after its downstream work lands and record whether the reading returned in band. A fix that does not move its sensor re-opens the finding; a sensor that became unreadable in the meantime is reported as unmeasurable, never as a pass.
- [ ] OT-P1-004 | Coverage drift detection | Diff every cell against the fleet's live command surface and report cells whose named sensor no longer resolves, plus `MISSING` cells whose gap a shipped verb could already close.
- [ ] OT-P1-005 | Operator board | A polished read-only operator panel presenting the cell grid, the ranked error surface, the trust distribution, and per-cell drilldown to the evidence behind each verdict. Instrument-panel dense rather than dashboard-decorative, WCAG 2.1 AA, and shipped vertical-by-vertical with the domain it presents.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Capability availability history | Consume each capability owner's persisted availability aggregate once owners expose it, replacing the interim read-time reachability proxy with owner-derived history.
- [ ] OT-P2-002 | Watchdog supervision | Supervise a watchdog tier's liveness, enumerated-action counts and claim-suppression rate as an eleventh projection, so a watchdog repeatedly restarting the same element surfaces as a heal-loop finding on the slow loop. Deferred until that tier is authorized.

## 🧱 Tech Direction Snapshot

- **Preferred stacks / frameworks**: Go API on Connect-RPC with a Go CLI built on cli-core `ScenarioApp`; React + Vite + Tailwind for the operator board. Matches the thin-aggregator shape already proven by `meta-optimization-manager`.
- **Data + storage expectations**: SQLite via `api-core/storage`, holding only the reading history with its trust verdicts, and the findings-to-sensor efficacy join. Spaces are read live from their owners, the setpoint is a checked-in file parsed per query, and band verdicts are never stored.
- **Integration strategy**: Typed Connect-RPC clients per control layer, resolved through `api-core/discovery` and bounded by a 10s per-source deadline, mirroring `meta-optimization-manager`'s numerator join. `vrooli-autoheal` has no proto surface today and gains a proper read-only one as part of this work; `vrooli capacity` is control-plane `internal/` and is therefore the one legitimate CLI read. Never re-run or re-implement a source's measurement; read its derived output.
- **Non-goals / guardrails**: This scenario never actuates. No restarts, no policy-lever changes, no degrade or preempt, no privileged host mutation. It never authors a space or a bar, never exposes a write path to the setpoint, never enumerates a capability owner's members, never caches a derived set, never stores a band verdict, and never reports an uninstrumented cell as healthy. Live incident response and remediation remain with autoheal, system-monitor, agent-manager and the operator.

## 🤝 Dependencies & Launch Plan

- **Required resources**: SQLite via `api-core/storage`. No new resource dependencies.
- **Scenario dependencies**: Read-only and independently degradable — `vrooli-autoheal` (supervision, availability and recovery projections, the check registry, and two of four trust rules), `storage-manager` (headroom), `test-genie` (validation cost), `system-monitor` (attribution), `data-backup-manager` (durability), `agent-manager` (agent throughput), `scenario-dependency-analyzer` (core-set closure for the derived supervised set). Control-plane reads: `vrooli capacity`.
- **Operational risks**: the typed autoheal surface and Gap 10 trust inputs are shipped, and owner space documents are present; condition history remains explicitly unmeasurable below the derived retention floor, capability availability history and some peer source joins remain unavailable, and setpoint confidence remains `SKETCH` until the obligation list is approved. The scenario template still reports `quarantined` in the registry, which remains an upstream provenance risk. The team loop has been paused since 2026-07-24, so no current-state value is a live baseline until it resumes.
- **Launch sequencing**: Resolve the template quarantine; migrate the setpoint and the instrumentation roadmap out of the team plan of record into this scenario; author the design system; then ship `coverage`, `condition` and `focus` as verticals, each with API, CLI and its board page. The autoheal typed surface and Gap 10 land before the condition vertical, because half the trust vocabulary depends on them. Capability availability history and watchdog supervision stay deferred to explicit later decisions.

## 🎨 UX & Branding

- **Look & feel**: Instrument-panel rather than dashboard-decorative. Dense, scannable rows over large tiles; state encoded in form as well as number so an out-of-band or untrusted reading reads at a glance. A cell grid is the primary object on the page, not a chart. Light and dark themes both first-class through the template's design tokens.
- **Accessibility**: WCAG 2.1 AA as the floor. Status is never carried by colour alone — every verdict pairs a hue with a label or shape, which matters more here than usual because the whole surface is status. Keyboard-navigable grids with visible focus, `aria-*` on live-updating regions, and the template's accessibility primitives and `data-testid` selectors preserved.
- **Voice & messaging**: Plain and unhedged. A verdict names the offending signal and its value — a bare DEGRADED is not a valid verdict. Unavailable states state the reason verbatim rather than apologising. No ratio appears without the denominator-confidence that makes it honest, and no condition figure appears without the coverage denominator it was computed against.
- **Branding hooks**: Inherit the `vrooli-default` design language and token plumbing from the template. No scenario-specific brand identity; this is internal operator tooling and should read as part of the platform, not as a product.

## 📎 Appendix

Canon this scenario implements or depends on:

- `docs/agent-system/TARGET_MODEL.md` — the instrument contract: the control chain, the six invariants, and the deviation catalogue this scenario is built to satisfy.
- `docs/infra-health/operating/OPERATING_MODEL.md` — the Platform Under Control layer map and the routing rules, including "supervise, don't operate".
- `docs/infra-health/governance/changelog.md` — the record of the team's hand-maintained target list, instrumentation roadmap and cross-platform ledger being retired onto computed surfaces. The setpoint is now checked-in data here; open-loop cells are computed by `coverage`.
- `docs/concepts/RECURSIVE_SELF_IMPROVEMENT.md` § Control topology — how this loop sits beside meta-optimization and above the fast platform loops.
- `scenarios/meta-optimization-manager/docs/concepts/{COVERAGE-MODEL,CONDITION-MODEL}.md` — the sibling instrument's models; the worked precedent for owner-held denominators, derived populations, and denominator-confidence.
