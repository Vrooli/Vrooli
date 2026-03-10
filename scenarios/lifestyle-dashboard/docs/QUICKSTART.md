# Quick Start

Get the Lifestyle Dashboard running in 5 minutes.

## Prerequisites

- Go 1.22+ ([CODE: api/go.mod])
- Node.js 18+ with pnpm
- SQLite (embedded, no external server needed)

## Installation

```bash
cd scenarios/lifestyle-dashboard

# Start via Makefile (recommended)
make start

# Or via Vrooli CLI
vrooli scenario start lifestyle-dashboard
```

## Verify Installation

After startup, you should see:
- API: `http://localhost:${API_PORT}/health` returns healthy status
- UI: `http://localhost:${UI_PORT}` shows the dashboard

Check health:
```bash
curl http://localhost:$(vrooli scenario port lifestyle-dashboard API_PORT)/health
```

## First Steps

### 1. View the Dashboard

Open the UI in your browser. You'll see:
- **Timeline chart** - Events by day and domain
- **Domain cards** - Registered health/wellness scenarios
- **Event feed** - Recent events across all domains

### 2. Register a Domain

Domains are health/wellness scenarios that integrate with the dashboard:

```bash
curl -X POST http://localhost:${API_PORT}/api/v1/domains \
  -H "Content-Type: application/json" \
  -d '{
    "name": "nutrition",
    "display_name": "Nutrition Tracker",
    "capabilities": ["meal_logging", "calorie_tracking"],
    "health_url": "http://localhost:3001/health"
  }'
```

See [CODE: api/handlers/domains.go#RegisterDomain] for the registration handler.

### 3. Create an Event

Events are the core data unit - time-series data points from domains:

```bash
curl -X POST http://localhost:${API_PORT}/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "nutrition",
    "event_type": "meal.logged",
    "payload": {"meal": "breakfast", "calories": 450}
  }'
```

See [CODE: api/handlers/events.go#CreateEvent] for the event handler.

### 4. Query Data

Retrieve events with filtering:

```bash
# All events from last 7 days
curl "http://localhost:${API_PORT}/api/v1/events?limit=100"

# Events from specific domain
curl "http://localhost:${API_PORT}/api/v1/events?domain=nutrition"

# Timeline statistics
curl "http://localhost:${API_PORT}/api/v1/stats/timeline?days=7"
```

## Architecture Overview

```
UI (React/Vite) → API (Go/Gorilla) → SQLite
     pages/           handlers/         lifestyle.db
     components/      repository/
     lib/api.ts       domain/
```

For detailed architecture, see [concepts/ARCHITECTURE.md](concepts/ARCHITECTURE.md).

## Next Steps

- [Architecture](concepts/ARCHITECTURE.md) - Understand the system design
- [Storage Audit](internal/STORAGE_AUDIT.md) - Storage architecture decisions
- [PRD](../PRD.md) - Full product requirements
