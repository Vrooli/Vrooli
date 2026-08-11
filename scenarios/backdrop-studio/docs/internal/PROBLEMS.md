# Problems — Backdrop Studio

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

### 2026-08-11 — image-tools ships no deterministic treatment operations

**Symptom:** Every Backdrop Studio style, in all four strategies, terminates in
a treatment chain (`RND-001`). None of the operations that chain names exist.
There is no halftone, line screen, ordered dither, error diffusion, duotone,
posterization, risograph, grain, bloom, chromatic aberration, typographic
mosaic, or scrim anywhere in the repository.

**Root cause:** `image-tools`' deterministic operation surface covers
`grayscale`, `sepia`, `blur`, `sharpen`, `invert`, and geometry (crop, resize,
rotate, format conversion). Everything else in its catalog
(`api/internal/operations/operations.go`) is a model-backed operation —
`text_to_image`, `upscale`, `inpaint`, `segment`, and similar. The reprographic
and quantization layer was never built because nothing needed it until now.

**Workaround:** None. This is the critical path. Backdrop Studio cannot produce a
single releasable candidate without it, and no local substitute is acceptable —
implementing treatments here would violate `CMP-004` and bury a generic
capability inside one product.

**Real fix:** Add the treatment operations to `image-tools` as pure-Go
deterministic ops, registered in the operations catalog alongside the existing
`grayscale`/`sepia` family. They are model-free, resolution-independent, and
golden-image testable.

Only **filters** belong upstream — anything that transforms an input image. The
generators (`voronoi`, `reaction_diffusion`, `flow_field`, and the rest) are
content and stay in this scenario's `scaffold` domain (D-003). The full split is
in `../reference/taxonomy.md`.

**Tier 1 — unblocks the procedural lanes.** Nothing ships without these:

| Op | Parameters | Notes |
|---|---|---|
| `duotone` | ink, paper, ramp stops | Simplest; unlocks palette lock on its own |
| `posterize` | levels, inks | Duotone with a quantized ramp |
| `halftone` | lpi, angle, dot shape | Rotated grid, dot radius from luminance |
| `dither_ordered` | matrix, depth | Bayer threshold matrix |
| `dither_diffusion` | algorithm, depth | Floyd–Steinberg error propagation |
| `grain` | sigma, contrast, distribution | Keeps colour; lightest-touch treatment |
| `scrim` | direction, stops, opacity | Required by the legibility gate (`LEG-005`) |

**Tier 2 — the breadth that makes the catalog a showcase:**

| Op | Parameters | Notes |
|---|---|---|
| `line_screen` | pitch, angle | Same as halftone, modulating stroke weight |
| `stipple` | density, jitter, dot size | Irregular positions; reads hand-drawn where halftone reads mechanical |
| `engraving` | line weight, spacing, follow | Tonal value as hatching |
| `aberration` | dispersion, bloom threshold | Radial channel separation plus highlight bleed |
| `bloom` | threshold, radius, intensity | Also serves `godray` |
| `curve` | per-channel control points | Serves `solarization` and `cross_process` |
| `defocus` | radius, aperture blades | Serves `bokeh` |
| `motion_blur` | angle, distance | Serves `long_exposure` |
| `ascii_mosaic` | cell, ramp, font | Luminance to glyph density |
| `pixel_sort` | axis, key, threshold band | Kim Asendorf's technique; streaked and unmistakably digital |
| `displacement` | source, amplitude | Also composes `letterpress` and `fluted_glass` |

**Tier 3 — completes the declared axis:**

`glitch`, `kaleidoscope`, `slit_scan`, `fluted_glass`, `photomosaic`, `resample`.

The taxonomy declares 44 treatments deliberately — it is an open list, and a
value with no operation behind it is simply unbuilt rather than wrong. Tier 1 is
the critical path; tiers 2 and 3 are additive and can land in any order.

**Value beyond this scenario:** this is a permanent capability deposit. Once
landed, every scenario in the portfolio can screen, quantize, and palette-lock an
image — and the `procedural-treated` strategy becomes a deterministic-code
replacement for what would otherwise be model calls.

**Owner:** unassigned — `image-tools`

**Refs:** `scenarios/image-tools/api/internal/operations/operations.go`,
`scenarios/image-tools/api/internal/looks/compiler.go` (the `STEP_KIND_AI` vs
deterministic step split that already anticipates these)

---

### 2026-08-11 — asset-studio conformance verdicts are structurally identity-coupled

**Symptom:** Backdrop Studio must release model-backed candidates through
`asset-studio` (`REL-002`) and record a verdict against them. Its verdict is a
*placement fitness* judgement — measured contrast under an overlay region. The
`asset-studio` verdict table cannot represent that.

**Root cause:** Verified in `scenarios/asset-studio/api/internal/studio/schema.sql`:

