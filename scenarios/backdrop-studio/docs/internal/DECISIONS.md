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

**Decision.** The legibility gate reports the minimum contrast ratio across each
overlay region (`LEG-001`), computed from linearised sRGB luminance
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
classification, placement, reserved-region geometry, gates, and lineage — the layout
judgement that is this scenario's reason to exist. Collapsing them would push
landing-page concerns into a general-purpose image toolbox.

**Watch condition.** If a third consumer needs classified recipes, the
classification layer may deserve promotion out of Backdrop Studio. One consumer
does not justify a shared abstraction. Tracked in `PROBLEMS.md`.

---

## D-009 — `copy_safe` generalises to reserved regions with a declared kind

**Date:** 2026-08-11 · **Status:** accepted · **Supersedes** the original `copy_safe` rectangle

**Decision.** A style carries `reserved_regions[]` rather than a single
`copy_safe` rectangle. Each region declares whether foreground content
**overlays** it (text, gated on contrast) or **occludes** it (a device frame or
card, behind which contrast is meaningless).

**Why.** Extending beyond page surfaces to store listings introduced a foreground
element that is not text. A device frame does not need contrast beneath it — it
needs the focal point *not* to be under it. Those are different gates, and a
single untyped rectangle cannot express which one applies. Without the kind, the
system would either measure contrast under an opaque frame (a meaningless number
that blocks release) or skip measurement on a caption band (a real failure that
ships).

**Cost of not doing it now.** Regions are written into every style record and
every released backdrop's reference payload. Adding a discriminator later means
migrating rows whose original intent is no longer recoverable — the same
reasoning `asset-studio` applies to its own P0 schema-shape targets.

**Consequence.** `OT-P0-010` is stated in terms of reserved regions, `OT-P0-011`
measures per overlay region, and the scaffold draws every reserved region as a
flat void rather than just the copy area.

---

## D-010 — Mockup chrome fidelity is chosen by surface kind, not by preference

**Date:** 2026-08-11 · **Status:** accepted

**Decision.** A `product` surface previews inside chrome derived from the target
scenario's design tokens. A `store` surface previews inside a facsimile of the
destination store listing. The `kind` field on the surface record decides;
nothing else may.

**Why.** The two previews answer different questions. For a landing page hero the
question is *does this belong to our product*, which only brand-derived furniture
can answer. For a store screenshot the question is *does this hold up against
that store's furniture and the listings beside it*, which brand chrome actively
obscures — an App Store screenshot previewed in our own design language looks
right and then loses to its neighbours.

**Boundary.** A product mockup draws an impression, never real components.
Importing the target's actual components would make this scenario depend on every
scenario it can preview, which inverts the dependency direction the whole design
rests on. A store facsimile is recognisable as a mockup and reproduces no more
trade dress than the layout judgement needs.

---

## D-011 — Backdrop Studio composes store assets but never captures or submits them

**Date:** 2026-08-11 · **Status:** accepted

**Decision.** For store listing assets, this scenario supplies the backdrop, the
placement composition, and the surface geometry. The application screenshot is an
**input it receives**. Submission to a store is not its concern.

**Why.** `scenario-to-android` and `scenario-to-ios` already declare screenshot
capture from journey evidence (`OT-P1-007` in each) and already own the store
relationship. Capturing screenshots here would re-implement a capability that
exists, against a device matrix this scenario has no business knowing about — the
same rule that keeps raster operations in `image-tools`.

**Consequence.** The store lane is a genuine composition: they produce
screenshots, we produce backdrops and geometry, and the composed asset is handed
back. It also means their `OT-P1-007` targets acquire a producer, which is the
main reason the store surfaces are worth building at all.

---

## D-012 — Surface records may be seeded freely; surface *targets* are admitted on demand

**Date:** 2026-08-11 · **Status:** accepted

