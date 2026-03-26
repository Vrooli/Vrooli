# Brand Manager – Full Branding Lifecycle for All Scenarios

Manages the full branding lifecycle for all Vrooli scenarios — generating, storing, applying, and validating brand identity. Serves both human designers (via UI wizard) and autonomous agents (via CLI/API) equally.

## Architecture

**Two-layer branding:**
1. **Design Language File** (`docs/DESIGN_LANGUAGE.md` per scenario) — LLM-generated prose from structured brand data + user notes
2. **Brand Manager DB + Assets** — SQLite for metadata, filesystem for binary assets

**Three surfaces:** UI (React dashboard + wizard), CLI, REST API

See [Architecture](docs/concepts/ARCHITECTURE.md) for detailed system design and data flow.

## Tech Stack
- **API**: Go with SQLite (WAL mode)
- **UI**: React + Vite + Tailwind
- **CLI**: Go
- **AI**: Ollama-first with OpenRouter fallback (AIProviderChain pattern)
- **Storage**: `~/.vrooli/brand-manager/brand-manager.db` + `~/.vrooli/brand-manager/assets/{brand_id}/`

## Quick Start
```bash
cd scenarios/brand-manager
make start    # Start via lifecycle
make test     # Run tests
make logs     # View logs
make stop     # Stop
```

See [Quick Start Guide](docs/QUICKSTART.md) for a complete walkthrough.

## CLI Commands
```bash
# Health
brand-manager status                           # Check API health

# Brand CRUD
brand-manager create --name "Name"             # Create a new brand
brand-manager list [--name FILTER] [--limit N] # List brands with filtering
brand-manager get <id>                         # Get brand details
brand-manager update <id> --name "New"         # Update a brand
brand-manager delete <id>                      # Delete a brand
brand-manager versions <id>                    # View version history

# Assignments
brand-manager assign --brand <id> --scenario <name>  # Assign brand to scenario
brand-manager unassign <assignment-id>               # Remove assignment
brand-manager scenario-status <name>                 # Check scenario branding

# All commands support --json for machine-readable output
```

See [CLI Reference](docs/reference/cli-commands.md) for full documentation.

## API Endpoints
- `GET/POST /api/v1/brands` — Brand CRUD
- `GET/PUT/DELETE /api/v1/brands/{id}` — Single brand operations
- `GET /api/v1/brands/{id}/versions` — Version history
- `POST/DELETE /api/v1/assignments` — Brand-to-scenario assignments
- `GET /api/v1/scenarios/{name}/status` — Per-scenario branding status
- `POST /api/v1/contrast/check` — WCAG AA contrast ratio check (pair)
- `POST /api/v1/contrast/brand` — WCAG AA contrast validation (full palette)

**Features:**
- All mutating endpoints support `X-Dry-Run: true` header (runs validation without persisting)
- Request IDs via `X-Request-ID` header (auto-generated if not provided)
- Structured error responses: `{code, message, recovery}` on all errors

See [API Reference](docs/reference/api-endpoints.md) for full documentation.

## Documentation
- [Quick Start](docs/QUICKSTART.md) — Get running in 5 minutes
- [Architecture](docs/concepts/ARCHITECTURE.md) — System design and data flow
- [API Reference](docs/reference/api-endpoints.md) — REST API documentation
- [CLI Reference](docs/reference/cli-commands.md) — CLI commands
- [Configuration](docs/reference/configuration.md) — Environment variables and settings
- [PRD](PRD.md) — Product requirements and operational targets

### Internal (Developer/Agent)
- [Seams](docs/internal/SEAMS.md) — Module boundaries and testability zones
- [Storage Audit](docs/internal/STORAGE_AUDIT.md) — SQLite architecture assessment
- [Progress](docs/internal/PROGRESS.md) — Development history
- [Problems](docs/internal/PROBLEMS.md) — Open issues and deferred ideas
- [Research](docs/internal/RESEARCH.md) — Background research and references