```sql
CREATE TABLE studio_conformance_verdicts (
  ...
  identity_version_id TEXT NOT NULL,
  basis TEXT NOT NULL CHECK (basis IN
    ('reference-sheet','reference-image-set','conditioning-artifact','prose-only')),
  actor_kind TEXT NOT NULL CHECK (actor_kind = 'operator'),
  ...
);
```

A backdrop has no identity version, so `identity_version_id NOT NULL` cannot be
satisfied. None of the four permitted `basis` values describes a measurement.
And `actor_kind = 'operator'` forbids recording a verdict that a deterministic
gate produced rather than a human.

**Good news, and worth recording so nobody re-derives it:** the rest of the spine
is *already generic*. `studio_assets` carries `status`, `alt_text`, `disclosure`,
`ai_generated`, and `credential_claims` with **no identity column at all**. The
asset, disclosure, and release path is reusable as-is. Only the verdict table is
coupled.

**Workaround:** For the procedural lanes, none is needed — `REL-003` deliberately
releases those locally, so the dependency does not exist. The model-backed lanes
are blocked until this is resolved.

**Real fix:** Generalize the verdict from *identity conformance* to *release
conformance* in `asset-studio`:
1. Make `identity_version_id` nullable, or introduce a verdict-subject
   discriminator.
2. Extend the `basis` CHECK with a measurement value such as
   `automated-measurement`.
3. Relax `actor_kind` to permit a named non-human gate, while keeping the
   operator requirement for identity verdicts specifically — the human
   requirement on identity conformance is deliberate and must survive.

**Open question for the owner:** whether this is one generalized verdict table or
two verdict kinds sharing an asset. Backdrop Studio has no preference; it needs
only to record a measured verdict against a released asset.

**Owner:** unassigned — `asset-studio`

**Refs:** `scenarios/asset-studio/api/internal/studio/schema.sql`,
`scenarios/asset-studio/PRD.md` OT-P0-010, OT-P0-011, OT-P0-017

---

### 2026-08-11 — asset-studio spec composition may be specialised to character media

**Symptom:** Unverified risk, flagged rather than diagnosed. `asset-studio`'s
prompt and reference-image composition path is built around binding *identity
records* — characters, scenes, products — into a prompt template. Backdrop Studio
binds a scaffold and a palette instead. If the composition path assumes an
identity binding the way the verdict table does, the handoff needs more
generalization than the verdict fix alone.

**Root cause:** Not yet established. What *is* established: the asset and
disclosure tables are identity-free (see the entry above), and `OT-P0-016`
already models conditioning artifacts generically enough to include "a trained
adapter, a reference image set, **or a look**" — which suggests the design
anticipated non-character conditioning. That is encouraging but not proof.

**Workaround:** Backdrop Studio composes its own plan (`compose` domain) and hands
`asset-studio` a *result* to release rather than a spec to resolve. If the
composition path turns out to be character-coupled, this boundary keeps the
blast radius to the release call.

**Real fix:** Read `asset-studio`'s spec composition and render-submit path and
either confirm it accepts an identity-free spec or record the specific coupling.
This should happen before the model-backed lanes are planned, not before the
procedural lanes ship.

**Additional context:** `asset-studio` has never been exercised in production.
It is built but unvalidated, so Backdrop Studio should expect to be its first
real consumer and should budget for finding defects rather than assuming a
working dependency.

**Owner:** unassigned — `asset-studio`

**Refs:** `scenarios/asset-studio/api/internal/studio/{studio.go,dispatcher.go}`,
`scenarios/asset-studio/PRD.md` OT-P0-004, OT-P0-005, OT-P0-016

---

### 2026-08-11 — brand-manager palette-slot resolution surface is unverified

**Symptom:** `CMP-002` requires resolving `$brand.*` treatment parameters into
concrete colours for the active brand at render time. Whether `brand-manager`
exposes a query shaped for that is unconfirmed.

**Root cause:** Not diagnosed. `brand-manager` carries `brands`, `design`,
`contrast`, `apply`, `assets`, and an `imagetools` adapter domain, so the
authority and an existing integration path both exist. What is unverified is
whether a caller can ask "give me the named token values for brand X" and
receive a stable answer, versus the surface being shaped only for applying a
brand to a target.

**Workaround:** Define the palette binding behind a Backdrop Studio seam with a
local fake, so the `compose` domain and its tests proceed while the real client
lands later. This is standard seam practice here and costs nothing.

**Real fix:** Confirm or add a brand token read surface. If it does not exist, it
belongs in `brand-manager` rather than here — palette authority is explicitly not
this scenario's to own.

**Owner:** unassigned — `brand-manager`

**Refs:** `scenarios/brand-manager/api/internal/{brands,design,contrast,imagetools}`

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

### 2026-08-11 — landing-page-business-suite hero imagery is hardcoded

