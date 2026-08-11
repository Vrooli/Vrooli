# Output surfaces

A **surface** is a named output target. It carries exact pixel geometry, the
placements it permits, and a citation for where that geometry came from
(`SUR-001`).

Surfaces exist as their own registry for one reason: **some geometry is ours and
some is not.** A hero image is whatever width we decide. A Play feature graphic
is whatever Google says it is this quarter. Holding both as data with a cited
authority means a changed store requirement is a data update rather than a patch
release.

## Record shape

```jsonc
{
  "id": "play.feature-graphic",
  "kind": "store",                    // product | store | social | email | document
  "geometry": { "width": 1024, "height": 500 },
  "placements": ["feature_graphic"],  // intersected with the style's own set
  "authority": {
    "name": "Google Play Console Help — Graphic assets",
    "url": "https://support.google.com/googleplay/android-developer/answer/9866151",
    "confirmed_on": "UNVERIFIED"      // ISO date once a human has checked
  },
  "notes": "Play overlays its own furniture and may crop on some layouts; keep the centre quiet."
}
```

## The `kind` field decides the mockup

This is the field that routes preview fidelity, and it is the only thing that
should (`UIX-008`, `UIX-009`). The rule underneath the table is simple:
**a `product` surface previews in brand-derived chrome; every other kind previews
in a facsimile of its destination.**

| `kind` | Mockup chrome | The judgement being made |
|---|---|---|
| `product` | Derived from the target scenario's design tokens — type scale, control shapes, radii, spacing, neutrals | *Does this belong to our product?* |
| `store` | A facsimile of the destination store listing | *Does this hold up against that store's furniture and its neighbours?* |
| `social` | The destination platform's feed card and crop | *Does this survive the crop and the feed around it?* |
| `email` | A mail-client reading pane at a constrained width | *Does this hold up as a flat raster in a cramped column?* |
| `document` | A page or slide with surrounding body content | *Does this sit under real text without competing with it?* |

A product mockup draws an *impression* of the target, never its actual
components. The goal is "this could plausibly be that product," not a live embed
— importing real components would make this scenario depend on every scenario it
previews.

## Seed set

> **Two different kinds of uncertainty below, and they are not interchangeable.**
>
> **Product geometries are proposals.** They are ours to choose; change them
> freely to suit the design.
>
> **Store geometries are UNVERIFIED external facts.** Every store row must be
> checked against its cited authority and stamped with a `confirmed_on` date
> before it is used to produce a submitted asset (`SUR-004`). A wrong number is
> rejected at submission, the most expensive place to find it.
>
> **This catalogue is illustrative, not a commitment.** It shows the space this
> scenario is built to cover, so an implementing agent seeds broadly and aligns
> to the right shape. Seeding a surface record is data and costs nothing;
> committing a PRD target is not the same act (D-012). Nothing here is required
> to ship, and the list is expected to grow.

### Product · web and marketing — ours to choose

| id | Purpose | Permitted placements |
|---|---|---|
| `web.hero` | Landing page hero | `full_bleed`, `split_panel`, `framed_inset`, `corner_bleed` |
| `web.hero-mobile` | Hero at mobile viewport | `full_bleed`, `framed_inset` |
| `web.auth-panel` | Sign-in / sign-up side panel | `split_panel`, `full_bleed` |
| `web.section-band` | Interstitial band between sections | `full_bleed`, `corner_bleed` |
| `web.pricing-band` | Backdrop behind a pricing table | `full_bleed`, `corner_bleed` |
| `web.footer-wash` | Quiet closing band | `corner_bleed`, `full_bleed` |
| `web.error-page` | 404 / 500 | `full_bleed`, `type_mask` |

### Product · in-application — ours to choose

The largest latent slice by volume: every scenario in the portfolio has these
states. Read the admission-test caveat below before building — an empty-state
*illustration* is focal and out of scope; only the ambient wash behind one is in.

| id | Purpose | Permitted placements |
|---|---|---|
| `app.splash` | Desktop or mobile launch screen | `full_bleed` |
| `app.installer-background` | Installer window or DMG background | `full_bleed`, `corner_bleed` |
| `app.onboarding-panel` | First-run walkthrough panel | `split_panel`, `framed_inset` |
| `app.empty-state` | Ambient wash behind an empty view | `framed_inset`, `corner_bleed` |
| `app.error-state` | Ambient wash behind a failure view | `framed_inset`, `corner_bleed` |
| `app.cli-banner` | Terminal splash — a genuine fit for `typographic_mosaic` | `caption_only` |

### Social and syndication — platform-shaped

Geometries here are conventions rather than hard requirements, but crops are
real: assume the centre may be all that survives.

| id | Purpose | Permitted placements |
|---|---|---|
| `social.og-card` | Open-graph / link preview | `full_bleed`, `type_mask` |
| `social.repo-preview` | Repository social preview | `full_bleed`, `caption_only` |
| `social.profile-banner` | Profile header on a social platform | `full_bleed`, `corner_bleed` |
| `social.post-card` | Square or portrait feed post | `full_bleed`, `caption_only` |
| `email.header` | Marketing or transactional email header | `full_bleed`, `caption_only` |

