# Problems — Backdrop Studio

## Remaining after the treatment-layer plan

- `REL-006` remains unbuilt: sized-variant derivation with reserved-region
  preservation is a separate geometry capability and is not silently implied by
  the store/device-frame implementation. **Narrowed 2026-08-12:** delivery
  geometry now comes from the surface record and every style renders correctly
  at every surface it declares, so the *sizing* half is done. What remains is
  deriving several sizes from one rendered master and rejecting a crop that
  pushes the focal mass into the reserved region.
- ~~Asset Studio exposes no external byte-ingress RPC.~~ **Closed 2026-08-12.**
  `StudioService.IngestExternalAsset` admits bytes with their producing-scenario
  provenance, requires no identity record, and refuses a model-backed request
  that names no model or prompt. Backdrop Studio's publisher seam now calls it
  instead of refusing. The refusal itself survives and is tested: with
  asset-studio absent, a model-backed release fails naming the missing
  capability and produces no fallback asset — and a candidate whose provenance
  this process did not record is refused too, because its model and prompt exist
  nowhere else. Filed as `knw-1786507241786326657`; resolved.
- Tier-3 image treatments remain unbuilt by explicit plan decision: `glitch`,
  `kaleidoscope`, `slit_scan`, `fluted_glass`, `photomosaic`, and `resample`.

## Open after the 2026-08-12 output-quality repair

- ~~`Normalize` is not on the wire.~~ **Closed 2026-08-12.** `normalize` now
  exists on every tone-mapping proto message, along with `dark`/`light` on the
  Tier-2 ink-on-paper screens, `distance` on aberration and `spacing` on
  displacement. See the correction below — this was not merely a missing
  convenience, it was breaking renders.
- ~~Every scenario's `api/go.mod` is stale.~~ **Closed 2026-08-12.** Root cause:
  `api-core` and `packages/proto` had moved to `golang.org/x/sys v0.44.0` while
  every consumer still pinned `v0.42.0`, so MVS demanded a version the recorded
  requirement was below. Repaired through the governed gateway
  (`scenario-dependency-analyzer deps install`), never `go mod tidy`: 61 blocked
  modules down to 6. `go test` now runs normally here — the
  backup/`GOFLAGS=-mod=mod`/restore dance is no longer needed and should not be
  reintroduced. The 6 remaining are tool gaps and one governance decision, filed
  as `knw-1786539478410570891`; none of them are in backdrop-studio or
  image-tools.
- ~~`docs/evidence/` is 8.8 MB and nobody has decided where it lives.~~
  **Decided 2026-08-12, D-014.** It stays in git, at delivery resolution, under
  a 50 MB ceiling — the artifacts are reviewed by reading a diff, growth is
  bounded by the catalog rather than by time, and a blob seam would add a
  running service the evidence rule cannot depend on. It is 34 MB today. Two
  artifact sets were retired the same day for a stronger reason than size:
  neither had a producing command.
- **`ascii_mosaic` is the only treatment whose cell size is coupled to a
  font.** It blits a 7×13 bitmap face, so `block_size` values far from 7 resample
  the glyph. Legible, but not crisp at extremes.
- ~~The studio UX barely exists.~~ **Closed 2026-08-12.** Eleven routes resolved
  to one 181-line `WorkbenchPage` holding four hardcoded style rows and
  CSS-gradient "specimens", so the one question the studio exists to answer —
  what does this style actually look like — could not be asked of it. There are
  now ten pages, each its own component, each reading the real catalog:
  a faceted catalog browser, style detail with the chain and its resolved
  parameters and the perceptual margins, a seed-range variation grid, a remix
  flow that records lineage, per-placement previews, a compose plan, the surface
  registry with its citations, candidates, and an honest empty state for
  released backdrops. `routes.test.tsx` asserts no two routes share a component.

  Three fields had to reach the wire first: `Style.treatment_params`,
  `Style.inks` and `Style.parent_id` were all held by the store, declared by
  neither the proto nor the handler mapping, and therefore invisible to every
  consumer. The studio could show a treatment chain with no parameters under it
  and offer a fork it could not record.
- **No surface exists for Apple's current primary iPhone class.** Verified
  2026-08-12: the 6.9-inch class at 1320×2868 is now primary and the seeded
  6.7-inch record at 1290×2796 is a fallback Apple still accepts. Assets for the
  primary class therefore have no surface to render into. The fix is one seed
  row, not code — the device classes are data by design — but it needs the
  geometry checked against App Store Connect Help at the time it is added,
  because that is the number a submission is rejected on.
- **Six placements have no mockup and silently preview as `full_bleed`.**
  `render.composePlacement` has cases for `split_panel`, `framed_inset` and
  `corner_bleed`, and a `default` that everything else falls into — so
  `device_center`, `caption_above_device`, `caption_below_device`,
  `caption_only`, `feature_graphic` and `type_mask` all preview as a full-bleed
  landing page with a headline over it. That is the wrong picture for all six:
  a store screenshot puts a device in the middle of the frame and a feature
  graphic has no headline at all.

  The catalog work of 2026-08-12 seeded five styles that declare these
  placements, and they render correctly *at their surface geometry* — the
  delivery path takes its size from the surface record and knows nothing about
  mockups. It is only the preview that is wrong, and it is wrong silently, which
  is the same class of defect as the subject substitution the procedural lane
  used to do: the system answers a request it cannot honour with a nearby
  answer and says nothing.

  This is the baseline for the studio-UX phase, which owns the store-listing
  mockup. Until then, treat a store or syndication placement preview as
  decorative rather than as evidence.

