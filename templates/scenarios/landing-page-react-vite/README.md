# Landing Page Template (React + Vite)

A production-ready landing page template with A/B testing, Stripe payments, analytics, and an admin portal. Used by `template-manager` to generate monetizable landing pages for Vrooli scenarios.

Generated scenarios receive root-level `DESIGN.md` from the `vrooli-conversion-landing` design kit. Treat that file as the canonical landing-page design contract. `.vrooli/styling.json` and `.vrooli/style-packs/` are runtime/configuration layers that instantiate the contract for variants and admin customization.

## Features

- **React + Vite Frontend** - Modern SPA with public landing and admin portal
- **Go (Gin) API** - High-performance REST backend
- **A/B Testing** - Whole-page variant testing with analytics
- **Stripe Integration** - Subscriptions, one-time payments, credits
- **Admin Portal** - Content management without code changes
- **Agent Customization** - AI-driven landing page optimization
- **Canonical Design Contract** - Root `DESIGN.md` plus configurable style packs for conversion-focused variants

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

Port bands declared in `.vrooli/service.json` follow the platform policy: `API_PORT` in `15000-19999` and `UI_PORT` in `20000-24999`. All sit below Linux's ephemeral floor (32768). Add more ports only for separate listener processes. See [docs/reference/port-allocation.md](../../../docs/reference/port-allocation.md) before changing.

## CLI

- `.vrooli/service.json` is the source of truth for the CLI command, adapter kind, install strategies, and invocation contract.
- The template includes `cli/install.sh` and `cli/install.ps1` as adapter assets referenced by that manifest.
- Health check: `<scenario-id> status` (after install)
- `status` and `configure` come from `cli-core`; `status` hits root `/health`.

### CLI Extension Model

- This is the only recommended greenfield CLI shape for new scenarios. Do not start with flat `cmd_<domain>.go` files as the intended long-term architecture.
- `cli/main.go`: entrypoint only.
- `cli/app.go`: metadata + scaffold wiring only. Keep endpoint logic out of this file.
- `cli/domains/domains.go`: domain registration only.
- `cli/domains/<domain>/`: default place for command handlers, request/response shaping, and output formatting.
- Use `core.Get(...)` / `core.Request(...)` for versioned API routes and `core.GetRoot(...)` / `core.RequestRoot(...)` for root paths.
- Mark API-backed commands with `NeedsAPI: true` so stale-checking, token validation, and `--auto-start` keep working automatically.
- Prefer `SubcommandGroup` by default for real domains so the file layout mirrors scenario boundaries.
- Default human output should follow a command contract: operational commands use `Status -> Triage -> Next Steps`, list/read commands use `Summary -> Results -> Retrieval Hints`, and mutation commands use `Result -> What Changed -> Next Command`.
- Use `cliapp.RenderOperationalReport`, `RenderListReport`, and `RenderMutationReport` so output stays consistent across scenarios.
- When a command supports `--json`, render the same underlying structured report through `cliapp.PrintReportJSON(...)` instead of inventing a second output shape.

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
- `DESIGN.md` at the generated scenario root - canonical conversion landing-page design contract
- [Design System](docs/DESIGN_SYSTEM.md) - Runtime styling config and style-pack constraints
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
| `styling.json` | Runtime style-pack values that instantiate root `DESIGN.md` |
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
