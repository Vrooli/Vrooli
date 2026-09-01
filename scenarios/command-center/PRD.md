# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)
> **Supersedes**: the 2026-07-02 PRD, which described a read-only kiosk aggregator. Regenerated 2026-09-01 after the operator decision to make this scenario the Director Swarm's instrument.

## 🎯 Overview

- **Purpose**: Be the Director Swarm's instrument — the one address a member reads to answer "is the work we are doing producing results, and which sensor is worth building next?" It reads the objective set and every team's declared instrument, joins the authored outcome space against what actually resolves, qualifies each reading, renders the result as a board that stands up on a wall, and emits one ranked surface of what is not yet measurable. It surfaces and ranks; it never decides, never actuates, and never authors the setpoint it measures against.
- **Primary users/verticals**: `team:director-swarm` members (`portfolio-manager`, `outcome-strategist`) as the daily readers; the operator at the morning vision walk and at the wall display; peer teams and agents that need a typed read of whether Vrooli's work is producing outcomes.
- **Deployment surfaces**: Go API as the programmatic surface; a scenario CLI carrying the same reads; a full-bleed ambient board that runs unattended on a wall panel, a desktop browser, a phone, and a gamepad-controlled TV, reached through `tunnel-manager`.
- **Value promise**: Director Swarm's portfolio loop runs open-loop today. Predictions are recorded against named Command Center metrics with horizon dates, and none can be scored, because the payload carries labels and no values. This scenario closes that loop — and makes the instrument beautiful enough that people look at it, because an instrument nobody looks at generates no pressure to build the sensors it says are missing.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Readings carry values | Every metric carries a typed value, unit, format, observation time and source TTL; upstream readings reach the payload instead of stopping at the cache.
- [ ] OT-P0-002 | Three axes, never merged | Coverage (`NOW` / `IN-REACH` / `MISSING` / `UNREGISTERED`), condition carried as trust (`VALID` / `CACHED` / `UNAVAILABLE` / `UNTRUSTED`) and empirical (`NONE` / `PENDING` / `HIT` / `MISS` / `UNMEASURABLE`) are independent fields; none is folded into another or into a single status.
- [ ] OT-P0-003 | Authored sample readings | Sample values are registry-authored data, never generated at runtime and never derived from an upstream response; every sample is stamped so it cannot be mistaken downstream for a measurement.
- [ ] OT-P0-004 | Setpoint read, never authored | Parse and validate a setpoint this scenario does not own, exposing target and distance-to-target; no API or UI path may write it.
- [ ] OT-P0-005 | Derived board shape | The room list, metric set and source bindings are derived at read time from the objective set, every team's `instrument` declaration, and each live instrument's own surface — never authored in this scenario's code.
- [ ] OT-P0-006 | One ranked surface | A single ordered answer to "which sensor is worth building next," merging coverage gaps, untrusted readings, unavailable sources and unregistered outcomes, with each entry naming its owner.
- [ ] OT-P0-007 | Open-loop self-report | Count and date every `MISSING` cell and every unregistered outcome, including this scenario's own blind spots, so a hole is visible and ageing rather than silent.
- [ ] OT-P0-008 | One address | A describe endpoint and CLI verbs carrying the same reads as the board, so the team reads one surface programmatically and the declared two-addresses deviation is closed.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Provenance rendering contract | Every figure renders in a material that encodes its coverage and trust state; no figure appears without the qualifier that makes it honest, and no state is carried by colour alone.
- [ ] OT-P1-002 | Composed rooms | Each outcome category renders as one composed visual idea in both landscape and portrait, with a non-blank composed fallback frame and a first-frame render self-check.
- [ ] OT-P1-003 | Ambient display shell | Full-bleed, zero idle chrome, cycle rail, hidden control bar that reveals on input, fade-through-black page changes, safe-area aware.
- [ ] OT-P1-004 | One intent vocabulary | Gamepad, keyboard, touch and pointer resolve to a single intent set built on the shared `GamepadAction` vocabulary before anything reacts.
- [ ] OT-P1-005 | Audience modes | `samples=hide|mark|full` selects whether illustrative readings are shown, marked, or withheld, with a persistent legend whenever marked readings are on screen.
- [ ] OT-P1-006 | Capability ladder | A runtime capability probe selects the scene tier so the board is correct on a phone, a laptop, a gamepad-controlled TV and a wall panel without a per-device build.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Prediction binding | Metrics carrying a live prediction render their target and remaining horizon beside the value.
- [ ] OT-P2-002 | Reading history | Persist readings with their trust verdicts so trend and staleness-against-window are computable, recomputing verdicts against the current setpoint at query time.
- [ ] OT-P2-003 | Revenue pipeline consumption | Consume the monetization instrument's revenue and subscription surface once it exposes one, retiring the Ledger room's authored samples.
- [ ] OT-P2-004 | Marketing instrument consumption | Consume a marketing-crew aggregator once that team declares one; until then the Broadcast room reports the team's missing instrument as the finding rather than reporting six unrelated gaps.