## Open after the 2026-08-12 catalog seeding

### Closed 2026-08-12 by the wire-truth plan

- ~~Ten of sixteen styles fail with `422 invalid color "$brand.primary"`.~~
  `mergedParams` now fails closed with a typed `UnresolvedSlotError`, every
  style declares its own ink defaults, and the render path resolves an effective
  palette. All 16 render with and without a brand bound.
- ~~The catalog can never upgrade.~~ Seed content is versioned data under
  `api/internal/catalog/seed/`, applied by version with `origin` protection for
  operator rows. Proven against the real pre-plan database in
  `docs/evidence/baseline/catalog-upgrade-proof.md`.
- ~~Stale binaries are indistinguishable from missing features.~~ `/api/v1/build`
  reports a content fingerprint over the API sources, `/health` carries it as
  semver build metadata, and the integration lane refuses to render on a
  mismatch.
- ~~One delivery geometry for every surface.~~ `deliveryWidth`/`deliveryHeight`
  are gone; geometry comes from the seeded surface record and the job echoes
  which surface it resolved.
- ~~Generated candidates report a false size.~~ Every candidate's recorded
  geometry is measured from the bytes it carries, on every strategy branch.
- ~~Downscaling uses nearest-neighbour.~~ Replaced with a windowed filter
  (Catmull-Rom down, triangle up). `internal/render/resample_test.go` proves the
  old implementation fails the same golden.
- ~~The placement mockup draws black bars where copy goes.~~ It renders real
  headline, subhead and call-to-action type at the style's declared text colour.

### Found and fixed while making the lanes coherent (2026-08-12)

- **Two test fakes were lying, and both mattered.** The render suite's fake
  executor inverted the blue channel — a transform no treatment performs, and on
  a blue-dominant source it turns dark water bright and compresses the luminance
  range, so the perceptual gate refused styles that render correctly through the
  real wire. It now maps lightness onto a two-ink ramp with the same p1-p99
  stretch every seeded style requests. A fake that is not a plausible stand-in
  for the thing it stands in for makes every test above it a test of fiction.
- **The perceptual gate's chain rule was wrong.** It took the strictest family
  bar on every metric, so `posterize` + `halftone` was held to the tonal
  family's 0.80 survival floor because a posterize appeared earlier in the
  chain — and refused `ukiyo-tide` at 0.772 for an image that reads correctly.
  Structural licence now takes the *loosest* bar in a chain (a chain is entitled
  to the licence of its most destructive member); usability metrics still take
  the strictest.
- **Three new generators shipped broken and the existing scene tests caught all
  three.** Reaction-diffusion rendered blank because the Laplacian stencil was
  six times the canonical weights and diffusion swamped the reaction; caustics
  rendered as speckle because the ray displacement scaled by frame size instead
  of by wavelength; the flow field failed both coherence and
  resolution-independence because a one-pixel filament is a different fraction
  of the frame at every size. Each was a real defect, not a threshold quibble.
- **A structural rule contradicted the render path.** The catalog forbade a
  procedural style from carrying any scaffold block, while `render.go` had
  always read `scaffold.params_json` out of one. The rule now forbids only what
  is genuinely meaningless without a model — a ControlNet `conditioner` — and
  `scaffold.preset` is how a procedural style selects its generator.
- **Seed-count assertions were hardcoded.** Three tests asserted "16 styles" and
  had to be edited every time the catalog grew, which turns an assertion into a
  number someone bumps until it passes. They now derive the count from the
  embedded seed versions.

### Found and fixed while building the perceptual gate (2026-08-12)

- **`engraved-colonnade`'s moire was never a composition failure.** The obvious
  reading — "the treatment destroyed the subject" — is wrong, and it was ruled
  out with measurement rather than argument. The broken image scores 0.973 on
  subject survival, essentially the same as the styles that render correctly,
  and the arcade is plainly visible when the image is reduced to the
  low-frequency field the metric reads
  (`docs/evidence/perceptual/engraving-repair/lowfrequency-field-comparison.png`).
  Correlation was checked at five grid scales on both the tonal and the gradient
  field; none separated broken from working.
- **The real defect was a sub-pixel line.** `image-tools`' engraving computed
  `width := math.Max(0.6, (1-l)*spacing*0.55)`. A 0.6px line cannot be drawn; it
  rasterises to a dotted trail of aliased fragments, and because 0.6 was a floor
  rather than a cutoff, those fragments covered the highlights too. 31.8% of the
  result's ink runs were one or two pixels wide, against 1.5% for the same scene
  under a line screen. Now: below one whole pixel, no mark — which is what an
  engraver does with a pale tone.
- **Screens were being handed a tonal range they could not express.** The
  procedural scenes deliver a compressed range (the arcade's wall sits at L\*
  0.85 and its sea at 0.45), so a screen mapped it into a narrow band of mark
  widths and read as flat texture. Seed v3 turns on `normalize` for every
  screening treatment. `frequency_modulation` roughly doubled across the
  affected styles, and `engraved-colonnade` now renders a legible colonnade.

- **`pixel_sort` is a no-op on a horizontally uniform source.** It reorders
  bright runs *along a row*, so a source built from horizontal bands — which
  every `horizon` scaffold and scene is — has nothing to reorder, and the
  operation returns its input byte-for-byte. Not a defect in the treatment, but
  it means a style pairing `pixel_sort` with a horizon subject renders as if the
  treatment were absent, silently. Found by the treatment gallery's
  "a treatment must change its input" assertion. No style currently pairs them;
  a catalog-write-time check would be the durable fix.

