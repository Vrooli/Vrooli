# Monetization — Go Code Graph

This document records how the scenario could create revenue or support a monetizable Vrooli capability. Keep it honest: `not-applicable` is better than inventing a commercial story.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component, add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

## Role In Vrooli

Go Code Graph is **internal infrastructure**, not a direct product. It exists to support architecture-cartographer and future Go-static-analysis scenarios. It has no end-user surface and no path to direct revenue.

- **Direct product**: not-applicable. Wrapping `go/packages` behind Connect-RPC is not a commercial offering on its own.
- **Internal capability**: yes. Every Vrooli scenario that needs Go parsing depends on this one rather than rolling its own.
- **SKU/bundle component**: indirect — go-code-graph is a prerequisite for architecture-cartographer, which *may* become a SKU component. If cartographer ships in a "Refactoring & Maintenance" bundle, go-code-graph is bundled implicitly as a substrate dependency.
- **Revenue line**: none. Cost driver, not a revenue driver.

## Customer / Buyer

- **Primary user**: other Vrooli scenarios (architecture-cartographer, future siblings, react-component-library's planned TS-graph parallel).
- **End human buyer**: indirect — whoever buys a scenario that depends on go-code-graph. The scenario itself is invisible to the buyer.
- **Pain solved**: every Go-aware Vrooli scenario would otherwise reimplement a Go parser. This scenario eliminates that work and guarantees deterministic, type-checked output.
- **Existing alternatives**: roll-your-own with `go/parser` (loses type info), regex-based scanning (loses correctness), or per-scenario wrappers of `go/packages` (loses determinism + creates drift across scenarios).

## Packaging

| Packaging Option | Status | Notes |
|---|---|---|
| Standalone app | not-applicable | Infrastructure scenario; no user-facing value on its own. |
| Bundle component | conditional | Implicit dependency when bundling any consumer scenario (cartographer first). |
| Add-on | not-applicable | Nothing to add on. |
| Service/consulting assist | not-applicable | Consultants don't sell parser wrappers. |

## Pricing Hypothesis

Not applicable. Go Code Graph is a **cost driver**, not a revenue driver. Its cost contribution:

- Compute cost during `Extract` (proportional to module size; full type-loading is CPU-heavy).
- Negligible storage cost (P1 Operation Log only).
- Zero external service cost (no Ollama, no API calls).

If/when a consumer scenario monetizes and its hosted operation is metered, go-code-graph's compute is part of that scenario's cost model — track it there, not here.

## Validation Plan

- **Demand signal**: any new scenario that asks for Go parsing → use this one, not a custom parser. Track adoption by counting consumers in [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) cross-references.
- **Channel**: not-applicable — internal infrastructure.
- **Success threshold**: cartographer's MVP can be built without parser code of its own (the original motivating use case).
- **Revisit trigger**: if a second consumer outside cartographer materializes within 6 months of v1, the layered architecture decision (see [`../internal/DECISIONS.md`](../internal/DECISIONS.md)) is validated.

## Current Status

`not-applicable` — infrastructure scenario. Marked deferred in `docs/manifest.json` rather than stub-filled with invented commercial assumptions.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements (operational targets, not commercial)
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — also not-applicable
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for consumer analysis
