# Landing Page Business Suite

A production-ready SaaS landing page with A/B testing, analytics, Stripe payments, and an admin portal for content management.

## What's Included

- **React + Vite Frontend**: Modern UI with public landing and admin portal
- **Go (Gin) API**: High-performance backend with PostgreSQL
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
| Public Landing | http://localhost:3000/ |
| Admin Portal | http://localhost:3000/admin |
| API Health | http://localhost:8080/health |

**Default Admin:** `admin@localhost` / `changeme123` (change immediately in production)

## Documentation

| Getting Started | Reference | Internal |
|-----------------|-----------|----------|
| [Quick Start](docs/QUICKSTART.md) | [API Reference](docs/reference/api/README.md) | [Architecture](docs/concepts/ARCHITECTURE.md) |
| [Admin Guide](docs/guides/ADMIN_GUIDE.md) | [Configuration](docs/guides/CONFIGURATION_GUIDE.md) | [Seams](docs/internal/SEAMS.md) |
| [Content Guide](docs/guides/CONTENT_GUIDE.md) | [Security](docs/reference/SECURITY.md) | [Progress](docs/internal/PROGRESS.md) |
| [Deployment](docs/guides/DEPLOYMENT.md) | [Stripe Webhooks](docs/reference/STRIPE_WEBHOOKS.md) | [Known Issues](docs/internal/PROBLEMS.md) |

**Full documentation:** See [docs/manifest.json](docs/manifest.json) for the complete navigation structure.

## Configuration

Key configuration files in `.vrooli/`:

| File | Purpose |
|------|---------|
| `service.json` | Lifecycle, ports, dependencies |
| `branding.json` | Site branding (name, logo, colors) |
| `styling.json` | Design tokens and CSS theming |
| `variant_space.json` | A/B testing axes (persona, JTBD, style) |
| `variants/*.json` | Variant content snapshots |
| `fallback/fallback.json` | Offline-safe landing payload |

## Tech Stack

**Frontend:** React 18, TypeScript, Vite, TailwindCSS, shadcn/ui, Lucide icons

**Backend:** Go 1.21+, Gin framework, PostgreSQL 14+

## Requirements

See [PRD.md](PRD.md) for the full product requirements document with operational targets.