**Measured but deliberately not shipped as a check.** The mark-run statistic
(fraction of ink runs ≤2px) separates the three arcade cases cleanly — 0.318
broken, 0.060 and 0.015 working. It is not a gate, because `stipple-massif`
renders correctly and scores 0.232 on it: a stipple's marks are legitimately
1–2px dots. Any threshold that fails the broken engraving also fails a working
stipple. It is recorded as a diagnostic in
`docs/evidence/perceptual/engraving-repair/README.md`. A future version could
make it work by measuring ink and paper runs separately, or by measuring mark
*area* rather than row-run length; neither was in this phase's scope.

**Known limit of the shipped gate.** It scores a candidate against its own
source, so it cannot see that the source itself is weak. `engraved-colonnade`
passes today over a flat vector `arcade` scene; the same style over a
photographic source would be better art and score about the same. Source quality
is Phase 7's subject, and no metric here substitutes for it.

### Found and fixed while executing the resolution-relative phase (2026-08-12)

- **`displacement` declared a `spacing` parameter and threw it away.** It was on
  `ops.proto`, mapped through `handlers/ops/params.go`, and carried into
  `treatments.Params` — and the implementation then used two hardcoded
  wavelengths (`.12` and `.09`) and never read it. Every caller that set spacing
  got the same picture. The wavelength is now honoured, with defaults derived
  from those exact constants so an unparameterised call is byte-identical to
  what it always produced. This is the same defect class the plan's own rule
  names: *a parameter is not shipped until it round-trips*, and a parameter that
  round-trips but is ignored at the far end is worse, because the wire proves
  nothing.
- **Halftone rulings have a pixel floor nobody had measured.** `lpi` is a count
  of lines across the image width, so it is resolution-independent by
  construction — confirmed to within 0.5% at every ruling from 16 to 96. But a
  screen cell drawn from fewer than 8 pixels cannot carry tone: at `lpi=130` on
  a 768px frame (5.9px cell) the rendered density falls 29.6% short of the same
  ruling on a 2304px frame. The scenario's own default of `lpi: 120` would have
  resolved to a 3.25px cell on `web.hero-mobile` (390px wide). image-tools now
  clamps a ruling finer than the grid supports and reports the clamp in
  `OpResult.resolved_params`. `MinHalftoneCellPx` and the two tests around it
  are the durable record.
- **`image-tools ops halftone --help` documented `--lpi` as "Screen cell size in
  pixels".** Wrong unit and inverted direction — raising it makes the cell
  smaller, and it is not a pixel measurement at all. A caller reading only the
  help text would tune it backwards.
- **Rounding an ASCII cell down to the glyph advance was a 48% error.** The 7px
  bitmap advance is a coarse quantum: a 13.4px request floored to 7px, and the
  same relative value collapsed to the floor on a small frame while resolving
  proportionally on a large one — defeating the point of a relative unit.
  Snapping to *nearest* halves the worst-case error and keeps it symmetric.

**Still open upstream.** Two spatial parameters keep a pixel-only form because
nothing seeded uses them and no relative twin was added: `overlay`'s `font_size`
and `canny`'s hysteresis thresholds (the latter is a gradient magnitude, not a
distance, so it is not resolution-dependent at all). Neither is reachable from a
seeded style; `TestSeededStylesSendNoAbsoluteSpatialParameter` would catch it if
one became so.

### Found and fixed while executing that plan

- **image-tools serialises its REST and Connect edges with two different JSON
  name conventions.** `POST /api/v1/ai/{op}` returns `job_id` (proto names)
  while `JobsService.WaitJob` returns `resultRef` (camelCase). Backdrop Studio
  decoded `jobId` and therefore discarded every generation job id, reporting
  "inference capability unavailable" — so the 2026-08-12 audit recorded a
  *working* local GPU lane as missing. Fixed on this side; the upstream
  inconsistency is still there and should be made uniform or documented.
- **Hardcoded routing sent every model-backed render to a paid cloud provider.**
  `qualityPolicy:"quality"` + `allowByok:true` resolved to
  `openrouter-image/byok-cloud` while an installed `sd-1.5` on local GPU served
  the same request in ~15s. Local-first constants fixed the leak but replaced it
  with the opposite limit: a style needing more than the installed local model
  set could not be made at all. **Closed 2026-08-13** by the capability router
  (`internal/render/routing.go`, decision D-015): the lane is a declared
  `quality_tier` on the style, the tier is a ceiling on spend as well as a floor
  on capability, and every candidate records the lane, model, execution tier and
  cost it was served by.
- **Depth-graded screens are the repair for the raster styles, and the vector
  styles are the only plated ones.** Phase 8 of the output-quality plan asks for
  a seed version giving each plated style "a coarser screen on its far plane and
  a finer screen on its near plane". Applied to the four styles that currently
  declare plates, that instruction would undo the reason those styles exist.

  All four are vector. They declare empty treatment chains deliberately: tone is
  carried by line density, and putting a screen over line work is precisely the
  defect the 2026-08-13 verdict pass diagnosed in seven of eight below-bar
  styles. Screening the colonnade's arcade plate would produce dots in the shape
  of an arch — the same failure, one layer down.

  **So no screens were applied, and the mechanism is in place for when the
  styles that need it exist.** Per-plate chains resolve, a plate inherits the
  style's chain unless it declares one, each plate is scored by the perceptual
  gate against its own source, and the screen-resolution rule extends to plates.
  `TestSeededPlateScreensResolveFinerThanTheGateSamples` reports that no plate
  declares a ruling rather than passing silently, because a rule that passes
  vacuously reads as coverage.

  The styles that would gain from depth-graded screens are the raster horizon
  and terrain ones, whose depth cue really was flattened away — and they cannot
  declare plates until the raster generators are separated. The instruction is
  right; its subject does not exist yet.

