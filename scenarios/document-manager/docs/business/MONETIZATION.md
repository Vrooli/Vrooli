# Monetization — Document Manager

This document records how the scenario could create revenue or support
a monetizable Vrooli capability. Keep it honest: `not-applicable` is
better than inventing a commercial story.

> **Scope boundary.** This file records the scenario-local hypothesis.
> Whether to charge, at what price, and which bundle it joins is
> operator-curated canon under `path:docs/monetization/` and is never
> decided here. The engineering contract for wiring free / metered /
> gated is `path:docs/concepts/PAID_FEATURES.md`.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

## Role In Vrooli

- **Direct product: yes.** It has a real external buyer and a defensible
  wedge, which is unusual for an enabler.
- **Internal capability: yes, and this comes first.** It is the ingestion
  half of the infinite ledger. Even with zero external revenue it earns
  its place by making every Vrooli research, proof-search and
  truth-seeking capability able to work from real source documents with
  citations.
- **SKU/bundle candidate: likely both bundles.** The SKU map already
  lists "note-taking / knowledge-base" as a pending candidate expected to
  serve business and lifestyle alike. This scenario is the closest
  existing thing to that slot. Placement is an operator call.
- **Revenue line: metered AI usage**, through the existing LPBS credit
  path. No new billing mechanism is proposed or needed.

## Customer / Buyer

- **Primary user:** anyone who has documents they cannot upload. In
  practice: legal and eDiscovery teams, healthcare handling PHI, finance
  and insurance under audit obligations, researchers working from papers,
  and privacy-motivated self-hosters.
- **Buyer:** in regulated verticals the buyer is frequently *compliance*
  rather than the end user — which matters, because compliance buys
  evidence, not features. In prosumer contexts user and buyer are the
  same person and the purchase is driven by capability.
- **Pain:** the market has split in two and neither half serves them.
  Hosted parsers offer understanding but require uploading the document,
  which chain-of-custody and minimum-necessary obligations increasingly
  forbid. Self-hosted tools keep the file local but stop at OCR and
  tagging — no understanding, no citable anchors, no audit trail.
- **Existing alternatives:** Paperless-ngx, Zotero, Docling and Marker
  (local, free, no understanding layer); LlamaParse, Reducto, Azure
  Document Intelligence, AWS Textract and Google Document AI (hosted,
  capable, structurally unable to keep the document local). Nobody
  currently offers local understanding *with* a custody record.

## The Wedge

The differentiator is not "private AI." It is **a per-document receipt
showing where every processing step executed**, which is buildable
because AI Gateway already records `privacy_class`,
`selected_provider`, `selected_locality`, `profile`, `policy_reasons`
and redaction flags for every inference, and because the
`privacy-sensitive` profile is documented to fail closed rather than
fall back remote.

Regulators have stopped accepting vendor attestations about residency
and now want technical verification plus a record of what was processed
where. A hosted competitor cannot produce that record, because for them
the answer never varies.

**Caveat that must not be lost:** `RouteEvidence` has no caller
correlation key today, so a receipt assembled now is self-attested by
this scenario. Closing that gap upstream is what upgrades the pitch from
self-report to corroborated evidence — see
[`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md).

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Standalone app | plausible | The custody story stands alone for regulated buyers. Would need its own onboarding and a deployment story beyond local lifecycle. |
| Bundle component | **preferred** | Fits the pending "note-taking / knowledge-base" slot expected to span both bundles. Placement is operator canon, not decided here. |
| Add-on | no | It is not an extension of another SKU; it is a capability other scenarios build on. |
| Service/consulting assist | plausible, later | Compliance-configuration work (retention, hold, redaction policy) is genuinely consultative, but only after the product exists. |

## Pricing Hypothesis

- **Model:** free local tiers, metered remote inference. Nothing gated.
  The subscription sells convenience and integrated routing, never
  access — a self-hoster with their own GPU gets the entire capability
  free, and that is the intent rather than a leak.
- **Comparable products** (order-of-magnitude anchors, vendors' own
  published figures, not independently benchmarked):
  - Hosted parse: LlamaParse ~$0.003/page; Reducto ~$0.005–0.015/page;
    cloud OCR ~$1.50 per 1,000 pages, with Textract tables/forms
    ~$15 per 1,000 pages.
  - Adjacent subscriptions: small-firm legal document automation
    $50–200/month; purpose-built legal AI $69–149/month; Clio Duo
    $49–59/month as an add-on; Harvey ~$1,000–2,000/seat/month at
    mid-market.
  - Local incumbents: Paperless-ngx is free and well-liked, which sets
    the bar the free tier must clear on capability rather than price.
- **Willingness-to-pay evidence:** none collected directly yet. The
  inference is that regulated buyers already pay subscription prices for
  weaker tooling and are being pushed off hosted APIs by their own
  compliance functions — but that is a hypothesis, not a measurement.
- **Cost drivers:** tiers 1 and 2 are local compute with no marginal
  cost. Tier 3 and enrichment carry real token cost. Embeddings are the
  highest-frequency paid operation, which makes the unresolved OpenRouter
  `embedding.default` gap a commercial issue rather than only a technical
  one.

## Free / Metered / Gated

Per `path:docs/concepts/PAID_FEATURES.md`:

| Capability | Mode | Reasoning |
|---|---|---|
| Tier-1 and tier-2 parse, anchoring, versioning, corpus, custody receipts, export | **free** | Deterministic, local, no marginal cost. Permanently free — it competes with genuinely good free tools, and gating it would gate something a self-hoster could already run. |
| Tier-3 vision parse, enrichment, embeddings | **metered** | Real token cost. Reserve → execute → finalize through LPBS. |
| Bulk re-derivation at corpus scale | **metered** | Same mechanism, larger volume. |
| — | **gated** | Nothing. Deliberate: the value is real compute or nothing. A gated feature appearing later signals the framing drifted. |

**BYOK must remain free of credit charge** (`DOC-P1-022`). This is a hard
rule from canon, not a concession.

## Validation Plan

- **Demand signal needed:** evidence that a regulated buyer will accept
  a locally-produced residency attestation as satisfying an internal
  control. That is the load-bearing assumption — if compliance teams
  will not accept it, the wedge collapses to "private AI," which is
  crowded.
- **Cheapest test:** produce a sample attestation from the P0 receipt
  and put it in front of two or three compliance reviewers before
  building any of the P1 commercial layer.
- **Channel:** see [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- **Success threshold:** defined by project-level monetization taxonomy,
  not here.
- **Revisit trigger:** when `DOC-P0-014` (processing receipt) is real
  and can produce an artifact worth showing someone.

## Current Status

`hypothesis` — the PRD identifies a customer, a wedge, and a metering
mechanism that already exists. Nothing is validated with a real buyer,
and the central assumption (that a self-produced attestation is
acceptable evidence) is untested. Do not treat the comparables above as
a pricing decision.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — the upstream gaps that affect the pitch
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- Project-level monetization strategy: `path:docs/monetization/README.md`.
