# Configuration

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `API_PORT` | Yes | Port for the Go API server (range 15000-19999) |
| `UI_PORT` | Yes | Port for the React UI (range 35000-39999) |
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `PROMPT_MANAGER_API_URL` | Yes | URL of prompt-manager's API (e.g., `http://localhost:18XXX/api/v1`) |
| `VROOLI_PROJECT_ROOT` | No | Override for Vrooli project root path (auto-detected by default) |
| `DTV_CLI_TIMEOUT_DEFAULT` | No | Default timeout for CLI tool assertions in seconds (default: 60) |

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
