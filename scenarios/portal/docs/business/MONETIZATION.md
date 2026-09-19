# Monetization — Portal

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

- Direct product: candidate after native release acceptance.
- Internal capability: active shared front door for local operators and agents.
- SKU/bundle candidate: Portal desktop companion with optional owner providers.
- Revenue line: unvalidated; no launch claim is made.

## Customer / Buyer

- Primary user: Vrooli operator or builder moving between chat, browser, and
  owned desktop capabilities.
- Buyer: unknown; validate separately for individual and team workflows.
- Pain: repeated context transfer and unclear provider/platform readiness.
- Existing alternatives: not yet captured.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Personal local companion | hypothesis | Local-first Portal chat and optional native control; no hosted service required by default. |
| Team operator bundle | hypothesis | Shared Portal plus owner-managed agent, browser, and device capabilities; requires account, support, and security validation. |
| Hosted relay add-on | deferred | Requires measured relay cost, tenancy, and deployment-owner evidence. |

## Pricing Hypothesis

- Model: unpriced hypothesis; do not publish a price until receipts and buyer
  interviews exist.
- Comparable products: none captured yet.
- Willingness-to-pay evidence: none captured yet.
- Cost drivers: local runtime is primarily host cost; optional inference and
  relay cost is estimated as `prompt_tokens * input_rate + completion_tokens *
  output_rate + relay_bytes * egress_rate + hosted_minutes * host_rate`.
- Measured inputs currently available: Portal persists prompt/completion token
  counts and provider/model in usage records. Input/output rates, relay bytes,
  hosted minutes, support effort, and platform signing cost are unmeasured.

## Validation Plan

- Demand signal needed: five operator interviews per proposed package and one
  repeat workflow using a receipt-backed supported platform.
- Channel: internal operator cohort first; public channel remains deferred.
- Success threshold: repeat usage plus measured cost coverage; no causal value
  claim from the current telemetry alone.
- Revisit trigger: native release acceptance, security review, and a named
  buyer segment.

## Current Status

`hypothesis` — the product role and cost model are explicit, but buyer,
pricing, and hosted-cost measurements are still missing.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- [`../../../../docs/monetization/README.md`](../../../../docs/monetization/README.md) — project-level monetization strategy
