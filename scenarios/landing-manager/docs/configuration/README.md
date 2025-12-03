---
title: "Configuration Guide"
description: "Complete configuration reference for generated landing pages"
category: "reference"
order: 8
audience: ["developers"]
---

# Configuration Guide

This guide describes all configuration files in the `.vrooli/` directory of generated landing pages.

## Quick Reference

| File | Purpose | Edit Frequency |
|------|---------|----------------|
| `service.json` | Lifecycle, ports, dependencies | Rarely |
| `variant_space.json` | A/B testing axes | When adding segments |
| `styling.json` | Design system tokens | When changing visuals |
| `variants/*.json` | Fallback content | When updating defaults |
| `testing.json` | Test configuration | When adding test phases |
| `lighthouse.json` | Performance thresholds | When adjusting targets |

---

## service.json

Defines the scenario lifecycle, ports, and resource dependencies.

### Complete Schema

```json
{
  "version": "1.0.0",
  "service": {
    "name": "my-landing",
    "displayName": "My Landing Page",
    "description": "Landing page for my product",
    "type": "business-application",
    "category": "marketing",
    "tags": ["landing-page", "saas"],
    "version": "1.0.0"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999",
      "description": "Go API server"
    },
    "ui": {
      "env_var": "UI_PORT",
      "range": "35000-39999",
      "description": "React UI server"
    }
  },
  "lifecycle": {
    "version": "2.0.0",
    "health": {
      "startup_grace_period": "30s",
      "endpoints": [
        { "name": "api", "path": "/health", "port_env": "API_PORT" },
        { "name": "ui", "path": "/", "port_env": "UI_PORT" }
      ]
    },
    "setup": {
      "condition": "first_run_or_changed",
      "steps": [
        { "name": "install-deps", "command": "cd ui && pnpm install" },
        { "name": "build-api", "command": "cd api && go build -o landing-api ." },
        { "name": "init-db", "command": "psql $DATABASE_URL < initialization/postgres/schema.sql" }
      ]
    },
    "develop": {
      "steps": [
        { "name": "api", "command": "cd api && ./landing-api", "background": true },
        { "name": "ui", "command": "cd ui && pnpm run dev", "background": true }
      ]
    },
    "production": {
      "steps": [
        { "name": "build-ui", "command": "cd ui && pnpm run build" },
        { "name": "api", "command": "cd api && ./landing-api", "background": true },
        { "name": "ui", "command": "cd ui && node server.js", "background": true }
      ]
    },
    "stop": {
      "steps": [
        { "name": "stop-all", "command": "pkill -f landing-api || true" }
      ]
    }
  },
  "dependencies": {
    "resources": {
      "postgres": { "enabled": true, "required": true },
      "redis": { "enabled": false, "required": false }
    }
  }
}
```

### Key Fields

| Field | Description |
|-------|-------------|
| `service.name` | URL-safe identifier (used in paths) |
| `service.displayName` | Human-readable name |
| `ports.*.range` | Port range for auto-allocation |
| `lifecycle.health.startup_grace_period` | Time to wait before health checks |
| `dependencies.resources.*.required` | If true, scenario won't start without it |

---

## variant_space.json

Defines the dimensions (axes) for A/B testing variants.

### Complete Schema

```json
{
  "_name": "landingPageVariantSpace",
  "_description": "Defines the variant space for landing page A/B testing",
  "axes": {
    "persona": {
      "description": "Target audience segment",
      "variants": [
        {
          "id": "ops_leader",
          "label": "Operations Leader",
          "description": "Director/VP of Operations running multi-scenario deployments",
          "defaultWeight": 0.4
        },
        {
          "id": "automation_freelancer",
          "label": "Automation Freelancer",
          "description": "Solo builders and agency developers",
          "defaultWeight": 0.3
        },
        {
          "id": "product_marketer",
          "label": "Product Marketer",
          "description": "GTM and marketing leads",
          "defaultWeight": 0.3
        }
      ]
    },
    "jtbd": {
      "description": "Job to be done - what problem are they solving?",
      "variants": [
        {
          "id": "launch_bundle",
          "label": "Launch Bundle",
          "description": "First production deployment"
        },
        {
          "id": "scale_services",
          "label": "Scale Services",
          "description": "Turn automations into products"
        },
        {
          "id": "improve_conversions",
          "label": "Improve Conversions",
          "description": "Optimize existing flows"
        }
      ]
    },
    "conversionStyle": {
      "description": "How they prefer to buy",
      "variants": [
        {
          "id": "self_serve",
          "label": "Self-Serve",
          "description": "Direct checkout without sales contact"
        },
        {
          "id": "demo_led",
          "label": "Demo Led",
          "description": "Prefers to see a demo first"
        },
        {
          "id": "founder_led",
          "label": "Founder Led",
          "description": "Wants to talk to the founder"
        }
      ]
    }
  },
  "constraints": {
    "disallowedCombinations": [
      {
        "persona": "automation_freelancer",
        "conversionStyle": "demo_led",
        "jtbd": "improve_conversions",
        "reason": "Freelancers rarely request demos for optimization work"
      }
    ]
  }
}
```