## 🧱 Tech Direction Snapshot

- **Preferred stacks / frameworks**: Go API on `api-core/server` with the shared `api/handlers/capabilities` module; Go CLI on `cli-core`; React 18 + Vite + TypeScript for the board, composed from React Component Library assets rather than scenario-local components; React Three Fiber for the ambient scenes behind a capability probe.
- **Data + storage expectations**: Filesystem and in-memory only for the P0 set — the outcome registry is checked-in versioned data, the setpoint is a checked-in file parsed per query, and numerators are computed live and never stored so a stale board is structurally impossible. Reading history (OT-P2-002) is the first thing that earns durable storage.
- **Integration strategy**: Read the objective set through `prompt-manager graph objectives` as a transmitter rather than absorbing the join; read every team's `instrument` block from the team record; read each live instrument through its own standard verb. Never re-implement a source's measurement — read its derived output. Every source is independently degradable and names itself when unreadable.
- **Non-goals / guardrails**: This scenario never actuates. No write path into Swarm Manager, no work-item filing, no setpoint writes, no mutation of any upstream. It never authors a denominator it measures against, never caches a derived set, never fabricates a reading, never reports an uninstrumented cell as healthy, and never renders a percentage over a denominator nobody authored. It is not the universal outcome surface: terminal objectives in personal-life domains are measured by the scenario that serves them.

## 🤝 Dependencies & Launch Plan

- **Required resources**: None for the P0 set. Reading history introduces SQLite via `api-core/storage`.
- **Scenario dependencies**: Read-only and independently degradable — `prompt-manager` (objective set and team instrument declarations, read as a transmitter), `swarm-manager` (portfolio and throughput state), `meta-optimization-manager` and `infrastructure-manager` (peer instruments), `offer-desk` and `money-ledger` (monetization), `landing-page-business-suite` (deployed funnel and revenue). Control-plane reads for scenario health.
- **Operational risks**: The board's honesty is inherited from its sources, and a room cannot be more instrumented than the team behind it — three of six teams have a partial or absent instrument, so several rooms will carry authored samples for some time by design. The Director Swarm loop has been paused since 2026-07-24, so no current-state value is a live baseline until it resumes. The declared archetype is `production-ledger`, which forbids reporting coverage percentages over undefensible denominators; any composite score must be re-expressed as ledger state.
- **Launch sequencing**: Land readings and the two honesty axes; root-cause the existing blank-scene defect; derive the board's shape from the fleet rather than from code; amend the design contract; build the provenance rendering primitives in the component library and ship one reference room on them; add the shell, cycle and input vocabulary; then the remaining rooms, with the all-sample room last as the final proof that the honesty system holds.

## 🎨 UX & Branding

- **Look & feel**: A command display, not an operational console — full-bleed, dark, one strong visual idea per room, one enormous readable figure, and generous void. Cinematic and technical without ever distorting a quantity. The governing contract is `DESIGN.md` (`vrooli-command-display`); this scenario does not use the default operational-console app shell.
- **Accessibility**: WCAG 2.1 AA as the floor, measured against live scenes rather than static frames. Provenance is carried by material first and colour second, so no state depends on hue. Declared quiet zones keep a contrast floor under every figure. Reduced motion freezes ambient scenes on a composed still, steps the freshness indicator instead of sweeping it, and swaps digits without rolling. Visible focus, 64px targets in kiosk mode, and no essential control behind hover.
- **Voice & messaging**: Sparse and unhedged, sized for a room. A degraded source states its reason verbatim rather than apologising. No figure appears without its provenance, no ratio without the confidence of the denominator behind it, and an absent metric states what is needed rather than showing an error. Gaps are presented as declared future capability, never as failures.
- **Branding hooks**: Six themes as React Component Library semantic token overrides — `--color-*`, `--space-*`, `--text-*` — so library components inherit the active room's palette. The scenario-private `--cc-*` vocabulary is retired.

## 📎 Appendix

Canon this scenario implements or depends on:

- `docs/agent-system/TARGET_MODEL.md` — the instrument contract: the control chain, the six invariants, the four modes, the two archetypes, and the deviation catalogue.
- `docs/director-swarm/evidence/OUTCOMES_CHARTER.md` — the outcome categories this board renders, the sensor map, and the gap-closure loop.
- `docs/director-swarm/operating/OPERATING_MODEL.md` — the portfolio control loop this board's readings close, and the prediction ledger that scores against them.
- `docs/director-swarm/strategy/OBJECTIVES.md` — the objective set every outcome category must trace upward to.
- `scenarios/meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md` — the worked precedent for owned denominators, denominator-confidence, and the two orthogonal attestation axes.
- `scenarios/infrastructure-manager/docs/concepts/SETPOINT-MODEL.md` — the worked precedent for a setpoint an instrument reads but does not own.
- `DESIGN.md` — the `vrooli-command-display` design contract governing every surface here.
