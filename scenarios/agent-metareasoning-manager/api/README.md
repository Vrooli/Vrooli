# Agent Metareasoning Manager API

A lightweight Go API for discovering and coordinating AI reasoning workflows across n8n and Windmill platforms.

## Architecture

This is a **simplified coordinator** that:
- Discovers workflows from n8n and Windmill at runtime
- Provides a unified interface for workflow execution
- Tracks minimal metadata for search and metrics
- Acts as a thin routing layer, not a storage system

## Quick Start

```bash
# Build
go build -o agent-metareasoning-manager-api cmd/server/main.go

# Or use Make
make build

# Run
./agent-metareasoning-manager-api
```

## Environment Variables

- `PORT` - API port (default: 8090)
- `N8N_BASE_URL` - n8n instance URL (default: http://localhost:5678)
- `WINDMILL_BASE_URL` - Windmill instance URL (default: http://localhost:5681)
- `DATABASE_URL` - PostgreSQL connection string

## API Endpoints

- `GET /health` - Health check
- `GET /workflows` - List discovered workflows
- `POST /workflows/search` - Search workflows by query
- `POST /execute/{platform}/{workflowId}` - Execute workflow on platform

## Design Philosophy

This API follows the principle of **discovery over storage**:
- Workflows live in their native platforms (n8n/Windmill)
- API discovers available workflows at startup
- No duplication or synchronization issues
- Always uses the latest workflow version

## Files

- `cmd/server/main_simplified.go` - Complete API implementation (400 lines)
- `go.mod` / `go.sum` - Dependencies
- `Makefile` - Build commands
- `main.go` - Compatibility wrapper

That's it! No complex architecture, just simple coordination.