### How Axes Work

Each variant is positioned in a 3D space:

```
                    Persona
                       │
                       │  ops_leader
                       │  ●────────────────┐
                       │                   │
    automation_        │                   │ JTBD
    freelancer ●───────┼───────────────────┼──────▶
                       │                   │
                       │  ●────────────────┘
                       │  product_marketer
                       │
                       ▼
               Conversion Style
```

See [Concepts - A/B Testing](../CONCEPTS.md#ab-testing-system) for the complete variant selection flow.

---

## styling.json

Defines the visual design system tokens used by the UI.

### Complete Schema

```json
{
  "_name": "landingPageStyling",
  "_description": "Design tokens for the landing page",
  "palette": {
    "background": "#07090F",
    "background_secondary": "#0F1219",
    "accent_primary": "#F97316",
    "accent_secondary": "#3B82F6",
    "text_primary": "#F3F4F6",
    "text_secondary": "#9CA3AF",
    "text_muted": "#6B7280",
    "border": "#1F2937",
    "success": "#10B981",
    "warning": "#F59E0B",
    "error": "#EF4444"
  },
  "typography": {
    "headline": {
      "font": "Space Grotesk",
      "weight": 600,
      "letterSpacing": "-0.02em"
    },
    "subheadline": {
      "font": "Space Grotesk",
      "weight": 500,
      "letterSpacing": "-0.01em"
    },
    "body": {
      "font": "Inter",
      "weight": 400,
      "size": "17px",
      "lineHeight": 1.6
    },
    "caption": {
      "font": "Inter",
      "weight": 400,
      "size": "14px"
    }
  },
  "spacing": {
    "section_padding": "96px",
    "container_max_width": "1280px",
    "card_padding": "24px"
  },
  "components": {
    "buttons": {
      "primary": {
        "style": "solid",
        "background": "accent_primary",
        "text": "white",
        "borderRadius": "8px",
        "padding": "12px 24px"
      },
      "secondary": {
        "style": "outline",
        "border": "accent_primary",
        "text": "accent_primary",
        "borderRadius": "8px"
      },
      "ghost": {
        "style": "text",
        "text": "text_secondary"
      }
    },
    "cards": {
      "background": "background_secondary",
      "border": "border",
      "borderRadius": "12px",
      "shadow": "0 4px 6px -1px rgba(0, 0, 0, 0.1)"
    }
  },
  "effects": {
    "gradient_hero": "linear-gradient(135deg, #07090F 0%, #1a1a2e 100%)",
    "gradient_accent": "linear-gradient(90deg, #F97316 0%, #FB923C 100%)",
    "blur_backdrop": "blur(12px)"
  }
}
```

### Using Design Tokens

In your React components, access tokens via CSS variables:

```tsx
// Tokens are available as CSS variables
<div className="bg-[var(--background)] text-[var(--text-primary)]">
  <h1 className="font-[var(--font-headline)]">Headline</h1>
</div>
```

Or use the Tailwind theme extensions configured in `tailwind.config.ts`.

---

## Fallback Variants

`variants/fallback.json` provides content when the API is unavailable, ensuring visitors always see something.

### Complete Schema

```json
{
  "variant": {
    "slug": "fallback",
    "name": "Fallback Variant",
    "status": "active"
  },
  "sections": [
    {
      "section_type": "hero",
      "order": 1,
      "enabled": true,
      "content": {
        "headline": "Welcome to Our Product",
        "subheadline": "The best solution for your needs",
        "cta_text": "Get Started",
        "cta_url": "/signup"
      }
    },
    {
      "section_type": "features",
      "order": 2,
      "enabled": true,
      "content": {
        "title": "Features",
        "items": [
          {
            "icon": "Zap",
            "title": "Fast",
            "description": "Lightning-fast performance"
          },
          {
            "icon": "Shield",
            "title": "Secure",
            "description": "Enterprise-grade security"
          },
          {
            "icon": "BarChart",
            "title": "Analytics",
            "description": "Built-in analytics dashboard"
          }
        ]
      }
    },
    {
      "section_type": "cta",
      "order": 3,
      "enabled": true,
      "content": {
        "headline": "Ready to get started?",
        "cta_text": "Sign Up Free",
        "cta_url": "/signup"
      }
    },
    {
      "section_type": "footer",
      "order": 4,
      "enabled": true,
      "content": {
        "copyright": "© 2024 Your Company",
        "links": []
      }
    }
  ],
  "branding": {
    "site_name": "Your Product",
    "tagline": "Your tagline here"
  }
}
```

### When Fallback is Used

The UI loads fallback content when:
- API is unreachable (network error)
- API returns 5xx error
- API response times out (>5 seconds)

---

## testing.json

Configuration for the test suite.

### Schema

```json
{
  "phases": {
    "unit": {
      "enabled": true,
      "command": "cd api && go test ./... -v",
      "timeout": "5m"
    },
    "integration": {
      "enabled": true,
      "command": "cd test && bats integration.bats",
      "timeout": "10m"
    },
    "e2e": {
      "enabled": false,
      "command": "cd ui && pnpm run test:e2e",
      "timeout": "15m"
    }
  },
  "coverage": {
    "minimum": 70,
    "exclude": ["**/test/**", "**/mocks/**"]
  }
}
```

---

## lighthouse.json

Performance and accessibility thresholds.

### Schema

```json
{
  "thresholds": {
    "performance": 80,
    "accessibility": 90,
    "best-practices": 90,
    "seo": 90
  },
  "config": {
    "formFactor": "desktop",
    "throttling": {
      "rttMs": 40,
      "throughputKbps": 10240,
      "cpuSlowdownMultiplier": 1
    }
  },
  "urls": [
    { "path": "/", "name": "Landing Page" },
    { "path": "/admin", "name": "Admin Portal" }
  ]
}
```

---

## Environment Variables

The lifecycle system sets these environment variables automatically:

| Variable | Description | Example |
|----------|-------------|---------|
| `API_PORT` | Go API server port | `15432` |
| `UI_PORT` | React UI port | `35432` |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@localhost:5432/mydb` |
| `STRIPE_PUBLISHABLE_KEY` | Stripe public key | `pk_test_...` |
| `STRIPE_SECRET_KEY` | Stripe secret key | `sk_test_...` |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook signing secret | `whsec_...` |
| `VROOLI_LIFECYCLE_MANAGED` | Whether Vrooli manages lifecycle | `true` |
| `VROOLI_SCENARIO_NAME` | Current scenario name | `my-landing` |
| `VROOLI_SCENARIO_PATH` | Absolute path to scenario | `/home/.../scenarios/my-landing` |

### Setting Custom Variables

Add custom environment variables in `initialization/configuration/<scenario-name>.env`:

```bash
# Custom variables
ANALYTICS_ENABLED=true
CUSTOM_DOMAIN=https://example.com
FEATURE_FLAG_NEW_PRICING=false
```

---

## Database Configuration

Database schema is defined in `initialization/postgres/schema.sql`. Key tables:

| Table | Purpose |
|-------|---------|
| `variants` | A/B test variants |
| `sections` | Landing page content sections |
| `events` | Analytics events |
| `subscriptions` | Stripe subscription records |
| `admin_users` | Admin portal authentication |
| `branding` | Site branding settings |

### Connection

The `DATABASE_URL` is constructed from PostgreSQL resource settings:

```
postgres://{user}:{password}@{host}:{port}/{database}
```

Default database name matches the scenario slug.

---

## Common Configuration Tasks

### Change the Primary Accent Color

1. Edit `styling.json`:
   ```json
   "palette": {
     "accent_primary": "#your-color-hex"
   }
   ```

2. Rebuild UI: `cd ui && pnpm run build`

### Add a New A/B Test Segment

1. Edit `variant_space.json`, add to appropriate axis
2. Create new variant in admin portal or via API
3. Assign sections to new variant

### Increase Performance Threshold

1. Edit `lighthouse.json`:
   ```json
   "thresholds": {
     "performance": 90
   }
   ```

2. Run `make test` to verify

### Add Stripe Keys

1. Set in `initialization/configuration/<name>.env`:
   ```bash
   STRIPE_PUBLISHABLE_KEY=pk_live_...
   STRIPE_SECRET_KEY=sk_live_...
   STRIPE_WEBHOOK_SECRET=whsec_...
   ```

2. Restart scenario: `make stop && make start`

---

## See Also

- [Concepts](../CONCEPTS.md) - Architecture overview
- [Admin Guide](../ADMIN_GUIDE.md) - Using the admin portal
- [API Reference](../api/README.md) - REST API documentation
