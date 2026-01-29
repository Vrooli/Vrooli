# Quick Start

Get Swarm Manager running in 5 minutes.

## Prerequisites

- Node.js 18+ (for UI development)
- Go 1.21+ (for API/CLI development)
- Vrooli CLI installed (`vrooli` command available)

## Start the Scenario

```bash
# Navigate to the scenario
cd scenarios/swarm-manager

# Full setup (build API, CLI, UI, start dependencies)
make setup

# Start development servers
make start
```

This starts:
- **API** at `http://localhost:15000` (port from service.json)
- **UI** at `http://localhost:35000` (port from service.json)

## Verify It's Working

```bash
# Check API health
curl http://localhost:15000/api/v1/health

# Or use the CLI
swarm-manager status
```

Expected output:
```json
{
  "status": "healthy",
  "readiness": true,
  "version": "1.0.0"
}
```

## Access the UI

Open http://localhost:35000 in your browser. You'll see:

1. **Backlog** tab - Manage research, ideas, fixes, and execution tasks
2. **Scenarios** tab - View scenario catalog and status
3. **Recommendations** tab - Review system suggestions
4. **Settings** tab - Configure behavior

## Common Commands

```bash
# View logs
make logs

# Run tests
make test

# Stop all services
make stop

# Rebuild after changes
make setup
```

## Development Mode

For active development with hot reload:

```bash
# Terminal 1: API (auto-rebuilds on changes)
cd api && go run .

# Terminal 2: UI (hot module replacement)
cd ui && npm run dev
```

## CLI Usage

```bash
# Check API health
swarm-manager status

# Configure API URL (if not auto-detected)
swarm-manager configure
```

## Next Steps

- Read [Architecture](concepts/ARCHITECTURE.md) for the mental model
- Check [Configuration](reference/configuration.md) for tunable settings
- Review [PRD.md](../PRD.md) for product requirements

## Troubleshooting

### API won't start

1. Check logs: `make logs`
2. Verify port 15000 is available: `lsof -i :15000`

### UI shows connection errors

1. Verify API is running: `curl http://localhost:15000/health`
2. Check browser console for CORS errors
3. Ensure `VITE_API_BASE_URL` is set correctly

### CLI can't find API

1. Run `swarm-manager configure` to set the API URL
2. Or set `SWARM_MANAGER_API_BASE` environment variable

For more help, see [docs/PROBLEMS.md](PROBLEMS.md) for known issues.
