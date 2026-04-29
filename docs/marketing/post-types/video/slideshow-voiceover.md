# Post Type: Slideshow Voiceover

**Status:** v0 (skeleton — strategic canon authored 2026-04-28).

**Paired skill:** `x-slideshow-voiceover` *(planned)*
**Primary author:** `subscription-advertiser`
**Notebook home for emerging craft patterns:** `../../notebook/SLIDESHOW_VOICEOVER_CRAFT.md` (created on first observation)

## Purpose

A slideshow voiceover is the **video rendering of a slideshow with TTS or voice-model narration synced to frame transitions**. The same arc as the image-medium [`../image/slideshow-tips-then-plug.md`](../image/slideshow-tips-then-plug.md) or [`../image/slideshow-listicle.md`](../image/slideshow-listicle.md), but pushed onto TikTok / Reels / Shorts surfaces where text-only carousels don't perform as well.

Use this format when the source content is already authored as a slideshow (image medium) and a video variant is wanted for short-video surfaces, or when the topic suits voice narration better than per-frame reading.

## Audience

Primary: same as the image-medium siblings — lifestyle bundle on short-video surfaces. Persona voice typical.

## Conversion goal

Same as image-medium siblings: click-through to link-in-bio; save/share secondary.

## Structure

Inherits the slideshow arc from the image-medium sibling chosen as the source:

- For tips-then-plug arc: see [`../image/slideshow-tips-then-plug.md`](../image/slideshow-tips-then-plug.md) for the structure.
- For listicle arc: see [`../image/slideshow-listicle.md`](../image/slideshow-listicle.md).

Video-specific additions:

- **Frame timing.** Each slide gets 4-6 seconds; transitions match voiceover pacing.
- **Voiceover sync.** TTS or voice-model narration locked to frame; no off-by-one drift.
- **First frame is the entire hook.** Cover frame must land in 1.5s; voiceover opens with the hook line.
- **Captions burned in.**

Length: 30-45s typical.

## Asset requirements

- **Source slideshow assets** (the per-frame images) generated via the rich-media schema at [`../../rich-media/`](../../rich-media/).
- **Voice profile** in the persona's character JSON drives the voiceover. Consistent voice across the persona's content slate.
- **Vertical 9:16.**
- **Disclosure label.**

## Voice

Persona voice typical. Brand voice possible for technical-listicle slideshow voiceovers (rare).

## Contrarian failure modes

Inherits all the image-medium sibling's failure modes, plus video-specific:

| Type-level failure mode | What it looks like | Framework-level mode |
|---|---|---|
| **Voice-frame-drift** | Voiceover doesn't match the on-frame text or topic; audience confused. | mode 9 specialization. |
| **TTS-uncanny-cadence** | Voice-model output has unnatural pacing or pronunciation; reads as robot. | mode 9 specialization. |
| **Timing-collapse** | Frames cycle too fast or too slow for the voice; loss of comprehension. | mode 9 specialization. |
| Plus all from [`../image/slideshow-tips-then-plug.md`](../image/slideshow-tips-then-plug.md) or [`../image/slideshow-listicle.md`](../image/slideshow-listicle.md). | | |

Honesty flags: same as the image-medium sibling, plus:

- `voice_profile_match=verified` — voice matches the persona's character JSON voice profile.
- `voiceover_frame_sync=verified` — voiceover and frame text aligned.

## Cross-references

- [`../image/slideshow-tips-then-plug.md`](../image/slideshow-tips-then-plug.md), [`../image/slideshow-listicle.md`](../image/slideshow-listicle.md) — image-medium siblings whose arc is reused here.
- [`../../strategies/ai-ugc-personas.md`](../../strategies/ai-ugc-personas.md).
- [`../../rich-media/`](../../rich-media/).
- Sibling video formats: [`narrative-talking-head.md`](narrative-talking-head.md), [`day-in-life-ugc.md`](day-in-life-ugc.md), [`problem-agitate-solve.md`](problem-agitate-solve.md).
