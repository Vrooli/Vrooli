# Landing Page Template (React + Vite)

This template is the canonical landing runtime used by landing-manager when generating Browser Automation Studio landing scenarios. Every generated scenario inherits this structure, so agents editing this directory are effectively editing **all future landing pages**.

## Orientation: Where Everything Lives

```
landing-page-react-vite/
├── api/                         # Go backend copied into each generated scenario
│   ├── *_handlers.go             # HTTP surfaces (auth, variants, downloads, metrics, etc.)
│   ├── *_service.go              # Domain + DB integration seams
│   └── initialization/           # Scenario bootstrap + migrations
├── .vrooli/                     # Source of truth for AI/runtime configs
│   ├── variant_space.json        # Axis definitions agents must respect
│   ├── styling.json              # Tone, palette, CTA, and typography guardrails
│   ├── variants/*.json           # Control + fallback variants copied into runtime
│   └── schemas/                  # JSON schema for sections, styling, variant payloads
├── ui/                          # React + Vite frontend used for both admin + public surfaces
│   ├── src/app/providers/        # Global providers (variant selection, admin auth, etc.)
│   ├── src/shared/               # API clients, hooks, lib helpers shared by surfaces
│   └── src/surfaces/
│       ├── public-landing/       # Public landing route + sections
│       └── admin-portal/         # Admin/editor experiences
└── docs/, requirements/, test/  # Reference documents, feature specs, and integration tests
```

### Config Anchors Agents Should Read First

| File | Why it matters |
| --- | --- |
| `.vrooli/variant_space.json` | Defines personas, JTBD, and conversion-style axes. Admin UI reads this file to populate controls, and AI agents must pick one variant per axis. |
| `.vrooli/styling.json` | Establishes voice, palette, CTA patterns, and component guardrails. React surfaces lean on these rules for default props/variants. |
| `.vrooli/variants/*.json` | Control and fallback payloads the public surface can render with zero API calls. |
| `ui/src/shared/lib/fallbackLandingConfig.ts` | Explains how fallback JSON becomes a runtime-safe config and how sections/pricing/downloads are normalized. |
| `ui/src/surfaces/public-landing/routes/PublicLanding.tsx` | Main renderer for landing sections + download rail. |

Read those five files together before attempting copy or styling overrides. They keep variant axes, language, and UI expectations in lockstep.

## Customization Workflow

1. **Pick axes + styling context**  
   - Update (or reference) `.vrooli/variant_space.json` to confirm persona/JTBD/conversion style constraints.  
   - Align tone and palette with `.vrooli/styling.json`. Add new guidance there instead of sprinkling constants through components.

2. **Author or tweak variant payloads**  
   - Control edits go in `.vrooli/variants/control.json`.  
   - Offline/health-check payloads live in `.vrooli/variants/fallback.json`.  
   - Each payload must satisfy `.vrooli/schemas/variant.schema.json`, so keep `section_type`, `axes`, and downloads consistent.

3. **Render + preview sections**  
   - `ui/src/surfaces/public-landing/routes/PublicLanding.tsx` converts `LandingSection` entries into React components using the shared registry (see below).  
   - Section components read `styling.json`-inspired props (color tokens, CTA treatments) via hooks/utilities under `ui/src/shared`.

4. **Persist runtime expectations**  
   - Backend tables enforce `section_type` constraints via `initialization/postgres/schema.sql`.  
   - When adding a new type, update the schema, JSON schema, TS types, and the section registry in a single PR so generated scenarios stay coherent.

## Adding or Updating Section Types

All section components now live in `ui/src/surfaces/public-landing/sections`. Each section exports a React component that accepts `content` plus any bespoke props (e.g., `pricingOverview`) and registers itself with the renderer.

1. **Create the component**  
   - Add `CountdownSection.tsx` under `ui/src/surfaces/public-landing/sections`.  
   - Mirror the existing pattern (props interface + `useMetrics` for CTA events).

2. **Register it for rendering**  
   - Update the `SECTION_COMPONENTS` map in `PublicLanding.tsx` so the renderer picks it up automatically. No more multi-case switch blocks.

3. **Update shared types**  
   - Extend `SectionType` in `ui/src/shared/api/types.ts`.  
   - Ensure `LandingSection` fixture data includes the new `section_type`.

4. **Lock down schemas + DB**  
   - Update `.vrooli/schemas/sections/<name>.schema.json` with validated content.  
   - Append the type to the CHECK constraint in `initialization/postgres/schema.sql`.  
   - Reference it in `.vrooli/schemas/variant.schema.json` so variant payloads validate.

5. **Seed / fallback data**  
   - Populate `.vrooli/variants/*.json` entries (control + fallback) with at least one instance of the new section for smoke testing.

### Section Component Pattern

```tsx
interface CountdownSectionProps {
  content: {
    eyebrow?: string;
    cta_label?: string;
    expires_at?: string;
  };
}

export function CountdownSection({ content }: CountdownSectionProps) {
  const { trackConversion } = useMetrics();

  return (
    <section className="...">
      {/* Render content */ }
    </section>
  );
}
```

### Files to Update When Adding/Removing Sections

| Change | Files |
| --- | --- |
| React implementation | `ui/src/surfaces/public-landing/sections/*Section.tsx` |
| Renderer wiring | `ui/src/surfaces/public-landing/routes/PublicLanding.tsx` (`SECTION_COMPONENTS`) |
| Schema / validation | `.vrooli/schemas/sections/*.json`, `.vrooli/schemas/variant.schema.json`, `initialization/postgres/schema.sql` |
| Types + fixtures | `ui/src/shared/api/types.ts`, `.vrooli/variants/*.json`, `.vrooli/variant_space.json` if new axes interplay |

### Section Coverage Status

- **Implemented**: hero, features, pricing, cta, testimonials, faq, footer, video, download rail.  
- **Schema-only (needs React implementation before use)**: benefits, social-proof, lead-form, preview.

## Why `.vrooli/styling.json` + `variant_space.json` Matter During Code Edits

These files are the only place agents should encode tone, CTA pairings, and persona-specific guardrails. React components read them indirectly via controllers/providers; if you need additional context (e.g., show/hide downloads by conversion style), add it to the JSON and plumb it through `useLandingVariant` rather than scattering `if` statements across components.

Key takeaways:

- Update `styling.json` when adjusting typography, CTA proportions, or iconography so both admin tools and AI instructions stay synchronized.  
- Touch `variant_space.json` when you introduce a new axis or need to disable a combination. Admin axes selectors + tests rely on those IDs—avoid hard-coding them elsewhere.  
- The fallback config loader (`ui/src/shared/lib/fallbackLandingConfig.ts`) normalizes `.vrooli/variants/fallback.json`, guaranteeing deterministic ordering and enabled flags even offline. Extend that helper instead of duplicating normalization logic inside components.
