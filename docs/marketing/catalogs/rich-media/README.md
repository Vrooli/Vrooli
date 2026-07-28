# Rich Media — Plan of Record

This folder is the **structured-data substrate** for AI-generated images and videos used in marketing. Persona-actor characters, scenes, products, and prompt templates live here as JSON so they can be reused across image-gen and video-gen tools — and across multiple frames within a single artifact — without visual drift.

This doc is the navigation hub. Per-entity schemas and templates live in subfolders.

## Why structured data, not free-text prompts

Three converging reasons:

1. **JSON prompting is now the dominant interface for production-grade image and video generation.** Veo 3.1 and Seedance 2 use compatible JSON prompt structures, so a single schema can drive both without rewriting. Image models including Flux and Qwen-Image natively support JSON prompts. The 2026 industry has converged on this; reasoning over free-text is no longer state of the art.
2. **Multi-frame and multi-shot consistency is the hardest production challenge.** Slideshows (image medium) and short videos (video medium) both depend on a character looking the same across frames, a scene having the same lighting across cuts, a product being identifiable across angles. Free-text prompts drift; structured JSON with frozen "identity blocks" doesn't.
3. **Reuse compounds.** A character JSON authored once is reused across every campaign that persona stars in; a scene JSON for "homelab desk" is reused across every video filmed in that environment. The marginal cost of the second campaign is much lower than the first.

Per the operator's framing: *"It's kind of like building a video game"* — the structured data **is** the asset, the rendered images and videos are derivative outputs.

## Folder structure

```
rich-media/
  README.md              # this file
  characters/            # persona-actor and scenario-character entries
    README.md            # schema doc
    _template.json       # template — copy and rename to <slug>.json
    <slug>.json          # one per character
    <slug>.character-sheet.png   # composite front/3-4/side/back reference
  scenes/                # reusable environments
    README.md
    _template.json
    <slug>.json
  products/              # Vrooli scenario / product visual descriptors
    README.md
    _template.json
    <slug>.json
  templates/             # prompt skeletons composing characters + scenes + products
    image-prompt.template.json    # Identity + CoreTraits + Clothing + Pose + Lighting/Camera + Background
    video-prompt.template.json    # Veo/Seedance-compatible Cinematography + Subject + Action + Context + Style+Ambiance
  assets/                # ground-truth uploads
    README.md
    logos/               # canonical logo files (logo.svg, etc.)
    voice-samples/       # persona voice references
    product-shots/       # scenario UI screenshots, packaging shots
    character-sheets/    # composite reference images per character
```

## Composition flow

When producing an image or video, the agent (today) or the future image/video skill composes:

```
character.<persona>.json       (the identity)
  +
scene.<environment>.json       (the where)
  +
product.<vrooli-scenario>.json  (the what — when product appears)
  +
templates/image-prompt.template.json  (or video-prompt.template.json)
  +
prompt-specific overrides       (this frame's specific pose, action, line)
  ↓
JSON prompt sent to image-gen / video-gen tool
  ↓
rendered output stored alongside source JSON for reproducibility
```

The composition is **manual today** (operator + agent assemble the JSON). The future state is a CLI in a rich-media-studio scenario that takes references by slug and emits the final prompt JSON. The schema authored here is designed to be CLI-friendly when that scenario lands.

## Schema design discipline

Three design choices the research argued for, and we adopt:

1. **Mirror Veo/Seedance field names in `video-prompt.template.json`.** Cross-tool compatibility is free if you use their conventions; expensive if you invent your own.
2. **The character-sheet composite is a first-class asset, not metadata.** Some image-gen tools take JSON + reference image; the JSON without the sheet is materially weaker. Each character's `character-sheet.png` lives next to its `.json`.
3. **Identity block is frozen across all renders of a character.** Pose, action, lighting, scene change per render; identity (face structure, body, voice) does not. This is the single largest contributor to multi-frame consistency.

## Schema mutation policy

Schemas in this folder will mutate. The pattern:

- Bake one character + one scene + one product end-to-end through a single campaign.
- Capture what is missing or awkward in a typed `marketing-craft-observation/rich-media/<slug>` entry, or raise `capability-gap` when the issue is blocked by missing tooling.
- After 2-3 campaigns' worth of usage, propose schema changes via `brand-guideline-update` decisions.

Schemas authored before real use are theoretical; real use mutates them. Build for one, generalize after three.

## Disclosure and AI-UGC tie-in

Rich-media content depicting people is governed by [`../../strategy/patterns/ai-ugc-personas.md`](../../strategy/patterns/ai-ugc-personas.md). Each character entry's `disclosure_required` field reflects the platform-disclosure expectations; persona-actors require disclosure, scenario-mascot characters typically do too, abstract style imagery (no person depicted) does not.

The `do_not_resemble` field on each character is the contrarian's check against [STRATEGY.md mode 15 (real-person-impersonation)](../../strategy/STRATEGY.md#anti-patterns).

## Future home

This folder is **incubating data** per `docs/agent-system/OPERATING_GRAPHS.md` §"State belongs to scenarios; prose holds judgment": structured entity state held in the PoR only because no scenario serves it yet, with `scenario:brand-manager` (or a future `rich-media-studio`) as the named promotion target. When the owning scenario serves these entities, this folder's schema migrates into structured database entries with versioning, the markdown stand-ins under `assets/` get replaced by real asset storage, and this folder compresses to a pointer plus the judgment sections above. Note 2026-07-28: `scenario:campaign-content-studio` has been retired and is no longer a promotion target. `scenario:brand-manager` exists but owns visual identity rather than rich-media entities, and `scenario:content-desk` is deliberately scoped to editorial production and holds no rendered assets. Promotion therefore still awaits an asset-production scenario; until one exists and an approved migration lands, this folder remains the operator-curated stand-in.

## Cross-references

- [`../../strategy/patterns/ai-ugc-personas.md`](../../strategy/patterns/ai-ugc-personas.md) — persona-actor account discipline and disclosure rules; consumes character entries.
- [`../post-types/image/`](../post-types/image/), [`../post-types/video/`](../post-types/video/) — image and video post types that consume these schemas.
- [`../../strategy/ASSETS.md`](../../strategy/ASSETS.md), [`../../strategy/IMAGE_STYLE.md`](../../strategy/IMAGE_STYLE.md) — brand canon; characters and scenes must be checkable against the brand palette and aesthetic.
- [`../../strategy/CHANNELS.md`](../../strategy/CHANNELS.md) — per-channel format support for the rendered outputs.
- Reference reading on JSON prompting (researcher's `marketing-craft` scope tracks updates):
  - Veo 3.1 prompting guide (Google Cloud Blog, 2026)
  - JSON prompting for AI image generation (industry coverage 2026)
  - Character consistency formula: `Identity + Core Traits + Clothing/Style + Pose + Lighting/Camera + Background`
  - Video formula: `Cinematography + Subject + Action + Context + Style & Ambiance`
