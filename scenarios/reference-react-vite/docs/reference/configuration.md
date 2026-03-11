# Configuration

Environment variables and service configuration for the reference-react-vite scenario.

## Tunable Levers

The API exposes a small set of meaningful, safe configuration levers that shape its behavior.
All levers have sensible defaults designed for development and production use.

[CODE: api/config/config.go]

### Pagination

Controls how list endpoints return results across tasks, projects, and notes.

| Variable | Default | Range | Description |
|----------|---------|-------|-------------|
| PAGINATION_DEFAULT_LIMIT | 20 | 1-MaxLimit | Items returned when no limit specified |
| PAGINATION_MAX_LIMIT | 100 | 1-1000 | Maximum allowed items per request |

**Tradeoffs:**
- Higher limits = more data at once, larger responses, potential memory pressure
- Lower limits = more requests needed, smaller responses, better for mobile

### Validation

Controls input validation constraints for user-provided data.

| Variable | Default | Description |
|----------|---------|-------------|
| VALIDATION_TASK_TITLE_MAX | 255 | Maximum characters for task titles |
| VALIDATION_PROJECT_NAME_MAX | 100 | Maximum characters for project names |
| VALIDATION_NOTE_CONTENT_MAX | 10000 | Maximum characters for note content |

**Why these defaults:**
- Task titles: 255 chars balances expressiveness with UI display
- Project names: 100 chars keeps navigation compact
- Note content: 10KB allows substantial notes without enabling abuse

### CORS

Controls Cross-Origin Resource Sharing behavior.

| Variable | Default | Description |
|----------|---------|-------------|
| CORS_ALLOWED_ORIGINS | http://localhost:* | Comma-separated allowed origins |
| CORS_MAX_AGE | 86400 | Preflight cache duration (seconds) |

**Patterns:**
- `http://localhost:*` matches any localhost port (development)
- `https://example.com` exact origin match (production)
- `*` allows all origins (not recommended)

### Server

Server-level settings.

| Variable | Default | Description |
|----------|---------|-------------|
| HEALTH_VERSION | 1.0.0 | Version string in health check responses |

---

## Environment Variables

### API Server

| Variable | Default | Description |
|----------|---------|-------------|
| API_PORT | 15000 | HTTP server port |
| CORS_ALLOWED_ORIGINS | http://localhost:* | Comma-separated allowed origins |
| LOG_LEVEL | info | Logging verbosity |

### Database (PostgreSQL)

[CODE: api/main.go] (see `api.GetPostgresConnectionString()`)

These are injected by the Vrooli resource system:

| Variable | Default | Description |
|----------|---------|-------------|
| POSTGRES_HOST | localhost | Database hostname |
| POSTGRES_PORT | 5433 | Database port (Vrooli default) |
| POSTGRES_USER | vrooli | Database user |
| POSTGRES_PASSWORD | (generated) | Database password |
| POSTGRES_DB | reference-react-vite | Database name |
| DATABASE_URL | (constructed) | Full connection string |

### UI (Vite)

| Variable | Default | Description |
|----------|---------|-------------|
| VITE_API_URL | http://localhost:15000 | API base URL for UI |

## Service Configuration

### .vrooli/service.json
[CODE: .vrooli/service.json]

Defines scenario metadata, ports, lifecycle, and dependencies:

```json
{
  "profile": {
    "name": "Reference React Vite",
    "description": "Golden reference scenario...",
    "tags": ["react-ui", "go-api"]
  },
  "ports": {
    "api": 15000,
    "ui": 35000
  },
  "lifecycle": {
    "setup": [...],
    "develop": [...],
    "test": [...],
    "stop": [...]
  },
  "dependencies": {
    "resources": {
      "postgres": {
        "enabled": true,
        "required": true,
        "schema": "reference-react-vite"
      }
    }
  }
}
```

### Port Allocation

| Service | Port | Range |
|---------|------|-------|
| API | 15000 | 15000-19999 |
| UI | 35000 | 35000-39999 |

Port ranges follow Vrooli conventions to avoid conflicts between scenarios.

## Lifecycle Commands

### Setup

Initializes the scenario environment:

```bash
make setup
# Or
vrooli scenario setup reference-react-vite
```

Steps:
1. Install Go dependencies
2. Install Node dependencies
3. Build CLI
4. Run database migrations

### Develop

Starts development servers:

```bash
make start
# Or
vrooli scenario start reference-react-vite
```

Steps:
1. Start API server (with auto-rebuild)
2. Start Vite dev server

### Test

Runs test suites:

```bash
make test
# Or
vrooli scenario test reference-react-vite all
```

### Stop

Stops all processes:

```bash
make stop
# Or
vrooli scenario stop reference-react-vite
```

## Health Checks

### .vrooli/health.json

Defines health check endpoints:

```json
{
  "checks": [
    {
      "name": "api",
      "type": "http",
      "url": "http://localhost:15000/health",
      "critical": true
    },
    {
      "name": "ui",
      "type": "http",
      "url": "http://localhost:35000/health"
    }
  ]
}
```

## CORS Configuration

The API accepts cross-origin requests based on `CORS_ALLOWED_ORIGINS`:

[CODE: api/main.go] (see `corsMiddleware()`)

```bash
# Development (default)
CORS_ALLOWED_ORIGINS="http://localhost:*"

# Production
CORS_ALLOWED_ORIGINS="https://app.example.com,https://admin.example.com"

# Allow all (not recommended)
CORS_ALLOWED_ORIGINS="*"
```

The `*` wildcard in port position (e.g., `http://localhost:*`) matches any port on that host.

## What Is NOT Exposed (And Why)

Some internal behaviors are intentionally kept fixed:

| Internal | Value | Rationale |
|----------|-------|-----------|
| Default task priority | Medium (2) | Standard UX expectation |
| New task status | Pending | Logical initial state |
| New project status | Active | Projects start active by definition |
| Allowed methods | GET, POST, PATCH, DELETE, OPTIONS | Standard REST verbs |
| Content-Type | application/json | API is JSON-only |

These are not levers because changing them would violate semantic expectations
or introduce unnecessary configuration complexity.

## Related

- [API Reference](api-endpoints.md) - Endpoint documentation
- [Data Model](data-model.md) - Database schema
- [QUICKSTART](../QUICKSTART.md) - Getting started
