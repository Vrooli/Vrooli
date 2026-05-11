# Brand

Visual identity and voice overview for Vrooli. Brand-manager (the member agent) is the strategic steward; the `brand-manager` scenario will eventually be the structured storage layer.

**Write rule:** operator-curated via accepted `brand-guideline-update` decisions. Brand-manager proposes; does not edit directly.

## Where each piece of the brand lives

This file is the navigation hub. The actual canonical content lives in dedicated files:

| Aspect | Canonical location |
|---|---|
| **Voice canon** (positioning, audience framing, voice samples, anti-patterns, dev-log narrative principles) | [`STRATEGY.md`](STRATEGY.md) |
| **Visual assets** (logos, favicons, OG image, fonts, usage rules) | [`ASSETS.md`](ASSETS.md) |
| **AI image generation style** (palette, aesthetic, prompt directives) | [`IMAGE_STYLE.md`](IMAGE_STYLE.md) |
| **Audience personas** | [`AUDIENCES.md`](AUDIENCES.md) |
| **Channel rules** | [`CHANNELS.md`](CHANNELS.md) |
| **Project-identity narrative** (pitch, story, FAQ, press kit, deck outline) | [`path:docs/narrative/`](../../narrative/) |

Single-source-of-truth discipline — do not duplicate content here that lives in the dedicated files.

## Visual identity (high-level summary)

- **Logo:** rabbit shaped from the letters V-R-O-O-L-I, conveying speed and a small Easter egg. See [`ASSETS.md`](ASSETS.md) for the full registry.
- **Palette:** dark blue and deep purple base with neon green accents. See [`IMAGE_STYLE.md`](IMAGE_STYLE.md) for prompt directives.
- **Typography:** brand font (`sakbunderan`) is logo-only; UI / body typography not yet canonically declared. To be filled in by the `brand-manager` scenario.
- **Aesthetic:** abstract, futuristic, neon. No stock photos. No corporate-clipart. No photorealistic AI humans.

## Voice (high-level summary)

- **Builder, not marketer.** First person, conversational, technically credible.
- **Agents as protagonists.** The agents themselves are visible; that's the unique signal.
- **Honest about struggles.** Failures land harder than polished wins.
- **Specific over vague.** Real numbers, real names, real artifacts.

Full canon in [`STRATEGY.md`](STRATEGY.md), including dev-log narrative principles, voice samples, and anti-patterns.

## Long-term direction

When the `brand-manager` *scenario* ships (currently a draft skill — see [`scenarios/prompt-manager/store/skills/packs/core/brand-manager/SKILL.md`](../../../scenarios/prompt-manager/store/skills/packs/core/brand-manager/SKILL.md)), it will provide structured storage for logos, favicons, color systems, typography, and voice snippets — replacing the markdown-based canon in `ASSETS.md` and `IMAGE_STYLE.md`. At that point those files become pointers at the scenario's registry; this file remains the navigation hub.
