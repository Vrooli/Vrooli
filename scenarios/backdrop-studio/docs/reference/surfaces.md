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

> **Reading the `Seeded` column.** It says whether this row exists as a record in
> `api/internal/catalog/seed/`, and it is the reconciliation this document owes
> its reader: every row is either seeded, or carries a reason it is not. A row
> marked *aspirational* is a description of the space, not a gap in the build —
> seeding a surface nothing renders into is inventory, and this scenario has
> already paid once for a catalog that named more than it drew.
>
> Compare this document against the running install with
> `backdrop-studio surfaces list --json`.

### Product · web and marketing — ours to choose

| id | Geometry | Purpose | Permitted placements | Seeded |
|---|---|---|---|---|
| `web.hero` | 1440x720 | Landing page hero | `full_bleed`, `split_panel`, `framed_inset`, `corner_bleed` | yes (v1) |
| `web.hero-mobile` | 390x844 | Hero at mobile viewport | `full_bleed`, `framed_inset` | yes (v1) |
| `web.auth-panel` | 640x900 | Sign-in / sign-up side panel | `split_panel`, `full_bleed` | yes (v1) |
| `web.section-band` | 1440x420 | Interstitial band between sections | `full_bleed`, `corner_bleed` | yes (v5) |
| `web.pricing-band` | 1440x520 | Backdrop behind a pricing table | `full_bleed`, `corner_bleed` | yes (v5) |
| `web.footer-wash` | 1440x360 | Quiet closing band | `corner_bleed`, `full_bleed` | yes (v5) |
| `web.error-page` | 1440x900 | 404 / 500 | `full_bleed`, `type_mask` | yes (v5) |

### Product · in-application — ours to choose

The largest latent slice by volume: every scenario in the portfolio has these
states. Read the admission-test caveat below before building — an empty-state
*illustration* is focal and out of scope; only the ambient wash behind one is in.

| id | Geometry | Purpose | Permitted placements | Seeded |
|---|---|---|---|---|
| `app.splash` | — | Desktop or mobile launch screen | `full_bleed` | aspirational |
| `app.installer-background` | — | Installer window or DMG background | `full_bleed`, `corner_bleed` | aspirational |
| `app.onboarding-panel` | — | First-run walkthrough panel | `split_panel`, `framed_inset` | aspirational |
| `app.empty-state` | — | Ambient wash behind an empty view | `framed_inset`, `corner_bleed` | aspirational |
| `app.error-state` | — | Ambient wash behind a failure view | `framed_inset`, `corner_bleed` | aspirational |
| `app.cli-banner` | — | Terminal splash — a genuine fit for `typographic_mosaic` | `caption_only` | aspirational |

**Why the whole slice is aspirational.** Every geometry here belongs to the
*consuming* scenario, not to this one: an installer background is whatever size
that installer's window is, and an empty-state wash is whatever the view is.
Seeding a guess would put a number in the registry with no authority behind it,
which is the one thing this registry exists to prevent. These rows land when a
consumer declares its geometry — and the admission test below is what decides
whether it should.

### Social and syndication — platform-shaped

Geometries here are conventions rather than hard requirements, but crops are
real: assume the centre may be all that survives.

| id | Geometry | Purpose | Permitted placements | Seeded |
|---|---|---|---|---|
| `social.og-card` | 1200x630 | Open-graph / link preview | `full_bleed`, `type_mask` | yes (v5) |
| `social.repo-preview` | 1280x640 | Repository social preview | `full_bleed`, `caption_only` | yes (v5) |
| `social.profile-banner` | 1500x500 | Profile header on a social platform | `full_bleed`, `corner_bleed` | yes (v5) |
| `social.post-card` | 1080x1350 | Portrait feed post | `full_bleed`, `caption_only` | yes (v5) |
| `email.header` | 600x240 | Marketing or transactional email header | `full_bleed`, `caption_only` | yes (v5) |

