# Quick Start

Get Brand Manager running and create your first brand in under 5 minutes.

## Prerequisites

- Vrooli CLI installed (`vrooli --version`)
- Brand Manager scenario available (`vrooli scenario status brand-manager`)

## Start the Scenario

```bash
cd scenarios/brand-manager
make start
```

This starts both the Go API server and the React UI. The API listens on the port allocated by Vrooli (check with `vrooli scenario port brand-manager API_PORT`).

## Create a Brand via CLI

```bash
brand-manager create --name "My Scenario Brand" --description "Professional branding for my scenario"
```

## Create a Brand via API

```bash
curl -X POST http://localhost:$(vrooli scenario port brand-manager API_PORT)/api/v1/brands \
  -H "Content-Type: application/json" \
  -d '{"name": "My Scenario Brand", "description": "Professional branding"}'
```

## List Brands

```bash
# CLI
brand-manager list

# API
curl http://localhost:$(vrooli scenario port brand-manager API_PORT)/api/v1/brands
```

## Assign a Brand to a Scenario

```bash
curl -X POST http://localhost:$(vrooli scenario port brand-manager API_PORT)/api/v1/assignments \
  -H "Content-Type: application/json" \
  -d '{"brand_id": "<brand-id>", "scenario_name": "my-scenario"}'
```

## Check Scenario Branding Status

```bash
curl http://localhost:$(vrooli scenario port brand-manager API_PORT)/api/v1/scenarios/my-scenario/status
```

## What's Next

- See [API Reference](reference/api-endpoints.md) for all endpoints
- See [CLI Reference](reference/cli-commands.md) for all commands
- See [Architecture](concepts/ARCHITECTURE.md) for system design
