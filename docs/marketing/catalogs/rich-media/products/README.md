# Rich Media — Products

Each `<slug>.json` in this folder defines one Vrooli scenario, bundle, or product as it appears visually in marketing renders. Products carry physical descriptors (logo placement, packaging, on-screen UI), placement rules, and constraints that prevent visual misrepresentation.

When a render shows a Vrooli scenario being used, the product entry constrains how the scenario's UI / mobile app / terminal output is depicted to keep it accurate to current state.

## Schema

Copy [`_template.json`](_template.json) and fill in. Field-by-field:

### Required

- **`slug`** — string; URL-safe slug; matches filename without extension. Typically matches the scenario slug under `path:scenarios/<slug>/`.
- **`display_name`** — human-readable scenario / product / bundle name.
- **`product_kind`** — "scenario" | "bundle" | "platform" | "merchandise"
- **`physical_descriptors`** — object. Whatever applies:
  - `ui_surfaces`: array of objects, each describing one UI surface (e.g., web app, mobile app, terminal, CLI output). Each:
    - `surface_kind`: "web-app" | "mobile-app" | "terminal" | "cli-output" | "physical-object"
    - `key_visual_elements`: array of strings (e.g., "dark-mode header with neon-green accent line", "monospace terminal with hex prompt")
    - `do_not_distort`: array of strings — UI elements that must remain accurate (e.g., "pricing tier labels must match current TIERS.md", "feature names must match current scenario state")
  - `packaging`: object (for bundles or physical merch) — colors, layout, key text. Optional.
- **`brand_element_placement_rules`** — object:
  - `logo_placement`: where the logo goes; whether it's required in this product's renders
  - `palette_lock`: which palette colors apply (typically the full brand palette per `IMAGE_STYLE.md`)
  - `tagline_rules`: any tagline / motto usage constraints (per `docs/narrative/PITCH.md`)
- **`tier_alignment_required`** — boolean. `true` for any scenario / bundle where features are tier-gated and renders must show only features available at the depicted tier (per `docs/monetization/strategy/TIERS.md`).

### Optional

- **`on_screen_ui_assets`** — array of relative paths to canonical screenshot assets in [`../assets/product-shots/`](../assets/product-shots/). Used as multi-reference inputs for image-gen so the rendered UI matches reality.
- **`do_not_pair_with`** — array of product slugs that should not appear in the same render (e.g., a placeholder UI vs the new design).
- **`notes`** — free-text annotations.

## Production discipline

- **Tier alignment is the most error-prone constraint.** Renders that show a "Pro" tier user accessing a feature that's actually at "Enterprise" tier mislead buyers. Contrarian validates `tier_alignment_required: true` products against the scenario's current entry in Offer Desk (`offer-desk offers catalog-edges`).
- **UI-state freshness.** A product entry should be reviewed when the underlying scenario ships UI changes. Stale entries propagate stale renders.
- **No fabricated features.** Renders must not invent UI elements (buttons, screens, capabilities) that don't exist in the actual scenario. Contrarian's `capability-inflation` check (mode 1) extends to visual depictions.

## Cross-references

- [`../templates/image-prompt.template.json`](../templates/image-prompt.template.json), [`../templates/video-prompt.template.json`](../templates/video-prompt.template.json) — prompt templates that compose product entries.
- [`../../../../monetization/strategy/TIERS.md`](../../../../monetization/strategy/TIERS.md), [`../../../../monetization/catalogs/CATALOG.md`](../../../../monetization/catalogs/CATALOG.md), and `offer-desk offers catalog-list` / `catalog-edges` — tier alignment authority.
- [`../../../strategy/ASSETS.md`](../../../strategy/ASSETS.md), [`../../../strategy/IMAGE_STYLE.md`](../../../strategy/IMAGE_STYLE.md) — brand canon.
