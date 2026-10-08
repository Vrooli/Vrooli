# Go To Market — Architecture Cartographer

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

Cartographer is an internal Vrooli tool in v1. External go-to-market
is deferred per [`MONETIZATION.md`](MONETIZATION.md). This document
captures the internal rollout plan and the trigger conditions for
revisiting external positioning.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

- **v1 audience (internal)**: Vrooli migration agents and scenario
  maintainers. The cartographer is invoked when:
  - A new scenario template version ships and existing scenarios
    need realignment.
  - A scenario has accumulated drift between its documented
    architecture and its code.
  - Cycles or boundary violations are blocking new feature work.
  - A planned migration is large enough that manual audit cost would
    be prohibitive.
- **v1 positioning**: "The L5 programmatic-drift-checks tool — built
  so screaming-architecture audits stop being expensive prose
  exercises." This is the framing in `PRD.md`.
- **Main internal claim**: cartographer-driven migrations cost a
  fraction of manual screaming-architecture audits and avoid the
  big-bang failure mode that reverted swarm-manager on 2026-05-13.
- **Proof needed for that claim**: time-tracking on N≥5 cartographer-
  assisted migrations vs. equivalent baseline migrations. Captured
  through analytics once the scenario is in active use.
- **External positioning**: deferred. Not in v1 scope.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Internal: vision-walk + agent skills | Agents working on migrations will reach for cartographer because it is the canonical L5 tool referenced in the screaming-architecture audit doc. | Updated audit-skill prose pointing to cartographer commands; cross-link from `screaming-architecture-audit/SKILL.md`. | Skill load events trigger cartographer use; analytics show migrations starting through `arch-cart migrate start`. |
| Internal: scenario template authors | Template authors will reach for cartographer when validating that new templates produce well-structured scenarios. | Sample manifest published as part of the react-vite template; CI gate that validates the template's example scenario produces zero drift findings. | Manifest schema becomes the template's documented contract for "intended architecture." |
| Internal: knowledge-observatory + ui-health integrations | These existing health-validation scenarios pick up cartographer's findings and surface them in their reports. | Connect-RPC integration with knowledge-observatory's manifest-validation and ui-health's drift surface. | Cross-scenario validation reports cite cartographer findings. |
| External: open-source Vrooli adopters | Deferred. Cartographer's value depends on the manifest pattern that defines "ideal" architecture; adopters need to commit to that pattern. | Documentation, marketing landing page, comparative analysis vs. SonarQube/NDepend/dep-cruiser. | Not pursued in v1. |
| External: commercial Vrooli tier | Deferred. Would require multi-tenant deployment hardening per `SECURITY.md`. | Auth, authorization, multi-tenancy, support model. | Not pursued in v1. |

## Launch Motion

Internal-only in v1. Sequence:

1. **Stage 0 — Scaffold (this stage)**: PRD + requirements + docs
   complete. Status: shipping 2026-05-21.
2. **Stage 1 — Language-graph dependencies**: build `go-code-graph`
   and `typescript-code-graph` as standalone scenarios. Cartographer
   cannot be implemented before these exist.
3. **Stage 2 — Cartographer MVP**: read-only graph extraction +
   manifest comparison + Conflict envelope + cycle detection + CLI
   workbench. Internal demo to the team.
4. **Stage 3 — First migration**: cartographer-assisted migration of
   a low-stakes scenario. Time and friction are measured against
   manual baseline.
5. **Stage 4 — Skill integration**: update
   `screaming-architecture-audit` skill to call cartographer
   commands. Document the change. Capture before/after audit-cost
   metrics.
6. **Stage 5 — Self-dogfooding**: cartographer validates its own
   architecture in CI. Any drift in cartographer's own code fails the
   build.
7. **Stage 6 — Rollout across active scenarios**: opportunistic use
   on scenarios undergoing real maintenance. Capture migration
   metrics for the validation experiments below.
8. **Stage 7 — Evaluate external positioning**: after N≥5
   cartographer-assisted migrations with measurable savings, revisit
   `MONETIZATION.md` and this document with real evidence.

## Messaging

| Message | Audience | Evidence Needed | Status |
|---|---|---|---|
| "Cartographer makes screaming-architecture audits programmatic." | Migration agents, audit-skill authors | Working MVP + first migration completed | in_progress (stage 0) |
| "Cartographer is the canonical L5 tool from the screaming-architecture doctrine." | Internal — referenced in the audit doc | Audit skill updated to call cartographer | deferred to Stage 4 |
| "Per-domain apply avoids the swarm-manager big-bang failure mode." | Migration agents | Per-domain apply ships with build-green guard | deferred to Stage 2 |
| "Every cartographer verdict is explainable: signals + reasons + evidence." | Skeptical maintainers who don't trust automated decisions | CLI output contract demonstrably surfaces reasons | deferred to Stage 2 |
| External-facing messages | n/a | n/a | not-applicable in v1 |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Audit-skill integration: measure tokens-per-audit before vs. after cartographer adoption | Internal | ≥50% reduction in token spend across N≥3 audits | Strong: keep cartographer as the audit substrate. Marginal: keep but iterate on CLI suggestions. Negative: investigate why the substrate isn't cheaper. |
| First migration metrics: time and conflicts-touched per migrated domain | Internal | Per-domain migration completes in <2× the time of manual file-move work on the same scope | Strong: rollout to next scenarios. Marginal: identify which step is bottlenecking. Negative: blocker on rollout. |
| Self-dogfooding CI gate | Internal | Zero unresolved drift findings against cartographer's own manifest in 30 consecutive CI runs | Pass: declare cartographer mature for self-application. Fail: fix the drift; the cartographer's own architecture takes precedence. |
| External positioning discovery | Deferred | Triggered by ≥1 unsolicited request for cartographer outside Vrooli | Run a discovery study; revisit `MONETIZATION.md`. |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging and pricing hypothesis (deferred)
- [`../../PRD.md`](../../PRD.md) — operational targets
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry that supports validation experiments
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — dogfooding decision and other durable choices
