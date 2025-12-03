---
title: "Agent Guide"
description: "Guide for AI agents customizing landing page sections"
category: "integration"
order: 2
audience: ["developers", "agents"]
---

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
│                                                                     │
│  Section Registry: generated/test/ui/src/components/sections/      │
│                    registry.tsx                                     │
│                                                                     │
│  Contains: Component mappings, types, metadata                      │
│  Auto-derives: SectionType union, rendering logic                  │
└─────────────────────────────────────────────────────────────────────┘
                                  │
          ┌───────────────────────┼───────────────────────┐
          │                       │                       │
          ▼                       ▼                       ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│  Section Schema │   │ React Component │   │   Database      │
│   {id}.json     │   │ {Name}Section   │   │                 │
│                 │   │     .tsx        │   │  - schema.sql   │
│ Defines fields, │   │                 │   │    CHECK        │
│ validation,     │   │ Renders the     │   │    constraint   │
│ constraints     │   │ section UI      │   │                 │
└─────────────────┘   └─────────────────┘   └─────────────────┘
```

## Adding a New Section (3 Steps Only!)

### Before: 6 files to update ❌
### After: 3 files to update ✅

The registry pattern eliminates:
- Switch statement updates
- TypeScript union updates
- PublicHome.tsx imports

### Step 1: Create the Component

Create `generated/test/ui/src/components/sections/MyNewSectionSection.tsx`:

```tsx
interface MyNewSectionSectionProps {
  content: {
    title?: string;
    // Add fields matching your schema
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

### Step 2: Register in Registry

Add to `generated/test/ui/src/components/sections/registry.tsx`:

```tsx
// Import at top
import { MyNewSectionSection } from './MyNewSectionSection';

// Add to SECTION_REGISTRY
'my-new-section': {
  component: MyNewSectionSection,
  name: 'My New Section',
  category: 'value-proposition',
  defaultOrder: 50,
},
```

**That's it for the UI!** The SectionType and PublicHome rendering are handled automatically.

### Step 3: Update Database Schema (if needed)

Add to `initialization/postgres/schema.sql` CHECK constraint (~line 88):

```sql
section_type VARCHAR(50) NOT NULL CHECK (section_type IN ('hero', ..., 'my-new-section')),
```

### Optional: Create Schema Definition

If you want field validation/documentation, create `api/templates/sections/my-new-section.json`:

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

And register in `api/templates/sections/_index.json`.

## Files You'll Touch

| Purpose | Path | When to Modify |
|---------|------|----------------|
| **Component Registry** | `generated/test/ui/src/components/sections/registry.tsx` | **Always** - register new sections here |
| **React Components** | `generated/test/ui/src/components/sections/{Name}Section.tsx` | Create/modify section UI |
| **Database Schema** | `initialization/postgres/schema.sql` | Add section type to CHECK constraint |
| **Section Schemas** (optional) | `api/templates/sections/{id}.json` | Document field definitions |
| **Section Index** (optional) | `api/templates/sections/_index.json` | Update if using schema |

## Files You DON'T Need to Touch

These files auto-derive from the registry:

| File | What's Auto-Derived |
|------|---------------------|
| `generated/test/ui/src/lib/api.ts` | `SectionType` union (re-exported from registry) |
| `generated/test/ui/src/pages/PublicHome.tsx` | Component rendering (uses `getSectionComponent()`) |

## Removing a Section

1. Remove from `SECTION_REGISTRY` in `registry.tsx`
2. Delete `generated/test/ui/src/components/sections/{Name}Section.tsx`
3. Remove from `schema.sql` CHECK constraint
4. (Optional) Delete `api/templates/sections/{id}.json`
5. (Optional) Remove from `_index.json`
6. Run `./scripts/manage-sections.sh validate`

## Modifying Section Content

**Styling changes only**:
- Edit the component TSX file
- No other files need changes

**Schema changes** (new fields):
- Update the component's props interface
- Update the component to render new fields
- (Optional) Update the JSON schema

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

## Critical Rules

1. **Register in registry.tsx** - This is the single source of truth for components
2. **Match section IDs** - Registry key must match DB section_type
3. **Provide defaults** - Components should render gracefully with empty content
4. **Keep sections independent** - Sections should not depend on each other
5. **Validate after changes** - Run `./scripts/manage-sections.sh validate`

## Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Section ID | kebab-case | `social-proof` |
| Registry key | Same as ID | `'social-proof'` |
| Component file | `{PascalCase}Section.tsx` | `SocialProofSection.tsx` |
| Component name | `{PascalCase}Section` | `SocialProofSection` |
| DB type | Same as ID | `'social-proof'` |

## Content Storage

Sections store content as JSONB in PostgreSQL:
- Flexible: Any JSON structure
- Queryable: Can search within content
- Validated: By component at render time

## Registry Pattern Benefits

The registry pattern (`registry.tsx`) provides:

1. **Single registration point** - One file to update for new sections
2. **Auto-derived types** - SectionType updates automatically
3. **No switch statements** - Components looked up dynamically
4. **Metadata co-location** - Component, name, category together
5. **Type safety** - Invalid section types caught at compile time
