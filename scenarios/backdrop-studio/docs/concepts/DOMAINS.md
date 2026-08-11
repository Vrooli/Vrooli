# Domains

Eight bounded contexts. The dependency chain decides the build order: a domain may
read the domains above it and must not read the domains below it.

## Domain Inventory

| # | Domain | Source Paths | Owns | Reads |
|---|---|---|---|---|
| 1 | `surfaces` | `api/handlers/surfaces`, `api/internal/catalog` | Output surface records, declared pixel geometry, permitted placements, geometry authorities | — |
| 2 | `catalog` | `api/handlers/catalog`, `api/internal/catalog` | Style records, five-axis classification, strategy declaration, reserved-region geometry, versioning, remix lineage | `surfaces` |
| 3 | `scaffold` | `api/handlers/scaffold`, `api/internal/scaffold` | Procedural scene generators, composition scaffold presets, conditioning-image rendering | `catalog` |
| 4 | `compose` | `api/handlers/compose`, `api/internal/compose` | Style + brief → inspectable execution plan; palette-slot resolution; device-frame composition; licensing gate | `catalog`, `scaffold`, `surfaces` |
| 5 | `render` | `api/handlers/render`, `api/internal/render` | Job lifecycle, candidate sets, selection, execution-path recording | `compose` |
| 6 | `legibility` | `api/handlers/legibility`, `api/internal/legibility` | Overlay-region measurement, worst-pixel contrast, verdicts, scrim solving | `render`, `catalog` |
| 7 | `release` | `api/handlers/release`, `api/internal/release` | Disclosure derivation, geometry conformance, `asset-studio` handoff, consumer reference surface | `render`, `legibility`, `surfaces` |
| 8 | `workbench` | `ui/src/pages`, `ui/src/features` | Operator surface: axis browsing, dual-view judgement, mockup chrome fidelity, accessibility obligations | all of the above |

`surfaces` sits above `catalog` because a style's reserved regions are expressed
against a surface geometry. A style that declares no surface has nowhere to put a
reserved region, and a reserved region with no geometry cannot be measured.

---

## 1 · `surfaces`

**Where externally mandated geometry lives, so it is never a constant in code.**

A surface is a named output target carrying exact pixel dimensions, the
placements it permits, and a citation for where its geometry comes
from (`SUR-001`). Web surfaces are ours to choose. Store surfaces are not — a
Play feature graphic and an App Store screenshot are dictated by their stores and
change without asking us.

Two consequences shape the design:

- **Geometry is data with a cited authority and a confirmation date** (`SUR-004`).
  A store requirement that moved is a data update, not a patch release.
- **Placements are constrained per surface** (`SUR-002`). A caption-above-device
  arrangement is meaningful on a phone screenshot and meaningless on a feature
  graphic, and the surface record is where that belongs.

This domain is what lets one style serve a landing page and an app listing
without either use case knowing about the other.

## 2 · `catalog`

**The vocabulary layer.** Everything else is specified against it, which is why
it is built first and why it is worth over-investing in.

A **Style** is the unit. It carries:

- the **five axes** — `role`, `subject`, `treatment`, `lineage`, `placement`
- exactly one **strategy** — `procedural` | `procedural-treated` | `guided` | `synthesized`
- a **scaffold binding** (guided only) — preset id, parameters, conditioner kind
- a **generation block** (model-backed only) — `ai-gateway` role, routing profile, prompt template, negative prompt
- a **treatment chain** — ordered operations with parameters, where a colour may be a `$brand.*` slot
- **reserved regions** — proportional rectangles, each declaring overlay or occlusion
- **gates** — the contrast threshold and scrim policy
- a **lineage reference** — the style version this was forked from

The axis enums are the taxonomy. Their canonical definitions live in
`../reference/taxonomy.md`; the catalog validates against them (`CAT-001`).

**Invariants.** A style declares one strategy and carries only the fields that
strategy permits (`CAT-002`). A style version referenced by a released backdrop
is immutable (`CAT-005`) — provenance recorded against released work must keep
meaning.

## 3 · `scaffold`

**Where content lives, as opposed to verbs.** A treatment like halftone is a
generic verb and belongs in `image-tools`. A generator that draws a classical
arcade is *content* — no other scenario wants it — so it lives here. That split
is the ownership rule applied cleanly, and it is the answer whenever a new
drawing capability appears.

Two responsibilities:

- **Procedural scene generators.** Seeded, deterministic, no wall-clock or
  unseeded randomness (`SCAF-001`). These produce the base image for the two
  code-only strategies.
- **Composition scaffolds.** Parameterised presets (`SCAF-004`) rendered into a
  conditioning image — a depth field or an edge drawing — with each reserved
  region drawn as a flat featureless area (`SCAF-003`).

