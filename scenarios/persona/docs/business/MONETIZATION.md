# Monetization — Persona

This document is the scenario-specific monetization hypothesis. Pricing,
bundle membership, and whether to monetize at all are **operator-curated
canon** owned by `docs/monetization/`; this file records the shape and
the reasoning, not the decision.

## Purpose Of This Document

Use this document to answer:

- Who would pay for this, and for what exactly?
- Which capabilities are free, metered, or gated?
- What is the pricing hypothesis and how will it be tested?
- What is the current validation status?

## Role In Vrooli

An **interface enabler** with a high multiplier. Persona is not a
product people go looking for; it is the prerequisite that makes a whole
class of agent action possible — purchasing, enrolment, signup, trial
activation, marketplace registration, support correspondence. Its
commercial value is mostly captured by what it unblocks, and only partly
by what it charges for directly.

It is also the scenario with the strongest standalone story outside
Vrooli, for an uncomfortable reason: it is the one that asks you to
store a passport scan.

## Customer / Buyer

**Primary buyer**: the operator running agents against real services who
has concluded that a hosted identity product is not an acceptable place
for their government ID and their mailbox credentials. They are not
price-sensitive on this axis; they are custody-sensitive.

**Secondary buyers**:

- Solo operators and small studios managing several legal entities, who
  currently keep persona state in a password manager and their head.
- Compliance-shaped teams that need every outbound action attributable
  to an authorising human, and for whom "the agent did it" is not an
  acceptable audit answer.

**Not the buyer**: anyone wanting a KYC vendor, a verification service,
or an identity provider. This scenario is none of those and should not
be sold as one.

## Market Context

Know Your Agent is a funded category with a live standards track.
Vouched raised a $17M Series A on a KYA suite and donated its framework
to the Decentralized Identity Foundation, where it became KYA-OS;
Skyfire is building agent identity as a core product; Sumsub frames KYA
as an agent whose activity is explicitly authorised by a real human,
here and now.

**Every shipping product in the category is hosted.** That is the
opening. The wedge is not better identity modelling — it is *custody*:
your persona, your mailbox, your documents, your box.

## Packaging

Following the ecosystem's stated posture: the subscription buys
convenience and integrated access, never access to capability a
self-hoster could run with their own keys.

| Capability | Tier | Reasoning |
|---|---|---|
| Persona object, kinds, legal basis | **Free, permanently** | Deterministic, local, no marginal cost. Gating the core object would gate the whole scenario. |
| Persona ACL and act-as | **Free, permanently** | This is the security model. Charging for it would mean shipping a less safe free tier, which is not an acceptable trade. |
| Action journal and export | **Free, permanently** | An audit trail that costs money to read is not a credible audit trail. |
| Handoff queue and completion | **Free, permanently** | The built-in queue is the floor; it must always work. |
| Email OTP adapter | **Free, permanently** | Runs against the operator's own mailbox with their own credentials. |
| Document binding and release | **Free, permanently** | The custody work is `document-manager`'s; this is a pointer and a verb. |
| SMS / phone-number provisioning | **Metered** | Real per-message and per-number cost from a provider. Metered because the cost is real, not to create a paywall. |
| Cross-device handoff relay | **Gated** | Pure convenience — the local queue delivers the same capability. A plan differentiator that costs the user nothing to decline. |
| Hosted mailbox provisioning | **Gated** | Convenience over self-hosting a mailbox; the self-hosted path stays fully supported. |
| Paid human handoff marketplace (P2) | **Metered** | Marketplace economics; the operator is buying someone's time. |

**BYOK stays valid throughout.** Every metered or gated row above has a
bring-your-own path that is never degraded.

## Pricing Hypothesis

Untested, and stated as a hypothesis rather than a plan:

- Metered rows price at pass-through cost plus a small margin, because
  the buyer can trivially check the provider's rate and a large markup
  reads as bad faith on a security product.
- Gated rows belong in whichever bundle the monetization canon places
  the agent-operations scenarios; they are not strong enough to anchor a
  tier alone.
- **The scenario should probably not be sold standalone first.** Its
  natural motion is as the thing that makes `treasury` and agent
  purchasing work, with standalone KYA positioning as a later, separate
  bet.

## Validation Plan

| Question | Experiment | Signal |
|---|---|---|
| Is custody actually the wedge, or just our preference? | Ask five operators running agents where they currently keep persona credentials and documents | If most already accept a hosted product, the wedge is weaker than assumed and positioning should shift to attribution |
| Does the handoff model match how blocked work really feels? | Instrument time-to-completion on real handoffs once P0 ships | Handoffs sitting for days mean the delivery story, not the model, is wrong |
| Is the relay worth gating? | Ship it gated and measure decline rate | High decline means it should be free and something else should carry the tier |
| Does anyone want this without `treasury`? | Offer standalone before building P1 | No pull means bundle it and stop treating it as a separate product |

## Current Status

**Hypothesis only. Nothing validated.** No pricing has been set, no
bundle membership decided, and no experiment run. The scenario has a
PRD and a documentation contract; it has no implementation, so every row
above is a claim about a product that does not exist yet.

The most likely way this document is wrong: assuming custody-sensitivity
is a large enough segment to matter. That is the first thing to test.

## Cross-References

- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — audience, channels, messaging
- [`../../PRD.md`](../../PRD.md) — operational targets and appendix
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — what each capability is
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — the posture the wedge rests on
