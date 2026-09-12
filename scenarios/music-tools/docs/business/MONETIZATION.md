# Monetization — Music Tools

How this scenario relates to what Vrooli sells.

> **Canon boundary.** Pricing, bundles, and SKU membership are operator-curated
> monetisation canon under `docs/monetization/`. Agents do not edit that canon.
> This document *cites* it and records scenario-local implications; it never sets a
> price or claims a bundle membership that the catalogue does not already record.

## Purpose Of This Document

Use this document to answer:

- Is this scenario sold directly, or does it enable something that is?
- What must be true here for a paid product built on it to be lawful and viable?
- What is decided, and what is still open?

## Role In Vrooli

**This scenario is not sold on its own.** It is a capability primitive, in the same
relationship to saleable products that `image-tools` has to `backdrop-studio` and
`asset-studio`. Its commercial role is to make consuming scenarios possible and to
keep their unit economics intact.

Two properties carry that role:

1. **Zero marginal cost per operation.** Models run locally on hardware the operator
   already owns. There is no per-generation royalty, no per-request billing, and no
   third-party service that can change its terms. Hosted competitors in this space
   now carry per-generation royalties under licensing settlements; this scenario
   carries none.
2. **Licence lanes.** Commercial usability is recorded per model, so a product built
   on this scenario can be configured to use only models whose licences permit
   commercially distributed output.

The second is the load-bearing one. Several of the strongest analysis models are
non-commercial or share-alike, and at least one useful tool declares no licence at
all. Without a lane split, a product built on this scenario would be unshippable —
and the split is far cheaper to maintain from the first commit than to retrofit.

## Customer / Buyer

No direct buyer. The buyer of a consuming product benefits from this scenario
indirectly: unmetered operations, no dependence on a third-party music API, and
provenance recorded against every artifact.

The first consumer is `music-library`, intended for the lifestyle bundle.

## Packaging

Not packaged or sold independently. It ships as a dependency of whatever product
scenario needs it, and its delivery tier follows that product's.

The **no-container decision** is a packaging constraint as much as an architectural
one: the desktop delivery tier cannot assume a container runtime, so both managed
resources provision natively.

## Pricing Hypothesis

**Not applicable.** This scenario has no price. Its commercial contribution is cost
structure, not revenue.

The relevant hypothesis belongs to consuming products: that local, unmetered
generation and analysis is worth paying for precisely because hosted alternatives
must meter it. That hypothesis is theirs to validate, not this scenario's.

## Validation Plan

What must be demonstrated here before a paid product can depend on it:

1. **A permissive-lane build works.** Configured to the permissive lane, the
   scenario must still satisfy the composition and analysis contracts end to end.
   This is `OT-P1-002` and it is the single most important commercial gate.
2. **Lane enforcement is tested, not conventional.** A test must fail if a
   permissive build can resolve a restricted model.
3. **Provenance is complete.** Every artifact records model, licence lane, and
   applied profile rung, so a consumer can meet disclosure obligations.
4. **Cost claims are measured.** "Zero marginal cost" is true of licensing but not
   of electricity or wall-clock. Consuming products should quote measured throughput,
   not the phrase.

## Current Status

| Item | Status |
|---|---|
| Sold directly | No — capability primitive |
| SKU membership | None, and none expected |
| Lane split designed | Yes — `docs/concepts/ARCHITECTURE.md`, `docs/reference/model-registry.md` |
| Lane split implemented | **No** — no implementation exists |
| Permissive-lane build proven | **No** — `OT-P1-002` not started |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — licence lanes
- [`../reference/model-registry.md`](../reference/model-registry.md) — per-model licence and lane
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — licence violation as a threat
- `docs/monetization/` (repo root) — the canon this document cites