- **Two vector styles were shipping a picture with a whole depth plane missing,
  and nothing refused them.**

  `radiant-orb` and `halftone-horizon` each draw FOUR depth planes. The styles
  built on them, `pale-moon` and `tidal-halftone`, named THREE plates and did
  not merge the fourth into any of them, so the frontmost plane was never
  composited and never reached the delivered image. `pale-moon` shipped without
  its ground; `tidal-halftone` shipped without its SEA, which is the subject of
  the picture.

  This is not a limit of the plate model. `maxPlates` is 3 and `colonnade` also
  draws four planes, carrying two of them on one plate — exactly the
  accommodation the model provides. These two omitted rather than merged.

  It was invisible to every existing test for a specific and instructive reason.
  The flatten-equivalence proof compares the generator's own planes against the
  generator's own flat document, and BOTH of those are complete — the defect
  lives in the STYLE's plate spec, which that proof never consults. So the
  strongest structural guarantee in the plate model was, by construction, unable
  to see a missing layer.

  Found by a legibility measurement reading a reserved region as uniform blank
  paper at 0.911 across the whole rectangle. A halftone of an ocean is not
  uniform anything; the region was blank because the sea was not there.

  Seed v13 merges them back. `TestEveryDeclaredPlaneReachesAPlate` now refuses
  the class, and was confirmed to catch it by reverting the seed and watching it
  name both styles and both dropped planes.

- **A reserve cut into a moving plate is legible for exactly one frame.**

  The vector reserve worked at rest immediately and failed the moment the page
  scrolled: `tidal-halftone` measured **8.15 at rest and 1.00 at half a screen
  of scroll**, and the Phase 10 parallax gate refused the render outright.

  Two distinct causes, and the first fix exposed the second. The reserve slides
  with the plate it is cut into, so it has to be a VOLUME — extended downward by
  that plate's whole travel. And reserving only the frontmost plate leaves the
  plate behind it showing wherever the front one has slid away; that plate is
  usually the ground, a full sheet of opaque paper that does not move at all.
  Both are recorded in D-021.

  The gate that caught this was built in Phase 10 for a different reason and had
  never refused anything before. It is worth noting what would have shipped
  without it: a backdrop that measures 8.15 in every still evidence artifact this
  repo produces, and is illegible as soon as a reader scrolls.

- **CORRECTION: every reserved-copy legibility number reported before this
  entry was produced by a measurement that searched the wrong rectangle.**

  `legibility.cropBounds` computed the search area's top edge as `y * r.Y` — a
  multiply where the three terms beside it add, against an origin that is always
  zero. So the top edge was pinned to the top of the FRAME however far down the
  copy actually sat, and every region was measured as though it began at y=0.
  The band above the copy was swept into the worst-pixel search and its darkest
  ink was returned as the headline's contrast.

  The error ran in the direction that hides work: a region measured taller than
  it is can only score lower, never higher. So a repair could be complete and
  still be reported as a failure — which is exactly what happened. Two styles
  appeared to respond to the generator quiet zone while twenty-one did not, and
  **the two were the ones whose copy sits at the top of the frame**, the only
  place where the defect was harmless. That pattern was visible in the data and
  read as a fact about treatment chains rather than as a fact about the ruler.

  Withdrawn: "2 of 24 pass", "3 of 24 pass", "the other twenty-one sit between
  1.00 and 2.60", and the reading that only pure tone-mapping chains could hold
  a clearing. `survey-relief` at 8.00 stands — it was measured over a region
  that starts near the top, and the conclusion drawn from it (a generator that
  composes around its copy beats any post-process) is still the one the current
  data supports.

  `cropBounds` is shared with `legibility.FindPlacements`, so placement search
  was scoring candidates over the same wrong rectangle and is repaired by the
  same fix.

  Two tests now pin the search geometry, and it is worth saying why neither
  existed before. Every test in the package built a picture, measured it, and
  checked the number — and a fixture that is uniform above the copy scores
  identically whether the top edge is right or wrong. The defect was only
  visible against a picture whose bands DIFFER, which is every real backdrop and
  none of the fixtures. The new tests assert on the search AREA instead: ink
  strictly outside the region and paper strictly inside it, then the reverse
  with one dark pixel in each corner to catch a search that is too small.

- **Reserved-copy legibility, measured through the real engine: 20 of 23
  measured styles pass, ratios 8.57 to 19.82 against a 4.50 bar.**

  `integration/legibility_test.go` submits each style through the really running
  API and measures the picture that comes back. `docs/evidence/legibility/reserved-copy.md`
  carries the table; `make integration-evidence` reproduces it.

  Four mechanisms, and it took all four. The raster generators open a clearing
  (`scenes.QuietZone`); the treatment chain honours it (`Knockout`, D-020); the
  vector generators cut their own, in both polarities and as a volume (D-021);
  and the ruler that had been measuring the wrong rectangle was repaired. The
  first two looked ineffective until the fourth landed, which is recorded above.

  The three failures are model-backed styles — `guided-botanical`,
  `guided-industrial`, `synth-celestial` — and all three declare LIGHT copy. The
  treatment reserve serves dark copy only, and D-022 records why on measurement
  rather than on principle: a solid large enough to give light copy dark ground
  cost `synth-celestial` its perceptual floor, 0.707 subject survival against a
  declared 0.800, and refused three styles that had been rendering correctly. The
  gate is right; the reserve gave way.

  Their answer is compositional, as it was for the vector lane. A model-backed
  style should have the darkness where the copy goes put there by the
  conditioning or a derived plate, which costs no subject survival because it is
  the subject. That belongs with the model lane's plate work.

  Styles that did not render on this host are listed as not-measured rather than
  counted as passes.

