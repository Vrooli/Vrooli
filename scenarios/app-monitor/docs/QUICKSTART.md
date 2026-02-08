# App Monitor Quick Start

## Run Locally
1. `cd scenarios/app-monitor`
2. `make start`
3. Open the scenario URL printed by lifecycle output.

## Core Views
- Single preview: `/apps/:appId/preview`
- Workspace view: `/apps/workspace`

## Test Before Changes
1. `make test`
2. For focused UI checks: `cd ui && pnpm test`

## Stop Scenario
- `make stop`
