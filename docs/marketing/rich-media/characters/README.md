# Rich Media — Characters

Each `<slug>.json` in this folder defines one character — typically a persona-actor used in AI-UGC content, occasionally a scenario mascot or operator stand-in. Paired with a `<slug>.character-sheet.png` composite reference image for production use.

## Schema

Copy [`_template.json`](_template.json) and fill in. Field-by-field:

### Required

- **`slug`** — string; URL-safe slug; matches filename without extension. Cross-referenced from `marketing-crew/shared/personas/<slug>/profile.json`.
- **`display_name`** — string; clearly fictional name (never a real person's name; never a near-likeness of a real person's name). Used internally; persona's social-handle-name lives in the persona's `accounts.json` (untracked per `CHANNELS.md` secrets rule).
- **`identity_block`** — immutable across all renders of this character. Object containing:
  - `gender_presentation`: "feminine" | "masculine" | "androgynous" | "non-binary"
  - `age_range`: e.g., "60-70", "25-35"
  - `ethnicity`: descriptive string; respects realistic representation; does not target a specific real-world person's appearance
  - `body_type`: descriptive (e.g., "average-build", "tall-slim", "stocky")
  - `face_structure`: descriptive (e.g., "round face, wide-set eyes, soft jaw")
  - `hair`: object with `color`, `style`, `length`
  - `eyes`: object with `color`, `notable_features` (e.g., "crow's feet", "narrow")
  - `distinctive_features`: array of strings (e.g., "freckles across nose", "scar on left cheek"). May be empty.
- **`core_traits`** — immutable across renders. Object containing:
  - `demeanor`: e.g., "warm, deliberate", "energetic, conversational"
  - `voice_profile`: object with `pitch_range`, `pace`, `accent_or_dialect`, `notable_features`. Used to drive TTS or voice-model selection.
  - `niche`: the persona's content niche (e.g., "home organization for retirees", "homelab tinkering")
  - `audience_target`: the audience persona this character speaks to (mapped to entries in [`../../AUDIENCES.md`](../../AUDIENCES.md))
- **`wardrobe_variants`** — array of variant objects. Each variant:
  - `id`: variant slug (e.g., "casual-home", "outdoor-walk", "kitchen-prep")
  - `description`: clothing details
  - `palette_notes`: how variant interacts with brand palette (does not introduce off-brand colors when on screen)
- **`character_sheet_uri`** — relative path to the composite reference image (`<slug>.character-sheet.png`). Required.
- **`disclosure_required`** — boolean. `true` for persona-actor characters subject to platform AI-content disclosure per [`../../strategies/ai-ugc-personas.md`](../../strategies/ai-ugc-personas.md).
- **`do_not_resemble`** — array of strings. Real-person likenesses the contrarian validates against (celebrities, public figures, named competitor founders). Initially populated with a generic "any specific identifiable real person" entry; specific names added if a character's generated outputs trend toward a likeness.

### Optional

- **`lora_uri`** — relative path to a LoRA fine-tune file or hosted URL. Use only when reference-based generation (Midjourney Omni Reference, Ideogram Character) is insufficient for consistency. Authoring a LoRA is a higher-investment step; defer until a character has shipped 5+ artifacts and consistency is still drifting.
- **`do_not_pair_with`** — array of other character slugs. Persona-actors that should never appear in each other's content (cross-account cannibalization risk).
- **`tied_skus`** — array of Vrooli SKU keys this character is the primary spokesperson for. Used by attribution and conversion-lift tracking.
- **`notes`** — free-text annotations about emerging consistency issues, prompt-tuning learnings, etc. Promoted into structured fields when they stabilize.

## Production discipline

- The `identity_block` and `core_traits` are **frozen**. Changes to these fields produce a fundamentally different character; if the character needs to evolve (e.g., aging, hair-style change), retire the old entry and author a new one with a new slug.
- The `wardrobe_variants` and `lora_uri` are mutable; they evolve with usage.
- Every render's prompt JSON references the character entry by slug; reproduce-from-source is a property of the schema.

## Character sheet composite

A character sheet is the production-grade reference: a single composite image showing the character from front, 3/4, side, and back angles, in neutral pose, with consistent lighting. Most image-gen tools accept this composite as a multi-reference upload and return more consistent generations than text alone. This is the gold standard workflow per 2026 industry practice.

Storing the composite at `<slug>.character-sheet.png` lets the prompt template reference it by relative path; assets live in [`../assets/character-sheets/`](../assets/character-sheets/) for the canonical version with extras (additional poses, expressions, lighting variants) once a character matures.

## Cross-references

- [`../../strategies/ai-ugc-personas.md`](../../strategies/ai-ugc-personas.md) — persona-actor discipline; characters are the visual substrate for personas.
- [`../templates/image-prompt.template.json`](../templates/image-prompt.template.json), [`../templates/video-prompt.template.json`](../templates/video-prompt.template.json) — prompt templates that consume character entries.
- [`marketing-crew/shared/personas/<slug>/`](../../../scenarios/prompt-manager/store/teams/marketing-crew/shared/) — per-persona profile / accounts / slate / link-in-bio (folder created on persona activation).
