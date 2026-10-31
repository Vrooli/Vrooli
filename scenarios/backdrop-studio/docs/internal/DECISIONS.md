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

## D-015 — A quality tier is a floor on capability and a ceiling on spend

**Decided 2026-08-13.** Which lane draws a picture used to be two constants in
`internal/render/render.go`: every model-backed style got the same policy pair,
so a style that needed more than the installed local model set simply could not
be made, and a style that needed nothing could not say so. The choice is now a
declared `quality_tier` on the style, and `internal/render/routing.go` is the one
place that turns a tier into a lane.

**The tier binds in both directions, and both halves are load-bearing.**

As a *floor on capability*: a procedural generator does not draw a photographic
interior. Serving a `frontier_model` style from a local checkpoint would ship a
picture wearing a label it did not earn, and every consumer downstream — the
release path, the disclosure record, an operator reading the verdict ledger —
would believe it.

As a *ceiling on spend*: a style authorises the money its tier costs and no more.
This is the direction with a recorded incident behind it. The predecessor of the
current constants hardcoded `("quality", "any", BYOK on)` and billed every
model-backed render to a cloud provider while an installed local GPU served the
same request in about fifteen seconds — billable, slower, and silent. So a lane
above the declared tier is never tried, not even when the declared lane fails.

Together these mean exactly one lane serves a tier. The ladder is still walked
cheapest-first and every lane it passes is recorded on the job with the reason it
did not serve, so an operator reads the escalation rather than only its result. A
tier no lane can meet is a typed `LaneRefusedError` naming the tier and every
lane tried, mapped to `FailedPrecondition` — because the fix is an install, a
credential, or a catalog edit, and none of those is a retry.

**The catalog cannot express the disagreement.** `ValidateTierCoherence` refuses
a model-backed strategy at the procedural tier and a generator strategy at a
model tier. That is what makes "a procedural style never reaches a model" a shape
rather than a behaviour a later edit could regress — the render-path test asserts
the rule, and the store makes the violating style unwritable.

**Judged over the settled catalog, not per seed version**, exactly like
`ValidateSubjectCoherence`. The field postdates most shipped seed versions and a
shipped seed value is never edited in place, so v1's `guided-botanical` is
allowed to predate the tier and v9 is where it acquires one.

**Revisit when** a tier gains a lane that is genuinely interchangeable with
another at the same rank — two frontier providers, say. The walk already supports
it; the one-lane-per-tier outcome is a consequence of the current ladder, not an
assumption baked into its shape.

## D-016 — Generation geometry is asked for, never assumed

**Decided 2026-08-13.** `sdNativeEdge`, `sdSizeQuantum` and `sdMaxEdge` are
deleted. They were named for Stable Diffusion 1.5 and applied to whichever model
`image-tools` selected, which on an SDXL or FLUX host throws away half the
trained resolution and quantises to a stride the model does not use.

The deeper problem was ownership: `OT-P0-004` gives `image-tools` sole ownership
of model configuration, so three numbers describing a model family lived in the
scenario least able to notice when they stopped being true. They are now facts on
`image-tools`' registry, served on `Model.geometry`, and read through
`ModelsService.SelectModel` — a preview that runs the same selection a submit
would and executes nothing, which is precisely the question "how big should I ask
for?" needs answered before spending a generation.

`imageengine.GeometryProbe` is a separate interface from `Generator` so a caller
can size a request without being able to spend a model call. An engine that
cannot answer is an error, not a fallback constant: falling back is how three
SD-1.5 numbers came to size generations on hosts running other architectures.

## D-017 — Text generation reaches `ai-gateway` directly; D-006 governs pixels

**Decided 2026-08-13.** Authoring a vector generator asks a model to write
source code. That request goes straight to `ai-gateway`'s `ExecuteRoute` with a
role, and does not pass through `image-tools`.

**This does not weaken D-006.** That decision routes *image* inference through
`image-tools`, and its reason is specific: image generation has a local lane, so
calling the gateway directly would duplicate `image-tools`' host probing and
tier selection and produce a second, divergent answer to "what can this machine
do right now".

None of that applies to text. `image-tools`' AI catalog is generation and
enhancement of pixels; it serves no text at all. There is no local lane to
duplicate and no host-capability question to answer twice. Routing a text
request through an image scenario to reach a text model would be indirection
with nothing behind it.

