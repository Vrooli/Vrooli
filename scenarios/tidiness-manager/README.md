# Tidiness Manager

> Central code cleanliness orchestrator combining static analysis, AI-powered scanning, and campaign-based tidiness enforcement

## Overview

Tidiness Manager prevents scenarios from decaying into unmaintainable chaos through:

- **Tidiness Scanning**: Fast maintainability findings for long files, complexity, duplication, technical debt, and coupling
- **Light Scanning**: Backward-compatible metrics collection and parser support for stored issue workflows
- **Smart Scanning**: AI-powered deep analysis using resource-claude-code for refactoring suggestions
- **Campaign Management**: Automatic, progressive tidiness campaigns using visited-tracker
- **Agent API**: HTTP/CLI interface for other agents to request tidiness recommendations
- **Human Dashboard**: React UI for managing campaigns, viewing issues, and triggering scans

## What You Get

Tidiness Manager gives agents and maintainers a focused maintainability system: scan endpoints, CLI commands, stored issues, prioritization, campaign workflows, and a dashboard for inspecting cleanup work. It does not own lint, type, or static-quality policy; Quality Health owns that contract.

## Quick Start

```bash
# Setup (build API, UI, install CLI)
vrooli scenario run tidiness-manager --setup

# Start development servers
make dev

# Check health
tidiness-manager status

# Scan a scenario for tidiness issues
tidiness-manager scan browser-automation-studio --type tidiness

# List issues
tidiness-manager issues browser-automation-studio --limit 10
```

## Architecture

```
tidiness-manager/
├── api/                    # Go API server
│   ├── cmd/server/        # Main entry point
│   ├── scanner/           # Light scanning (Makefile integration)
│   ├── analyzer/          # Smart scanning (AI integration)
│   └── campaigns/         # Campaign management
├── cli/                   # CLI wrapper for agent integration
├── ui/                    # React dashboard
│   ├── src/pages/        # Dashboard, scenario detail, campaign manager
│   └── src/components/   # File tables, issue viewers, scan controls
├── requirements/         # Requirements registry
└── docs/                # Concepts, reference, operations, and internal ledgers
```

## Core Capabilities

### Light Scanning
- Collects file metrics and supports legacy parser workflows
- Computes per-file line counts and flags files exceeding thresholds
- Completes in <60s for small scenarios, <120s for medium scenarios

### Tidiness Scanning
- Reports normalized maintainability findings from file size, TODO/FIXME/HACK density, coupling, complexity, and duplication metrics
- Emits agent guidance fields: why the issue matters, recommended remediation, and campaign grouping hints
- Excludes lint/type/static-quality contract findings; those belong to quality-health and Test Genie's `quality` phase

### Smart Scanning
- Uses resource-claude-code/resource-codes for AI analysis
- Detects dead code, duplication, complexity, style issues
- Batches files (configurable: 10 files/batch, 5 concurrent batches)
- Integrates with visited-tracker to prioritize unvisited/stale files
- Never analyzes same file twice per session

### Campaign Management
- Auto-campaigns run across up to K scenarios concurrently (default K=3)
- Configurable session limits, file batching, priority rules
- Auto-completes when all files visited or max sessions reached
- Pause/resume/terminate controls with error detection

### Agent API
- `GET /api/v1/scenarios/{name}/issues` - Query issues with filters
- `POST /api/v1/scenarios/{name}/scan` - Trigger light/smart scans
- `GET /api/v1/campaigns` - List campaigns
- CLI commands: `scan`, `issues`, `campaigns`, `status`

### Human Dashboard
- Global view: scenario table with issue counts, visit %, campaign status
- Scenario detail: file table (sortable, filterable) with metrics
- Issue management: mark resolved/ignored, view remediation steps
- Campaign controls: start/pause/stop campaigns, configure thresholds

## Integration Points

### Dependencies
- **postgres** (required): Issue storage, campaign state, scan history
- **redis** (optional): Caching for expensive operations
- **resource-claude-code** (optional): AI-powered analysis
- **visited-tracker** (optional): Campaign-based file tracking
- **code-smell** (optional): Pattern-based smell detection

### Consumers
- Development agents needing refactor guidance
- Maintenance scenarios (scenario-auditor, app-issue-tracker)
- CI/CD pipelines wanting code health metrics

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `API_PORT` | Go API server port | Assigned by lifecycle (15000-19999) |
| `UI_PORT` | React UI port | Assigned by lifecycle (20000-24999) |
| `DATABASE_URL` | PostgreSQL connection | From postgres resource |
| `REDIS_URL` | Redis connection (optional) | From redis resource |
| `CLAUDE_CODE_CLI` | Path to resource-claude-code CLI | Auto-detected |

