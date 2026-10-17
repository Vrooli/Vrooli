# Monetization — Image Tools

This document records how the scenario could create revenue or support
a monetizable Vrooli capability. Keep it honest: `not-applicable` is
better than inventing a commercial story. Everything below is an early,
pre-implementation hypothesis derived from `PRD.md`, not a committed plan.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

## Role In Vrooli

image-tools plays a deliberate dual role, and both sides compound:

- **Internal capability primitive (primary, near-term).** It is the
  permanent, reusable image layer the rest of the system builds on. Other
  scenarios and agents — campaign-content-studio, landing-page-business-suite,
  palette-gen, and the future rich-media-studio — compose image-tools instead
  of reinventing resize/convert/watermark, AI generation/upscale/restore, or
  OCR/analysis. Every consumer raises the platform's aggregate value without
  needing a direct dollar attached. This is the compounding-intelligence side
  of the dual mandate.
- **Standalone deliverable (secondary, later).** The same headless-complete
  core can ship on its own as a tool, a SaaS edge, or a desktop app for users
  who want a private, scriptable image toolbox.
- **Differentiator: local-first / private by default.** Deterministic ops run
  with zero downloads on the user's own hardware; AI ops run via opt-in local
  models or the user's own cloud keys. "Your images never leave your machine"
  is a structural advantage over cloud editors for privacy-sensitive work.

## Customer / Buyer

- **Primary user (early):** internal Vrooli scenarios and automation agents
  that need programmatic image operations. They are the first and most certain
  consumers, validated by dogfooding.
- **Prosumers / creators:** individuals producing social, marketing, or
  portfolio imagery who want pro editing plus on-demand AI without a cloud
  subscription.
- **Small marketing teams:** need batch processing, watermarking, format
  conversion, and quick AI generation/cleanup at predictable cost.
- **Developers / automation builders:** want every operation scriptable from a
  CLI and callable from an API for pipelines and watch-folders.
- **Privacy-sensitive users (legal, healthcare, enterprise):** cannot send
  source images to third-party clouds; local-first execution is a hard
  requirement, not a preference.
- **Buyer vs. user:** for prosumers the user is the buyer; for teams/enterprise
  the buyer is a team lead or IT/compliance owner who values the local-first and
  no-lock-in posture.
- **Pain:** cloud editors charge per-operation or per-seat, create vendor
  lock-in, and exfiltrate source images; heavyweight desktop suites are not
  headless or composable.
- **Existing alternatives:** cloud image APIs and SaaS editors, heavyweight
  desktop suites, and ad-hoc per-scenario reimplementation inside Vrooli (the
  status quo this scenario is meant to retire).

## Packaging

| Packaging Option | Status | Notes |
|---|---|---|
| Standalone app | hypothesis | Desktop/PWA packaging of the local-first core for prosumers and privacy-sensitive users; revisit after the first real domain ships. |
| Bundle component | hypothesis | Embeddable UI component + stable internal API contract (audio-tools voice-button pattern) so rich-media-studio and marketing scenarios consume it; the primary near-term role. |
| Add-on | hypothesis | Sells as a capability add-on inside SKUs that need image work (landing-page-business-suite, campaign-content-studio). |
| Service/consulting assist | hypothesis | Accelerates done-for-you content delivery where image generation/cleanup is part of the engagement. |

Free/cheap layer: all deterministic ops (resize, crop, convert, compress,
watermark, metadata, thumbnails) are zero-download and effectively free at the
margin. AI layer: opt-in local models (no per-call cost, user pays in hardware
and disk) **or** BYOK cloud (user pays the provider directly, with a mandatory
pre-op cost estimate). Premium candidates: a managed/curated model catalog,
hosted GPU for users without one, and team/enterprise features (shared recipes,
provenance, audit).

## Pricing Hypothesis

- **Model (hypothesis):** local-first means near-zero marginal cost per
  operation, so direct per-op pricing is the wrong lever. Monetize instead via
  (a) desktop/SaaS packaging, (b) managed-model-catalog convenience, (c) hosted
  GPU for users without local acceleration, and (d) team/enterprise features.
- **BYOK pass-through:** cloud-tier AI ops pass provider cost straight to the
  user, surfaced as a pre-op estimate before execution (OT-P0-011 /
  OT-P1-007). Vrooli does not mark up the user's own keys.
- **Comparable products:** cloud image-editing APIs and SaaS editors price
  per-operation, per-credit, or per-seat; desktop suites price per-license or
  subscription. None captured with hard numbers yet.
- **Willingness-to-pay evidence:** none captured yet — this is pre-implementation.
- **Cost drivers:** deterministic ops are local and cheap; the real cost drivers
  are model download/disk, local GPU/CPU compute, and (only on the BYOK tier)
  third-party provider API spend borne by the user.

## Validation Plan

- **Dogfood first.** Land image-tools as a dependency of marketing scenarios
  (campaign-content-studio, landing-page-business-suite) and palette-gen before
  any external sale. Internal consumption is the first demand signal.
- **Demand signal needed:** measurable adoption by other scenarios/agents (count
  of consuming scenarios and call volume), the op-usage mix (deterministic vs.
  AI), and the local-vs-BYOK split per AI op.
- **Telemetry source:** the OT-P0-012 measure blocks (op latency, throughput,
  queue wait, fallback-tier usage) plus consumption counts feed this directly —
  see [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md).
- **Gate paid tiers on demonstrated demand.** Only build the managed-catalog,
  hosted-GPU, or team tiers once usage shows users hitting the local-hardware
  ceiling or asking for convenience, not on speculation.
- **Channel:** see [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- **Revisit trigger:** first real domain reaches validated scenario tests and
  has a clear internal consumer or external user.

## Current Status

Pre-implementation. PRD, requirements, and the docs foundation were authored
2026-06-16; monetization is hypothesis-stage. The near-term, low-risk path is
the internal capability primitive; standalone/SaaS/desktop monetization is
deferred until adoption data justifies it.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- [`../../../../docs/monetization/README.md`](../../../../docs/monetization/README.md) — project-level monetization strategy
