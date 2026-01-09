# Landing Template Design System

The landing-page-react-vite template now ships with a **structured design system** so every generated scenario can describe its look-and-feel with the same fidelity as its runtime configuration. This document explains how `.vrooli/styling.json`, the new `style-packs/` directory, and the landing-manager prompts work together to avoid “AI slop” gradients and enforce Clause-style case-study layouts by default.

## Files to read first

| Path | Purpose |
| --- | --- |
| `.vrooli/styling.json` | Active style pack loaded by the React surfaces at runtime. |
| `.vrooli/style-packs/*.json` | Reusable packs (Clause case study today, more soon). Copy one of these into `styling.json` or ask landing-manager to randomize against them. |
| `.vrooli/schemas/styling.schema.json` | Extended schema defining palette tokens, layout systems, component kits, imagery slots, and randomization metadata. |
| `ui/src/shared/lib/stylingConfig.ts` | Imports styling.json and exposes it to Admin + Public surfaces. |
| `docs/DESIGN_SYSTEM.md` (this file) | Human-readable briefing for agents and designers. |

## Schema updates (TL;DR)

| Addition | Why it matters |
| --- | --- |
| `style_id`, `influences`, `mood` | Document the design lineage so prompts can cite real references (“Clause case study”, “Swiss editorial”, etc.). |
| Expanded `palette` tokens | Surfaces now know about semantic surfaces (`surface_primary`, `surface_alt`, `accent_primary`, etc.) so they can render alternating sections without inventing gradients. |
| `layout` + `depth` | Define container width, hero layout (case-study vs. centered), section spacing, and shadow/border tokens. |
| `component_kits` + `imagery_slots` | Tell agents what UI primitives already exist (hero panels, brand strip, process timeline) and which screenshots/assets they must supply. |
| `section_variants` | Guardrails for how each section should look when this style is active. |
| `randomization` | Metadata for future “style randomizer” prompts (family name, allowed pairings, alternative fonts). |

Validate every change with `yarn lint-styling` (coming soon) or manually run `jq` against the schema:\
`node ./scripts/scenarios/templates/landing-page-react-vite/ui/scripts/generate-selector-manifest-helper.mjs --check-styling`

## Clause-style case study (default)

The default `styling.json` mirrors the research-backed Clause reference:

- **Palette**: Charcoal foundations (`#07090F` background, `#0F172A` hero surface) with a single bold accent (`#F97316`) and cyan annotations.
- **Typography**: Space Grotesk headings (64/48/32px scale) paired with Inter 17px body copy. CTA buttons reuse Space Grotesk for confident letterforms.
- **Layout**: 12-column grid, 160px vertical rhythm, 2-column hero that stacks layered dashboard cards on the right and copy/CTAs on the left.
- **Artifacts over gradients**: Every story block should include an artifact (full page poster, device macro, palette strip, or wireframe trio). Texture is limited to <8% diagonal noise on select sections.
- **CTA discipline**: Pill-shaped solid buttons only. Secondary CTAs appear as outlined pills or text links—no gradient backgrounds.
- **Story cadence**: Sequence hero → metrics → full-preview → wireframes → features → pricing → CTA, with alternating dense artifacts and airy copy zones.

When customizing:
1. Start by summarizing the “Design intent” (mood + influences) for the agent.
2. Highlight the hero kit and imagery slots you expect them to touch.
3. Provide screenshot URLs for `ui_panels`, `full_page_preview`, and `device_macro` so the hero never falls back to empty gradients.

## Style packs & randomization

- Drop new packs inside `.vrooli/style-packs/`. Follow the exact schema—these packs should be copy-paste ready for `.vrooli/styling.json`.
- Landing-manager’s upcoming style randomizer will:
  1. Pick a pack (Clause case study, ChronoTask floating panels, Fitme warm lifestyle, etc.).
  2. Optionally mutate `pairings`/`allowed_styles` to avoid repetition.
  3. Copy the chosen pack into `.vrooli/styling.json` before handing off to the agent.
- Until that automation lands, you can manually swap packs via:
  ```bash
  cp scripts/scenarios/templates/landing-page-react-vite/.vrooli/style-packs/<pack>.json \
     scripts/scenarios/templates/landing-page-react-vite/.vrooli/styling.json
  ```

## Guardrails for agents

Whenever you ask an AI to restyle the landing page:

1. Attach the entire `.vrooli/styling.json` to the prompt. Landing-manager now does this automatically when you use the “Agent Customization” UI/CLI.
2. Include the anti-slop checklist:
   - **One accent color + one support accent** maximum.
   - **Real artifacts** only (screenshots, device frames, palettes). No mesh gradients or blobby shapes.
   - **Consistent component kit** (hero, panels, brand strip). Reuse them instead of inventing new ones per section.
   - **Section cadence**: Case-study hero → preview → wireframes → features/pricing.
   - **Typography**: Use the provided fonts; if substitutes are required, note them explicitly in `typography.scale`.
3. Require a plan before code. Landing-manager now asks agents to summarize the vibe, palette, typography, section list, and key components prior to editing JSX/CSS.

## Updating the design

1. Modify `.vrooli/styling.json` (or create a new pack).
2. Run `pnpm --filter landing-template-ui test` to ensure none of the admin UI expectations break.
3. Update `docs/DESIGN_SYSTEM.md` with what changed and why, especially if new component kits or imagery slots were added.
4. Mention the new pack in `TEMPLATE.md` so other agents know it exists.

## Future work

- **Style randomizer CLI**: `landing-manager styles randomize --family case-study` that copies a pack and logs the change.
- **Device frame component**: React primitive for layering screenshots with drop shadows according to `imagery.device_frames`.
- **Schema validation hook**: add a dedicated check script (after we land `sg`) so any CI failure points directly to schema violations.

Until then, keep the Clause pack pristine and link to this document whenever you hand off the landing page to another agent.
