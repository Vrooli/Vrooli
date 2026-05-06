# Post Type: Single-Image Ad

**Status:** v0 (skeleton — strategic canon authored 2026-04-28; will harden as the type runs in production and the marketing notebook accumulates entries).

**Paired skill:** `x-single-image-ad` *(planned)*
**Primary lane/member:** Usually subscription lane (`subscription-advertiser`) for drafting, with publisher/operator handoff for posting.
**Notebook home for emerging craft patterns:** `../../notebook/SINGLE_IMAGE_AD_CRAFT.md` (created on first observation)

## Purpose

A single-image ad is **one stylized still image plus a short caption**, designed to stop a scroll on a visual feed (Instagram, LinkedIn, X-with-image, paid placements). The image carries the hook; the caption carries the conversion-rung. Distinct from `infographic.md` (data-heavy single image — different production cost, different reader contract) and `slideshow-*` (multi-frame).

It is **not**:

- A scenario-spotlight (those are text-led with attached video — see [`../text/scenario-spotlight.md`](../text/scenario-spotlight.md)).
- An infographic (those compress data into one image — see [`infographic.md`](infographic.md)).
- A slideshow (those carry the reader through multiple frames — see [`slideshow-listicle.md`](slideshow-listicle.md), [`slideshow-tips-then-plug.md`](slideshow-tips-then-plug.md)).

## Audience

Primary: subscription-buyer persona who scrolls visual feeds (Instagram Feed, LinkedIn, paid placements). The audience does not stop unless the image lands in the first half-second.

Secondary, for OSS-bundle-adjacent content: the dev/builder persona on X (image attached to a hook tweet).

## Conversion goal

- **Click-through.** The image's job is to stop the scroll and earn a tap on the caption or link. Sign-up is downstream of click-through; do not try to convert in one frame.
- A single-image ad is **not** the place for sign-up CTAs that demand multi-step commitment. The reader has consumed roughly one second; respect that.

## Structure

- **Image carries the hook.** A frame that says nothing in 0.5s is dead. Strong patterns: one bold contrast (object vs background), one specific moment of recognition (someone doing something the audience does), or one bold textual hook overlaid (≤6 words).
- **Caption carries the next step.** One line of context, one CTA. Don't recap the image; advance from it.
- **Brand consistency** with [`../../ASSETS.md`](../../ASSETS.md) and [`../../IMAGE_STYLE.md`](../../IMAGE_STYLE.md) is mandatory. Logos, palette, OG-image style.

Per-channel size targets:

- Instagram Feed: 1080×1080 (square) or 1080×1350 (portrait).
- LinkedIn: 1200×627 (landscape).
- X with attached image: 1200×675 (landscape).
- Paid placements: per-vendor specs.

## Asset requirements

- **One image.** Not a series. If the message requires multiple frames, this is the wrong post type.
- **Generation source:** the rich-media schema at [`../../rich-media/`](../../rich-media/) drives generation. If the image features a person/character, the character JSON in [`../../rich-media/characters/`](../../rich-media/characters/) is the substrate. Disclosure rules in [`../../strategies/ai-ugc-personas.md`](../../strategies/ai-ugc-personas.md) apply if the image is AI-generated and substantial.
- **Brand consistency check:** the asset must be checkable against `IMAGE_STYLE.md` palette and the brand asset registry at `ASSETS.md`.

## Voice

Image-led posts have voice in **two surfaces**: the image overlay text (if any) and the caption. Both follow `STRATEGY.md` voice canon — builder-not-marketer applies to brand-voice content; persona-voice rules in `ai-ugc-personas.md` apply to persona-account content.

## Contrarian failure modes (single-image-ad-specific specializations)

| Type-level failure mode | What it looks like | Framework-level mode |
|---|---|---|
| **Dead-frame** | The image carries no recognizable hook in 0.5s; the reader scrolls past before reading. | mode 9 (narrative-flatness) — applied to a single frame. |
| **Brand-asset drift** | Image uses outdated logo, off-palette colors, or pre-canon style. | mode 13 (persona-disclosure-violation) when AI-generated, plus brand-canon violation per `ASSETS.md`. |
| **Caption recapping the image** | Caption restates what the image already shows; wastes the next-step opportunity. | mode 12 (what-without-why) — variation: image is the *what*, caption must be the *why* / next step. |
| **Hook over-promising** | Image text or implied claim overpromises a feature/result. | mode 1 (hype-drift). |
| **Persona-disclosure-violation** | AI-generated image with a person, posted without platform AI-content label or `#ad` for sponsorship. | mode 13. |
| **Real-person-impersonation** | Generated image resembles a specific identifiable real person. | mode 15. |

Honesty flags for a single-image-ad draft:

- `feature_claims=measured | overclaimed | uncertain` — claims in image overlay or caption.
- `persona_disclosure=labeled | unlabeled | exempt-minor-edit-only` (when image features AI-generated person).
- `tier_alignment=verified | not-yet-checked` — implied features must match the CTA's tier.

## Cross-cutting techniques this type uses

- [Hook-vs-body length asymmetry](../../post-techniques/hook-vs-body-asymmetry.md) — image is the hook (extreme asymmetry); caption is the body.
- [No internal numbering externally](../../post-techniques/no-internal-numbering-externally.md) — no `p8` / `round-002` in image or caption.
- [Recommendation framing](../../post-techniques/recommendation-framing.md) — when applicable.

## Cross-references

- [`../../strategies/ai-ugc-personas.md`](../../strategies/ai-ugc-personas.md) — disclosure rules for AI-generated images.
- [`../../rich-media/characters/`](../../rich-media/characters/), [`../../rich-media/scenes/`](../../rich-media/scenes/), [`../../rich-media/products/`](../../rich-media/products/), [`../../rich-media/templates/image-prompt.template.json`](../../rich-media/templates/image-prompt.template.json) — generation substrate.
- [`../../ASSETS.md`](../../ASSETS.md), [`../../IMAGE_STYLE.md`](../../IMAGE_STYLE.md) — brand canon.
- [`../../CHANNELS.md`](../../CHANNELS.md) — per-channel size targets and disclosure expectations.
- Sibling: [`infographic.md`](infographic.md) (data-heavy), [`slideshow-listicle.md`](slideshow-listicle.md), [`slideshow-tips-then-plug.md`](slideshow-tips-then-plug.md).
