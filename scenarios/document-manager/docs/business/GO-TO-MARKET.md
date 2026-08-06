# Go To Market — Document Manager

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

- **Audience:** two groups with the same need and very different buying
  behavior. *Regulated professionals* (legal and eDiscovery, healthcare,
  finance and insurance) cannot upload documents and buy evidence.
  *Technical self-hosters and researchers* already run local tooling and
  buy capability. A third group — Vrooli's own agents — is not a market
  but is the reason this exists.
- **Positioning:** the third option in a market that split in two.
  Hosted parsers understand documents but require you to upload them;
  local tools keep documents home but only OCR and tag them. This does
  both, and proves it.
- **Main claim:** *your documents never leave your machine, and we can
  prove it* — a per-document receipt showing where every processing step
  executed.
- **Proof needed:** a sample residency attestation that a compliance
  reviewer will accept as satisfying an internal control. This is the
  single piece of evidence the whole strategy rests on, and it is
  currently untested.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Self-hosting and homelab communities | The Paperless-ngx audience already chose local for privacy reasons and will adopt a strictly better local tool. Lowest-friction, highest-affinity first audience. | Working free tier, a genuinely simple install, honest comparison against Paperless-ngx. | Installs and retention past the first corpus import. |
| Compliance and infosec practitioners | The people saying "no" to hosted parsers are an underserved audience for anything that lets them say yes. Reaching the blocker rather than the user is unusual and probably the highest-leverage move. | A sample attestation artifact, a plain-English residency explanation. | A reviewer confirming the attestation would satisfy a control. |
| Academic and research users | Equations, citations and reproducibility are unserved by Zotero, which owns the workflow but has no understanding layer. | Equation and citation fidelity on real papers; a Zotero-adjacent import path. | Researchers importing an existing library. |
| Legal technology channels | Chain-of-custody and privilege are the sharpest version of the pain, and small firms already pay subscription prices for weaker tooling. | Redaction defensibility story, audit-trail walkthrough. | A firm trialling on real matter documents. |
| Vrooli ecosystem (internal) | Other scenarios adopting it as their document substrate is the compound-value proof and comes free. | Clean Connect-RPC surface, declared agent tools. | Another scenario depending on it without being asked. |

## Launch Motion

Sequenced so the cheapest disproof comes first.

1. **Prove the receipt before building the commercial layer.** Ship P0
   through `DOC-P0-014`, generate a real attestation, and put it in front
   of compliance reviewers. If it is not accepted, the wedge is wrong and
   everything downstream should change — better to learn this before
   building redaction, metering and attestation export.
2. **Earn the local audience with the free tier.** Tier-1 and tier-2
   parse, anchoring and the Reader, competing directly with Paperless-ngx
   on capability. No paywall anywhere in this path.
3. **Add understanding where it is provably local.** Enrichment against a
   local model, so the first AI features a user meets are also the ones
   that cost nothing and never leave.
4. **Then the paid tier.** Tier-3 vision and hosted enrichment, metered,
   with routing preview before spend so metering reads as honest rather
   than extractive.
5. **Then the regulated verticals.** Redaction, access control, legal
   hold and attestation export — the P1 commercial layer — once the
   underlying claim is validated.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Your documents never leave your machine — and we can prove it." | All | Per-document receipt from AI Gateway route evidence; `privacy-sensitive` fails closed. | **Claim is mechanically grounded but not externally validated.** The receipt is self-attested until the upstream correlation-key gap closes. |
| "Everything you self-host for, plus the machine actually reads the document." | Self-hosters | Working tier-1/tier-2 parse with anchors, side by side with Paperless-ngx output. | Deferred until the free tier is real. |
| "Same understanding as a hosted parser, without the upload. The comparison isn't price — it's admissibility." | Regulated buyers | Attestation artifact plus fail-closed test evidence. | Deferred until a reviewer confirms acceptability. |
| "Every claim traces to a page and a character range." | Researchers, agents | Anchor resolution surviving re-derivation. | Deferred until `DOC-P0-009` is green. |
| "The corpus compounds — the fiftieth document is worth more than the first." | All | Corpus retrieval returning anchored units across documents (`DOC-P0-023`), then federated reach through search-hub (`DOC-P0-018`). | Deferred until retrieval is green. No longer depends on ledger work. |

Two things not to say: nothing about encryption strength (table stakes,
and it was the retired scenario's mistake), and no compliance-framework
badge wall (HIPAA/GDPR/SOX/FedRAMP logos claim certification we do not
have — the honest claim is that the architecture supports the
obligations, not that it is certified).

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Show a sample attestation to compliance reviewers | Compliance practitioners | At least 2 of 3 say it would satisfy an internal control, unprompted | Below threshold → the wedge is "private AI," not "provable custody"; re-plan the P1 commercial layer before building it. |
| Free tier against Paperless-ngx on a real personal corpus | Self-hosting communities | Users keep it past the first import and say what it does better | Below threshold → the free tier is not yet good enough to earn the audience; fix before any paid work. |
| Equation and citation fidelity on 20 real papers | Research users | Equations preserved and references resolvable at a rate users call usable | Below threshold → tier-1/tier-2 are insufficient for research and tier-3 moves earlier, which makes the gateway vision role urgent. |
| Routing preview before bulk spend | Paid-tier trialists | Users proceed after seeing projected cost and routing | Below threshold → metering is not perceived as honest; revisit pricing presentation, not price. |

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`MONETIZATION.md`](MONETIZATION.md) — buyer, packaging, and pricing hypothesis
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — upstream gaps affecting the claim
- Project-level monetization strategy: `path:docs/monetization/README.md`.
