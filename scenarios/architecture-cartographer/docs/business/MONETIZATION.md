# Monetization — Architecture Cartographer

This document records how the scenario could create revenue or support
a monetizable Vrooli capability. Keep it honest: `not-applicable` is
better than inventing a commercial story.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

## Role In Vrooli

- **Direct product: not-applicable in v1.** Cartographer is an
  internal developer tool. There is no buyer outside Vrooli for v1.
- **Internal capability: yes.** Cartographer is the L5
  "programmatic drift checks" tool that makes screaming-architecture
  audits cheap. Every Vrooli scenario maintained, migrated, or
  reviewed benefits — that is its monetizable role.
- **SKU/bundle candidate: deferred.** Could become part of a future
  "developer productivity" bundle if Vrooli ships such a thing.
- **Revenue line: not-applicable.** No direct revenue attribution in
  v1.

## Customer / Buyer

- **Primary user**: Vrooli migration agents, scenario maintainers,
  template authors, screaming-architecture auditors.
- **Buyer (internal)**: Vrooli leadership — the value is in reduced
  manual audit cost, faster scenario migrations, and lower
  scar-tissue risk (no repeat of swarm-manager 2026-05-13).
- **Pain it solves**: today, screaming-architecture audits burn
  significant tokens and human time per scenario. Migrations risk
  big-bang failure modes. Cycle detection and resolution is manual.
  Cartographer makes the expensive manual steps cheap.
- **External buyer**: not pursued in v1. If a future tier exposes
  cartographer to external teams (open-source Vrooli adopters,
  enterprise installations), the buyer hypothesis would be
  engineering managers responsible for maintainable codebases — but
  pricing that against existing developer tools (SonarQube, NDepend,
  IntelliJ's architecture analyzer) is a separate investigation that
  is not in v1 scope.

## Packaging

| Packaging Option | Status | Notes |
|---|---|---|
| Standalone app | not-applicable in v1 | Cartographer ships as a Vrooli scenario, not a standalone product. |
| Bundle component | deferred | Could be part of a future "developer productivity" bundle alongside other code-quality scenarios (knowledge-observatory, scenario-auditor, ui-health). |
| Add-on | not-applicable | Cartographer is foundational, not an extension to a specific other scenario. |
| Service/consulting assist | deferred | A consulting offering ("we'll migrate your codebase to clean architecture") could use cartographer as an accelerator. Out of scope until cartographer itself is mature. |

## Pricing Hypothesis

- **Model**: not-applicable in v1.
- **Internal value attribution**: estimated savings of N hours per
  screaming-architecture audit × audits-per-month. Quantitative
  estimate deferred until the cartographer is in active use across
  ≥3 scenarios and time-savings data exists.
- **Comparable external products** (informational only): SonarQube's
  architecture rules, NDepend, IntelliJ's dependency analysis, dep-cruiser.
  None handle the manifest-driven domain-modeling that cartographer
  centers on; they are dependency-graph tools without the ideal-vs-actual
  comparison axis.
- **Willingness-to-pay evidence**: none captured. Not pursued in v1.
- **Cost drivers**: local runtime; SQLite analytics; no third-party
  API costs; depends on two new scenarios (go-code-graph,
  typescript-code-graph) whose cost drivers are their own.

## Validation Plan

- **Demand signal for internal value**: cartographer reaches "active
  use in ≥3 scenarios" milestone, with each scenario reporting
  measurable reduction in audit/migration time vs. pre-cartographer
  baseline.
- **Demand signal for external monetization**: not pursued in v1.
  If exploration begins, requires market discovery (interviews,
  competitive analysis vs. existing tools) — deferred.
- **Channel**: internal only in v1. See
  [`GO-TO-MARKET.md`](GO-TO-MARKET.md) for the limited internal
  rollout plan.
- **Success threshold (internal)**: cartographer-driven migrations
  ship with strictly fewer manual audit cycles than pre-cartographer
  equivalents, measured against analytics history once enough data
  exists (N ≥ 5 migrations).
- **Revisit trigger**: re-evaluate the monetization stance when (a)
  cartographer is in active use across ≥3 active scenarios, AND (b)
  external Vrooli adoption produces unsolicited demand for the same
  capability.

## Current Status

`deferred` — cartographer's commercial story is intentionally
out-of-scope in v1. Its monetizable role is captured under "Internal
capability." This document will be revisited per the trigger above.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements (operational targets, not revenue targets)
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — internal rollout plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry that would support business validation
- [`../../../../docs/monetization/README.md`](../../../../docs/monetization/README.md) — project-level monetization strategy
