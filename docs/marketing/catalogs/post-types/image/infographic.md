# Post Type: Infographic

**Status:** v0 (skeleton — strategic canon authored 2026-04-28).

**Paired skill:** `x-infographic` *(planned)*
**Primary lane/member:** Subscription lane (`subscription-advertiser`) for lifestyle/general subjects; OSS lane (`oss-advertiser`) for technical subjects.
**Notebook home for emerging craft patterns:** `../../../notebook/INFOGRAPHIC_CRAFT.md` (created on first observation)

## Purpose

An infographic is **one image that compresses data, comparison, or process into a visual that reads faster than the equivalent text**. The reader gets the gist in 2-5 seconds and decides whether to engage further. Distinct from `single-image-ad.md` (no data compression — just a stylized hook image) and from `slideshow-listicle.md` (multi-frame, list-shaped).

Use cases at Vrooli:

- Architecture diagram condensed into one image (which scenarios depend on which resources).
- Comparison chart (Vrooli vs alternative on N axes).
- Pricing-tier visualization.
- Workflow / data-flow diagram.

## Audience

Primary: dev-tool audience for technical infographics; subscription-buyer for product-comparison infographics. Different shape per audience: dev audience tolerates density; lifestyle audience demands generous whitespace.

## Conversion goal

- **Save / share** is the primary signal — infographics convert by being saved as a reference and shared as evidence.
- **Click-through** secondary, via caption + link-in-bio.
- **Reading-time anchor:** infographics that get studied (not just scrolled past) compound reach in algorithm-fed surfaces.

## Structure

Infographics fail when they try to be slideshows compressed into one image. Discipline:

- **One central message.** If asked "what is this image showing?" the answer should be one sentence.
- **Hierarchy.** Eye should land on the title, then the main visual, then supporting detail. Not all-at-once.
- **Honest data presentation.** Per `STRATEGY.md`'s source-material discipline — every numeric claim is `measured` or carries its honesty flag visibly. Charts that mislead via axis-truncation or scale-skew fail the contrarian.

## Asset requirements

- **Single image.** Format depends on channel:
  - LinkedIn: 1200×627 or 1200×1500.
  - Instagram Feed: 1080×1350 (portrait works for dense content).
  - X with attached image: 1200×675.
  - Blog embed: any reasonable format; use the original full-resolution asset.
- **Brand assets** per [`../../../strategy/ASSETS.md`](../../../strategy/ASSETS.md), [`../../../strategy/IMAGE_STYLE.md`](../../../strategy/IMAGE_STYLE.md). Infographics often deviate slightly from `IMAGE_STYLE.md` because data-viz palettes need additional non-brand colors for clarity; the deviation must be intentional and documented in the draft.
- **Not generated end-to-end by image-gen.** Infographics typically use a layout tool (Figma, slide template) rather than diffusion image-gen, because text in generated images is unreliable and infographics depend on text accuracy.

## Voice

The infographic itself carries little voice — voice lives in the caption. Caption follows `STRATEGY.md` voice canon. Brand-voice infographics dominate; persona-voice rare.

## Contrarian failure modes (infographic-specific specializations)

| Type-level failure mode | What it looks like | Framework-level mode |
|---|---|---|
| **Data-honesty violation** | Chart axes truncated or skewed to overstate the differentiator; numbers without sources. | mode 3 (hallucinated engagement metrics) — extended to any numerical claim. |
| **Density-overload** | Image is busy beyond the audience's tolerance; nothing reads. | mode 9 (narrative-flatness) — visual narrative collapses. |
| **Brand-asset drift** | Off-palette without explicit data-viz justification. | brand-canon violation. |
| **Comparison-without-genuine-differentiator** | Comparison chart is a strawman. | mode 17 (recommendation-framing-without-basis) generalized + see [`../../../methods/post-techniques/competitive-comparison.md`](../../../methods/post-techniques/competitive-comparison.md). |
| **Stale-data** | Infographic reflects state from N months ago, no longer accurate. | mode 1 (hype-drift) when the staleness inflates current claims. |

Honesty flags:

- `data_source=measured | unverified-third-party-claim | aspirational | pending-telemetry` for every number on the image.
- `freshness_date=<YYYY-MM-DD>` — when the data was captured.
- `comparison_basis=measured | partial | aspirational` — when the infographic compares against a named alternative.

## Cross-cutting techniques this type uses

- [Competitive comparison](../../../methods/post-techniques/competitive-comparison.md) — when the infographic compares.
- [Hook-vs-body length asymmetry](../../../methods/post-techniques/hook-vs-body-asymmetry.md) — title is hook; visual is body.
- [No internal numbering externally](../../../methods/post-techniques/no-internal-numbering-externally.md).

## Cross-references

- [`single-image-ad.md`](single-image-ad.md) — sibling format without data compression.
- [`../../../methods/post-techniques/competitive-comparison.md`](../../../methods/post-techniques/competitive-comparison.md) — applies when the infographic is comparison-shaped.
- [`../../../strategy/ASSETS.md`](../../../strategy/ASSETS.md), [`../../../strategy/IMAGE_STYLE.md`](../../../strategy/IMAGE_STYLE.md).
