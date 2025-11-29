# SaaS Landing Page Template

This template generates a complete landing page scenario with A/B testing, analytics, and Stripe payments.

## For Agents: Working with This Template

This is the **source template** that gets copied when landing-manager generates new landing page scenarios.

### Structure

```
payload/
├── api/                      # Go backend (copied to generated/*/api/)
│   ├── main.go              # API entry point
│   ├── *_service.go         # Business logic services
│   └── *_handlers.go        # HTTP handlers
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

1. **Create component** in `payload/ui/src/components/sections/CountdownSection.tsx`
   - Copy an existing component as template
   - Define the `content` props interface
   - Implement the rendering

2. **Update PublicHome.tsx** in `payload/ui/src/pages/PublicHome.tsx`
   - Add import: `import { CountdownSection } from '../components/sections/CountdownSection';`
   - Add switch case: `case 'countdown': return <CountdownSection {...commonProps} />;`

3. **Update TypeScript type** in `payload/ui/src/lib/api.ts`
   - Add 'countdown' to `section_type` union in `ContentSection` interface

4. **Update database schema** in `payload/initialization/postgres/schema.sql`
   - Add 'countdown' to the CHECK constraint on `section_type`

5. **Create JSON schema** in `scenarios/landing-manager/api/templates/sections/countdown.json`
   - Define the content fields for admin customization

6. **Update section registry** in `scenarios/landing-manager/api/templates/sections/_index.json`
   - Add the new section entry

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

Check `scenarios/landing-manager/api/templates/sections/_index.json` for status:

**Implemented** (have both schema + component):
- hero, features, pricing, cta, testimonials, faq, footer, video

**Schema-only** (need component creation):
- benefits, social-proof, lead-form, preview
