# Configuration Guide

This document describes all configuration files in the `.vrooli/` directory and how to customize them for your landing page.

## Configuration Files Overview

| File | Purpose | When to Edit |
|------|---------|--------------|
| `service.json` | Lifecycle, ports, dependencies | Rarely - deployment/infrastructure changes |
| `variant_space.json` | A/B testing axes (personas, JTBD, conversion styles) | When adding new audience segments or conversion strategies |
| `styling.json` | Design system (colors, typography, components) | When changing visual identity |
| `variants/*.json` | Fallback variant content | When updating offline/default content |
| `endpoints.json` | API endpoint inventory | Auto-generated, rarely manual edit |
| `schemas/*.json` | JSON Schema validation | When adding new section types or fields |
| `testing.json` | Test phase configuration | When adding new test phases |
| `lighthouse.json` | Performance thresholds | When adjusting performance targets |

---

## service.json

Defines the scenario lifecycle, ports, and resource dependencies.

### Structure

```json
{
  "$schema": "../../../../.vrooli/schemas/service.schema.json",
  "version": "1.0.0",
  "service": { ... },
  "ports": { ... },
  "lifecycle": { ... },
  "dependencies": { ... }
}
```

### service

Basic scenario metadata.

| Field | Type | Description |
|-------|------|-------------|
| `parent` | string | Parent service (always `"vrooli"`) |
| `name` | string | Scenario ID (use `landing-page-business-suite` placeholder) |
| `displayName` | string | Human-readable name |
| `description` | string | Scenario description |
| `version` | string | Semantic version |
| `type` | string | `"application-template"` for landing pages |
| `category` | string | Category for discovery |
| `tags` | string[] | Search tags |

### ports

Port allocation for services.

```json
{
  "api": {
    "env_var": "API_PORT",
    "description": "Go API server",
    "range": "15000-19999"
  },
  "ui": {
    "env_var": "UI_PORT",
    "description": "React UI",
    "range": "35000-39999"
  }
}
```

The lifecycle system allocates ports from the specified range and exports them as environment variables.

### lifecycle

Defines setup, develop, test, production, and stop phases.

```json
{
  "version": "2.0.0",
  "health": {
    "startup_grace_period": 15000,
    "endpoints": {
      "api": "/health",
      "ui": "/health"
    },
    "checks": [
      {
        "name": "api_endpoint",
        "type": "http",
        "target": "http://localhost:${API_PORT}/health",
        "timeout": 10000,
        "interval": 30000,
        "critical": true
      }
    ]
  },
  "setup": {
    "steps": [
      { "name": "build-api", "run": "cd api && go build -o landing-page-business-suite-api ." }
    ]
  },
  "develop": {
    "steps": [
      { "name": "start-api", "run": "cd api && exec ./landing-page-business-suite-api", "background": true }
    ]
  }
}
```

**Lifecycle phases:**
- `setup` - Build binaries, install dependencies, apply database schema
- `develop` - Start services for development
- `production` - Start services with production settings
- `test` - Run test suite
- `stop` - Stop background processes

### dependencies.resources

Resource requirements for the scenario.

