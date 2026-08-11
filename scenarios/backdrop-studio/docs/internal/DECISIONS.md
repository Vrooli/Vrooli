# Decisions

Durable decisions and the reasoning behind them. A decision here is one a future
agent should not re-open without new information.

---

## D-001 — A new scenario, not an extension of `asset-studio`

**Date:** 2026-08-11 · **Status:** accepted

**Decision.** Backdrop Studio is a separate scenario that *consumes*
`asset-studio` rather than living inside it.

**Why.** The two products have different units of value. `asset-studio` exists so
that the same character renders the same way in six weeks — its gate is an
operator confirming a frame depicts the subject it claims to. Backdrop Studio has
no subject to conform to; its unit of value is fitness for a layout and its gate
is measured contrast under overlaid type. Folding one into the other would mean
an identity registry with no identities in it.

**What was considered.** Extending `asset-studio` with a backdrop mode. Rejected:
it would push landing-page layout concerns into a scenario whose whole design is
organised around identity conformance, and would make both harder to reason
about.

**The narrow place they meet.** Every render that invokes a model or costs money
releases through `asset-studio`, so there is exactly one system of record for
spend, provenance, and disclosure. Procedural renders do not (see D-004).

---

## D-002 — Every generation strategy terminates in a deterministic treatment pass

**Date:** 2026-08-11 · **Status:** accepted · **Load-bearing**

**Decision.** Raw model output is never released. The style's treatment chain is
the final stage of every render regardless of strategy (`RND-001`).

**Why.** Two properties follow, both structural rather than prompt-dependent:

1. **Palette lock.** Duotone, posterization and dithering discard source colour
   and remap luminance onto the active brand's inks, so every image is forced
   into the palette without art-directing each one.
2. **Distinctiveness.** Everything that makes generic model output recognisable —
   smooth photoreal gradients, arbitrary palettes, absence of visible process — is
   destroyed by the treatment. The result does not depend on anyone writing a
   sufficiently good prompt.

**Consequence.** A style with an empty treatment chain is refused at catalog
write. Prompt-only output is not a supported product.

---

## D-003 — Treatments belong to `image-tools`; scene generators belong here

**Date:** 2026-08-11 · **Status:** accepted

**Decision.** Deterministic raster treatments are added to `image-tools`. The
procedural scene generators and composition scaffolds stay in Backdrop Studio.

**Why.** The ownership rule: *a capability any scenario could want belongs in the
engine; a judgement only landing pages need belongs in the studio.* "Halftone at
60lpi, 15°" is a generic verb — deposited once, every scenario can screen an
image forever. "Draw a classical arcade" is content; no other scenario wants it.

**Consequence.** Backdrop Studio cannot ship until the treatment operations land
upstream. That is accepted as the critical path rather than worked around,
because implementing them locally would bury a general capability inside one
product and make it unavailable to everything else.

---

## D-004 — Procedural work releases without `asset-studio`

**Date:** 2026-08-11 · **Status:** accepted

**Decision.** Candidates produced without any model invocation release locally.

**Why.** A procedural render incurs no spend, carries no disclosure obligation,
and reproduces exactly from its seed. Routing it through a cost-and-disclosure
ledger would add a dependency that buys nothing — and would make the product
stop working whenever `asset-studio` is unavailable.

**Consequence, and the reason this is a requirement rather than an optimisation
(`REL-003`).** The procedural catalog must keep working with `asset-studio` down.
That is what makes the scenario deployable as a desktop product on an ordinary
machine, and it is the posture the monetisation story depends on.

---

## D-005 — Disclosure is derived from strategy and cannot be set

**Date:** 2026-08-11 · **Status:** accepted

**Decision.** The AI-generated flag is computed from the declared strategy.
Direct assignment is rejected (`REL-001`).

**Why.** Marking procedurally drawn output as AI-generated is a false claim.
Over-disclosure is an integrity failure in the same way under-disclosure is, and
a flag that any caller can set will eventually be set wrongly.

**Honest limitation, recorded so it is not later mistaken for a bug.** A
treatment pass that quantises to two inks will likely destroy an invisible pixel
watermark. The durable record is the signed manifest and `asset-studio`'s
provenance row, not something recoverable from the pixels. Any future claim about
watermark survivability must be measured, not assumed.

---

## D-006 — `ai-gateway` is reached transitively through `image-tools`

**Date:** 2026-08-11 · **Status:** accepted

**Decision.** No direct `ai-gateway` dependency. Styles name a role and a routing
profile; `image-tools` resolves the rest.

**Why.** `image-tools` already registers the gateway as a provider alongside its
local backends and selects between them from a probed host-capability inventory,
with an inspectable pre-execution resolution. Calling the gateway directly would
duplicate host probing and tier selection, and produce a second, divergent answer
to "what can this machine do right now".

**Consequence.** Backdrop Studio does not choose between local and routed
execution. It records which path ran (`RND-004`) so a degraded result is
attributable.

---

## D-007 — Contrast is measured at the worst pixel, never averaged

**Date:** 2026-08-11 · **Status:** accepted

**Decision.** The legibility gate reports the minimum contrast ratio across the
copy-safe region (`LEG-001`), computed from linearised sRGB luminance
(`LEG-002`), per placement crop (`LEG-004`).

**Why.** A single bright area behind one word makes a headline unreadable while
an average stays comfortable. Averages are the specific reason beautiful hero
imagery ships broken, so the gate has to measure the thing that actually fails.

**Consequence.** Measurement is O(pixels in region) per verdict rather than a
sampled approximation. This is affordable and is not a candidate for
optimisation-by-sampling; a sampled minimum is not a minimum.

---

## D-008 — Style is a superset of an `image-tools` Look, and compiles down to one

**Date:** 2026-08-11 · **Status:** accepted · **Watch item**

**Decision.** Keep Style as the outer record in Backdrop Studio and compile down
to a Look or step list when submitting to `image-tools`.

**Why.** A Look is a rendering recipe with no opinion about layout. A Style adds
classification, placement, copy-safe geometry, gates, and lineage — the layout
judgement that is this scenario's reason to exist. Collapsing them would push
landing-page concerns into a general-purpose image toolbox.

**Watch condition.** If a third consumer needs classified recipes, the
classification layer may deserve promotion out of Backdrop Studio. One consumer
does not justify a shared abstraction. Tracked in `PROBLEMS.md`.
