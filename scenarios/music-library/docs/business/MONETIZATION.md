# Monetization — Music Library

How this scenario relates to what Vrooli sells.

> **Canon boundary.** Pricing, bundles, and SKU membership are operator-curated
> monetisation canon under `docs/monetization/`. Agents do not edit that canon.
> This document records intent and scenario-local obligations; it does not set
> prices and does not claim a membership the catalogue has not recorded.

## Purpose Of This Document

Use this document to answer:

- Which bundle is this scenario intended for, and what is its actual status?
- What architectural obligations does that bundle impose?
- What has to be proven before it can be sold?

## Role In Vrooli

This scenario is intended as the **first lifestyle-bundle scenario**. Every
monetisation-track scenario built so far is business-oriented — development,
infrastructure, marketing, operations. This is the first built for personal use.

That makes it a proving ground for two things the lifestyle bundle needs and the
business bundle never exercised:

1. **Recommendation blindness in code.** The lifestyle-bundle rule is that the
   component producing a recommendation must not know what is sold or
   commission-bearing, with offer insertion strictly downstream. The
   `affiliate-commerce` revenue line is explicitly gated on that architecture
   existing in code. Building it here unblocks it generally.
2. **A consumer-facing surface for a non-developer audience**, which is a different
   design discipline from the operational consoles the business bundle ships.

## Customer / Buyer

A listener who owns music and is dissatisfied with commercial recommendation —
specifically with its opacity and its reluctance to follow them into new territory.
Not a developer, not an operator. Household or single-listener, consistent with the
"no seat proliferation" pricing principle.

Two capabilities have no commercial equivalent, because the incumbents' economics
forbid them: an inspectable and editable taste model, and unmetered generation.
Hosted generators meter output because each generation carries a royalty; local
generation does not.

## Packaging

Intended as a lifestyle-bundle scenario. Bundle membership is recorded in the
catalogue's scenario-to-SKU map, which is operator-curated — **this scenario has no
entry there yet**, and adding one is an operator decision, not an agent action.

Per the pricing principle that a bundle's scenarios do not vary by tier, this
scenario would ship identically across delivery tiers, with only the delivery mode
differing.

An important dependency: this scenario is only sellable in a configuration where
every model it depends on permits commercially distributed output. That is the
permissive lane in `music-tools`, and it is a weaker analysis stack than the
personal configuration. **The personal build and the sellable build are not the same
build**, and that difference should be settled before it is discovered.

## Pricing Hypothesis

**None stated.** The lifestyle bundle's pricing is `TBD` in the pricing matrix, and
setting it is an operator decision informed by market benchmarking. Inventing a
number here would contradict the canon.

What can be said: the value proposition is unmetered use plus an inspectable taste
model, and the open-source self-hosted path with bring-your-own-keys sets the price
floor for it — as it does for every Vrooli tier.

## Validation Plan

1. **Prove the blindness boundary in code**, with a test that fails if `ranking` can
   observe offer state. This is the gate for the whole revenue line, not just this
   scenario.
2. **Establish that generated music survives repeat listening.** This is the
   product's largest unvalidated assumption. Background music must be inoffensive
   once; music a listener chooses must reward a fifth play. Test it early and
   cheaply, before the surrounding product is built.
3. **Prove the permissive-lane configuration is a usable product**, not just a
   working build.
4. **Measure how many comparisons are needed before ranking beats random.** That
   number is the first-run experience, and it is the honest measure of whether the
   taste model works at all.

## Current Status

| Item | Status |
|---|---|
| Target bundle | Lifestyle — intent, not yet recorded |
| Lifestyle bundle status | `candidate` (business is the only `active` base bundle) |
| Entry in the scenario-to-SKU map | **None** — operator action |
| Pricing | `TBD` in canon; none proposed here |
| Blindness boundary | Designed and documented; **not implemented, not tested** |
| Repeat-listening assumption | **Unvalidated** |

## Cross-References

- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — the blindness boundary
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — blindness as an integrity control
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — why generation is load-bearing
- `docs/monetization/` (repo root) — the canon this document cites
