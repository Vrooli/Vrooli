# Integrations


This document records resource and service dependencies, their failure modes, and ownership.

## Dependency Inventory

| Dependency | Manifest status | Runtime contract |
|---|---|---|
| PostgreSQL | Required (`must_start`) | Stores regions, observations, weather data, predictions, reports, and trips through the shared database helper. |
| Ollama | Required (`must_start`) | Predictions use `resource-ollama gateway generate --role summarize.default`; failures use the typical-week prediction fallback. |
| Qdrant | Required (`must_start`) | Declared in `.vrooli/service.json` and refreshed by the lifecycle setup step. No runtime Qdrant client was found in the current API implementation. |
| Vrooli lifecycle | Platform contract | Owns API/UI process startup, dependency setup, ports, and proxy metadata. |

The legacy `redis-cache` service tag is not a dependency declaration: Redis is
absent from `dependencies.resources` and has no implementation use. The same is
true of the former optional browser-automation-studio fallback; neither is a
runtime dependency of this scenario today.

## Vrooli Resources

### PostgreSQL

PostgreSQL stores regions, observations, weather data, predictions, reports, and trips. The API connects through the shared Vrooli database helper in [CODE: api/main.go#initDB]. Schema ownership lives in [CODE: api/internal/<domain>/schema.sql].

### Ollama

Ollama powers advanced peak prediction through the resource gateway. The API
calls `resource-ollama gateway generate --role summarize.default`; prediction
failures fall back to the typical peak week. Implementation:
[CODE: api/main.go#generateWithGateway].

### Qdrant

Qdrant is a required resource declaration and the lifecycle refreshes its
embeddings during setup. The current scenario API does not contain a Qdrant
client, so this setup artifact does not establish a runtime query dependency.

## Scenario Dependencies

This scenario declares no scenario-to-scenario runtime dependencies. App
Monitor's iframe bridge is a UI embedding contract, not a scenario peer.

## Third-Party Services

No external third-party production service is required. Weather data is currently modeled as PostgreSQL records.

## App Monitor Iframe Bridge

The UI initializes the iframe bridge when embedded and resolves proxy metadata before falling back to loopback. Implementation: [CODE: ui/src/app.js#resolveApiBase].

## Weather

Weather data is read from PostgreSQL via `GET /api/weather`. The PRD now describes direct API/CLI flows rather than n8n workflows for weather data collection.

## Failure Modes

PostgreSQL failures degrade read-only discovery but block writes. Ollama failures should fall back to typical peak-week prediction. UI proxy metadata failures fall back to loopback and can cause smoke failures when the API is not running.

## Cross-References

- [DOC: docs/reference/configuration.md]
- [DOC: docs/operations/RUNBOOK.md]
