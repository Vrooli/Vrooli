# Progress — Backdrop Studio

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append entries when work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-11 | claude | partial | Scenario generated from `react-vite` 1.6.5 with the `vrooli-default` design kit, then documented ahead of implementation at the operator's request. Landed: `PRD.md` with 18 P0, 8 P1 and 4 P2 operational targets in EARS shape; a seven-module requirements registry carrying 40 requirements (`catalog`, `scaffold`, `compose`, `render`, `legibility`, `release`, `workbench`), replacing the starter `01-foundation` module; `docs/concepts/{ARCHITECTURE,DOMAINS,DATA,FLOWS,INTEGRATIONS,UI-ARCHITECTURE}.md`; `docs/reference/taxonomy.md` defining the five axis enums; `docs/internal/DECISIONS.md` with eight durable decisions; `docs/business/{MONETIZATION,GO-TO-MARKET}.md`; and eight L0 experience page specs replacing the generated dashboard/notes placeholders. Six upstream dependency gaps recorded in `PROBLEMS.md`. **No product code written** — `api/`, `cli/` and `ui/` remain the generated scaffold, so `scaffold-health`, `first-real-vertical-slice` and `example-domain-removed` gates are correctly unmet. Validation: all requirements and experience JSON parse; `make orient` reports the documentation gates. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | status | Summary of what landed, how it was validated, and what remains |
```

Status values: `partial`, `complete`, `blocked`, `reverted`.

## Next

The build order is set in `../concepts/DOMAINS.md`. In dependency order:

1. **Upstream first.** Treatment operations must land in `image-tools`
   (`PROBLEMS.md`, first entry). Nothing here produces a releasable candidate
   until they do, and no local substitute is acceptable — implementing them in
   this scenario would violate `CMP-004` and bury a generic capability inside
   one product.
2. **`surfaces` + `catalog` domains.** The taxonomy and the web surface
   geometries become real and queryable. Nothing renders yet, and that is fine.
3. **`scaffold` + `compose` + `render`, procedural lanes only.** This is the
   milestone that matters: a complete, useful product with zero model
   dependency, working offline at no cost. If the procedural lanes work end to
   end, the architecture is proven and everything after is addition rather than
   risk.
4. **`legibility` + `release` + dual-view `workbench`**, then the first consumer
   integration in `landing-page-business-suite`.
5. **`guided` and `synthesized` lanes**, which depend on the `asset-studio`
   verdict generalization (`PROBLEMS.md`, second entry).
6. **Store surfaces** — the store surface pack, device-frame composition, and
   platform-accurate preview; first integration into `scenario-to-android` and
   `scenario-to-ios`.
7. **Workbench depth** — contact sheet, brand-themed chrome, remix.

## 2026-08-11 — documentation review applied; scope extended to output surfaces

**Review fixes.** Six findings from the documentation review were applied:

- Three load-bearing invariants that had been traced to targets not stating them
  now have their own P0 targets — style version immutability (`OT-P0-019`),
  render lifecycle contract (`OT-P0-020`), and alt text as a release
  precondition (`OT-P0-021`). `CAT-005`, `RND-005`, and `REL-005` were repointed.
- All experience pages and journeys moved from `active` to `draft`. The schema
  defines `active` as "the build is expected to conform"; there is no build, and
  claims and bindings are still empty. The index's own description already said
  so — the status field now agrees with it.
- A filter fixture used `treatment=ascii_mosaic`, which is not a taxonomy value.
  Corrected, and `taxonomy.md` now carries an explicit axis-value to
  operation-name mapping, since the two are separate namespaces and nothing had
  said so.
- `workbench` and `surfaces` are now declared bounded contexts. The requirements
  had a seventh module with no context behind it.
- `OT-P1-005` gained `CMP-007`; it had been the only non-P2 target with no
  requirement.
- Orphaned BAS experience-spec cases for `dashboard` and `notes` were removed —
  neither is a page in this scenario's contract.

**Scope extension.** Two capabilities were added, both operator-directed:

- **In-context judgement is now P0** (`OT-P0-024`, `UIX-007`). Every candidate is
  presented standalone *and* composed into a mockup of its target surface with
  foreground content in place. Mockup chrome fidelity is chosen by surface kind
  (D-010): product surfaces preview in chrome derived from the target scenario's
  design tokens, store surfaces inside a facsimile of the destination store.
- **App-store listing assets** are now in scope. This introduced the `surfaces`
  domain — a registry of output targets with declared pixel geometry and a cited
  external authority — plus store placements, device-frame composition
  (`CMP-008`), and a geometry conformance gate (`SUR-003`).

**The model generalization this forced.** `copy_safe` became `reserved_regions[]`
with a declared kind (D-009). A device frame *occludes* rather than *overlays*:
contrast beneath it is meaningless, but focal detail placed there is wasted.
Those are different gates and one untyped rectangle could not express which
applies. This is schema shape, which is why it is P0 rather than deferred.

**Boundary held.** Backdrop Studio does not capture application screenshots and
does not submit to stores (D-011). `scenario-to-android` and `scenario-to-ios`
already own both; their `OT-P1-007` listing-asset targets simply had no producer
for the imagery half. Recorded in `PROBLEMS.md`.

**Also added.** `docs/reference/surfaces.md` and `docs/reference/starter-catalog.md`
(both registered in the docs manifest); a `surfaces` experience page; a
`recover-from-a-failed-gate` journey, since both existing journeys ran clean and
this scenario's distinguishing behaviour is that it refuses things; `LEG-006`,
which requires an available amendment on every contrast refusal, because scrim
solving is P1 and a P0 refusal that states only a number contradicts the PRD's
own voice contract.

**Still open.** Store geometries in `surfaces.md` are marked `UNVERIFIED` and
must be confirmed against their cited authority before producing a submitted
asset. The starter catalog's style list is an art-direction call left to the
operator; only its coverage rule is settled.


## 2026-08-11 — surface scope surveyed; admission test recorded instead of new targets

Surveyed every scenario PRD for declared, unmet imagery needs. Found exactly two
— `scenario-to-android` and `scenario-to-ios` `OT-P1-007` — both already claimed
by the store lane. `chart-generator` is evidential and out of scope;
`document-manager`'s raster target is intake, not production.

**No new targets were added.** Other genuinely good fits exist — extension store
tiles, desktop splash, email headers, deck backgrounds, repository social
previews, in-product empty and error states — but none is requested, and the
`surfaces` domain already makes each of them a data row rather than a feature.
Committing to them now would inflate P1 without making any arrive sooner (D-012).

Added instead: an **admission test** in `reference/surfaces.md` with a worked
table recording which candidates were accepted, deferred, or refused, so the
scope question is answered once. Two non-obvious boundaries recorded — print
collateral is the best conceptual fit and the least free (sRGB pipeline, WCAG
gate, no CMYK or bleed), and in-product imagery is the largest latent use but is
usually focal rather than ambient.

**Open for the operator:** `scenario-to-extension` has no listing-asset target at
all, though the Chrome Web Store requires promotional tiles the same way Play
does. That is a gap in *that* scenario, not this one — worth a look, but not ours
to file.


## 2026-08-11 — taxonomy and surface catalogue broadened for implementation

Both open lists were widened so an implementing agent can build a real showcase
rather than inferring the intended scope. Web research informed the treatment
expansion; several genuinely common techniques were missing.

**Taxonomy.** `treatment` went from 17 values to 44, reorganised into eight
families and split by **where each is implemented** — filters belong in
`image-tools`, generators stay in `scaffold` (D-003). New: stipple, engraving,
letterpress, thermal, bokeh, godray, solarization, cross-process, CRT scanline,
anaglyph, noise field, metaball, voronoi, reaction-diffusion, cellular automata,
wave function collapse, truchet, L-system, strange attractor, contour, pixel
sort, glitch, displacement, fluted glass, kaleidoscope, slit scan, photomosaic.
`subject` went 7 → 14 and `lineage` 10 → 24. Every list is marked open.

**Upstream ask tiered.** `PROBLEMS.md` now splits the `image-tools` request into
Tier 1 (seven ops, the critical path), Tier 2 (eleven ops, the breadth that makes
a showcase), and Tier 3 (the tail). A declared axis value with no operation is
unbuilt, not wrong.

**Starter catalogue re-tiered.** The old rule — every treatment appears at least
once, in fifteen to twenty styles — became arithmetically false at 44 treatments.
Coverage is now Tier 1 (~15 styles, every treatment *family* represented), Tier 2
(~40, the showcase), Tier 3 (the tail).

**Surface catalogue expanded** from 11 to 35 records across five groups: web and
marketing, in-application (splash, installer, onboarding, empty and error states,
CLI banner), social and syndication (OG, repo preview, profile banner, post card,
email header), document and presentation (slides, covers, section headers), and
stores (Play, App Store, and Chrome Web Store).

**Two consistency repairs the expansion forced.** The `kind` enum gained `email`
and `document`, since neither previews in brand chrome nor in a store facsimile;
the underlying rule is now stated directly — product surfaces get brand-derived
chrome, every other kind gets a facsimile of its destination. And D-012 was
reconciled: it now governs **targets**, not records. Seeding a surface record is
data and costs nothing; committing a PRD target is a different act needing a real
consumer. The heading had asserted the opposite of the body.

**Unchanged:** no new PRD targets. The catalogue is illustrative, store
geometries remain `UNVERIFIED`, and `chrome.*` rows are seeded ahead of any
consumer — `scenario-to-extension` still declares no listing-asset target.


## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues and upstream dependencies
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and rationale
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — bounded contexts and build order
