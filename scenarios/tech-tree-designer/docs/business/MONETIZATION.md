# Monetization — Tech Tree Designer

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

- Direct product: deferred.
- Internal capability: scenario-interface graph and contract-first planning surface.
- SKU/bundle candidate: deferred.
- Revenue line: deferred.

## Customer / Buyer

- Primary user: Vrooli operators and implementation agents planning scenario interfaces.
- Buyer: not applicable for the current internal meta-capability.
- Pain: scenario dependency drift and late interface design create avoidable rework across the fleet.
- Existing alternatives: ad hoc diagrams, manual proto browsing, and scenario-specific planning notes.

## Packaging

| Packaging Option | Status | Notes |
|---|---|---|
| Standalone app | deferred | Possible only if planning surfaces become customer-facing. |
| Bundle component | candidate | Could support enterprise architecture/planning installations as an internal operator tool. |
| Add-on | deferred | Use only when scenario clearly extends another SKU. |
| Service/consulting assist | candidate | Helps scope and de-risk done-for-you scenario delivery by validating contracts early. |

## Pricing Hypothesis

- Model: deferred.
- Comparable products: none captured yet.
- Willingness-to-pay evidence: none captured yet.
- Cost drivers: local runtime, SQLite storage, and proto generation; no third-party AI/API cost in the shipped scope.

## Validation Plan

- Demand signal needed: repeated operator use during scenario planning and evidence that early proto validation reduces implementation rework.
- Channel: internal Vrooli planning workflows first; external packaging only after user-facing demand is explicit.
- Success threshold: material percentage of new scenario plans start as validated proto contracts.
- Revisit trigger: TTD becomes part of a repeatable paid implementation or enterprise deployment workflow.

## Current Status

`internal-meta-capability` — shipped for Vrooli's planning loop. No direct pricing hypothesis is active.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- [`../../../../docs/monetization/README.md`](../../../../docs/monetization/README.md) — project-level monetization strategy
