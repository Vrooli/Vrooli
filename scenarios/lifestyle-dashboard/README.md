# Lifestyle Dashboard

> Unified personal lifestyle intelligence dashboard — the foundation for cross-domain health insights.

## Overview

The Lifestyle Dashboard is the **foundation scenario** for a personal lifestyle intelligence system. It provides:

- **Shared event schema & storage** — SQLite-based time-series events with JSON payloads
- **Domain registration & discovery** — Dynamic registration for domain-specific scenarios
- **Cross-domain correlation engine** — Statistical analysis across health domains
- **Unified analytics UI** — Mobile-first dashboard with daily briefs and Lifestyle Score
- **Automation-first philosophy** — Everything runs hands-off by default

**Target:** Single user (Matt) on personal Vrooli server. No multi-tenant, no auth, no SaaS complexity.

**Value proposition:** Cross-domain health insights that no single-domain app can provide.

## Quick Start

### Prerequisites
- Go 1.22+
- Node.js 18+ with pnpm
- SQLite (embedded, no external server needed)

### Installation
```bash
cd scenarios/lifestyle-dashboard

# Install UI dependencies
corepack pnpm install --dir ui --ignore-workspace

# Build and start via lifecycle
make start
```

### Connecting First Domain
The recommended first domain scenario is **nootropics-tracker**. See [PRD.md](./PRD.md) for domain integration patterns.

## Features

### P0 (Core)
- Unified event storage (SQLite with JSON payloads)
- Domain registration & discovery
- Cross-domain query API
- Dashboard UI (mobile-first)
- Daily briefs (morning 7am / evening 9pm)
- Storage management settings

### P1 (Intelligence)
- Correlation engine with significance tracking
- Composite Lifestyle Score (0-100 daily)
- Weekly digest ("What Changed?" Sundays)
- Experiment framework (N=1 experiments)
- Natural language queries (Ollama-powered)

## Architecture

### Event Schema
All events follow a common envelope:
```sql
CREATE TABLE events (
    id TEXT PRIMARY KEY,           -- UUID
    timestamp TEXT NOT NULL,       -- ISO-8601
    domain TEXT NOT NULL,          -- e.g., "nootropics", "sleep"
    event_type TEXT NOT NULL,      -- e.g., "intake.logged"
    payload TEXT NOT NULL,         -- JSON blob
    is_intervention INTEGER DEFAULT 0,  -- 0 = observation, 1 = intervention
    hypothesis_id TEXT,            -- UUID if part of experiment
    created_at TEXT DEFAULT (datetime('now'))
);
```

### SQLite Storage Rationale
Per accepted suggestion, SQLite was chosen over PostgreSQL/Redis for:
- **Portability** — Single file, easy backup/restore
- **Future deployment** — Works identically on mobile (Capacitor) or desktop (Electron)
- **Simplicity** — No external database server to manage
- **Performance** — Sufficient for single-user with thousands of events

### Domain Integration Contract
- Register via `POST /api/v1/domains/register`
- Emit events to the events table via dashboard API
- Expose `/api/v1/health` for health checks
- Provide structured brief content via `GET /api/v1/brief/contribute`

## Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| Morning brief time | 7:00 AM | When to generate morning brief |
| Evening review time | 9:00 PM | When to generate evening review |
| Correlation threshold | 14 data points | Minimum samples before correlations |
| Lifestyle Score weights | Configurable | Sleep (high), exercise (high), etc. |

## API Reference

See `api/` and `.vrooli/endpoints.json` for full documentation. Key endpoints:

### Events (P0-001, P0-003)
- `POST /api/v1/events` — Create event with JSON payload
- `GET /api/v1/events?domain=&start=&end=&limit=` — Query events with filters
- `GET /api/v1/events/{id}` — Get specific event

### Domains (P0-002)
- `POST /api/v1/domains` — Register a domain scenario
- `GET /api/v1/domains` — List registered domains
- `GET /api/v1/domains/{name}` — Get domain details
- `PATCH /api/v1/domains/{name}` — Update domain status
- `GET /api/v1/domains/{name}/health` — Check domain health

### Statistics (P0-003, P0-004)
- `GET /api/v1/stats/timeline?days=7` — Activity by day/domain
- `GET /api/v1/stats/summary` — Dashboard summary stats

## Related Scenarios

| Phase | Scenario | Notes |
|-------|----------|-------|
| 1 | nootropics-tracker | First to build — lowest friction, no hardware |
| 1 | sleep-tracker | Blocked on wearable purchase |
| 2 | diet-nutrition | Optimizes existing habits |
| 2 | exercise-planner | Structures existing activity |
| 3 | skincare-manager, biomarkers, meditation, learning, socialization | Lower urgency |

## Development

```bash
# Start development servers
make dev

# Run tests
make test

# Build for production
make build
```

See [PRD.md](./PRD.md) for operational targets and [requirements/](./requirements/) for detailed specifications.

## Environment Variables

The lifecycle exports everything automatically when you run `vrooli scenario run`. Key variables:

| Variable | Purpose |
|----------|---------|
| `API_PORT` | Port assigned to the Go API server |
| `UI_PORT` | Port assigned to the Vite dev server / production UI |
| `SQLITE_PATH` | Path to SQLite database file (default: `lifestyle.db`) |
| `VITE_API_BASE_URL` | UI → API bridge (set to `http://localhost:${API_PORT}/api/v1`) |

## CLI Usage

The control plane installs the declared Go CLI when the scenario starts. Commands:
- `lifestyle-dashboard status` — Show dashboard and domain health
- `lifestyle-dashboard events list` — Query recent events
- `lifestyle-dashboard brief morning` — Get morning brief
