# Agent Guide: Landing Page Sections

**READ THIS FIRST** when customizing landing page sections.

## Quick Start

```bash
# From scenario root (scenarios/landing-manager/)

# List all sections
./scripts/manage-sections.sh list

# Validate consistency (run after any changes!)
./scripts/manage-sections.sh validate

# Get help adding a new section
./scripts/manage-sections.sh add <section-id>

# Get info about a section
./scripts/manage-sections.sh info hero
```

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                    SINGLE SOURCE OF TRUTH                          │
│              api/templates/sections/_index.json                     │
│                                                                     │
│  Contains: All section metadata, paths, status, content fields     │
└─────────────────────────────────────────────────────────────────────┘
                                  │
          ┌───────────────────────┼───────────────────────┐
          │                       │                       │
          ▼                       ▼                       ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│  Section Schema │   │ React Component │   │   Data Layer    │
│   {id}.json     │   │ {Name}Section   │   │                 │
│                 │   │     .tsx        │   │  - api.ts types │
│ Defines fields, │   │                 │   │  - schema.sql   │
│ validation,     │   │ Renders the     │   │  - PublicHome   │
│ constraints     │   │ section UI      │   │    .tsx switch  │
└─────────────────┘   └─────────────────┘   └─────────────────┘
```

## Files You'll Touch

| Purpose | Path | When to Modify |
|---------|------|----------------|
| **Section Registry** | `api/templates/sections/_index.json` | Add/remove any section |
| **Section Schemas** | `api/templates/sections/{id}.json` | Add/modify section fields |
| **React Components** | `generated/test/ui/src/components/sections/{Name}Section.tsx` | Add/modify section UI |
| **Section Renderer** | `generated/test/ui/src/pages/PublicHome.tsx` | Add/remove section imports & switch cases |
| **TypeScript Types** | `generated/test/ui/src/lib/api.ts` | Add/remove from SectionType union |
| **Database Schema** | `initialization/postgres/schema.sql` | Add/remove from CHECK constraint (~line 88) |

## Adding a New Section (Step-by-Step)

### 1. Run the helper script
```bash
./scripts/manage-sections.sh add my-new-section
```
This prints templates and instructions for all files you need to modify.

### 2. Create the schema file
Create `api/templates/sections/my-new-section.json`:
```json
{
  "$schema": "./_schema.json",
  "id": "my-new-section",
  "name": "My New Section",
  "description": "What this section does",
  "category": "value-proposition",
  "fields": {
    "title": {
      "type": "string",
      "label": "Section Title",
      "required": true,
      "max_length": 80
    }
  },
  "ui_component": "MyNewSectionSection",
  "default_order": 50
}
```

### 3. Create the component
Create `generated/test/ui/src/components/sections/MyNewSectionSection.tsx`:
```tsx
interface MyNewSectionSectionProps {
  content: {
    title?: string;
  };
}

export function MyNewSectionSection({ content }: MyNewSectionSectionProps) {
  return (
    <section className="py-24 bg-slate-950">
      <div className="container mx-auto px-6">
        <h2 className="text-4xl font-bold text-white text-center">
          {content.title || 'Default Title'}
        </h2>
      </div>
    </section>
  );
}
```

### 4. Register in _index.json
Add to the `sections` array in `api/templates/sections/_index.json`:
```json
{
  "id": "my-new-section",
  "name": "My New Section",
  "description": "What this section does",
  "category": "value-proposition",
  "schema": "api/templates/sections/my-new-section.json",
  "component": "generated/test/ui/src/components/sections/MyNewSectionSection.tsx",
  "status": "implemented",
  "default_order": 50,
  "content_fields": ["title"]
}
```

### 5. Update PublicHome.tsx
Add import at top:
```tsx
import { MyNewSectionSection } from '../components/sections/MyNewSectionSection';
```

Add case in `renderSection()` switch (before `default:`):
```tsx
case 'my-new-section':
  return <MyNewSectionSection {...commonProps} />;
```

### 6. Update api.ts
Add to `SectionType` union:
```tsx
export type SectionType =
  | 'hero'
  | 'features'
  // ... existing types ...
  | 'my-new-section';  // Add here
```

### 7. Update schema.sql
Modify the CHECK constraint (around line 88):
```sql
section_type VARCHAR(50) NOT NULL CHECK (section_type IN ('hero', 'features', ..., 'my-new-section')),
```

### 8. Validate
```bash
./scripts/manage-sections.sh validate
```

## Removing a Section

```bash
# Get removal instructions
./scripts/manage-sections.sh remove <section-id>
```

Then:
1. Delete `api/templates/sections/{id}.json`
2. Delete `generated/test/ui/src/components/sections/{Name}Section.tsx`
3. Remove entry from `api/templates/sections/_index.json`
4. Remove import and switch case from `PublicHome.tsx`
5. Remove from `SectionType` union in `api.ts`
6. Remove from CHECK constraint in `schema.sql`
7. Run `./scripts/manage-sections.sh validate`

## Modifying Section Content

**Schema changes only** (new fields):
1. Edit `api/templates/sections/{id}.json`
2. Update `content_fields` in `_index.json`
3. Update component props interface
4. Update component to render new fields

**Styling changes only**:
1. Edit `generated/test/ui/src/components/sections/{Name}Section.tsx`

## Implemented Sections

| ID | Component | Category | Description |
|----|-----------|----------|-------------|
| `hero` | HeroSection | above-fold | Main headline, CTA |
| `features` | FeaturesSection | value-proposition | Product features grid |
| `video` | VideoSection | engagement | Embedded video |
| `pricing` | PricingSection | conversion | Pricing tiers |
| `testimonials` | TestimonialsSection | social-proof | Customer quotes |
| `faq` | FAQSection | trust-building | Q&A accordion |
| `cta` | CTASection | conversion | Call-to-action block |
| `footer` | FooterSection | navigation | Site footer |

## Schema-Only Sections (Need Components)

| ID | Component Needed | Category |
|----|------------------|----------|
| `benefits` | BenefitsSection | value-proposition |
| `social-proof` | SocialProofSection | social-proof |
| `lead-form` | LeadFormSection | conversion |
| `preview` | PreviewSection | engagement |

## Critical Rules

1. **Field names must match** - Schema field names MUST exactly match component prop names
2. **Always provide defaults** - Components should render gracefully with empty content
3. **Validate after changes** - Run `./scripts/manage-sections.sh validate`
4. **Keep sections independent** - Sections should not depend on each other
5. **Test your changes** - Run `make test` from scenario root

## Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Section ID | kebab-case | `social-proof` |
| Schema file | `{id}.json` | `social-proof.json` |
| Component file | `{PascalCase}Section.tsx` | `SocialProofSection.tsx` |
| Component name | `{PascalCase}Section` | `SocialProofSection` |
| DB type | Same as ID | `'social-proof'` |

## Content Storage

Sections store content as JSONB in PostgreSQL:
- Flexible: Any JSON structure
- Queryable: Can search within content
- Validated: By component at render time

## Common Tasks

### Add a field to existing section
1. Edit schema JSON to add field definition
2. Update `content_fields` in `_index.json`
3. Update component interface and rendering

### Change section styling
1. Edit component TSX file only
2. Use Tailwind classes
3. No other files need changes

### Reorder default sections
1. Edit `default_order` in schema JSON
2. Update `default_order` in `_index.json`