- **CORRECTION, same day: the legibility numbers below were measured through the
  unit suite's FAKE executor and are not yet confirmed against the real engine.**

  The fake applies one fixed duotone whatever treatment chain it is handed, so a
  scrim, a halftone and a bloom all come back as the same picture. Every figure
  in the entry that follows is therefore a real measurement of the generator
  output for UNTREATED styles and a measurement of the fake for treated ones.
  The tell was a scrim polarity flip that produced byte-identical ratios: a
  black scrim and a white scrim cannot both leave a number unchanged, so nothing
  was being applied.

  This repo already records the rule that was broken — a style's exact bytes
  have to make a round trip through a running `image-tools` before anything is
  claimed about them. It was applied to renders and not to the measurement OF
  renders.

  `integration/legibility_test.go` now measures each style through the really
  running API and writes `docs/evidence/legibility/reserved-copy.md`, so the
  number is reproducible rather than asserted. Until it has run, treat the
  counts below as UNVERIFIED. What is unaffected: the capabilities are
  independently tested, and the two placement repairs in seed v12 —
  `ember-mesh` and `feature-band-mesh` — were verified against real renders at
  their predicted ratios.

- **ZERO of the twenty styles that declare an overlay region keep their declared
  text colour legible.** Measured 2026-08-13 over the settled catalog, rendering
  each style at `web.hero` and running worst-pixel contrast on its own declared
  regions with its own declared text colour and threshold.

  Twenty styles declare an overlay region. **None passes.** The best is
  `ember-mesh` at 2.81 against a 4.50 threshold. Eight sit at or below 1.20 —
  `swiss-contour` and `terrazzo-truchet` at 1.00, `store-tile-truchet` at 1.01,
  `engraved-colonnade` at 1.03 — which is type the same colour as the thing it
  sits on. Fifteen more styles declare no overlay region at all, so nothing was
  measured for them.

  **The render path has never checked this.** `internal/legibility` exists and
  works; it is reachable only through a standalone RPC that nothing in the
  render or release path calls. So a reserved region has been, in practice, a
  decorative annotation: the catalog says where copy goes and what colour it is,
  and no gate has ever confirmed the copy would be readable there.

  This matters more than it first sounds. The whole point of a reserved region
  is that a designer can drop a headline on the backdrop without editing it, and
  that is one half of the written bar — "a designer would put it on a paying
  customer's landing page without editing it". A backdrop that reads beautifully
  and cannot carry its own declared copy has not met it.

  **The parallax-sweep gate deliberately does not enforce this.** It refuses a
  MOTION-INDUCED failure — legible at rest, illegible somewhere in the sweep —
  because enforcing rest inside a motion feature would refuse the entire catalog
  under the wrong banner, and because repairing twenty styles' regions is
  catalog-maturation work with its own phase. The delta rule is tested; this
  entry is the record of what the delta is being measured against.

  **The repair is per style and is art direction**, not a threshold change: move
  the region to a quiet part of the composition, change the declared text
  colour, or add the scrim the amendment already computes. The gate reports a
  scrim opacity for every failure, so the mechanical option exists — but a scrim
  over a picture chosen for its beauty is the last resort, not the first.

  **Which repair each style needs was then measured.** `legibility.FindPlacements`
  holds a region's size — the author's decision, since a headline needs the room
  it needs — and sweeps its position over a 5x5 grid, reporting worst-pixel
  contrast at each. Searching by hand across twenty styles is how a catalog ends
  up with regions nobody checked; the search is a measurement even though the
  choice is not, which is why the function reports every passing placement and
  chooses none.

  With the DECLARED text colour, only **three of twenty-four** measured regions
  can be fixed by moving them: `ember-mesh` 2.81 → 4.83, `feature-band-mesh`
  2.15 → 5.65, `pale-moon` 3.67 → 6.57. The rest cannot be repaired by placement
  at any position in the frame.

  **The reason is visible in the numbers.** Eight styles cluster at exactly 1.18
  and several more at 1.00-1.10 - a ratio that flat across a whole frame means
  the type is the same tone as the picture EVERYWHERE, not that it landed badly.

  **Every text colour was then tried, and none is enough.** The search was
  re-run over each style's own three ink slots plus pure black and pure white,
  at every grid position. Flipping to the ink roughly doubles the best ratio -
  `engraved-colonnade-vector` 1.01 to 4.49, `demoscene-terrain` 1.18 to 2.21,
  `survey-relief` 1.18 to 2.17, `swiss-contour` 1.10 to 2.07 - and **not one
  crosses the 4.50 threshold.** The best of all twenty-one lands at 4.49, a
  hundredth short.

  **That is a structural result, not a tuning failure.** Worst-pixel contrast
  asks about the LEAST legible pixel in the region, and a pictorial backdrop has
  both light and dark pixels nearly everywhere: a halftone has white paper and
  black dots in every square inch, so dark type meets a dot and light type meets
  the paper, and either way one pixel ruins the ratio. No colour and no position
  can fix that, because the picture is doing what it was designed to do.

  **So the repair is one of two things, and both are real work.**

  Either a *localised* scrim - a gradient behind the reserved region only, which
  is what every design system that puts type on photography actually does. The
  existing `scrim` operation is directional over the whole frame, so a
  region-scoped one is a new capability in `image-tools`.

  Or a generator-drawn quiet zone: the generator leaves the reserved region
  genuinely flat, rather than the style declaring a region over whatever the
  generator happened to draw. That is what `survey-relief`'s repair already did
  by keeping its upper-left open, and what the perceptual gate's `reserved_quiet`
  metric measures - but only the four vector styles have a generator that knows
  about their region at all.

  The three styles fixable by placement should be moved regardless; that is a
  seed edit with measured before-and-after numbers and no new capability.