Only one of these five has a vendor behind it. `social.repo-preview` cites
[GitHub's own documentation](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/customizing-your-repositorys-social-media-preview),
which states 1280x640 recommended and 640x320 minimum. `social.og-card` cites
the 1.91:1 sharing convention and says so in its `authority` field — the Open
Graph protocol itself specifies no size, and a record claiming otherwise would be
a fabricated citation. The remaining three are conventions with no authority to
cite, and their records say that too. None of them is a submission target, so a
wrong number here costs a recrop rather than a rejection.

**Email is raster-only.** No CSS, no live export, no web fonts. A style destined
for `email.header` must survive being a flat image at a fixed width, which rules
out anything depending on `OT-P2-001` live export.

### Document and presentation — ours to choose

| id | Geometry | Purpose | Permitted placements | Seeded |
|---|---|---|---|---|
| `deck.title-slide` | — | Presentation title background | `full_bleed`, `caption_only` | aspirational |
| `deck.section-divider` | — | Slide section break | `full_bleed`, `type_mask` | aspirational |
| `doc.cover` | — | Report or PRD cover | `full_bleed`, `framed_inset` | aspirational |
| `doc.section-header` | — | Chapter or section banner | `corner_bleed`, `caption_only` | aspirational |

**Why this slice is aspirational.** `document-manager` owns the render toolchain
these would feed and has none built yet, so a seeded `doc.cover` would be a
surface with no consumer. The 16:9 and A4 geometries are not in doubt; the
consumer is.

### Store · app and extension — externally mandated

| id | Geometry | Purpose | Permitted placements | Seeded |
|---|---|---|---|---|
| `play.feature-graphic` | 1024x500 | Play listing banner | `feature_graphic` | yes (v1) |
| `play.phone-screenshot` | 1080x1920 | Play phone screenshot | `device_center`, `caption_above_device`, `caption_below_device`, `caption_only` | yes (v1) |
| `play.tablet-screenshot` | 1920x1080 | Play tablet screenshot | same as phone | yes (v1) |
| `app-store-6.7-screenshot` | 1290x2796 | App Store screenshot, primary iPhone class | same as phone | yes (v1) |
| `app-store-6.5-screenshot` | 1284x2778 | App Store screenshot, secondary iPhone class | same as phone | yes (v1) |
| `app-store-12.9-screenshot` | 2048x2732 | App Store screenshot, iPad class | same as phone | yes (v1) |
| `chrome.marquee` | — | Chrome Web Store marquee tile | `feature_graphic` | aspirational |
| `chrome.small-tile` | — | Chrome Web Store small promotional tile | `feature_graphic`, `caption_only` | aspirational |
| `chrome.screenshot` | — | Chrome Web Store screenshot | `device_center`, `caption_above_device` | aspirational |

### Verification status, 2026-08-12

Two store rows were checked against the authority their record cites, and both
are correct as seeded:

- `play.feature-graphic` at 1024×500, against
  [Play Console Help — Add preview assets](https://support.google.com/googleplay/android-developer/answer/9866151).
  Google requires exactly this size; there is no tolerance.
- `app-store-6.7-screenshot` at 1290×2796, against
  [App Store Connect Help — Screenshot specifications](https://developer.apple.com/help/app-store-connect/reference/app-information/screenshot-specifications/).

**The 6.7-inch class is no longer Apple's primary iPhone class.** The current
primary is 6.9-inch at 1320×2868; 1290×2796 is accepted as a fallback. This
catalog has no 6.9-inch surface, so an operator producing assets for the primary
class today has no record to render into. That is a gap in the seeded set rather
than a wrong number in it, and it is the reason this section exists: the device
classes are data, and Apple revises them as hardware ships.

`app-store-6.5-screenshot` (1284×2778), `app-store-12.9-screenshot`
(2048×2732), `play.phone-screenshot` and `play.tablet-screenshot` carry their
2026-08-11 stamp and were **not** re-checked today. Their `confirmed_on` dates
say when someone last looked, which is exactly what that field is for — do not
read a stamp as a guarantee that the vendor has not moved since.

**The App Store ids name the device class, not the marketing name.** This
document previously called them `appstore.iphone-primary` and its siblings while
the seeded records were `app-store-6.7-screenshot` and its siblings, so the two
disagreed on the identifier an operator would type. The seeded ids win, because
released assets reference them and a rename would break that reference — and
because the screen diagonal is the fact Apple's requirement is actually stated
in, while "primary iPhone class" is a fact that changes every autumn.

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
