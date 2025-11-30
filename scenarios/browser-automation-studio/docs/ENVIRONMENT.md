# Environment Variables

This document lists all environment variables used by the Browser Automation Studio scenario.

## Required Variables

These variables MUST be set for the scenario to function. There are NO fallback values - the services will fail to start if these are missing.

### Core Service Ports
- `API_PORT` - Port for the API server (range: 20000-24999)
- `UI_PORT` - Port for the UI server (range: 40000-44999)  
- `WS_PORT` - Port for WebSocket connections (range: 25000-29999)

### Resource Configuration

#### PostgreSQL
- `DATABASE_URL` - Full PostgreSQL connection string

#### MinIO Storage
- `MINIO_PORT` - MinIO service port
- `MINIO_ACCESS_KEY` - MinIO access key credential
- `MINIO_SECRET_KEY` - MinIO secret key credential
- `MINIO_BUCKET_NAME` - Bucket name for screenshot storage

#### Browserless
- `BROWSERLESS_PORT` - Browserless service port

## Optional Variables

These variables have sensible defaults but can be overridden for different deployment scenarios.

### Host Configuration
- `API_HOST` - API server host (default: `localhost`)
- `UI_HOST` - UI server host (default: `localhost`)
- `WS_HOST` - WebSocket server host (default: `localhost`)
- `MINIO_HOST` - MinIO server host (default: `localhost`)
- `BROWSERLESS_HOST` - Browserless server host (default: `localhost`)

### CORS Configuration
- `CORS_ALLOWED_ORIGINS` - Comma-separated list of allowed origins (supports `*` for development)
- `ALLOWED_ORIGINS` - Legacy alias for `CORS_ALLOWED_ORIGINS`
- `CORS_ALLOWED_ORIGIN` - Legacy single-origin setting (still honored)

When no CORS variables are provided, the API and UI automatically allow requests from the lifecycle-managed UI port,
`https://app-monitor.itsagitime.com`, and sandboxed iframe contexts (`Origin: null`).

### Screenshot Configuration
- `SCREENSHOT_DEFAULT_WIDTH` - Default width for screenshots when not detected
- `SCREENSHOT_DEFAULT_HEIGHT` - Default height for screenshots when not detected

### Alternative URL Configuration
- `BROWSERLESS_URL` - Full Browserless URL (overrides host/port)
- `MINIO_ENDPOINT` - Full MinIO endpoint (overrides host/port)
- `BROWSER_AUTOMATION_API_URL` - Full API URL for CLI (overrides host/port)
- `BAS_EXPORT_PAGE_URL` - Absolute URL to the replay composer page used by the Browserless renderer. Derived automatically from `BAS_UI_BASE_URL`/`UI_*` when present, but override it when hosting the UI under a custom domain or path.

## Lifecycle Variables

These are set automatically by the Vrooli lifecycle system:

- `VROOLI_LIFECYCLE_MANAGED` - Must be "true" for services to start
- `LOG_LEVEL` - Logging level (debug, info, warn, error)

## Example Configuration

```bash
# Required - set by lifecycle system
export API_PORT=20100
export UI_PORT=40100
export WS_PORT=25100
export DATABASE_URL="postgresql://user:pass@localhost:5432/browser_automation"
export MINIO_PORT=9000
export MINIO_ACCESS_KEY="minioadmin"
export MINIO_SECRET_KEY="minioadmin"
export MINIO_BUCKET_NAME="browser-automation-screenshots"
export BROWSERLESS_PORT=3000
export VROOLI_LIFECYCLE_MANAGED="true"

# Optional - for containerized deployment
export API_HOST="api.browser-automation.local"
export UI_HOST="ui.browser-automation.local"
export ALLOWED_ORIGINS="https://app.example.com,https://dashboard.example.com"
```

## Timeout Hierarchy

Understanding the timeout relationships between components is critical for debugging "context deadline exceeded" errors and ensuring long-running operations complete successfully.

### Timeout Stack (outer to inner)

```
┌─────────────────────────────────────────────────────────────┐
│ Go API HTTP Client Timeout: 5 minutes                      │
│   (playwright_engine.go:48)                                 │
│   ┌─────────────────────────────────────────────────────┐   │
│   │ Playwright Driver Request Timeout: 5 minutes       │   │
│   │   (playwright-driver/src/config.ts:7)              │   │
│   │   ┌─────────────────────────────────────────────┐   │   │
│   │   │ Workflow Execution Timeout:                 │   │   │
│   │   │   - 90s for simple workflows (default)      │   │   │
│   │   │   - 120s for workflows with subflows        │   │   │
│   │   │   - Configurable via executionTimeoutMs     │   │   │
│   │   │   (simple_executor.go:918-935)              │   │   │
│   │   │   ┌─────────────────────────────────────┐   │   │   │
│   │   │   │ Step Timeout: per-instruction      │   │   │   │
│   │   │   │   - Uses workflow execution timeout│   │   │   │
│   │   │   │   - Plus 2s HTTP buffer            │   │   │   │
│   │   │   │   (simple_executor.go:783-784)     │   │   │   │
│   │   │   └─────────────────────────────────────┘   │   │   │
│   │   └─────────────────────────────────────────────┘   │   │
│   └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Key Timeout Values

| Component | Timeout | Location | Notes |
|-----------|---------|----------|-------|
| Go HTTP Client | 5 min | `playwright_engine.go:48` | Outer bound for all driver calls |
| Driver Request | 5 min | `config.ts:7` | Must be <= Go HTTP timeout |
| Workflow Execution | 90-120s | `simple_executor.go:918-935` | Configurable via metadata |
| Step + HTTP Buffer | timeout + 2s | `simple_executor.go:783-784` | Prevents network overhead from causing step timeouts |
| Health Check | 5s | `main.go:performStartupHealthCheck` | Startup-only |

### Best Practices

1. **Inner timeouts should be smaller than outer timeouts** to ensure errors are reported correctly:
   - Workflow timeout < Driver timeout < HTTP Client timeout

2. **Use `executionTimeoutMs` in workflow metadata** to customize timeout for long-running workflows:
   ```json
   {
     "metadata": {
       "executionTimeoutMs": 180000
     }
   }
   ```

3. **Debug "context deadline exceeded"**:
   - If from Go API: Check workflow timeout vs step complexity
   - If from driver: Check Playwright operation timeout vs page load times
   - If from HTTP layer: Network issues between API and driver

## Important Notes

1. **No Fallbacks**: This scenario follows a fail-fast philosophy. If required environment variables are missing, services will exit with clear error messages rather than using fallback values.

2. **Port Ranges**: The service.json defines specific port ranges for each service to avoid conflicts with other scenarios.

3. **Host Variables**: The `*_HOST` variables allow the scenario to work in containerized environments where services may not be on localhost.

4. **Lifecycle Management**: This scenario MUST be run through the Vrooli lifecycle system which sets up the required environment variables automatically.