**Email is raster-only.** No CSS, no live export, no web fonts. A style destined
for `email.header` must survive being a flat image at a fixed width, which rules
out anything depending on `OT-P2-001` live export.

### Document and presentation — ours to choose

| id | Purpose | Permitted placements |
|---|---|---|
| `deck.title-slide` | Presentation title background | `full_bleed`, `caption_only` |
| `deck.section-divider` | Slide section break | `full_bleed`, `type_mask` |
| `doc.cover` | Report or PRD cover | `full_bleed`, `framed_inset` |
| `doc.section-header` | Chapter or section banner | `corner_bleed`, `caption_only` |

### Store · app and extension — externally mandated

| id | Purpose | Permitted placements |
|---|---|---|
| `play.feature-graphic` | Play listing banner | `feature_graphic` |
| `play.phone-screenshot` | Play phone screenshot | `device_center`, `caption_above_device`, `caption_below_device`, `caption_only` |
| `play.tablet-screenshot` | Play tablet screenshot | same as phone |
| `appstore.iphone-primary` | App Store screenshot, primary iPhone class | same as phone |
| `appstore.iphone-secondary` | App Store screenshot, secondary iPhone class | same as phone |
| `appstore.ipad` | App Store screenshot, iPad class | same as phone |
| `chrome.marquee` | Chrome Web Store marquee tile | `feature_graphic` |
| `chrome.small-tile` | Chrome Web Store small promotional tile | `feature_graphic`, `caption_only` |
| `chrome.screenshot` | Chrome Web Store screenshot | `device_center`, `caption_above_device` |

Apple and Google both revise their required device classes as hardware ships.
Treat the class list itself as data too — adding a class must not require a code
change. The `chrome.*` rows are seeded ahead of a consumer: `scenario-to-extension`
declares no listing-asset target today, so these stay unbuilt until it does.

## Admitting a new surface

Adding a surface is **data, not architecture** — a record, sometimes a placement
enum value. That cheapness is the point of this domain, and it is also the risk:
a registry that admits anything stops meaning anything.

A candidate qualifies when all four hold:

1. **The imagery is `ambient`.** It sits behind or beside foreground content. If
   the image *is* the message — a logo, an avatar, an illustration explaining a
   concept, a chart — it is `focal` or `evidential` and belongs elsewhere.
2. **The geometry is fixed and knowable.** A surface with no declared dimensions
   cannot be conformance-gated (`SUR-003`), which is most of what a surface is
   for.
3. **The foreground is expressible as reserved regions.** Something must be
   overlaid or occluded, or there is no layout judgement to make and any image
   would do.
4. **Somebody will produce against it.** A surface record may be *seeded*
   speculatively — it is data, and the catalogue above does exactly that. What
   requires a real consumer is a **PRD target or a requirement** committing to
   build for it. See D-012; the distinction is the whole of that decision.

### Worked examples

| Candidate | Verdict | Why |
|---|---|---|
| Landing page hero, auth panel, section band | **in registry** | The original case |
| Open-graph / share card | **in registry** | Fixed geometry, title overlaid, ambient |
| App-store listing assets | **in registry** | `scenario-to-android` / `-ios` declared the need |
| Extension store promotional tiles | **qualifies** | Structurally identical to the app-store case. `scenario-to-extension` does not declare listing assets today; if it gains that target, this is a surface record and nothing more |
| Desktop splash / installer background | **qualifies** | Fixed geometry, logo as an occlusion region |
| Email header | **qualifies** | Fixed width, headline overlaid. Raster only — no CSS, no live export |
| Slide / deck background | **qualifies** | Title and body as overlay regions |
| Repository social preview | **qualifies** | Fixed geometry, ambient, trivial to add |
| In-product empty, error and onboarding states | **qualifies, with care** | The largest latent use by volume — every scenario has these. But an empty-state *illustration* is usually focal; only the ambient wash behind one qualifies. Draw that line before building, not after |
| Print collateral — cards, one-pagers, posters | **needs work first** | Conceptually the best fit here, since halftone and riso *are* print processes. But the pipeline is sRGB and the gate is WCAG, which is a screen standard. CMYK, bleed and DPI are real work, and contrast under ink is not the same measurement. Do not assume this is free |
| Avatars, logos, app icons | **does not qualify** | Identity-bearing and focal. `brand-manager` and `asset-studio` own these |
| Charts, dashboards, data imagery | **does not qualify** | Evidential. `chart-generator` owns it |
| Per-user personalised imagery | **does not qualify** | A different product with different privacy, cost and caching properties |

## What this scenario does not do

Backdrop Studio supplies the backdrop, the placement composition, and the
geometry. It does **not** capture the application screenshot that sits inside a
device frame — `scenario-to-android` and `scenario-to-ios` already produce those
from journey evidence, and it does not submit anything to a store. The screenshot
is an input this scenario receives.

## Related

- [`taxonomy.md`](taxonomy.md) — the five axes, including the placement enums
  referenced above
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) §1 — the `surfaces` context
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) D-010, D-011
