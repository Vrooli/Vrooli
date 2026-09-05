# Quick Start

## Setup

From the scenario root:

```bash
make setup
```

This delegates to `vrooli scenario setup fall-foliage-explorer`, builds the Go API, installs UI dependencies, builds the static UI bundle, and populates required resources through the lifecycle configuration in [CODE: .vrooli/service.json].

## Run

```bash
make start
make status
make logs
```

The lifecycle system assigns API and UI ports from the scenario ranges in [DOC: docs/reference/configuration.md#ports]. The API exposes `GET /health`; the UI exposes `GET /health` through [CODE: ui/server.js].

## Validate

```bash
vrooli scenario status fall-foliage-explorer
scenario-completeness-scoring score get fall-foliage-explorer --json
scenario-auditor audit fall-foliage-explorer --timeout 240
vrooli scenario test fall-foliage-explorer all
vrooli scenario ui-smoke fall-foliage-explorer
```

For faster iteration, run phase-specific checks such as `vrooli scenario test fall-foliage-explorer structure` or `vrooli scenario test fall-foliage-explorer unit`.

## Stop

```bash
make stop
```

Shutdown is owned by the lifecycle stop steps in [CODE: .vrooli/service.json].
