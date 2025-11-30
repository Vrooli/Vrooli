# Landing Page Template (React + Vite)

This template generates complete landing page scenarios with A/B testing, analytics, and payment integration. Used by landing-manager to create both SaaS landing pages and lead-magnet pages.

## For Agents: Working with This Template

This is the **source template** that gets copied when landing-manager generates new landing page scenarios.

### Structure

```
landing-page-react-vite/
├── api/                      # Go backend (copied to generated/*/api/)
│   ├── main.go              # API entry point
│   ├── *_service.go         # Business logic services
│   └── *_handlers.go        # HTTP handlers
├── .vrooli/                  # Scenario-level configs (copied to generated/.vrooli)
│   ├── variants/            # Runtime-ready variant configs + fallback
│   ├── schemas/             # JSON Schemas for variants + sections + styling
│   └── styling.json         # Brand/tone/style guardrails for agents
├── ui/                       # React frontend (copied to generated/*/ui/)
│   └── src/
│       ├── components/
│       │   └── sections/    # Landing page section components
│       │       ├── HeroSection.tsx
│       │       ├── FeaturesSection.tsx
│       │       ├── PricingSection.tsx
│       │       ├── CTASection.tsx
│       │       ├── TestimonialsSection.tsx
│       │       ├── FAQSection.tsx
│       │       ├── FooterSection.tsx
│       │       └── VideoSection.tsx
│       ├── pages/
│       │   └── PublicHome.tsx   # Main renderer (uses switch statement)
│       └── lib/
│           └── api.ts           # API client + TypeScript types
├── initialization/           # Database setup
│   └── postgres/
│       ├── schema.sql       # Table definitions (section_type CHECK)
│       └── seed.sql         # Initial data
└── requirements/            # Feature modules
```

### Adding a New Section Type

To add a new section type (e.g., "countdown"):

1. **Create component** in `ui/src/components/sections/CountdownSection.tsx`
   - Copy an existing component as template
   - Define the `content` props interface
   - Implement the rendering

2. **Update PublicHome.tsx** in `ui/src/pages/PublicHome.tsx`
   - Add import: `import { CountdownSection } from '../components/sections/CountdownSection';`
   - Add switch case: `case 'countdown': return <CountdownSection {...commonProps} />;`

3. **Update TypeScript type** in `ui/src/lib/api.ts`
   - Add 'countdown' to `section_type` union in `ContentSection` interface

4. **Update database schema** in `initialization/postgres/schema.sql`
   - Add 'countdown' to the CHECK constraint on `section_type`

5. **Create JSON schema** in `.vrooli/schemas/sections/countdown.schema.json`
   - Define the content fields for admin customization and validation

6. **Update variant schema references** in `.vrooli/schemas/variant.schema.json`
   - Add new `section_type` enum entry + condition so validation passes

### Section Component Pattern

All section components follow this pattern:

```tsx
interface {Name}SectionProps {
  content: {
    // Define expected content fields from JSON schema
    title?: string;
    // ...
  };
}

export function {Name}Section({ content }: {Name}SectionProps) {
  const { trackCTAClick } = useMetrics();  // For analytics

  return (
    <section className="...">
      {/* Render section content */}
    </section>
  );
}
```

### Files to Update When Adding/Removing Sections

| Change | Files to Update |
|--------|-----------------|
| Add section | `*Section.tsx`, `PublicHome.tsx`, `api.ts`, `schema.sql`, `*.json` schema |
| Remove section | Same files (remove entries) |
| Modify section fields | `*Section.tsx` (component), `*.json` (schema) |
| Change section styling | `*Section.tsx` only |

### Implemented vs Schema-Only Sections

Check `.vrooli/schemas/sections/` for status:

**Implemented** (have both schema + component):
- hero, features, pricing, cta, testimonials, faq, footer, video

**Schema-only** (need component creation):
- benefits, social-proof, lead-form, preview