## CLI Commands

```bash
# Check API health
tidiness-manager status

# Trigger tidiness scan (default maintainability audit)
tidiness-manager scan <scenario> --type tidiness

# Trigger light scan (legacy metrics/parser path)
tidiness-manager scan <scenario> --type light [--wait]

# Trigger smart scan (slow, AI-powered)
tidiness-manager scan <scenario> --type smart [--wait]

# List tidiness issues
tidiness-manager issues <scenario> \
  [--category dead_code|duplication|length|complexity] \
  [--severity critical|high|medium|low] \
  [--limit N]

# List campaigns
tidiness-manager campaigns list

# Start auto-campaign for a scenario
tidiness-manager campaigns start <scenario> \
  [--max-sessions N] \
  [--files-per-session N]

# Pause/resume/stop campaign
tidiness-manager campaigns pause <scenario>
tidiness-manager campaigns resume <scenario>
tidiness-manager campaigns stop <scenario>
```

## Testing

```bash
# Full test suite
make test

# Specific phases
make test-unit          # Parser logic, ranking algorithms
make test-integration   # Makefile execution, AI mocking, API endpoints
make test-business      # End-to-end workflows, campaign lifecycles
make test-performance   # Scan timing, Lighthouse scores
```

See `requirements/README.md` for requirement tracking and validation strategy.

## Documentation Map

- [Start Here](docs/START-HERE.md) - orientation for agents and maintainers
- [Architecture](docs/concepts/ARCHITECTURE.md) - surfaces, boundaries, and data flow
- [Domains](docs/concepts/DOMAINS.md) - bounded contexts and ownership
- [API Endpoints](docs/reference/api-endpoints.md) - HTTP contract
- [CLI Commands](docs/reference/cli-commands.md) - installed command surface
- [Configuration](docs/reference/configuration.md) - lifecycle, env, and scenario contracts
- [Runbook](docs/operations/RUNBOOK.md) - operational commands and triage
- [Problems](docs/internal/PROBLEMS.md) - known issues and deferred work
- [Progress](docs/internal/PROGRESS.md) - historical implementation record

## Development Workflow

1. **Read the PRD** (`PRD.md`) to understand operational targets
2. **Check requirements** (`requirements/`) for detailed technical specs
3. **Start with light scanning** (foundation for all other modules)
4. **Tag tests** with `[REQ:TM-XX-NNN]` to link to requirements
5. **Update docs/internal/PROGRESS.md** when completing modules

## Configuration

### Configurable Thresholds
- Long file line count (default: 400-600 lines)
- Max scans per file (default: 3 visits)
- Max concurrent campaigns (default: K=3)
- AI batch size (default: 10 files/batch)
- Max concurrent AI batches (default: 5)

Stored in postgres `config` table; UI provides management interface.

## Links

- **PRD**: [PRD.md](PRD.md) - Operational targets and technical direction
- **Requirements**: [requirements/](requirements/) - 8 modules, 58 requirements
- **Research**: [docs/RESEARCH.md](docs/RESEARCH.md) - Uniqueness analysis
- **Progress**: [docs/internal/PROGRESS.md](docs/internal/PROGRESS.md) - Development log
- **Problems**: [docs/internal/PROBLEMS.md](docs/internal/PROBLEMS.md) - Known issues

## Customize Safely

- Keep lint, type checking, static-quality contracts, and protective config comments aligned with Quality Health.
- Keep Tidiness Manager changes focused on maintainability scans, issue workflows, recommendations, and campaigns.
- Use lifecycle commands rather than direct process execution.
- Update docs and requirements whenever API, CLI, persistence, or campaign behavior changes.

## Comparison to Related Scenarios

- **scenario-auditor**: Standards compliance (security, schema) vs. code cleanliness (length, organization)
- **code-smell**: Pattern violations vs. structural issues - complementary, can integrate
- **visited-tracker**: Provides file tracking; tidiness-manager is a consumer

## Notes for Implementers

- Light scanning is the foundation - implement first
- Smart scanning depends on light scanning infrastructure
- Agent API and UI can be developed in parallel
- Auto-campaigns require both scanning engines functional
- Build data auditability in from the start, not retrofitted

---

**Generated from react-vite template** | Last updated: 2025-11-21
