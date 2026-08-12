# Problems — Backdrop Studio

## Remaining after the treatment-layer plan

- `REL-006` remains unbuilt: sized-variant derivation with reserved-region
  preservation is a separate geometry capability and is not silently implied by
  the store/device-frame implementation. **Narrowed 2026-08-12:** delivery
  geometry now comes from the surface record and every style renders correctly
  at every surface it declares, so the *sizing* half is done. What remains is
  deriving several sizes from one rendered master and rejecting a crop that
  pushes the focal mass into the reserved region.
- Asset Studio currently exposes metadata/render/release RPCs but no external
  backdrop byte-ingress RPC. Backdrop Studio therefore owns an injectable
  publisher seam and refuses model-backed release without it. This keeps
  provenance and disclosure from being duplicated or fabricated.
  **Filed 2026-08-12 as `knw-1786507241786326657`** (scenario-qa,
  `bug-inbox/code-defect/asset-studio-exposes-no-external-byte-ingress-rpc`).
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
- **`docs/evidence/` is 8.8 MB across 36 PNGs** now that evidence renders at
  delivery resolution. `treatments/grain.png` alone is 2.9 MB because noise does
  not compress. Delivery resolution was the point — a screen cannot be judged at
  64×48 — but whether this belongs in git or behind a blob seam is an owner
  decision that has not been made.
- **`ascii_mosaic` is the only treatment whose cell size is coupled to a
  font.** It blits a 7×13 bitmap face, so `block_size` values far from 7 resample
  the glyph. Legible, but not crisp at extremes.

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
  the same request in ~15s. Now local-first (`balanced`, `local_only`, BYOK off,
  batch priority with reclaim).
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

- **The catalog is 16 styles across 4 scenes.** Adding a genuinely new *subject*
  (botanical, celestial, figure, industrial) needs a new scene generator, not a
  new catalog row — those subjects are only reachable through model-backed
  strategies today, and `scenePreset` refuses them procedurally rather than
  silently substituting a field.
- ~~`TreatmentParams` is unvalidated on write.~~ **Closed 2026-08-12.**
  `validateStyle` now calls `imageengine.ValidateChain`, which rejects malformed
  JSON, non-object values, parameters naming an operation the style does not
  run, and — most importantly — any field image-tools' proto will not accept.
  Both write paths are covered (`CreateStyle` and `ImportStylePack`). The
  wire-format knowledge lives in `internal/imageengine`, so the catalog asks
  "will the engine take this?" without learning protobuf.
- **Two seeded styles cannot be released.** `guided-botanical` and
  `constructivist-figure` are model-backed and blocked on asset-studio byte
  ingress (`knw-1786507241786326657`). They are seeded deliberately so the lanes
  have real coverage the moment that lands.
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

**Additional context:** `asset-studio` has now been exercised by conformance and dispatcher tests.
Backdrop Studio is an early generic consumer, so any new conditioning kind still
Backdrop Studio is an early generic consumer, so any new conditioning kind still
needs an explicit contract test before it is treated as stable.

**Owner:** unassigned — `asset-studio`

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
