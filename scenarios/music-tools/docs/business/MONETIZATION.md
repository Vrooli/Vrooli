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

   As of 2026-09-18 this is a **recorded decision with revisit triggers**, not an
   assumption — see `../internal/DECISIONS.md`. It was previously stated here as a
   bare property, which made a real fork look like a fact of nature. The siblings
   disagree about this fork and neither is a house pattern: `image-tools` routes a
   BYOK rung through `ai-gateway` with no subscription and no markup, while
   `audio-tools` runs a Vrooli rung through LPBS with `ai_credits` and
   `voice_minutes` meters. Music is the modality where staying local has the
   strongest case, because hosted music generation is precisely where the royalty
   and training-data-provenance exposure lives.

   Two mechanical blockers back the decision up: the gateway's `RequestKind` enum
   has no audio member, and no hosted music provider is wired in `resources/`.
   Adding the lane is real work, not a configuration flag.

   **Correction, same day.** An earlier version of this section implied that a
   hosted rung would require this scenario to declare meters and entitlement. It
   would not, and the argument has been withdrawn. `image-tools` calls the gateway
   as an ordinary registered provider and declares no `.vrooli/monetization.json`,
   no entitlement, and no quota logic; subscription and metering live entirely in
   `ai-gateway`, which declares `ai_credits` and `requires_entitlement` on its own
   behalf. The no-hosted-rung decision now rests on licence exposure alone, which
   is the leg that holds.

   **The cost of staying local, stated plainly:** composition has no CPU rung and
   will not get one, so an operator without a capable GPU gets no composition at
   all. `image-tools` degrades to CPU and still serves that user; this scenario
   refuses. "Local-first" is therefore also "GPU owners only", and that is now a
   recorded consequence with its own revisit trigger rather than an unexamined
   side effect.
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
| Meter declaration | **None** — no `.vrooli/monetization.json`, because there is no metered path. Note this would still be true with a hosted rung: `image-tools` has a BYOK rung and declares no meters either, because `ai-gateway` owns metering. |
| Hosted/gateway rung | **Decided against, 2026-09-18**, on licence exposure alone, with three revisit triggers in `../internal/DECISIONS.md`. The provider-chain *seam* is adopted; the rung is not built. |
| Hardware reach | **Bounded.** No CPU rung for composition, so no GPU means no composition. Recorded as a decision, 2026-09-18. |
| Lane split designed | Yes — `docs/concepts/ARCHITECTURE.md`, `docs/reference/model-registry.md` |
| Lane split implemented | **No** — no implementation exists |
| Permissive-lane build proven | **No** — `OT-P1-002` not started |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — licence lanes
- [`../reference/model-registry.md`](../reference/model-registry.md) — per-model licence and lane
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — licence violation as a threat
- `docs/monetization/` (repo root) — the canon this document cites
