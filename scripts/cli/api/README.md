# Vrooli App Management API

A minimal HTTP API server for managing generated Vrooli apps. Inspired by the clean architecture of agent-metareasoning-manager.

## Architecture

```
app-commands.sh (CLI) --HTTP--> main.go (API) --FileSystem--> ~/generated-apps/
```

- **main.go** (261 lines): HTTP API server using gorilla/mux
- **app-commands.sh** (406 lines): CLI wrapper that calls the API
- **Total**: 700 lines (vs 1700+ lines in the previous approach)

## API Endpoints

- `GET /health` - Health check
- `GET /apps` - List all apps with status
- `GET /apps/{name}` - Get specific app details
- `POST /apps/{name}/protect` - Protect app from regeneration
- `DELETE /apps/{name}/protect` - Remove protection
- `POST /apps/{name}/backup` - Create backup
- `POST /apps/{name}/restore` - Restore from backup

## Running

```bash
# Start the API server (default port 8094)
./start-api.sh

# Or run directly
go run main.go

# Custom port
PORT=8095 go run main.go
```

## CLI Usage

The CLI automatically checks if the API is running and provides instructions if not:

```bash
vrooli app list        # List all apps
vrooli app status foo  # Show app details
vrooli app protect foo # Protect from regeneration
```

## Design Principles

1. **Minimal surface area** - Only essential operations
2. **Clean separation** - API handles logic, CLI handles display
3. **Standard HTTP/JSON** - Easy to integrate with other tools
4. **No binary artifacts** - Run directly with `go run`