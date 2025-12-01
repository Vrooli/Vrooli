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
| `.vrooli/styling.json` | Establishes voice, palette, CTA patterns, and component guardrails. React surfaces lean on these rules for default props/variants. See `docs/DESIGN_SYSTEM.md` for the full schema + Clause pack briefing. |
| `.vrooli/variants/*.json` | Control and fallback payloads the public surface can render with zero API calls. |
| `ui/src/shared/lib/fallbackLandingConfig.ts` | Explains how fallback JSON becomes a runtime-safe config and how sections/pricing/downloads are normalized. |
| `ui/src/surfaces/public-landing/routes/PublicLanding.tsx` | Main renderer for landing sections + download rail. |

Read those five files together before attempting copy or styling overrides. They keep variant axes, language, and UI expectations in lockstep.

## Customization Workflow

1. **Pick axes + styling context**  
   - Update (or reference) `.vrooli/variant_space.json` to confirm persona/JTBD/conversion style constraints.  
   - Align tone and palette with `.vrooli/styling.json`. If you need a different look, copy a pack from `.vrooli/style-packs/` or author a new one using the schema described in `docs/DESIGN_SYSTEM.md`.

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
- When swapping design languages, capture the entire pack under `.vrooli/style-packs/` and describe the intent (mood, artifacts, imagery slots) in `docs/DESIGN_SYSTEM.md` so landing-manager can cite it in its prompts.  
- Touch `variant_space.json` when you introduce a new axis or need to disable a combination. Admin axes selectors + tests rely on those IDs—avoid hard-coding them elsewhere.  
- The fallback config loader (`ui/src/shared/lib/fallbackLandingConfig.ts`) normalizes `.vrooli/variants/fallback.json`, guaranteeing deterministic ordering and enabled flags even offline. Extend that helper instead of duplicating normalization logic inside components.

## Experience Architecture Audit – Landing Page React Vite Template

**Purpose statement**: Landing Manager’s runtime lets marketers spot conversion signals and react without touching the factory—Analytics explains what’s happening, Customization lets them adjust variants/sections, and the public landing should guide prospects directly to a CTA or download.

### Personas & key jobs
- **Experiment Operator (ops/marketing)** – check live traffic source, find red/yellow variants, open analytics for the right time range.
- **Content Author** – locate the variant or section that needs edits, update copy/weighting, preview updates on the public landing.
- **Prospect/Visitor** – skim the public page, jump to the section they care about (features, pricing, downloads), and click the hero CTA.

### Flow insights (current vs. ideal)
- Ops users previously landed on `/admin` with two vague buttons. They needed to guess that “Analytics” tells them what happened and “Customization” fixes it. The ideal home screen should state the two jobs explicitly and give a preview link to validate the public experience.
- Content authors had to scan dozens of variant cards; stale or underperforming variants were only hinted at in Experience Ops. Ideal flow lets them filter the grid down to the problematic experiments directly from the same signals.
- Visitors on `/` had no navigation guidance—they had to scroll from hero → features → pricing manually, and admins couldn’t see runtime health without dev tools. Ideal landing flow provides a sticky header that surfaces runtime state, anchor links, and a persistent CTA.
- Once an ops user filtered analytics to a variant/time range, there was no persistent indicator tying that view back to runtime state. Ideal flow keeps the “what am I looking at?” context in the viewport and links straight to customization/preview.
- Variant cards didn’t show *why* a variant needed attention—just that filters existed. Ideal flow explains the reason (stale copy, never customized, lowest conversion) inline so the author knows the next edit to ship.

### Changes implemented this loop
- **Admin Experience Guide (AdminHome)** – Added a purpose banner plus three quick flows (audit performance, ship a variant, preview landing) with direct buttons so first-time admins know exactly where to start.
- **Variant filters + needs-attention focus (Customization)** – Introduced a search/attention filter bar, applied counts to the active grid, and wired the Needs Attention panel to focus the list, shrinking the “find the broken experiment” loop to one click. Added `highlight variant` and `clear filters` selectors for automation.
- **Section-focused deep links** – Customization now resolves `focusSectionId` / `focusSectionType` query params so AdminHome, Analytics, and Ops widgets can jump straight into the Section Editor (defaulting to hero) without forcing users to hunt for the right block.
- **Public landing navigation rail** – Inserted `LandingExperienceHeader` with runtime pills, anchor navigation (desktop + mobile), and a sticky hero CTA button so visitors, operators, and agents can jump to sections instantly while seeing whether fallback copy is active.
- **Analytics focus rail** – Added `AnalyticsFocusBanner` so ops users always know which variant/time range they’re analyzing, how it compares to the live runtime, and can reset filters or jump to customization/preview without scrolling.
- **Variant status storytelling** – Added `VariantListSummary` plus inline badges on each variant card that call out last edit time and attention reasons (“Stale · 12d”, “Lowest conversion”). This turns the Customization grid into a to-do list instead of an undifferentiated catalog.
- **Download CTA surfacing** – The sticky landing header now includes a dedicated download button (when assets exist) so end-users chasing installers don’t need to scroll through the marketing narrative to reach entitlements.
- **Admin health digest (AdminHome)** – Snapshot panel now fetches variant and analytics data to surface live runtime state, traffic allocation, and the highest-priority attention variant with direct actions (“Open analytics”, “Review in customization”, “Adjust weights”) so ops personas can triage before diving into a mode.
- **Cross-surface deep links** – Customization accepts `?focus=<slug>` to auto-filter and scroll to that variant, letting AdminHome (and future surfaces) highlight the broken experiment in one click instead of forcing users to reapply filters manually.

### Opportunities for future loops
1. **Customization > Section intent signals** – Surface which section triggered an alert (hero vs. pricing vs. download rail) so deep links can target that block instead of defaulting to hero when context is missing.
2. **Analytics > Saved views** – Persist filter presets (e.g., “Last 30 days · Variant Bravo”) and surface them on AdminHome to eliminate the repeated query building for ops personas.
3. **Public landing trust rails** – Add lightweight trust indicators (customer logos, uptime badges) to the sticky header or hero so prospects see proof before scrolling.
4. **Health event log** – Capture recent runtime events (traffic allocation changes, fallback activation, agent jobs) in the AdminHome digest so follow-up actions can be audited without leaving the portal.
