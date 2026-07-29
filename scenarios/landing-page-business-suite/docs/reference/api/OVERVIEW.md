# API Overview

A SaaS landing page and operations suite with A/B testing, analytics, Stripe payments, subscriptions, credits, and an admin portal for content management.

## What's Included

- **React + Vite Frontend**: Modern UI with public landing and admin portal
- **Go HTTP API**: Backend service with PostgreSQL
- **A/B Testing**: Whole-page variant testing with weight-based traffic allocation
- **Stripe Integration**: Subscriptions, one-time payments, credits, customer portal
- **Analytics Dashboard**: Page views, clicks, conversions, per-variant metrics
- **Admin Portal**: Content management without code changes

## Quick Start

```bash
# Start the scenario
make start

# View logs
make logs

# Stop
make stop
```

**Access Points:**
| Surface | URL |
|---------|-----|
| Public Landing | `http://localhost:${UI_PORT}/` |
| Admin Portal | `http://localhost:${UI_PORT}/admin` |
| API Health | `http://localhost:${API_PORT}/health` |

Vrooli allocates `API_PORT` and `UI_PORT` during lifecycle startup. Configure
administrator credentials through the scenario's supported secret/configuration
surface; do not rely on development defaults in a deployed environment.

## Documentation

| Getting Started | Reference | Internal |
|-----------------|-----------|----------|
| [Quick Start](../../QUICKSTART.md) | [API Reference](OVERVIEW.md) | [Architecture](../../concepts/ARCHITECTURE.md) |
| [Admin Guide](../../guides/ADMIN_GUIDE.md) | [Configuration](../../guides/CONFIGURATION_GUIDE.md) | [Seams](../../internal/SEAMS.md) |
| [Content Guide](../../guides/CONTENT_GUIDE.md) | [Security](../SECURITY.md) | [Progress](../../internal/PROGRESS.md) |
| [Deployment](../../guides/DEPLOYMENT.md) | [Stripe Webhooks](../STRIPE_WEBHOOKS.md) | [Known Issues](../../internal/PROBLEMS.md) |

**Full documentation:** See [the documentation manifest](../../manifest.json) for the complete navigation structure.

## Configuration

Key configuration files:

| File | Purpose |
|------|---------|
| `.vrooli/service.json` | Lifecycle, ports, dependencies |
| `config/branding.json` | Site branding (name, logo, colors) |
| `.vrooli/styling.json` | Design tokens and CSS theming |
| `config/variant_space.json` | A/B testing axes (persona, JTBD, style) |
| `config/variants/*.json` | Variant content snapshots |
| `.vrooli/fallback/fallback.json` | Offline-safe landing payload |

## Tech Stack

**Frontend:** React 18, TypeScript, Vite, TailwindCSS, shadcn/ui, Lucide icons

**Backend:** Go, `net/http`, PostgreSQL

## Requirements

See [PRD.md](../../../PRD.md) for the full product requirements document with operational targets.