**What this still owes D-006 is its consequence.** The model that answers is
recorded on the generator, so an asset drawn by an authored generator can name
both the model that wrote the generator and the model — if any — that drew the
pixels.

**A role, never a model slug.** `ai-gateway`'s conformance policy refuses a
concrete model name in a runtime surface, and the reason is not bureaucratic: a
slug here would have to be edited whenever the provider catalog moved, by
someone who does not know what this scenario needs. `author.generator` was added
to `ai-gateway`'s inference-role catalog and says what is needed — long-form
code authoring, hosted-first because the local candidates in that catalog are
sized for short schema-constrained values.

## D-018 — An authored generator is a template, and the envelope is not the author's to write

**Decided 2026-08-13.** A model cannot add a Go function to a running binary, so
an authored generator is a `text/template` producing SVG *marks*. The document
element, the depth plane and the ink substitution are added around its output by
`internal/vector`.

**The expressive difference is real and is stated rather than papered over.** A
template cannot trace marching-squares iso-lines; the four hand-written
generators remain the shipping lane. What a template can do is compose — arrange,
repeat, scale and vary marks across a frame — which is what most art directions
actually are.

**Keeping the envelope means an authored generator cannot opt out of the
guarantees a hand-written one has.** A generator that emitted its own `<svg>`
could set a viewBox disagreeing with the requested size, skip the depth planes
the plate model reads, or place an `<image>` where validation did not look.

**Validation is a precondition of storage, not of rendering.** A stored
generator is one a later code path may reasonably assume was checked, so the
checks run before the row exists: the template parses, every parameter it reads
is declared with a range that contains its default, two renders of one seed are
byte-identical, the composition survives a four-fold size reduction, an unbound
palette is refused rather than written into the document, the id does not shadow
a built-in preset, and the marks contain no script, external reference, remote
URL, data URI, event handler or entity declaration.

**The active-content check runs on the author's marks, not the wrapped
document** — the envelope legitimately carries `xmlns="http://www.w3.org/2000/svg"`,
and a remote-URL check over the whole document refuses every generator ever
written, including a correct one. It is a deny list, which is normally the weaker
choice; the alternative is a second SVG implementation in a scenario that owns no
rasterizer. It is paired with two structural controls that do not depend on it:
the author never writes the document element, and the template function surface
is closed and arithmetic-only.

**Arithmetic helpers coerce int and float.** `range $i := seq 12` yields an int,
so `mul .W $i` — the obvious thing to write — would otherwise fail with a type
error. The plan's stated risk for this work is that authored generators fail
validation often enough to be useless, and a paid round trip spent on a type
error rather than on the art direction is the largest avoidable part of that.

## D-019 — A candidate is a plate stack whose flat composite is never optional

**Decided 2026-08-13.** A backdrop is not flat. A colonnade sits in front of a
sea which sits in front of a sky, and the generators already know that — the
vector family draws each as its own SVG group. Flattening at the moment of
render threw the separation away, so a consumer wanting parallax had to infer
depth from a picture that no longer contained it.

A candidate now carries an ordered `Plate` list. **`image_png` stays the
deliverable and is always present**, and that is the load-bearing half of this
decision: every consumer in this system decodes one image, and a stack that
could not be flattened would be a contract change rather than a capability.

**One assembly path, not two.** A style declaring no plate spec renders as a
stack of length one carrying the whole picture, materialised by
`EffectivePlateSpec`. There is no flat branch beside a stacked branch, because
two paths is how the flat one comes to differ from the stacked one in a way
nobody notices.

**The single-plate path does not call the compositor.** It applies the chain and
returns those exact bytes as both the composite and the plate. That is what
makes "an existing style is byte-identical to its pre-plate output" a property
rather than a hope: a PNG that round-trips through a second encoder is not
guaranteed to be the same bytes even when it is the same picture, and a golden
would catch that drift a release too late.

**Three blend modes, and the set is not arbitrary.** Normal is placement.
Multiply is how ink sits on paper, which is what every screen and duotone in
this system models. Screen is how light adds, which is what a glow or a star
field needs. A mode outside the set is refused by name at both ends — the
catalog will not store it and the compositor will not run it — rather than
approximated with the nearest one, because substituting a blend silently changes
what a picture depicts.

**Depth is explicit, not list position.** A caller reordering a stack should not
have to rebuild the list, and two plates silently claiming one layer is a
mistake worth keeping visible: the catalog refuses it and the compositor sorts
stably so the picture never depends on map or slice order.

