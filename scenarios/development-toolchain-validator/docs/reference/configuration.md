# Configuration

This document describes all tunable levers (configuration options) for the Development Toolchain Validator. Configuration is organized into coherent groups that reflect how operators think about tuning behavior.

## Control Surface Overview

| Group | Lever | Environment Variable | Default | Impact |
|-------|-------|---------------------|---------|--------|
| Server | API Port | `API_PORT` | dynamic | Port for Go API server |
| Server | UI Port | `UI_PORT` | dynamic | Port for React UI |
| Server | CORS Origins | `CORS_ALLOWED_ORIGINS` | localhost | Origins allowed for API requests |
| Data | Database URL | `DATABASE_URL` | - | PostgreSQL connection |
| Pagination | Default Limit | `DTV_PAGINATION_DEFAULT_LIMIT` | 20 | Results per page when unspecified |
| Pagination | Max Limit | `DTV_PAGINATION_MAX_LIMIT` | 100 | Hard cap on results per page |
| Validation | Slug Min Length | `DTV_SLUG_MIN_LENGTH` | 2 | Minimum reference slug length |
| Validation | Slug Max Length | `DTV_SLUG_MAX_LENGTH` | 100 | Maximum reference slug length |
| Integration | prompt-manager URL | `PROMPT_MANAGER_API_URL` | auto-detect | External API integration |
| CLI | Default Timeout | `DTV_CLI_TIMEOUT_DEFAULT` | 60 | CLI tool assertion timeout |

## Environment Variables

### Required Variables

| Variable | Description |
|----------|-------------|
| `API_PORT` | Port for the Go API server (range 15000-19999) |
| `UI_PORT` | Port for the React UI (range 35000-39999) |
| `DATABASE_URL` | PostgreSQL connection string |
| `PROMPT_MANAGER_API_URL` | URL of prompt-manager's API (e.g., `http://localhost:18XXX/api/v1`) |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VROOLI_PROJECT_ROOT` | auto-detected | Override for Vrooli project root path |
| `DTV_CLI_TIMEOUT_DEFAULT` | 60 | Default timeout for CLI tool assertions in seconds |

## API Configuration

### Pagination Levers

These levers control list endpoint behavior. Higher limits increase response payload size but reduce API calls needed to retrieve all results.

| Variable | Default | Range | Description |
|----------|---------|-------|-------------|
| `DTV_PAGINATION_DEFAULT_LIMIT` | 20 | 1-MaxLimit | Applied when no limit is specified or limit is invalid |
| `DTV_PAGINATION_MAX_LIMIT` | 100 | 1-1000 | Upper bound for requested limits; prevents resource exhaustion |

**Tradeoff**: Higher default/max limits mean fewer API calls but larger response payloads. For mobile clients or slow networks, lower values may be preferable.

**Example**:
```bash
# Production with higher throughput needs
export DTV_PAGINATION_DEFAULT_LIMIT=50
export DTV_PAGINATION_MAX_LIMIT=200

# Constrained environment
export DTV_PAGINATION_DEFAULT_LIMIT=10
export DTV_PAGINATION_MAX_LIMIT=25
```

### Validation Levers

These levers control what input values are accepted by the API.

| Variable | Default | Range | Description |
|----------|---------|-------|-------------|
| `DTV_SLUG_MIN_LENGTH` | 2 | 1-SlugMaxLength | Minimum allowed slug length |
| `DTV_SLUG_MAX_LENGTH` | 100 | 1-255 | Maximum allowed slug length (capped by DB VARCHAR) |

**Why constrained**: Slug constraints ensure URL-friendly identifiers. The regex pattern `^[a-z0-9][a-z0-9-]*[a-z0-9]$` is fixed and cannot be changed via configuration to maintain cross-system consistency.

### CORS Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CORS_ALLOWED_ORIGINS` | localhost variants | Comma-separated list of allowed origins |

**Default origins** (for development):
- `http://localhost:3000`
- `http://localhost:5173`
- `http://127.0.0.1:3000`
- `http://127.0.0.1:5173`

**Production example**:
```bash
export CORS_ALLOWED_ORIGINS="https://app.example.com,https://admin.example.com"
```

## PostgreSQL

DTV uses PostgreSQL for all persistent data. The schema is initialized via migration files in `initialization/postgres/`.

Database name: `development-toolchain-validator` (configurable via `DATABASE_URL`).

## prompt-manager Integration

DTV connects to prompt-manager's API to:
- Fetch skill metadata, content, and version history
- Check for skill drift via content hashes

The API URL must be configured. DTV will attempt to auto-detect prompt-manager's port using `vrooli scenario port prompt-manager API_PORT` if `PROMPT_MANAGER_API_URL` is not set.

## CLI Configuration

The CLI stores configuration at:
- Linux: `~/.config/vrooli/development-toolchain-validator/config.json`
- Fallback: `~/.vrooli/config/development-toolchain-validator/config.json`

Configuration fields:
```json
{
  "api_base": "http://localhost:18XXX/api/v1",
  "token": ""
}
```

Set via:
```bash
development-toolchain-validator configure api_base <url>
development-toolchain-validator configure token <value>
```

## Assertion Timeout Configuration

Default timeouts can be overridden per-assertion when creating CLI tool expectations:

```bash
development-toolchain-validator expectations add cli-tool \
  --connection api-steer:reference-react-vite \
  --command "test-genie execute reference-react-vite --preset comprehensive --json" \
  --path "$.success" --op eq --value true \
  --timeout 900 \
  --description "Full test suite passes"
```

Recommended timeouts by tool:

| Tool | Recommended | Default |
|------|------------|---------|
| scenario-auditor | 240s | 60s |
| test-genie (comprehensive) | 900s | 60s |
| test-genie (quick) | 120s | 60s |
| scenario-completeness-scoring | 120s | 60s |
| knowledge-observatory | 60s | 60s |

## Service Configuration

The `.vrooli/service.json` file configures lifecycle, ports, and dependencies. See the generated file for the full configuration.

Key sections:
- **ports**: API (15000-19999), UI (35000-39999)
- **lifecycle**: setup (build API/UI/CLI), develop (start servers), test (test-genie), stop
- **dependencies**: PostgreSQL (required)

## What Is NOT Configurable (And Why)

Some values are intentionally **not** exposed as levers:

| Value | Location | Why Not Configurable |
|-------|----------|---------------------|
| Slug regex pattern | `domain/reference/service.go` | Cross-system consistency; changing would break URL routing |
| Health check paths | `.vrooli/service.json` | Infrastructure standard; changing breaks orchestration |
| Database schema | `initialization/postgres/schema.sql` | Migrations should handle changes, not runtime config |
| API version prefix | `handlers/reference.go` | Breaking changes need versioned endpoints, not config |

These represent **conscious architectural decisions** where runtime configurability would introduce more risk than benefit.