**Symptom:** The first consumer cannot reference a released backdrop. Hero
imagery in `landing-page-business-suite` is fixed in the component rather than
resolved from an asset reference, so nothing Backdrop Studio releases can reach
the page it was built for.

**Root cause:** No consumer contract existed when the section was written.

**Workaround:** None required until the release surface exists (`REL-004`).

**Real fix:** Change `HeroSection` to accept a backdrop reference and a
placement, resolving both at render. This is the retirement claim recorded in the
ecosystem-fit review, and it is deliberately small — the point is that *every
later* landing page inherits the library once this seam exists.

**Owner:** unassigned — `landing-page-business-suite`

**Refs:** `scenarios/landing-page-business-suite/ui/src/surfaces/public-landing/sections/HeroSection.tsx`

### 2026-08-11 — the experience-contract orientation gate requires a page literally named `dashboard`

**Symptom:** `make orient` cannot pass `experience-contract` for this scenario.
The gate asserts `file_exists: experience/pages/dashboard.json`, but Backdrop
Studio's home route is the style catalog and there is no dashboard.

**Root cause:** `.vrooli/orientation.json` is generated from the `react-vite`
template, which seeds `dashboard`, `notes`, and `settings` example pages. The
gate hardcodes the *generated placeholder filename* rather than checking the
property it means — that the experience index describes the scenario's real
routes. The neighbouring `example-domain-removed` gate explicitly requires the
`notes` example to be deleted, so the template already expects placeholders to
go; `dashboard` was evidently assumed to be a universal route name.

**Workaround:** None taken deliberately. Renaming the catalog page to
`dashboard` would satisfy the check while making the product's primary surface
worse named, so the gate is knowingly left unmet rather than gamed. The other
checks in this step — a populated `journeys` registry and a scenario-specific
index description — are satisfied.

**Real fix:** In `template-manager`, change the `experience-contract` check from
a fixed `dashboard.json` path to a property-based assertion: at least one page
exists, every `pages[].path` in the index resolves, and no page retains the
generated placeholder purpose text. That tests what the gate is actually for.

**Owner:** unassigned — `template-manager`

**Refs:** `scenarios/backdrop-studio/.vrooli/orientation.json` step
`experience-contract`; `templates/scenarios/react-vite`

---

### 2026-08-11 — the react-vite `notes` example domain is still present

**Symptom:** The orientation gate `example-domain-removed` cannot pass. The
generated `notes` example domain is still fully present under
`api/internal/notes/` — service, repository, SQLite store, attachments, mocks,
and tests.

**Root cause:** Expected at this stage. The scenario is documentation-only; no
product code has been written, so the template scaffolding has not yet been
displaced. Recorded here so it is not rediscovered as a defect, and so it is not
forgotten once code begins.

**Workaround:** None needed. The gate is honestly red.

**Real fix:** Delete `api/internal/notes/` when the first real domain lands —
`surfaces` or `catalog`, per the build order. Its BAS experience-spec cases were
already removed as orphans referencing pages this scenario does not have.

**Owner:** unassigned — this scenario, at first-code

**Refs:** `api/internal/notes/`, `.vrooli/orientation.json` step `example-domain-removed`

---

### 2026-08-11 — scenario-to-android and scenario-to-ios declare listing-asset targets with no producer

**Symptom:** Not a defect — an unclaimed integration, recorded before it is
forgotten. Both mobile deployment scenarios already declare store listing asset
generation:

- `scenario-to-android` `OT-P1-007` — "Generate store listing assets, screenshots
  from journey evidence, and a privacy policy from scenario metadata"
- `scenario-to-ios` `OT-P1-007` — "Generate listing assets and screenshots from
  journey evidence at the required device sizes"

Neither has a producer for the *imagery* half. Screenshot capture from journey
evidence is theirs and already scoped; the backdrop behind a device frame, the
feature graphic, and the surface geometry are not.

**Root cause:** No scenario owned ambient imagery when those targets were written.

**Workaround:** None required. The store lane is P1 here (`SUR-004`, `CMP-008`,
`UIX-009`) and sequenced after the procedural lanes prove the architecture.

**Real fix:** When the store surfaces land, wire the seam in both directions:
they supply a captured screenshot and a target surface id; Backdrop Studio
returns a composed asset conforming to that surface's geometry (`SUR-003`).
Neither side should acquire the other's responsibility — see D-011.

**Open question for the owners:** whether the composed asset is released through
this scenario's backdrop reference surface (`REL-004`) or handed back as bytes
for their existing submission pipeline. Backdrop Studio prefers the reference,
but the store submission path is theirs and may need the bytes.

**Owner:** unassigned — `scenario-to-android`, `scenario-to-ios`

**Refs:** `scenarios/scenario-to-android/PRD.md` OT-P1-007,
`scenarios/scenario-to-ios/PRD.md` OT-P1-007,
`docs/reference/surfaces.md`, `docs/internal/DECISIONS.md` D-011


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
