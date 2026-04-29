# Post Type: Slideshow Listicle

**Status:** v0 (skeleton — strategic canon authored 2026-04-28).

**Paired skill:** `x-slideshow-listicle` *(planned)*
**Primary author:** `subscription-advertiser`
**Notebook home for emerging craft patterns:** `../../notebook/SLIDESHOW_LISTICLE_CRAFT.md` (created on first observation)

## Purpose

A slideshow listicle is a **numbered series of frames (typically 5-10)** delivering a list — "5 ways to keep your house organized", "7 dev tools that saved my week", "10 reasons your terminal is slow". The reader swipes through; each frame delivers one item.

Distinct from `slideshow-tips-then-plug.md`: a listicle is a *list*, not a *tips → product-plug arc*. The listicle's last frame may include a CTA, but the structure is symmetric (each item is roughly equal weight); the tips-then-plug shape is asymmetric (early frames generic, final frame plugs the product).

## Audience

Primary: visual-feed audience (Instagram Feed, TikTok carousel, LinkedIn carousel) that consumes content via swipe rather than read. Listicles work because the format itself (numbered, scrollable, low-commitment) lowers the cost of starting.

Specific personas: subscription-buyer in the *lifestyle* category for general-audience listicles; *dev-tool* category for developer-tool listicles.

## Conversion goal

- **Save / share** as the primary engagement signal. Listicles convert by being shared.
- **Click-through** secondary, via a profile-link / link-in-bio rather than per-frame CTA.
- Sign-up is far downstream; do not push it.

## Structure

- **Cover frame:** the hook. The numbered count and the topic ("7 ways to ...") set expectations. The cover is the entire decision — readers swipe in or scroll past based on this single frame.
- **Item frames (one per number):** each frame delivers one item. Compact text (≤25 words). Visual continuity across frames is required (same character if a person is shown, same scene type, consistent color treatment).
- **Closer frame:** invitation. "Save this post / Follow for more / Link in bio." Not a hard sell.
- **Frame count:** 5-10 typical. Below 5 reads as thin; above 10 loses the swipe momentum.

## Asset requirements

- **Multi-frame consistency.** The single largest production risk: visual drift across frames. The rich-media schema at [`../../rich-media/`](../../rich-media/) is the substrate that prevents this — character/scene/product JSON keeps the visual identity locked frame-to-frame.
- **Per-frame layout consistency.** Same headline placement, same number-badge style, same background palette. Drift here reads as low-effort.
- **Brand assets** per [`../../ASSETS.md`](../../ASSETS.md) and [`../../IMAGE_STYLE.md`](../../IMAGE_STYLE.md).

## Voice

Listicles can be brand-voice or persona-voice. Brand-voice listicles follow `STRATEGY.md` voice canon. Persona-voice listicles follow [`../../strategies/ai-ugc-personas.md`](../../strategies/ai-ugc-personas.md) — including disclosure rules per channel.

Listicle voice across frames must stay locked. A frame-1 voice that drifts by frame-5 reads as machine-generated and fails.

## Contrarian failure modes (slideshow-listicle-specific specializations)

| Type-level failure mode | What it looks like | Framework-level mode |
|---|---|---|
| **Asset-style-drift-across-frames** | Visual identity, character, scene, or palette shifts noticeably between frames. | mode 9 (narrative-flatness, here at the *visual narrative* layer). |
| **Numerical-mismatch** | Cover promises N items; the slideshow delivers a different number. | mode 1 (hype-drift). |
| **Padded-list** | Items 6-10 are weaker than items 1-5; the count was inflated to look comprehensive. | mode 12 (what-without-why) — items without distinct why-it-mattered framing. |
| **Voice-drift-across-frames** | Voice changes between frames (frame 1 is conversational, frame 5 is corporate). | mode 2 (voice drift). |
| **Persona-disclosure-violation** | Persona-account listicle without platform AI-content label. | mode 13. |
| **CTA-blur** | Closer frame mixes save / share / click / sign-up; reader doesn't know what to do. | mode 7 (acquisition-only) — variation: blurred-acquisition. |

Honesty flags:

- `feature_claims=measured | overclaimed` — when the list mentions Vrooli scenarios, claims must hold.
- `persona_disclosure=labeled | exempt-minor-edit-only` (when persona-voice).
- `count_promised_matches_count_delivered=YES | NO` — must be YES.

## Cross-cutting techniques this type uses

- [Hook-vs-body length asymmetry](../../post-techniques/hook-vs-body-asymmetry.md) — cover frame is the hook; item frames are the body.
- [No internal numbering externally](../../post-techniques/no-internal-numbering-externally.md).

## Cross-references

- [`slideshow-tips-then-plug.md`](slideshow-tips-then-plug.md) — sibling format with asymmetric tips-then-plug arc.
- [`../../strategies/ai-ugc-personas.md`](../../strategies/ai-ugc-personas.md) — when persona-voice is used.
- [`../../rich-media/`](../../rich-media/) — multi-frame consistency substrate.
- [`../../strategies/hook-library.md`](../../strategies/hook-library.md) — listicle cover hooks are a populated category.