**Decision.** A **PRD target or requirement** committing to a surface is added
only when a consumer asks for it and the admission test in
`../reference/surfaces.md` passes. **Seeding a surface record is a separate act
and is not governed by this decision** — records are data, cost nothing, and the
seed catalogue deliberately covers more ground than any consumer has requested,
so that an implementing agent can see the intended shape of the space.

The line: *a seeded record says "this would work here." A target says "we will
build this."* Only the second needs a consumer.

**Why.** A survey of every scenario PRD found exactly two declared, unmet imagery
needs — `scenario-to-android` and `scenario-to-ios` `OT-P1-007` — and both are
now claimed. Everything else that *would* work here (extension store tiles,
desktop splash, email headers, deck backgrounds, repository social previews,
in-product empty states) is real but unrequested.

Enumerating them as targets would trade a genuine property for a false one. The
genuine property is that **adding a surface is a data row**, because the
`surfaces` domain was built to generalise exactly this. A speculative target list
would not make any of them arrive sooner; it would inflate a P1 set that is
already twelve items deep and turn "unbuilt" into "committed but unbuilt".

**What the seed catalogue is for.** It shows the intended shape of the space without committing to it, alongside an admission test, so a future agent adds a surface
in one step without reopening the scope question, and a table of worked examples
recording which candidates were considered and why each was accepted, deferred,
or refused.

**Two boundaries recorded because they are non-obvious:**

- **Print collateral is the best conceptual fit and the least free.** Halftone
  and risograph *are* print processes, so the treatment layer looks native to
  paper. But this pipeline is sRGB and the legibility gate is WCAG, which is a
  screen standard. CMYK separation, bleed, DPI, and a contrast measure that means
  something under ink are all real work. Nobody should assume it follows.
- **In-product imagery is the largest latent use and the easiest to get wrong.**
  Every scenario has empty, error and onboarding states. But an empty-state
  illustration is usually *focal* — a drawing that explains something — and only
  the ambient wash behind one is in scope. That line needs drawing before the
  work starts.

## D-014 — Evidence stays in git, at a stated resolution, with a stated ceiling

**Decided 2026-08-12.** `docs/evidence/` reached 34 MB across roughly seventy
PNGs once the catalog sheets and the generator sheet landed. `PROBLEMS.md` had
flagged the question as an owner decision nobody had made; this is the decision.

**It stays in git.** Three reasons, in order of weight.

The artifacts are *reviewed by reading a diff*. A contact sheet's job is to make
a catalog regression visible when someone changes a generator, and a blob seam
turns that into a fetch step that the reviewer skips. Evidence nobody looks at
is the same failure as evidence nobody can reproduce, arriving by a different
road.

They are also *small in the way that matters*. Thirty-four megabytes is a large
directory and a trivial repository — this repo carries far more in
dependencies — and the growth is bounded by the catalog, not by time: adding a
style adds one cell to one sheet, and the sheets are regenerated in place rather
than accumulated per run.

And a blob seam is a *dependency the evidence rule cannot afford*. The rule is
that every artifact is reproducible by a stated command; a seam adds a service
that must be running for the evidence to be readable at all, which is exactly
the coupling that makes a reader trust the summary instead of the picture.

**Two constraints keep it honest.**

*Resolution is deliberate, not incidental.* Every artifact renders at its
delivery geometry or at a sheet cell of 640×320, because the failures this
scenario has actually shipped — sub-pixel engraving lines, screen moire,
one-pixel filaments read as speckle — are all invisible below roughly a third of
delivery size. Thumbnails would make the directory small and the evidence
worthless.

*The ceiling is 50 MB.* Past that, the next artifact set does not get committed
until something is retired. `docs/evidence/placements/` and
`docs/evidence/phase-14/` were retired the same day for a stronger reason —
neither had a producing command — and that is the first thing to check when the
ceiling is reached: an artifact with no command is not costing space, it is
costing trust.

**Revisit when** a consumer outside this repository needs the artifacts, or when
per-seed evidence (rather than per-style) becomes worth keeping. Both would
change the growth from bounded to unbounded, which is the property this decision
rests on.