## 4 · `compose`

**The pure function at the centre of the scenario.** Given a style and a brief,
it returns an explicit plan: strategy, ordered operations, fully merged
parameters, conditioning inputs, expected execution path. It executes nothing
(`CMP-001`).

This mirrors `image-tools`' own `resolver` → `ExplainResolution` pattern
deliberately. A caller can read back exactly what *would* run before anything
runs, which makes the whole pipeline debuggable without spending money.

It also owns two refusals that must happen *before* execution:

- an unresolved `$brand.*` slot (`CMP-003`) — never silently defaulted, because a
  substituted colour defeats the palette lock the treatment layer exists for
- an adapter whose licence forbids commercial use (`CMP-006`) — evaluated at
  composition time so spend is not incurred against work that cannot ship

## 5 · `render`

**Execution and its record.** Submits the plan through the `image-tools` seam,
tracks the job lifecycle, produces candidate sets, and requires an explicit
selection before anything proceeds (`RND-003`) — including when the set holds
exactly one, so the selection record always names who chose.

Holds the invariant that every strategy terminates in the treatment chain
(`RND-001`), and records which execution path `image-tools` took (`RND-004`).

## 6 · `legibility`

**The gate that separates a picture from a backdrop.** Computes the WCAG
contrast ratio between the intended text colour and every pixel inside each
overlay region, and reports the **minimum** (`LEG-001`).

The minimum, not a mean — this is the whole point of the domain. A single bright
area behind one word makes a headline unreadable while an average stays
comfortable. Averages are why beautiful hero imagery ships broken.

Two correctness details worth their own requirements: luminance must be computed
from linearised sRGB (`LEG-002`), because a ratio derived from raw channel values
is wrong in a way that passes casual inspection; and measurement happens against
the region *as the placement crops it* (`LEG-004`), so a backdrop that passes as
a desktop hero and fails as a mobile crop is reported as failing for the crop.

## 7 · `release`

**The handoff, and the only place the `asset-studio` boundary is crossed.**

The split is by cost, not by convenience:

- **Model-backed candidates** release through `asset-studio` (`REL-002`), which
  already owns provenance, spend, and disclosure. No second provenance record is
  written here.
- **Procedural candidates** release locally (`REL-003`). They incur no spend,
  carry no disclosure obligation, and reproduce from a seed, so routing them
  through a cost-and-disclosure ledger would add a dependency that buys nothing —
  and would make the product unusable whenever `asset-studio` is down.

Also owns disclosure derivation (`REL-001`), the geometry conformance gate
(`SUR-003`), and the consumer reference surface, which serves metadata and never
bytes (`REL-004`).

---


## 8 · `workbench`

**The operator surface, and the one place judgement actually happens.**

It exists as its own context because two of its obligations are product
invariants rather than presentation choices:

- **Dual-view judgement** (`UIX-007`). Every candidate is shown standalone *and*
  composed into a mockup of its target surface with foreground content in place.
  An ambient backdrop judged alone is judged against the wrong question.
- **Chrome fidelity is chosen by surface kind, not by preference.** A product
  surface previews in chrome derived from the target scenario's design tokens
  (`UIX-008`); a store surface previews inside a facsimile of the destination
  store (`UIX-009`). The judgements differ — "does this belong to our product"
  versus "does this hold up next to competing listings" — so the mockups must.

Also owns the accessibility obligations a contrast tool cannot itself violate
(`UIX-003`) and the achromatic-chrome rule that keeps the specimen the only
saturated thing on screen (`UIX-006`).

---

## Build order

Each step is independently useful, which is deliberate — the scenario should be
worth running before it is finished.

1. **`surfaces` + `catalog`** — the taxonomy and the web surface geometries exist
   and are queryable. Nothing renders yet.
2. **`scaffold` + `compose` + `render`, procedural lanes only** — a complete,
   useful product with **zero model dependency**. Works offline, costs nothing.
3. **`legibility` + `release` + dual-view `workbench`** — backdrops become safe to
   consume and judgeable in context; first integration into
   `landing-page-business-suite`.
4. **`guided` and `synthesized` lanes** — model-backed generation with
   `asset-studio` release and derived disclosure.
5. **Store surfaces** — the store surface pack, device-frame composition, and
   platform-accurate preview; first integration into `scenario-to-android` and
   `scenario-to-ios`, whose listing-asset targets have no producer today.
6. **Workbench depth** — contact sheet, brand-themed chrome, remix.

Step 2 is the milestone that matters. If the procedural lanes work end to end,
the architecture is proven and everything after it is addition rather than risk.