- **A per-plate chain is not the same picture as the same chain applied once,
  and `normalize` is why.** Measured 2026-08-13 with the plate path isolated
  from the art direction: `city-pop-horizon` rendered flat scores
  `subject_survival 0.990`; the same style with the SAME chain declared on every
  plate scores **0.640**, against a 0.800 floor. Two styles with different
  chains both landed on exactly 0.640, which is what pointed at the path rather
  than at the grading.

  **The cause is per-layer normalization.** `posterize`, `duotone` and the
  screens take `normalize: true`, which maps the input's own tonal range onto
  the full ink ramp. Applied to a whole scene that range spans sky to
  foreground; applied to a plate it spans only that layer. So each plate is
  stretched against its own narrow range and the composite is three
  independently re-stretched bands — a different picture, and a legitimately
  different one, but not the one the style declared.

  **Consequences worth stating before anyone builds on this.** The chain a plate
  inherits from its style is exactly the case that fails: adding a plate spec to
  a treated style silently changes what it looks like, and the change is subtle
  enough that a reviewer would not catch it. Depth-graded chains remain the
  right idea — the mechanism is built, validated and tested — but an author
  using them has to know that each plate normalizes alone.

  **The proper repair is a normalization range that spans the stack**, so a
  plate's screen maps against the composite's tonal range rather than its own.
  That needs an explicit range parameter on the image-tools operations, which is
  a wire change in the scenario that owns them.

  **Seed v11 was written and then reverted.** It gave the three
  source-similarity-1.000 horizon styles depth-graded chains — band count for
  city-pop, grain for riso, bloom radius for solar-bloom. The gate refused two
  of the three, and the diagnosis above says the refusal is about the path and
  not the grading. Shipping art direction that cannot be verified, over a
  mechanism with a known distortion, would have put three styles into the
  catalog whose appearance nobody had checked.

- **No text inference role routes on this host, so generator authoring cannot
  reach a model here.** Measured 2026-08-13 with `RoutingService.PreviewRoute`.
  `author.generator` is refused with `role_not_exposed` on both candidates —
  and so is the pre-existing `classify.fast`, while `extract.structured` reaches
  openrouter and then fails `capability_mismatch`. This is a provider-policy
  state, not a defect in the authoring lane: ai-gateway matches the *logical*
  role name against what each provider's policy exposes, and openrouter's
  exposed set contains `code.default` but no gateway logical role by that name.

  **The role catalog entry was corrected once already.** Its first version named
  a `resource_role` no provider exposes at all; the candidates now point at
  `code.default` (openrouter) and `code.local` (ollama), both verified present
  in `ListProviderRoles`. That fixes the catalog half. The other half — exposing
  a logical role through provider policy — lives in the resource scenarios,
  which are outside this plan's change boundary.

  **What this means for the lane:** it is built, tested and wired end to end,
  and on this host it refuses by name. That is the designed behaviour and the
  same class of limit Phase 4's own risk register anticipates. Every validation
  path is proven against a fake client, because authoring costs money and a
  rejection path that can only be exercised by spending is a rejection path
  nobody tests.

