# Monetization — Scenario to Plugin

This document records how this scenario relates to Vrooli's commercial
model. Offer Desk is authoritative for live channel and offer state; this
page keeps the reasoning.

## Purpose Of This Document

Use this document to answer:

- Is this scenario sold, or does it enable something that is?
- Who is the buyer, and what would they be paying for?
- What has to be true before any pricing conversation is honest?
- What is the current, evidenced status?

## Role In Vrooli

**This scenario is not itself a product for sale. It is the mechanism that
makes the `skill-registries` channel possible.**

That distinction matters and should not blur over time. Scenario to Plugin
is infrastructure: it packages, verifies, and publishes *other* scenarios.
Its commercial contribution is entirely indirect — it lowers the cost and
raises the trustworthiness of putting a Vrooli capability in front of an
external agent, and that exposure is what may later convert.

The revenue line it feeds is [`subscription`](../../../../docs/monetization/catalogs/revenue-lines/subscription.md),
through the [`skill-registries`](../../../../docs/monetization/catalogs/channels/skill-registries.md)
channel. It does not feed `affiliate-commerce`, `consumer-products`, or any
recommendation surface.

### Recommendation-blindness

Both `consumer-products` and `affiliate-commerce` carry an architectural
rule that the agent producing a recommendation must not know what Vrooli
sells. Packages published through this ramp **do not** violate that rule:
a published plugin teaches an *external* agent to use a Vrooli capability;
it does not produce recommendations to a Vrooli end user.

If a published plugin ever calls into a scenario that then recommends
products back into a lifestyle-bundle context, the rule applies in full
and that plugin needs explicit review. Default assumption: plugins wrap
business-bundle capabilities, where this risk is low. This ramp should
record the bundle context of each declaration so the question is
answerable rather than assumed.

## Customer / Buyer

There are two distinct populations and conflating them produces bad
pricing reasoning.

| Population | Relationship | Pays? |
|---|---|---|
| External agent runtimes and their users | Install a published plugin; may never touch Vrooli otherwise | No, at first. Free usage is the validation instrument. |
| Vrooli operators publishing capabilities | Use this ramp internally | No. Internal infrastructure. |
| Downstream converts | Installed a plugin, then wanted the managed convenience layer — hosted gateway, hosted infrastructure, the broader bundle | Yes, through `subscription`. |

The buyer this scenario is ultimately serving is the third row, and that
row is a hypothesis until install-to-subscription correlation exists.

## Packaging

Free publication is deliberate, not leakage. A published plugin lets an
external agent validate that a Vrooli capability has standalone value
*before* the surrounding subscription surface is mature. The near-term
return is proof: installs, task fit, failure reports, registry trust
signals, and referrer traffic.

Where paid capability does appear, it appears as **entitlement inside a
published plugin**, never as a paywalled download:

- The plugin itself is free to install and free to inspect.
- A skill that wraps a paid capability packages a sign-in path that
  resolves entitlement at run time (`OT-P1-004`, `PLG-REHEARSE-ENTITLEMENT`).
- No credential is ever embedded in an artifact. The rehearsal proves this
  before publication rather than asserting it afterwards.

A plugin that could not be inspected or run without paying would fail the
channel on its own terms: registries downrank it, and an agent cannot
evaluate it.

## Pricing Hypothesis

**There is no pricing hypothesis for this scenario, and inventing one
here would create a number with no provenance.**

Pricing belongs to `subscription` and to the monetization team. What this
scenario owes that conversation is the measurement that makes it possible:
per-plugin installs, referrer origin, and downstream conversion
attribution (`OT-P1-003`, `PLG-DIST-ATTRIBUTION`). Until that exists,
aggregate channel numbers cannot distinguish a capability with real
standalone value from one that was merely indexed.

## Validation Plan

The channel's own activation criteria are the validation plan; this
scenario exists to satisfy the first two and instrument the third.

| Criterion | Owner | State |
|---|---|---|
| At least one Vrooli capability is standalone-installable | Upstream scenario owners | **Unmet.** This is the gating prerequisite and this ramp fails closed without it. |
| At least one signed, scanned package is live in a curated registry | This scenario | **Unmet.** No package has been composed. |
| 60+ days of install, referrer, and conversion telemetry exists | This scenario (`OT-P1-003`), Offer Desk | **Unmet.** Attribution is not implemented. |

Pilot discipline, when the first package ships: a fixed 60–90 day window,
one named capability, a stated validation hypothesis, and a sunset-or-scale
decision at the end. `workspace-sandbox` and `git-control-tower` are the
strongest pilot candidates — both are already Offer Desk deliverables and
both have sharp standalone value.

## Current Status

- Offer Desk channel `skill-registries`: **CANDIDATE**. It has no
  machine-evaluable trigger registered, so the channel cannot currently be
  promoted from evidence. Registering that trigger is cheap and should not
  wait for this scenario.
- This scenario: **pre-implementation.** Contract and documentation
  authored; no code.
- Nothing here is monetized, and no claim of revenue attribution should be
  made until per-plugin attribution is real.

## Cross-References

- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — audience, channels, launch motion
- [`../../../../docs/monetization/catalogs/channels/skill-registries.md`](../../../../docs/monetization/catalogs/channels/skill-registries.md) — channel doctrine and activation criteria
- [`../../../../docs/monetization/catalogs/revenue-lines/subscription.md`](../../../../docs/monetization/catalogs/revenue-lines/subscription.md) — the revenue line this feeds
- [`../../PRD.md`](../../PRD.md) — operational targets, including `OT-P1-003` and `OT-P1-004`
- [`../concepts/DATA.md`](../concepts/DATA.md) — what attribution stores and deliberately does not store
