# Quick Start

Get the reference-react-vite scenario running in minutes.

## Prerequisites

- Docker (for PostgreSQL)
- Go 1.21+
- Node.js 18+
- Vrooli CLI (`vrooli` command available)

## Start the Scenario

```bash
# From the repository root
cd scenarios/reference-react-vite && make start

# Or using vrooli CLI
vrooli scenario start reference-react-vite
```

This will:
1. Start the PostgreSQL database
2. Run schema initialization
3. Start the Go API on port 15000
4. Start the Vite UI on port 35000

## Verify It's Running

```bash
# Check scenario status
vrooli scenario status reference-react-vite

# Health check
curl http://localhost:15000/health
```

## Access the Application

- **UI**: http://localhost:35000
- **API**: http://localhost:15000/api/v1

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check with dependency status |
| GET | `/api/v1/projects` | List all projects |
| POST | `/api/v1/projects` | Create a project |
| GET | `/api/v1/tasks` | List all tasks |
| POST | `/api/v1/tasks` | Create a task |
| GET | `/api/v1/tasks/{id}/notes` | List notes for a task |
| POST | `/api/v1/tasks/{id}/notes` | Add a note to a task |

See [API Reference](reference/api-endpoints.md) for full documentation.

## Stop the Scenario

```bash
cd scenarios/reference-react-vite && make stop
# Or
vrooli scenario stop reference-react-vite
```

## Run Tests

```bash
# All tests
vrooli scenario test reference-react-vite all

# Specific test phases
vrooli scenario test reference-react-vite unit
vrooli scenario test reference-react-vite integration
```

## Useful Commands

```bash
# View logs
make logs

# Rebuild and restart
make restart

# Run linting
make lint

# Format code
make fmt
```

## Next Steps

- Read the [Architecture](concepts/ARCHITECTURE.md) to understand the domain model
- Check the [Configuration](reference/configuration.md) for environment variables
- See [internal/SEAMS.md](internal/SEAMS.md) for architectural boundaries
