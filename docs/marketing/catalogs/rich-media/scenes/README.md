# Rich Media — Scenes

Each `<slug>.json` in this folder defines one reusable environment — the *where* component of an image or video prompt. Scenes carry environment, lighting, palette, time-of-day, ambiance, and a list of typical props. Scenes are composed with characters and products to form the full prompt.

## Schema

Copy [`_template.json`](_template.json) and fill in. Field-by-field:

### Required

- **`slug`** — string; URL-safe slug; matches filename without extension.
- **`display_name`** — short human-readable name (e.g., "Homelab Desk", "Suburban Kitchen Morning").
- **`environment`** — object. The structural description:
  - `category`: "indoor-residential" | "indoor-workspace" | "outdoor-urban" | "outdoor-natural" | "studio-abstract" | other
  - `description`: one-paragraph descriptive prose
  - `notable_objects`: array of strings (e.g., ["wooden desk", "rack-mount server", "monstera plant"])
- **`lighting`** — object:
  - `time_of_day`: "early-morning" | "morning" | "afternoon" | "evening" | "night" | "studio-controlled"
  - `quality`: "soft-diffused" | "hard-directional" | "ambient" | "neon-accent" | other
  - `key_light_direction`: "from-left" | "from-right" | "from-front" | "from-above" | "backlit"
  - `notes`: free-text, e.g., "warm color temperature, slight haze in air"
- **`palette`** — object:
  - `dominant`: array of hex colors describing the dominant tones
  - `accent`: array of hex colors describing accent tones (often pulls from brand palette per [`../../../strategy/IMAGE_STYLE.md`](../../../strategy/IMAGE_STYLE.md))
  - `notes`: free-text on how palette interacts with brand canon
- **`ambiance`** — string; the *feeling* the scene should evoke (e.g., "calm, slightly aspirational, lived-in", "focused, slightly tense").

### Optional

- **`typical_props`** — array of objects, each:
  - `id`: prop slug
  - `description`: brief
  - `placement_notes`: how the prop interacts with characters or product placements
- **`brand_consistency_check`** — string; specific notes on how this scene aligns or deviates from `IMAGE_STYLE.md`. If the scene deliberately introduces non-brand palette elements (common — scenes need realism), document why.
- **`notes`** — free-text annotations.

## Production discipline

- A single scene is reused across many renders with different characters, actions, and times-of-day. The `lighting.time_of_day` may be overridden in the prompt for a specific render; the rest of the scene stays stable.
- When generating multi-shot content (slideshow, video) in the same scene, pinning the scene slug across all shots is the primary tool against environmental drift.
- Prefer specific over generic. "Suburban kitchen, late-morning sun through east window, oak cabinets, tile floor" generates more consistently than "kitchen, morning light".

## Cross-references

- [`../templates/image-prompt.template.json`](../templates/image-prompt.template.json), [`../templates/video-prompt.template.json`](../templates/video-prompt.template.json) — prompt templates that compose scene entries.
- [`../../../strategy/IMAGE_STYLE.md`](../../../strategy/IMAGE_STYLE.md) — brand canon for palette and aesthetic.
