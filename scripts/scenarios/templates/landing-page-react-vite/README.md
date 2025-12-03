# Landing Page Template (React + Vite)

A production-ready landing page template with A/B testing, Stripe payments, analytics, and an admin portal. Used by `landing-manager` to generate monetizable landing pages for Vrooli scenarios.

## Features

- **React + Vite Frontend** - Modern SPA with public landing and admin portal
- **Go (Gin) API** - High-performance REST backend
- **A/B Testing** - Whole-page variant testing with analytics
- **Stripe Integration** - Subscriptions, one-time payments, credits
- **Admin Portal** - Content management without code changes
- **Agent Customization** - AI-driven landing page optimization

## Quick Start

```bash
# If you have a generated scenario:
cd scenarios/<your-scenario>
make start

# Access your landing page:
# Public:  http://localhost:<UI_PORT>/
# Admin:   http://localhost:<UI_PORT>/admin
# Health:  http://localhost:<API_PORT>/health
```

See [QUICKSTART.md](docs/QUICKSTART.md) for detailed first-time setup.

## Documentation

| Document | Description |
|----------|-------------|
| [Documentation Index](docs/index.md) | Complete documentation hub |
| [Quick Start Guide](docs/QUICKSTART.md) | 5-minute first landing page |
| [Admin Guide](docs/ADMIN_GUIDE.md) | Managing content and A/B tests |
| [API Reference](docs/api/README.md) | REST API documentation |
| [Configuration Guide](docs/CONFIGURATION_GUIDE.md) | All config file reference |
| [Architecture](docs/ARCHITECTURE.md) | System design and components |

### For Different Audiences

**Marketers & Content Authors:**
- [Admin Guide](docs/ADMIN_GUIDE.md) - Portal usage
- [Content Guide](docs/CONTENT_GUIDE.md) - Writing effective copy

**Developers:**
- [Architecture](docs/ARCHITECTURE.md) - System design
- [API Reference](docs/api/README.md) - Endpoints
- [Seams & Testability](docs/SEAMS.md) - Code organization

**AI Agents:**
- [Design System](docs/DESIGN_SYSTEM.md) - Styling constraints
- [Configuration Guide](docs/CONFIGURATION_GUIDE.md) - File formats

## Directory Structure

```
landing-page-react-vite/
├── api/                    # Go backend (Gin framework)
│   ├── *_handlers.go       # HTTP route handlers
│   ├── *_service.go        # Business logic
│   └── initialization/     # DB schema and migrations
├── ui/                     # React + Vite frontend
│   └── src/
│       ├── surfaces/
│       │   ├── public-landing/   # Visitor-facing pages
│       │   └── admin-portal/     # Admin interface
│       └── shared/               # Common utilities
├── .vrooli/                # Configuration files
│   ├── service.json        # Lifecycle configuration
│   ├── styling.json        # Design tokens
│   ├── variant_space.json  # A/B testing axes
│   └── variants/           # Fallback content
├── docs/                   # Documentation
├── requirements/           # Feature specifications
└── test/                   # Test suites
```

## Configuration

Key configuration files in `.vrooli/`:

| File | Purpose |
|------|---------|
| `service.json` | Ports, lifecycle, dependencies |
| `styling.json` | Colors, typography, components |
| `variant_space.json` | A/B testing dimensions |
| `variants/*.json` | Fallback/default content |

See [Configuration Guide](docs/CONFIGURATION_GUIDE.md) for details.

## Requirements

- **Go** 1.21+
- **Node.js** 18+ with pnpm
- **PostgreSQL** 14+
- **Vrooli CLI** (for lifecycle management)

## Related Documentation

- [PRD](PRD.md) - Product requirements and roadmap
- [Template Internals](TEMPLATE.md) - Deep dive for template contributors
- [Troubleshooting](docs/TROUBLESHOOTING.md) - Common issues
- [FAQ](docs/FAQ.md) - Frequently asked questions

## License

Part of the Vrooli project. See repository root for license.