- **The render matrix renders one geometry, and a style can fail on the others.**
  Found 2026-08-13 by submitting `survey-relief` at each seeded surface rather
  than at the matrix's 1440x900. It passed there and was refused by the
  perceptual gate on six of the twelve surfaces its placements permit —
  frequency modulation 0.028 against a 0.030 floor at `web.hero` (2:1), tonal
  occupancy 0.277 against a 0.40 floor at `web.section-band` (3.4:1), and four
  more between them. Not systemic: the other three vector generators and the
  raster styles checked alongside it pass on the same band surfaces.

  **Half repaired.** The height field applied a hardcoded `1.7` aspect
  correction, so hills were round on a 1.7:1 frame and progressively flatter and
  sparser on everything else — a wide plate was the same land stretched rather
  than more land. Distance is now measured in short-edge units so a hill is
  round in pixels at any frame shape, and the summit count scales with the
  frame's area, placed on a golden-ratio sequence clear of the overlay band.
  `web.hero` — the flagship surface, and the one this found — now passes, and
  `TestARelieFieldGrowsWithTheFrame` pins the composition rule.

  **Still below bar on the wide, short bands**: `web.section-band` (3.4:1),
  `web.footer-wash` (4:1), `web.pricing-band` (2.8:1),
  `social.profile-banner` (3:1), `email.header` (2.5:1) and `social.og-card`
  (1.9:1, at 0.394 against 0.400). The remaining cause is stroke weight: it is a
  fraction of the short edge, so on a 1440x360 wash the pen is 0.4px and a
  drawing that is nothing but line work nearly vanishes.

  **An attempted fix was reverted, deliberately.** Scaling the pen to the
  geometric mean of the two edges cleared all five band surfaces and broke four
  others — `web.error-page` (the matrix's own geometry) fell to exactly 0.030 on
  frequency modulation, and the portrait frames followed, because thicker lines
  fill a tall frame more uniformly. Six surfaces passed before and six after: the
  failures moved rather than reduced. That is the signature of tuning to a metric
  instead of designing a picture, which is the failure this plan exists to fix,
  so the change was reverted rather than kept for the appearance of progress.

  What this needs is art direction across the aspect range — a plate that
  composes differently on a band, not the same plate with a heavier pen — and it
  belongs with the catalog-maturation work. **The gate is meanwhile doing its
  job**: nothing below bar ships, and each refusal names the metric and the
  bound it missed.

  **The matrix's single geometry is the wider gap, and it is not closed.**
  `make integration-evidence` still renders every style at one shape, so this
  class of defect stays invisible to it. The check that found this one was
  manual. Widening the matrix to a few extreme aspects — the 4:1 wash, the 2:1
  hero, the 0.46:1 mobile hero — is the fix.

- **A generation was sized from a model that was never going to draw it.**
  Found on the live wire 2026-08-13, after the router landed and before it
  shipped. `op-art-interior` at the frontier tier really reached OpenRouter, but
  the canvas had been sized from `sd-1.5`: the geometry probe sent
  `ModelsService.SelectModel` no routing policy, so image-tools previewed the
  local default and answered with its 512px native edge and 768px cap. The
  generation went out at 768x512 to a provider with no such limit.

  The probe was right to ask and image-tools was right to answer; the question
  was incomplete. `SelectModelRequest` now carries `quality_policy` and
  `allow_byok`, which is what makes its own documented claim — "preview which
  enabled model would run" — true. `TestTheGeometryProbeIsAskedTheServedLanesQuestion`
  pins it. Worth noting how it was caught: every unit test passed, because the
  fake answered the same geometry regardless of policy. Only a real submit
  against a really running image-tools showed it.
- **A field named `image_png` could carry JPEG.** The cloud provider returned
  2048x2048 JPEG; generated sources are now normalised to PNG through
  image-tools' `convert` op before the treatment chain.
- **The GPU is shared and diffusion loses the allocation race.** This host
  routinely holds ~10 GB of idle language models against a 16 GB card, so a
  model-backed render can fail with `ErrorOutOfDeviceMemory` through no fault of
  the catalog. The integration lane classifies that as `SKIP(gpu-capacity)` —
  never a pass, never a style failure.

  **Measured 2026-08-12** with ~5.9 GB reported free: `512x512` generates fine,
  `768x448` and `768x512` both fail on a single 1.81 GB Vulkan allocation. The
  ceiling is a per-allocation limit under fragmentation, not raw free memory, so
  "free VRAM looks sufficient" is not evidence that a render will succeed.

  Deliberately **not** worked around by generating smaller. The plan's own
  instruction is to record the contention rather than lower resolution to make a
  run finish: 768x512 is the correct canvas for a 16:9 hero on an SD-1.5-class
  model, and shrinking it to force a green light would trade a visible failure
  for an invisible quality loss. Re-run when the card is free, or raise
  generation priority against the other residents.

- **A new subject needs a new generator, not a new catalog row.** The catalog is
  40 styles across 13 generators as of seed v7, and the rule has not changed:
  `botanical`, `celestial`, `figure`, `industrial`, `interior`,
  `textile_material` and `object_metaphor` are reachable only through
  model-backed strategies, and `ResolvePreset` refuses them procedurally rather
  than silently substituting an abstract field. `cartographic` and `atmospheric`
  left this list when `contour` and `nebula` landed, which is what closing the
  gap looks like — a generator that genuinely depicts the subject, not a
  relabelled one that resembles it.
- ~~`TreatmentParams` is unvalidated on write.~~ **Closed 2026-08-12.**
  `validateStyle` now calls `imageengine.ValidateChain`, which rejects malformed
  JSON, non-object values, parameters naming an operation the style does not
  run, and — most importantly — any field image-tools' proto will not accept.
  Both write paths are covered (`CreateStyle` and `ImportStylePack`). The
  wire-format knowledge lives in `internal/imageengine`, so the catalog asks
  "will the engine take this?" without learning protobuf.
- ~~Two seeded styles cannot be released.~~ **Unblocked 2026-08-12** by the
  Asset Studio ingress above; the release path no longer refuses them by
  design. What still blocks them on this host is generation capacity, which is
  a different problem with a different answer: local diffusion runs out of
  device memory at hero aspect, so they are recorded as `SKIP(gpu-capacity)`
  rather than as passing. See `docs/evidence/catalog/coverage.md`.
- **Catalog visual evidence is not committed.** The 12 rendered style previews
  produced during seeding came from a throwaway probe and were removed rather
  than shipped, because unreproducible evidence is the defect this repair
  existed to fix. A repeatable path needs a running image-tools, so it belongs
  in an integration lane, not a unit test.

## Shipped-then-caught: the wire contract (2026-08-12)

The catalog seeded earlier that day requested `normalize` and brand inks on the
Tier-2 screens. Neither existed on `ops.proto`, and `protojson.Unmarshal`
**rejects unknown fields** — so eleven of sixteen styles would have failed their
render with `400 invalid params: unknown field "normalize"`.

Both unit suites stayed green throughout, and that is the lesson worth keeping:
backdrop-studio tests against a fake executor that never reaches the REST edge,
and image-tools tests its treatments below the wire. Neither side could see the
boundary between them. The gate now lives in
`backdrop-studio/internal/imageengine/wire_contract_test.go`, which resolves
every seeded style's parameters against a real brand and asserts the exact bytes
parse as `opsv1.OpParams` — plus `image-tools/handlers/ops/wire_test.go`, which
pins every parameter the engine accepts.

**Rule for future work here: a parameter is not shipped until it round-trips
through protojson.** Adding a knob to `treatments.Params` without extending the
proto produces a loud failure in production and silence in CI.

## Corrections to the 2026-08-11 audit

The audit that drove this repair got one finding wrong, recorded here so it is
not re-derived:

- **The placement preview never applied its placement.** `PreviewPlacements`
  resized the candidate to the viewport and ignored the placement argument
  entirely, so every placement returned identical pixels. It now composites four
  real layouts with scrim and copy chrome.
- **`drawScaled` stretched instead of cover-cropping**, so a 1600x1000 backdrop
  in a tall split panel rendered a circular sun as an oval. Now scaled by the
  larger axis ratio and centre-cropped.
- **"Treatments are not reachable from the CLI" was false.** All 18 were already
  registered in `image-tools/cli/domains/ops/register.go`, with param builders
  and proto messages. `cli/manifest.json` legitimately carries only
  Connect-bound calls; REST multipart run commands are hand-appended and
  documented in the manifest's `omitted` array. The audit tested a **stale
  installed binary** (`~/.vrooli/bin/image-tools`, a day older than the source).
  Rebuild before concluding a CLI surface is missing.

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

> **Note on scope.** Every entry below is work in *another* scenario that
> Backdrop Studio depends on. They are recorded here because they were
> discovered while designing this scenario and they gate its delivery.
> Each still needs filing against its owning scenario through
> `prompt-manager skill read report-bug` before it becomes scheduled work —
> this file is the design-time record, not the work queue.

---

### 2026-08-11 — asset-studio spec composition may be specialised to character media

**Symptom:** The spec path was reviewed for identity coupling. `asset-studio`'s
prompt and reference-image composition path is built around binding *identity
records* — characters, scenes, products — into a prompt template. Backdrop Studio
binds a scaffold and a palette instead. If the composition path assumes an
identity binding the way the verdict table does, the handoff needs more
generalization than the verdict fix alone.

**Root cause:** Identity-version resolution is optional; the dispatcher accepts
resolved creative intent and generic conditioning references. What *is* established: the asset and
disclosure tables are identity-free (see the entry above), and `OT-P0-016`
already models conditioning artifacts generically enough to include "a trained
adapter, a reference image set, **or a look**" — which suggests the design
anticipated non-character conditioning. That is encouraging but not proof.

**Workaround:** Backdrop Studio composes its own plan (`compose` domain) and hands
`asset-studio` a *result* to release rather than a spec to resolve. If the
composition path turns out to be character-coupled, this boundary keeps the
blast radius to the release call.

**Real fix:** Keep identity-version fields optional for generic creative
specifications and add a regression test whenever a new conditioning kind is
introduced. The model-backed handoff is identity-free and needs no workaround.

**Resolved 2026-08-12.** The worry was correct to record and turned out not to
bite. `StudioService.IngestExternalAsset` takes a generic
`ConditioningReference` — the same type `OT-P0-016` already used — so a scaffold
goes through as `kind: "scaffold"` with no identity anywhere in the request, and
the resulting asset carries no identity version ids. It is proven both below the
wire (`TestIngestAcceptsANonCharacterConditioningKind`) and across it
(`TestAssetStudioAcceptsAScaffoldConditionedIngress`, which asserts the
disclosure records `scaffold arcade@edge`). The standing instruction stands: a
new conditioning kind still needs an explicit contract test before it is treated
as stable.

**Owner:** resolved — `asset-studio`

**Refs:** `scenarios/asset-studio/api/internal/studio/{studio.go,dispatcher.go}`,
`scenarios/asset-studio/PRD.md` OT-P0-004, OT-P0-005, OT-P0-016

---

### 2026-08-11 — two recipe catalogs risk divergence with image-tools Looks

**Symptom:** Not a defect; a design tension worth recording before it becomes
one. `image-tools` already has a **Look** — a prompt template plus ordered AI and
deterministic steps with merged parameters, and a documented `Compile()` seam.
Backdrop Studio's **Style** is a superset of that shape. Two catalogs of
"recipes" in one repository can drift into two answers for the same question.

**Root cause:** The abstractions genuinely differ in scope. A Look is a
*rendering recipe* with no opinion about layout. A Style adds classification,
placement, reserved-region geometry, gates, and lineage — the layout judgement that is
this scenario's whole reason to exist. Collapsing them would push landing-page
concerns into a general-purpose image toolbox.

Worth noting the current seed pack is not a conflict in practice: `image-tools`
ships eleven Looks and all of them are consumer photo filters — Polaroid 600,
Noir, Golden Hour, Anime, Vivid Pop. None is a backdrop recipe. The shapes
overlap; the content does not.

**Workaround:** None needed. Keep Style as the outer record and compile *down* to
a Look or a step list when submitting to `image-tools`, so `image-tools` stays
the single authority on what a rendering step means.

**Real fix:** Revisit if a third consumer needs classified recipes. At that point
the classification layer may deserve promotion out of Backdrop Studio. Until
then, one consumer does not justify a shared abstraction.

**Owner:** unassigned — design watch item

**Refs:** `scenarios/image-tools/api/internal/looks/{compiler.go,seed.go}`

---

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet — scenario is documentation-only._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