```json
{
  "postgres": {
    "type": "postgres",
    "enabled": true,
    "required": true,
    "description": "Primary data store",
    "schema": "landing-page-business-suite"
  },
  "redis": {
    "type": "redis",
    "enabled": false,
    "required": false,
    "description": "Session caching (optional)"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Resource type (`postgres`, `redis`, `qdrant`, `resource`) |
| `enabled` | boolean | Whether to install on setup |
| `required` | boolean | Whether scenario can run without it |
| `description` | string | Purpose description |

---

## variant_space.json

Defines the A/B testing dimension space. Each axis represents a dimension along which variants can differ.

### Structure

```json
{
  "_name": "landingPageVariantSpace",
  "_schemaVersion": 1,
  "_note": "Axis definitions for AI agents",
  "_agentGuidelines": [
    "Pick exactly one variant per axis when proposing a new landing page."
  ],
  "axes": {
    "persona": { ... },
    "jtbd": { ... },
    "conversionStyle": { ... }
  },
  "constraints": {
    "disallowedCombinations": [ ... ]
  }
}
```

### Axis Definition

Each axis contains an array of variants:

```json
{
  "persona": {
    "_note": "Primary buyer persona that the copy should target.",
    "variants": [
      {
        "id": "ops_leader",
        "label": "Operations Leader",
        "description": "Director/VP of Operations running multi-scenario deployments.",
        "examples": {
          "headline": "Standardize automation launches across every team",
          "tagline": "Ops can spin up bundles with governance already baked in."
        },
        "defaultWeight": 0.4,
        "tags": ["b2b", "enterprise"],
        "status": "active",
        "agentHints": [
          "Emphasize governance, repeatability, and compliance."
        ]
      }
    ]
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (used in database) |
| `label` | string | Human-readable display name |
| `description` | string | Detailed description for agents/admins |
| `examples` | object | Example copy for this persona/axis |
| `defaultWeight` | number | Default traffic allocation (0.0-1.0) |
| `tags` | string[] | Categorization tags |
| `status` | string | `"active"` or `"experimental"` |
| `agentHints` | string[] | Instructions for AI agents |

### Standard Axes

**persona** - Target buyer persona
- `ops_leader` - Enterprise operations director
- `automation_freelancer` - Solo builders/agencies
- `product_marketer` - GTM/marketing leads

**jtbd** - Jobs to be done
- `launch_bundle` - First production deployment
- `scale_services` - Turn automations into products
- `improve_conversions` - Optimize existing flows

**conversionStyle** - Page conversion motion
- `self_serve` - Direct checkout
- `demo_led` - Book a demo
- `founder_led` - Founder/operator call

### Constraints

Prevent invalid axis combinations:

```json
{
  "disallowedCombinations": [
    {
      "persona": "automation_freelancer",
      "conversionStyle": "demo_led",
      "jtbd": "improve_conversions"
    }
  ]
}
```

---

## styling.json

Defines the visual design system. React components read these values for consistent styling.

### Top-Level Fields

| Field | Type | Description |
|-------|------|-------------|
| `style_id` | string | Unique identifier for this style pack |
| `brand` | object | Product branding info |
| `influences` | string[] | Design inspirations |
| `mood` | string[] | Emotional keywords |
| `palette` | object | Color tokens |
| `typography` | object | Font settings |
| `tone` | object | Voice and messaging rules |
| `layout` | object | Grid and spacing |
| `depth` | object | Shadows and borders |
| `motion` | object | Animation settings |
| `imagery` | object | Image guidelines |
| `components` | object | Component styling |
| `component_kits` | object | Pre-built component patterns |
| `imagery_slots` | object | Required image assets |
| `section_variants` | object | Per-section styling |
| `variant_guidance` | object | Per-variant styling hints |
| `usage_notes` | string[] | General design guidelines |
| `randomization` | object | Style randomizer metadata |

### palette

Color tokens used throughout the design:

```json
{
  "background": "#07090F",
  "gradients": ["#101828", "#1A2236"],
  "supporting": ["#F97316", "#22D3EE", "#10B981"],
  "text_primary": "#F3F4F6",
  "text_secondary": "#94A3B8",
  "surface_primary": "#0F172A",
  "surface_muted": "#1E2433",
  "surface_alt": "#F6F5F2",
  "card_border": "rgba(255,255,255,0.08)",
  "accent_primary": "#F97316",
  "accent_secondary": "#38BDF8",
  "accent_alert": "#EF4444",
  "neutral_scale": ["#090B14", "#111827", "#1F2937", "#374151", "#94A3B8"]
}
```

### typography

Font configuration:

```json
{
  "headline": {
    "font": "Space Grotesk",
    "weight": 600,
    "tracking": "-0.02em",
    "case": "Sentence"
  },
  "body": {
    "font": "Inter",
    "weight": 400,
    "size": "17px",
    "line_height": "1.65"
  },
  "cta": {
    "font": "Space Grotesk",
    "weight": 600,
    "transform": "none"
  },
  "scale": {
    "h1": "64px/1.1",
    "h2": "48px/1.15",
    "h3": "32px/1.2",
    "eyebrow": "12px/1 uppercase",
    "body": "17px/1.65",
    "caption": "14px/1.5"
  }
}
```

### tone

Voice and messaging guidelines for copy:

```json
{
  "voice": "Measured operator with a product-case-study bias",
  "keywords": ["process", "case study", "dashboard", "workflow"],
  "avoid": ["vague 'AI magic' claims", "gradient-orb filler copy"],
  "sentence_rules": [
    "State the outcome first, then mention the subsystem enabling it.",
    "Tie every feature to a concrete workflow."
  ]
}
```

### layout

Grid and spacing configuration:

```json
{
  "container_max_width": "1200px",
  "grid": {
    "columns": 12,
    "gutter": "32px"
  },
  "section_spacing": "160px",
  "hero": {
    "type": "case-study",
    "columns": 2,
    "includes_process_diagram": true,
    "ui_focus": "layered_panels"
  },
  "section_sequence": ["hero", "metrics", "full-preview", "wireframes", "features", "pricing", "cta"]
}
```

### components

Styling for individual component types:

```json
{
  "buttons": {
    "primary": {
      "style": "Solid accent_primary with 2px outline on dark surfaces",
      "icon": "ArrowRight",
      "padding": "px-7 py-4",
      "corners": "9999px pill"
    },
    "secondary": {
      "style": "Surface_muted background, 1px card_border",
      "usage": "Preview links or supporting CTAs"
    }
  },
  "cards": {
    "border": "1px solid rgba(255,255,255,0.08)",
    "hover": "translateY(-4px) and border-color accent_secondary",
    "background": "surface_muted with 92% opacity"
  },
  "icons": {
    "set": "Lucide",
    "default_color": "#F97316",
    "size": 24
  }
}
```

### variant_guidance

Per-variant styling hints:

```json
{
  "control": {
    "promise": "Audit + ship automation bundles",
    "primary_cta": "Book a live review",
    "secondary_cta": "Download runbook",
    "color_story": "Use accent_primary for hero CTA, accent_secondary for metrics.",
    "recommended_components": ["hero panels", "brand guidelines strip"],
    "notes": ["Hero headline should read like a case-study title."]
  }
}
```

---

## variants/*.json

Fallback variant payloads used when the API is unavailable.

### fallback.json

Default content rendered when API fails or during initial load:

```json
{
  "variant": {
    "slug": "fallback",
    "name": "Fallback",
    "status": "active"
  },
  "sections": [
    {
      "section_type": "hero",
      "order": 1,
      "enabled": true,
      "content": {
        "headline": "...",
        "subheadline": "...",
        "cta_text": "...",
        "cta_url": "..."
      }
    }
  ],
  "pricing": { ... },
  "downloads": [ ... ]
}
```

### control.json

The control variant content (copied to database on seed).

---

## schemas/

JSON Schema files for validation.

### variant.schema.json

Validates variant payload structure including sections and axes.

### styling.schema.json

Validates styling.json structure with all design tokens.

### variant-space.schema.json

Validates variant_space.json axis definitions.

### sections/*.json

Per-section-type content schemas:
- `hero.schema.json`
- `features.schema.json`
- `pricing.schema.json`
- `cta.schema.json`
- `testimonials.schema.json`
- `faq.schema.json`
- `footer.schema.json`
- `video.schema.json`
- `downloads.schema.json`

---

## testing.json

Test phase configuration.

```json
{
  "phases": ["structure", "unit", "integration", "business", "performance"],
  "timeout": 300000,
  "parallel": false
}
```

---

## lighthouse.json

Lighthouse performance thresholds.

```json
{
  "thresholds": {
    "performance": 90,
    "accessibility": 90,
    "best-practices": 90,
    "seo": 90
  },
  "urls": ["/", "/admin"]
}
```

---

## Style Packs

Additional style packs are stored in `.vrooli/style-packs/`. To use a different style:

```bash
cp .vrooli/style-packs/<pack-name>.json .vrooli/styling.json
```

Each pack follows the same schema as `styling.json` but with different design tokens.

---

## Environment Variables

The lifecycle system exports these environment variables:

| Variable | Description |
|----------|-------------|
| `API_PORT` | Port for the Go API server |
| `UI_PORT` | Port for the React UI |
| `DATABASE_URL` | PostgreSQL connection string |
| `POSTGRES_HOST` | Database host (alternative) |
| `POSTGRES_PORT` | Database port (alternative) |
| `POSTGRES_USER` | Database user (alternative) |
| `POSTGRES_PASSWORD` | Database password (alternative) |
| `POSTGRES_DB` | Database name (alternative) |
| `VROOLI_LIFECYCLE_MANAGED` | Set to `"true"` when run via lifecycle |
| `ENABLE_ADMIN_RESET` | Set to `"true"` to enable demo data reset |

---

## Customization Checklist

When customizing a generated landing page:

1. **Update branding** - Edit `styling.json` > `brand` section
2. **Adjust colors** - Modify `styling.json` > `palette` tokens
3. **Change fonts** - Update `styling.json` > `typography` settings
4. **Define personas** - Edit `variant_space.json` > `axes.persona.variants`
5. **Set fallback content** - Update `variants/fallback.json` sections
6. **Configure resources** - Enable/disable in `service.json` > `dependencies.resources`

Always validate changes against the schemas in `schemas/` before deploying.
