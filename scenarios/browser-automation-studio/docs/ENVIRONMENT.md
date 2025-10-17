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
- `ALLOWED_ORIGINS` - Comma-separated list of additional allowed origins for CORS

### Screenshot Configuration
- `SCREENSHOT_DEFAULT_WIDTH` - Default width for screenshots when not detected
- `SCREENSHOT_DEFAULT_HEIGHT` - Default height for screenshots when not detected

### Alternative URL Configuration
- `BROWSERLESS_URL` - Full Browserless URL (overrides host/port)
- `MINIO_ENDPOINT` - Full MinIO endpoint (overrides host/port)
- `BROWSER_AUTOMATION_API_URL` - Full API URL for CLI (overrides host/port)

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

## Important Notes

1. **No Fallbacks**: This scenario follows a fail-fast philosophy. If required environment variables are missing, services will exit with clear error messages rather than using fallback values.

2. **Port Ranges**: The service.json defines specific port ranges for each service to avoid conflicts with other scenarios.

3. **Host Variables**: The `*_HOST` variables allow the scenario to work in containerized environments where services may not be on localhost.

4. **Lifecycle Management**: This scenario MUST be run through the Vrooli lifecycle system which sets up the required environment variables automatically.