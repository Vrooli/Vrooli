# Quick Start

Get Swarm Manager running in 5 minutes.

## What Swarm Manager Is (Current Model)

Swarm Manager is the execution control plane for scenario changes:
- Prompt Manager teams do research/recommendations and write backlog items (`idea`, `research`, `fix`, `execute`).
- Swarm Manager stores and governs backlog state.
- Agent runs are managed from the **Execution** surface (pending, running, completed, failed).

Swarm Manager does not run a recommendation engine.

## Prerequisites

- Node.js 18+ (UI development)
- Go 1.21+ (API/CLI development)
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
- API at `http://localhost:15000`
- UI at `http://localhost:35000`

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

Open `http://localhost:35000`.

You should see the **Plan board** (`/plan`) — a single lens-driven workspace over
backlog, scenarios, initiatives, and execution runs. The old standalone
Scenarios and Execution list tabs were absorbed into it (their routes redirect to
`/plan`); scenario and execution **detail** pages remain directly reachable. The
`/graph` and `/stats` lenses and the `/records` browser round out the surfaces.

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

Use lifecycle-managed commands from the scenario root:

```bash
# Start managed services
make start

# Watch logs while developing
make logs
```

## CLI Usage

```bash
# Check API health
swarm-manager status

# Configure API URL (if not auto-detected)
swarm-manager configure

# List execution runs
swarm-manager execution list
```

## Next Steps

- Read [Architecture](concepts/ARCHITECTURE.md)
- Check [Configuration](reference/configuration.md)
- Review [PRD](../PRD.md)

## Troubleshooting

### API won't start

1. Check logs: `make logs`
2. Verify port is open: `lsof -i :15000`

### UI shows connection errors

1. Verify API is running: `curl http://localhost:15000/api/v1/health`
2. Check browser console for CORS errors
3. Ensure `VITE_API_BASE_URL` is correct

### CLI can't find API

1. Run `swarm-manager configure`
2. Or set `SWARM_MANAGER_API_BASE`

For known issues, see [internal/PROBLEMS.md](internal/PROBLEMS.md).
