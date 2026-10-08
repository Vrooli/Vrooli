# Monetization — TypeScript Code Graph

This document records how the scenario could create revenue or support a monetizable Vrooli capability. Keep it honest: `not-applicable` is better than inventing a commercial story.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component, add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

## Role In Vrooli

TypeScript Code Graph is **internal infrastructure**, not a direct product. It exists to support architecture-cartographer, react-component-library, and future TS-static-analysis scenarios. It has no end-user surface and no path to direct revenue.

- **Direct product**: not-applicable. Wrapping `ts-morph` behind Connect-RPC is not a commercial offering on its own.
- **Internal capability**: yes. Every Vrooli scenario that needs TypeScript parsing or refactoring depends on this one rather than rolling its own.
- **SKU/bundle component**: indirect — typescript-code-graph is a prerequisite for architecture-cartographer (a possible SKU component) and a planned substrate for react-component-library's migration.
- **Revenue line**: none. Cost driver, not a revenue driver.

## Customer / Buyer

- **Primary user**: other Vrooli scenarios (architecture-cartographer, react-component-library, future TS-static-analysis siblings).
- **End human buyer**: indirect — whoever buys a scenario that depends on typescript-code-graph. The scenario itself is invisible to the buyer.
- **Pain solved**: every TS-aware Vrooli scenario would otherwise reimplement a TS parser (or rely on brittle regex, as react-component-library currently does). This scenario eliminates that work and guarantees deterministic, type-checked output with verbatim leading-comment fidelity.
- **Existing alternatives**: regex-based scanning (loses correctness and breaks on multi-line JSDoc — rcl's current pain), per-scenario `ts-morph` wrappers (loses determinism + creates drift), or a Go-native TS parser (loses `ts-morph` parity).

## Packaging

| Packaging Option | Status | Notes |
|---|---|---|
| Standalone app | not-applicable | Infrastructure scenario; no user-facing value on its own. |
| Bundle component | conditional | Implicit dependency when bundling any consumer scenario (cartographer first, rcl second). |
| Add-on | not-applicable | Nothing to add on. |
| Service/consulting assist | not-applicable | Consultants don't sell parser wrappers. |

## Pricing Hypothesis

Not applicable. TypeScript Code Graph is a **cost driver**, not a revenue driver. Its cost contribution:

- Compute cost during `Extract` (proportional to project size; `ts-morph` Project init is the dominant cost).
- Node sidecar memory footprint (a Node process per scenario instance).
- Negligible storage cost (P1 Operation Log only).
- Zero external service cost (no Ollama, no API calls).

If/when a consumer scenario monetizes and its hosted operation is metered, typescript-code-graph's compute is part of that scenario's cost model — track it there, not here.

## Validation Plan

- **Demand signal**: react-component-library migrates off regex onto this scenario's `Extract` + leading comments (PRD OT-P1-006). Track adoption by counting consumers in [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) cross-references.
- **Channel**: not-applicable — internal infrastructure.
- **Success threshold**: cartographer's MVP can be built without TS parser code of its own, AND rcl's regex parser can be retired.
- **Revisit trigger**: if rcl successfully migrates within 6 months of this scenario's v1 + a third consumer materializes within 12 months, the layered architecture decision (see [`../internal/DECISIONS.md`](../internal/DECISIONS.md)) is validated.

## Current Status

`not-applicable` — infrastructure scenario. Marked deferred in `docs/manifest.json` rather than stub-filled with invented commercial assumptions.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements (operational targets, not commercial)
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — also not-applicable
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for consumer analysis