**Plate pixels do not travel on the job record.** A three-plate candidate at
store geometry is tens of megabytes, and inlining them would make every list
call expensive for a field most callers ignore.

**A style declaring a stack nothing can fill is refused by name.** Until a
generator emits its layers, a multi-plate style is an error rather than a
silently flattened picture — a style that says it draws a sky behind a colonnade
and delivers one flat picture is the substitution this whole plan exists to
remove.

**Three plates is a plan boundary, not a technical limit.** The field is typed
as a list so raising it needs no migration; the cap exists so the first plate
work cannot quietly become an unbounded layer editor.

## D-020 — Reserved space is a knockout the whole chain honours, applied before each operation

A style declares where its copy sits. Every colour treatment in its chain
reserves that area, and reserves it by lifting the area to the top of the tonal
ramp **before** the operation runs.

**Why a knockout and not a scrim.** A scrim shades the picture where the copy
sits, so it fights the picture: it dims something chosen for its beauty in order
to rescue one corner, and on a screened backdrop it barely works anyway. A
knockout is what a printer actually does — a hole left in an ink layer so
something else can occupy the space — and it is what the reference art does. In
the halftone-flower and arcade posters this catalog is measured against, the
screen simply stops where the type goes.

**Why before and not after.** A repair applied afterwards has to invent a colour
to paint the area, and the only honest candidate is the untreated source: raw
grey in the middle of a duotone, which reads as a fault rather than as paper.
Fed white beforehand, a tone-driven operation lays no ink AND returns its own
paper — the same cream or blue the rest of the picture is printed on. The
mechanism produces the right colour instead of guessing it.

**Why the whole ramp and not nearly.** The first attempt lifted to 0.94 and
failed on every screened style. A screen fed 0.94 still lays a sparse pattern of
FULL-strength dots, and one dot decides worst-pixel contrast for the entire
area. Measured against a 0.941 patch: ordered dither 0.000, stipple 0.099, ASCII
mosaic 0.102, line screen 0.137. A knockout differs from a light patch in kind,
not in degree.

**Why it is a sibling of the operation and not a field on each.** `OpParams`
carries `knockout` next to the operation oneof, and `ops.treatment`/`ops.tier2`
apply it at one seam. Reserving space is a property of the picture being made,
not of the treatment making it: the author states the rectangle once and every
operation in the chain honours the same one. Five fields repeated across
seventeen parameter messages guarantees that some of them eventually differ.

**Geometry operations are excluded, deliberately.** A resize or a crop moves the
frame out from under the rectangle, so a reserve declared against the old frame
would name the wrong part of the new one.

**What it does not cover, and cannot.** A style with an empty treatment chain
never reaches the knockout, because there is no operation to reserve anything
in. Those styles have to compose around their copy in the generator, which is
what `survey-relief` does and why it is the only untreated style that keeps its
copy legible. Post-processing loses to composition; where there is no
post-processing at all, composition is the only move left.

**Consequences accepted.** Three treatments had to change to make the reserve
mean anything, and two of the three were carrying defects of their own:

- `stipple` deposited one full-strength pixel per cell however light the tone
  above it, because its dot bounds always covered their own centre. Every
  stippled highlight in the catalog was carrying about fifteen percent more ink
  than the tone called for. It now applies the rule `engraving` already applied:
  a mark too small to draw is not drawn.
- `ascii_mosaic` is block-quantised, so a cell straddling the edge read as dark
  and stamped a dense glyph whose ink landed inside. A glyph prints whole or not
  at all, so the reserve rounds outward to whole cells there.
- `displacement` fetches a pixel from elsewhere rather than deciding it from the
  tone in place, so lifting the area cannot hold it and the feather cannot
  either — the warp steps over the margin in one hop. The reserve damps the warp
  itself, easing back to full amplitude outside, because a wave that stops dead
  reads as a tear.

The tone mapper also excludes reserved space from its auto-level histogram. The
lift is white this package added, not a highlight the picture has; counted, it
becomes the new p99 and every real tone compresses downward to make room for it.
The distortion would scale with the area reserved, so it would be worst on
exactly the layouts that reserve most for their copy.

## D-021 — The vector lane reserves space itself, in both polarities, as a volume on every plane

A vector style declares no treatments, so the knockout of D-020 has nothing to
attach to: there is no operation in which to reserve anything. The generator
reserves the space itself, in its own document.

