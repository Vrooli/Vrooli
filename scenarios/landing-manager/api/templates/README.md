# Landing Page Templates

This directory contains the template system for Landing Manager - a factory that generates landing page scenarios.

## For Agents

**Start here**: Run the section management script from scenario root:
```bash
./scripts/manage-sections.sh help      # See all commands
./scripts/manage-sections.sh list      # List all sections
./scripts/manage-sections.sh validate  # Check consistency
```

**Full guide**: See [`AGENT.md`](./AGENT.md) for comprehensive instructions including:
- Step-by-step guides for adding, removing, and modifying sections
- Component templates and examples
- All file paths and their purposes
- Critical rules and architecture overview

**Single Source of Truth**: [`sections/_index.json`](./sections/_index.json) contains all section metadata with full paths.

## Structure

```
templates/
├── catalog.json              # Template registry (entry point)
├── README.md                 # This file
├── sections/                 # Reusable section building blocks
│   ├── _schema.json          # JSON schema for section definitions
│   ├── _index.json           # Section registry (single source of truth)
│   ├── hero.json             # Hero section (above-fold)
│   ├── features.json         # Features grid (value-proposition)
│   ├── pricing.json          # Pricing tiers (conversion)
│   ├── cta.json              # Call-to-action (conversion)
│   ├── testimonials.json     # Customer testimonials (social-proof)
│   ├── faq.json              # FAQ accordion (trust-building)
│   ├── footer.json           # Site footer (navigation)
│   ├── video.json            # Video embed (engagement)
│   ├── social-proof.json     # Logos/trust badges (social-proof)
│   ├── lead-form.json        # Lead capture form (conversion)
│   ├── benefits.json         # Benefits list (value-proposition)
│   └── preview.json          # Content preview (engagement)
└── catalog/                  # Templates organized by category
    ├── saas/
    │   └── landing-page.json
    └── lead-generation/
        └── lead-magnet.json
```

## How It Works

### Templates Compose Sections

Templates don't define section content inline. They reference reusable sections via `$ref`:

```json
{
  "sections": {
    "required": [
      { "$ref": "../../sections/hero.json", "id": "hero" },
      { "$ref": "../../sections/features.json", "id": "features" }
    ],
    "optional": [
      { "$ref": "../../sections/faq.json", "id": "faq" }
    ]
  }
}
```

### Section Categories

| Category | Purpose | Examples |
|----------|---------|----------|
| `above-fold` | First thing visitors see | hero |
| `value-proposition` | Communicate product value | features, benefits |
| `social-proof` | Build credibility | testimonials, social-proof |
| `conversion` | Drive action | pricing, cta, lead-form |
| `trust-building` | Answer objections | faq |
| `engagement` | Increase time on page | video, preview |
| `navigation` | Site structure | footer |

### Implementation Status

Check `sections/_index.json` for current status of each section:
- `implemented` - Both schema and React component exist
- `schema-only` - Schema exists, component needs creation

## Related Files

When working with sections, these files may need updates (paths relative to scenario root):

| File | Purpose | When to Update |
|------|---------|----------------|
| `api/templates/sections/_index.json` | Registry | Add/remove sections |
| `api/templates/sections/{id}.json` | Schema | Change section fields |
| `generated/test/ui/src/components/sections/{Name}Section.tsx` | Component | Change rendering |
| `generated/test/ui/src/pages/PublicHome.tsx` | Renderer | Add/remove section types |
| `generated/test/ui/src/lib/api.ts` | Types | Add/remove section types |
| `initialization/postgres/schema.sql` | DB | Add/remove section types |
