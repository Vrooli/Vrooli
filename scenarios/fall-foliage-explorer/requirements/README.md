# Requirements

The requirement registry is [CODE: requirements/index.json]. It maps the PRD operational targets in [DOC: ../PRD.md#operational-targets] to requirement IDs used by tests and documentation.

## Current Modules

- `01-core-viability/module.json` groups P0 health, database, API, weather, prediction, map UI, and lifecycle requirements.
- `02-user-engagement/module.json` groups P1 reports, time slider, trip planning, and photo gallery requirements.
- `03-expansion-readiness/module.json` groups P2 AI prediction, mobile responsive, and export requirements.

## Test Traceability

Existing Go tests include `[REQ:...]` tags for API, CLI, business, integration, and performance coverage. Full scenario test runs synchronize requirement coverage automatically.