**Both polarities, because a generator owns its ink as well as its paper.** Dark
copy gets a knockout — the plate carries no ink and the copy sits on paper.
Light copy gets a solid — the plate carries full ink and the copy sits on the
ink. D-020's treatment reserve can only lift toward paper, so it serves one of
the two and correctly refuses the other; here both are available and both are
used. `engraved-colonnade-vector` is served by a knockout, `pale-moon` and
`tidal-halftone` by solids.

**Emitted as SVG, not composed mark by mark.** A reserve is a property of the
plate — an area carrying no ink, or full ink — not a decision each mark makes
for itself. It also has to work for a generator whose subject is a single shape
the size of the picture, where "do not draw here" means nothing. This is not the
scrim that was tried and abandoned: a scrim is translucent, the picture shows
through, and against a catalog sitting near 1.0 everywhere it repaired nothing.
A reserve is opaque, so what the copy sits on is decided entirely by the reserve.

**A volume, not a rectangle.** The copy is laid out in the page and does not
move; the plate the reserve is cut into does. Cut to the copy's own rectangle,
the reserve slides out from under the copy on the first scroll. That is measured
rather than supposed: `tidal-halftone` scored **8.15 at rest and 1.00 half a
screen later**, and the Phase 10 parallax gate refused it. The reserve therefore
extends downward by the frontmost plate's whole travel — downward only, because
plates translate upward as the page scrolls, so the material that would arrive
over the copy is the material below it.

**On every plane, not only the frontmost.** Reserving the front plate alone buys
exactly one frame of legibility, the one nobody scrolls: wherever the front
plate's reserve has slid off the copy, the plate behind is showing, and the plate
behind is usually the ground — a full sheet of opaque paper that does not move at
all. Repeating the reserve costs nothing in the flat picture, since the reserves
are identical and opaque and only the frontmost is seen, and it preserves
flatten-equivalence: source-over is associative, so N stacked reserves in one
document and N plane rasters each carrying one composite to the same pixels.

**The rectangle is derived, never declared.** `vectorQuietZone` reads the style's
`regions`, the same source `quietZone` and `reservedSpace` read. Three lanes, one
statement of where the copy goes. An author maintaining it three times would
change one of them, and a reserve cut where the copy is not is invisible to every
test that does not measure contrast where the copy actually sits.

## D-022 — A treatment reserve serves dark copy only; light copy is served by composition

image-tools can cut a reserve either way. `Knockout.solid` drives the area to the
BOTTOM of the tonal ramp before the operation runs, mirroring the knockout that
drives it to the top, and it is built, tested and reachable by any caller.
Backdrop Studio uses only the knockout.

**Because the solid costs more picture than the quality bar allows.** It was
wired for light copy and withdrawn on measurement: `synth-celestial` fell to
0.707 subject survival against its own declared 0.800 floor, and three styles
that had been rendering correctly were refused outright. Reserving a third of a
frame as flat ink is a large claim on a picture. A knockout gets away with the
same claim only because paper is where a screen was already leaving gaps —
removing ink is a smaller change to a picture than adding it.

**The gate is right and the reserve is what gives way.** Loosening a quality
floor so a legibility number passes trades a measured defect for an unmeasured
one. This work has already made that mistake once, tuning a pen weight until a
perceptual score moved, and had to revert it when the failures turned out to
have moved rather than reduced.

**A second limit, measured, that bounds any future attempt.** A discrete screen
cannot lay a solid at all. Its mark is sub-cell, so coverage is capped below one
however hard it is driven: at full tone, halftone, line screen, stipple,
engraving and ASCII mosaic all leave between 0.93 and 1.00 paper showing.
Ordered dither is NOT in that set, and the difference is the whole rule — a
dither's mark is the pixel, so at full tone every pixel turns and the area goes
genuinely solid. Sub-cell marks cap; per-pixel marks do not.

**So light copy is served by composition.** The vector lane cuts its own solids
(D-021) and pays nothing for them, because there the generator owns the ink and
makes the ground dark as part of the drawing rather than as a claim laid over
it. For a model-backed style the same reasoning puts the reserve in the
conditioning or in a derived plate rather than in a post-hoc treatment: a
picture whose own darkness falls where the copy goes costs no subject survival
at all, because it is the subject.

This is the same finding this work keeps arriving at from different directions.
Post-processing loses to composition; where a picture must be changed rather
than merely masked, the change belongs in whatever drew it.
